package player

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"sync"

	"mygo-music/internal/ffmpeg"
)

// ffmpegRate and ffmpegChannels are what FFmpeg decodes to: the output's own
// rate and layout, so that it has nothing to resample for the player.
const (
	ffmpegRate     = 48000
	ffmpegChannels = 2
)

// ffmpegStream plays what the app's decoders don't, by leaving it to FFmpeg:
// APE, WMA, AIFF, DSD and the like. FFmpeg decodes to stereo PCM on a pipe,
// which the stream reads a chunk at a time, and a seek starts FFmpeg again
// where it goes: by the file's own offset when the song is on disk, or letting
// it decode and drop what comes before, which a pipe can't skip.
type ffmpegStream struct {
	dl     *download
	file   string // the song on disk, or "" to read the download as it comes
	rate   int
	frames int64 // the song's length, or -1 if unknown
	pos    int64 // the frame read next
	cmd    *exec.Cmd
	out    io.ReadCloser
	in     *ffmpegInput // the download, read into the pipe
	raw    []byte
	pre    []byte // what opening decoded, to play before the pipe
	log    *errorLog
	spec   *spectrum
	drop   func() // lets the download go, as the stream closes
	once   sync.Once
}

// openFFmpegStream opens dl with FFmpeg, which the app looks for beside itself,
// on the PATH or where MYGO_FFMPEG says.
func openFFmpegStream(dl *download, durationMs int64) (*ffmpegStream, error) {
	if !ffmpeg.Available() {
		return nil, ffmpeg.ErrMissing
	}
	s := &ffmpegStream{
		dl:   dl,
		rate: ffmpegRate,
		log:  &errorLog{},
		spec: newSpectrum(ffmpegRate),
		drop: dl.Close,
	}
	// A song all in on disk FFmpeg seeks by itself; one still coming is read
	// from the pipe as its bytes arrive.
	if dl.onDisk {
		dl.mu.Lock()
		if dl.file != nil {
			s.file = dl.file.Name()
		}
		dl.mu.Unlock()
	}
	s.frames = durationMs * int64(ffmpegRate) / 1000
	if s.frames <= 0 && s.file != "" {
		// The track came without its length: FFmpeg says of it.
		if info, err := ffmpeg.Probe(s.file); err == nil && info.Duration > 0 {
			s.frames = info.Duration.Milliseconds() * int64(ffmpegRate) / 1000
		}
	}
	if s.frames <= 0 {
		s.frames = -1
	}
	if err := s.start(0); err != nil {
		return nil, err
	}
	// What it decodes first says whether it can read the song at all, before
	// a listener waits on a song that never plays.
	if err := s.prime(); err != nil {
		s.stop()
		return nil, err
	}
	return s, nil
}

// prime reads what FFmpeg decodes first, to keep for the first Read, and fails
// for a song it can't decode.
func (s *ffmpegStream) prime() error {
	buf := make([]byte, 8<<10)
	got, _ := io.ReadFull(s.out, buf)
	if got -= got % 4; got <= 0 {
		if msg := s.log.last(); msg != "" {
			return errors.New(msg)
		}
		return errors.New("ffmpeg: decoded nothing")
	}
	s.pre = buf[:got]
	return nil
}

// start runs FFmpeg on the song from frame at, on a new pipe.
func (s *ffmpegStream) start(at int64) error {
	sec := strconv.FormatFloat(float64(at)/float64(s.rate), 'f', 3, 64)
	args := []string{"-hide_banner", "-loglevel", "error", "-nostats"}
	if s.file != "" {
		if at > 0 {
			args = append(args, "-ss", sec) // seek the file itself
		}
		args = append(args, "-i", s.file)
	} else {
		args = append(args, "-i", "pipe:0")
		if at > 0 {
			// A pipe can't be seeked: FFmpeg decodes from its start, and
			// drops what comes before the frame asked for.
			args = append(args, "-ss", sec)
		}
	}
	args = append(args,
		"-map", "0:a:0", "-vn", "-sn", "-dn",
		"-ac", strconv.Itoa(ffmpegChannels), "-ar", strconv.Itoa(s.rate),
		"-f", "s16le", "-acodec", "pcm_s16le", "pipe:1")
	cmd := ffmpeg.Cmd(nil, args...)
	cmd.Stderr = s.log
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if s.file == "" {
		s.in = newInput(s.dl)
		cmd.Stdin = s.in
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	s.cmd, s.out = cmd, out
	return nil
}

// stop ends the FFmpeg on its way, and lets it go.
func (s *ffmpegStream) stop() {
	if s.cmd == nil {
		return
	}
	// Let the read into its pipe go first: a song still coming would keep
	// FFmpeg waiting on bytes, and waiting on it with them.
	if s.in != nil {
		s.in.cancel()
	}
	if s.out != nil {
		s.out.Close()
	}
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	s.cmd.Wait() // reaps it
	s.cmd, s.out, s.in = nil, nil, nil
}

func (s *ffmpegStream) SampleRate() int { return s.rate }
func (s *ffmpegStream) Channels() int   { return ffmpegChannels }
func (s *ffmpegStream) Len() int64      { return s.frames }

func (s *ffmpegStream) Read(dst []float32) (int, error) {
	n := len(dst) / 2 * 2
	need := n * 2
	var b []byte
	var err error
	if len(s.pre) > 0 {
		take := min(len(s.pre), need)
		b, s.pre = s.pre[:take], s.pre[take:]
	} else {
		if s.out == nil {
			return 0, io.EOF
		}
		if cap(s.raw) < need {
			s.raw = make([]byte, need)
		}
		b = s.raw[:need]
		var got int
		got, err = io.ReadFull(s.out, b)
		b = b[:got]
	}
	b = b[:len(b)-len(b)%4]
	samples := len(b) / 2
	for i := range samples {
		dst[i] = float32(int16(binary.LittleEndian.Uint16(b[2*i:]))) / (1 << 15)
	}
	if samples > 0 {
		s.spec.feed(s.pos, dst[:samples])
		s.pos += int64(samples / 2)
		return samples, nil
	}
	if err == nil || errors.Is(err, io.ErrUnexpectedEOF) {
		// The song ended, or FFmpeg stopped: with a reason, it says it.
		s.stop()
		if msg := s.log.last(); msg != "" {
			return 0, errors.New(msg)
		}
		return 0, io.EOF
	}
	return 0, err
}

// SetPosition starts FFmpeg again at frame, dropping the one on its way.
func (s *ffmpegStream) SetPosition(frame int64) error {
	if frame < 0 {
		frame = 0
	}
	if s.frames > 0 && frame > s.frames {
		frame = s.frames
	}
	s.stop()
	s.log.reset()
	s.pre = nil // what came before the frame asked for
	s.pos = frame
	return s.start(frame)
}

func (s *ffmpegStream) Close() error {
	s.once.Do(func() {
		s.stop()
		s.drop()
	})
	return nil
}

func (s *ffmpegStream) source() (*download, *spectrum) { return s.dl, s.spec }

// errorLog keeps what FFmpeg said went wrong, up to what's worth showing.
type errorLog struct {
	mu sync.Mutex
	b  bytes.Buffer
}

const errorLogMax = 4 << 10

func (l *errorLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.b.Len() < errorLogMax {
		l.b.Write(p)
	}
	return len(p), nil
}

// last returns the last thing said, or "".
func (l *errorLog) last() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	lines := bytes.Split(bytes.TrimSpace(l.b.Bytes()), []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if line := bytes.TrimSpace(lines[i]); len(line) > 0 {
			return string(line)
		}
	}
	return ""
}

func (l *errorLog) reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.b.Reset()
}

// ffmpegInput feeds FFmpeg a song still coming, a chunk at a time, so that a
// seek or a close lets it go rather than waiting on bytes that never come.
type ffmpegInput struct {
	dl   *download
	pos  int64
	done chan struct{}
	once sync.Once
}

func newInput(dl *download) *ffmpegInput {
	return &ffmpegInput{dl: dl, done: make(chan struct{})}
}

// errInputClosed says the stream let its FFmpeg go.
var errInputClosed = errors.New("player: ffmpeg stopped")

func (r *ffmpegInput) Read(p []byte) (int, error) {
	// The read happens on a goroutine of its own, waiting on the download,
	// while this one waits on the cancel.
	type chunk struct {
		b   []byte
		n   int
		err error
	}
	got := make(chan chunk, 1)
	pos, b := r.pos, make([]byte, len(p))
	go func() {
		n, err := r.dl.readWait(b, pos)
		got <- chunk{b, n, err}
	}()
	select {
	case c := <-got:
		r.pos += int64(c.n)
		return copy(p, c.b[:c.n]), c.err
	case <-r.done:
		return 0, errInputClosed
	}
}

func (r *ffmpegInput) cancel() { r.once.Do(func() { close(r.done) }) }
