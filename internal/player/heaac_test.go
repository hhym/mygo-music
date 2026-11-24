package player

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/egoist/mygo/audio/mp4"

	"mygo-music/internal/ffmpeg"
)

// heAAC is seek-aac.m4a with its AudioSpecificConfig saying HE-AAC, whose
// SBR the app's decoder doesn't decode.
func heAAC(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/seek-aac.m4a")
	if err != nil {
		t.Fatal(err)
	}
	track, err := mp4.ReadTrack(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	cfg := track.Config
	at := bytes.Index(b, cfg)
	if at < 0 || len(cfg) == 0 {
		t.Fatal("no AudioSpecificConfig in the file")
	}
	patch := bytes.Clone(b)
	patch[at] = cfg[0]&0x07 | 5<<3 // HE-AAC's object type, the sample rate as it was
	return patch
}

// An .m4a of HE-AAC isn't the app's to decode: its decoders say so, and FFmpeg
// plays it, or says where to put one.
func TestHEAACLeftToFFmpeg(t *testing.T) {
	b := heAAC(t)
	if _, err := openBuiltin(downloaded(b), 0); !errors.Is(err, errOtherFormat) {
		t.Fatalf("openBuiltin: %v", err)
	}
	if !ffmpeg.Available() {
		if _, err := openStream(downloaded(b), 0); !errors.Is(err, ffmpeg.ErrMissing) {
			t.Fatalf("openStream without FFmpeg: %v", err)
		}
		return
	}
	s, err := openStream(downloaded(b), 0)
	if err != nil {
		t.Skipf("FFmpeg here doesn't decode it: %v", err)
	}
	defer s.Close()
	buf := make([]float32, 4096)
	if _, err := s.Read(buf); err != nil && err != io.EOF {
		t.Fatal(err)
	}
}
