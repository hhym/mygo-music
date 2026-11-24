package player

import (
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"mygo-music/internal/ffmpeg"
)

// A WMA, which none of the app's decoders opens, plays through FFmpeg, and
// the seek to the middle lands on the music there.
func TestFFmpegPlaysWMA(t *testing.T) {
	_, b := encodeWMA(t)
	// The app's own decoders take a WMA for an MP3, and fail.
	if _, err := openBuiltin(downloaded(b), 5000); err == nil {
		t.Fatal("the app decoded a WMA by itself")
	}
	s, err := openStream(downloaded(b), 5000)
	if err != nil {
		t.Fatal(err)
	}
	fs, ok := s.(*ffmpegStream)
	if !ok {
		t.Fatalf("opened a %T, not FFmpeg's", s)
	}
	defer fs.Close()
	if fs.SampleRate() != ffmpegRate || fs.Channels() != 2 {
		t.Fatalf("decodes %d Hz, %d channels", fs.SampleRate(), fs.Channels())
	}
	ref := decodeAll(t, b)
	if len(ref) < 4*ffmpegRate {
		t.Fatalf("decoded %d samples of a two second song", len(ref))
	}
	if err := fs.SetPosition(ffmpegRate); err != nil {
		t.Fatal(err)
	}
	buf := make([]float32, 4096)
	n, err := fs.Read(buf)
	if err != nil || n == 0 {
		t.Fatalf("read after a seek: %d samples, %v", n, err)
	}
	// A WMA seeks to its own blocks, a few milliseconds wide: the music is
	// the one a second in, within a block of it.
	at, got := align(buf[:n], ref[2*ffmpegRate:])
	if got < 10 || at < -2048 || at > 2048 {
		t.Errorf("the seek to a second plays other music: %.1f dB, %d samples off", got, at)
	}
}

// align returns what frame of ref got starts at, and the signal to noise of it
// there, for a codec that seeks to its blocks rather than to the frame.
func align(got, ref []float32) (int, float64) {
	const span, window = 2048, 1024 // frames to look within, and to compare
	frames := func(a []float32) int { return len(a) / 2 }
	bestAt, best := 0, -1e9
	for at := -span; at <= span; at += 4 {
		var dot, energy float64
		for k := 0; k < window; k++ {
			i, j := k, k+at
			if i >= frames(got) || j < 0 || j >= frames(ref) {
				break
			}
			dot += float64(got[2*i]) * float64(ref[2*j])
			energy += float64(ref[2*j]) * float64(ref[2*j])
		}
		if score := dot / (math.Sqrt(energy) + 1e-9); score > best {
			bestAt, best = at, score
		}
	}
	var sig, noise float64
	for k := 0; k < window; k++ {
		i, j := k, k+bestAt
		if i >= frames(got) || j < 0 || j >= frames(ref) {
			break
		}
		sig += float64(ref[2*j]) * float64(ref[2*j])
		d := float64(got[2*i] - ref[2*j])
		noise += d * d
	}
	return bestAt, 10 * math.Log10(sig/(noise+1e-9))
}

// A song still coming lets FFmpeg go at once when the stream closes, rather
// than waiting on bytes that never come.
func TestFFmpegLetsGoOfASongStillComing(t *testing.T) {
	_, b := encodeWMA(t)
	half := len(b) / 2
	d := &download{data: b[:half], n: int64(half), size: int64(len(b)), cancel: func() {}, users: 1}
	d.cond.L = &d.mu
	s, err := openFFmpegStream(d, 5000)
	if err != nil {
		t.Fatal(err)
	}
	closed := make(chan struct{})
	go func() {
		s.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("closing waited on bytes that never came")
	}
}

// A file FFmpeg can't read either says so, rather than playing silence.
func TestFFmpegSaysWhatFailed(t *testing.T) {
	if !ffmpeg.Available() {
		t.Skip("ffmpeg is not installed")
	}
	_, err := openStream(downloaded([]byte("not a song at all")), 0)
	if err == nil {
		t.Fatal("opened what is no song")
	}
	if errors.Is(err, ffmpeg.ErrMissing) {
		t.Fatalf("FFmpeg is there, yet it says missing")
	}
}

// encodeWMA encodes the fixture as a WMA, which the app's decoders don't read,
// and returns the file and its bytes, skipping a test where FFmpeg can't.
func encodeWMA(t *testing.T) (string, []byte) {
	t.Helper()
	if !ffmpeg.Available() {
		t.Skip("ffmpeg is not installed")
	}
	if _, err := os.Stat("testdata/seek.flac"); err != nil {
		t.Skip("no fixture to encode")
	}
	wma := filepath.Join(t.TempDir(), "seek.wma")
	cmd := exec.Command(ffmpeg.Path(), "-hide_banner", "-loglevel", "error", "-y",
		"-i", "testdata/seek.flac", "-c:a", "wmav2", "-b:a", "128k", wma)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg can't encode a WMA here: %v: %s", err, out)
	}
	b, err := os.ReadFile(wma)
	if err != nil {
		t.Fatalf("reading back the WMA: %v", err)
	}
	return wma, b
}
