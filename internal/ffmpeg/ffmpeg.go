// Package ffmpeg finds an FFmpeg to play and read what the app's own decoders
// can't: APE, WMA, AIFF, DSD and the rest. FFmpeg is not built into the app,
// so it is looked for beside it, on the PATH, or where MYGO_FFMPEG points.
package ffmpeg

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Env names the FFmpeg to run, before one beside the app or on the PATH.
const Env = "MYGO_FFMPEG"

// ErrMissing says no FFmpeg was found.
var ErrMissing = errors.New("ffmpeg: not installed")

var (
	pathOnce sync.Once
	pathBin  string
)

// Path returns the FFmpeg to run, or "" if there is none.
func Path() string {
	pathOnce.Do(func() { pathBin = find() })
	return pathBin
}

// Available reports whether there is an FFmpeg to run.
func Available() bool { return Path() != "" }

func find() string {
	if p := strings.TrimSpace(os.Getenv(Env)); p != "" {
		if p = resolve(p); p != "" {
			return p
		}
	}
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		for _, d := range []string{
			dir,
			filepath.Join(dir, "bin"),
			// A macOS app keeps what it ships in Resources or Frameworks.
			filepath.Join(dir, "..", "Resources"),
			filepath.Join(dir, "..", "Frameworks"),
		} {
			if p := resolve(filepath.Join(d, name)); p != "" {
				return p
			}
		}
	}
	return resolve(name)
}

// resolve returns where p is a binary runnable as it is, or "" if it isn't.
func resolve(p string) string {
	if strings.ContainsAny(p, `/\`) {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			return ""
		}
		if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
			return "" // there, but not something to run
		}
		return p
	}
	if found, err := exec.LookPath(p); err == nil {
		return found
	}
	return ""
}

// Cmd returns what runs args with the FFmpeg found, for reading by its
// caller: its output to a pipe, or its own.
func Cmd(ctx context.Context, args ...string) *exec.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, Path(), args...)
	hideWindow(cmd)
	return cmd
}

// Info is what FFmpeg reads of a file before playing it.
type Info struct {
	Codec      string
	SampleRate int
	Channels   int
	Duration   time.Duration
}

// Probe reads of a file what it holds, for a library to keep before playing it.
// FFprobe is asked when it is beside FFmpeg, or else FFmpeg itself.
func Probe(path string) (Info, error) {
	bin := Path()
	if bin == "" {
		return Info{}, ErrMissing
	}
	if probe := sibling(bin, "ffprobe"); probe != "" {
		if info, err := probeFile(probe, path); err == nil {
			return info, nil
		}
	}
	return probeFromFFmpeg(bin, path)
}

// probeFile asks ffprobe of a file, one field a line.
func probeFile(probe, path string) (Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, probe,
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_name,sample_rate,channels",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1",
		path)
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return Info{}, err
	}
	if info := parseFields(out); info.Codec != "" || info.SampleRate > 0 || info.Duration > 0 {
		return info, nil
	}
	return Info{}, errors.New("ffmpeg: ffprobe read nothing")
}

// parseFields reads what ffprobe wrote, as key=value lines.
func parseFields(out []byte) Info {
	var info Info
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "codec_name":
			info.Codec = value
		case "sample_rate":
			info.SampleRate, _ = strconv.Atoi(value)
		case "channels":
			info.Channels, _ = strconv.Atoi(value)
		case "duration":
			if secs, err := strconv.ParseFloat(value, 64); err == nil && secs > 0 {
				info.Duration = time.Duration(secs * float64(time.Second))
			}
		}
	}
	return info
}

// probeFromFFmpeg runs FFmpeg on the file with nothing to write, so that it
// says of it in its log what it holds.
func probeFromFFmpeg(bin, path string) (Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-hide_banner", "-i", path)
	hideWindow(cmd)
	out, _ := cmd.CombinedOutput() // said, with nothing to write, it fails
	var info Info
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "Duration:"); ok {
			if d, ok := parseDuration(after); ok && d > 0 {
				info.Duration = d
			}
			continue
		}
		if i := strings.Index(line, "Audio:"); i >= 0 {
			fields := strings.Split(line[i+len("Audio:"):], ",")
			info.Codec = strings.TrimSpace(fields[0])
			for _, f := range fields {
				f = strings.TrimSpace(f)
				if hz, ok := strings.CutSuffix(f, " Hz"); ok {
					info.SampleRate, _ = strconv.Atoi(strings.TrimSpace(hz))
				}
				switch {
				case strings.HasPrefix(f, "mono"):
					info.Channels = 1
				case strings.HasPrefix(f, "stereo"):
					info.Channels = 2
				}
			}
		}
	}
	if info.Duration <= 0 && info.SampleRate <= 0 && info.Codec == "" {
		msg := lastLine(string(out))
		if msg == "" {
			msg = "ffmpeg: read nothing"
		}
		return Info{}, errors.New(msg)
	}
	return info, nil
}

// parseDuration reads the " 00:03:12.34" of a log line as a duration.
func parseDuration(s string) (time.Duration, bool) {
	s, _, _ = strings.Cut(strings.TrimSpace(s), ",")
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 3 {
		return 0, false
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, false
	}
	secs, err3 := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err3 != nil {
		return 0, false
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(secs*float64(time.Second)), true
}

// sibling returns the tool beside bin, of the same name root, or on the PATH,
// or "" if there is none.
func sibling(bin, tool string) string {
	if runtime.GOOS == "windows" {
		tool += ".exe"
	}
	dir := filepath.Dir(bin)
	if p := resolve(filepath.Join(dir, tool)); p != "" {
		return p
	}
	return resolve(tool)
}

// lastLine returns the last line of s that says something, as FFmpeg's
// errors do.
func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := strings.TrimSpace(lines[i]); line != "" {
			return line
		}
	}
	return ""
}
