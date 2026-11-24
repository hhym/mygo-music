package ffmpeg

import (
	"testing"
	"time"
)

// FFprobe's fields say what a file holds, and how long it plays.
func TestParseFields(t *testing.T) {
	out := []byte("codec_name=ape\nsample_rate=44100\nchannels=2\nduration=192.500000\n")
	info := parseFields(out)
	if info.Codec != "ape" || info.SampleRate != 44100 || info.Channels != 2 {
		t.Errorf("read %+v", info)
	}
	if info.Duration != 192*time.Second+500*time.Millisecond {
		t.Errorf("length %v", info.Duration)
	}
}

// FFmpeg's log says how long a file is, in hours, minutes and seconds.
func TestParseDuration(t *testing.T) {
	for _, c := range []struct {
		line string
		want time.Duration
	}{
		{" 00:03:12.34, start: 0.000000, bitrate: 128 kb/s", 3*time.Minute + 12*time.Second + 340*time.Millisecond},
		{" 00:00:05.00", 5 * time.Second},
		{" 01:00:00.00", time.Hour},
	} {
		got, ok := parseDuration(c.line)
		if !ok || got != c.want {
			t.Errorf("%q: %v, %v", c.line, got, ok)
		}
	}
	if _, ok := parseDuration(" N/A"); ok {
		t.Error("read a length of N/A")
	}
}
