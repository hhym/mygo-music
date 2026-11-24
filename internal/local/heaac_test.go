package local

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/audio/mp4"
)

// An .m4a of HE-AAC, which the app's own decoder doesn't decode, is listed all
// the same: FFmpeg plays it when it is there.
func TestHEAACListed(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "player", "testdata", "seek-aac.m4a"))
	if err != nil {
		t.Fatal(err)
	}
	track, err := mp4.ReadTrack(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.Index(b, track.Config)
	if at < 0 || len(track.Config) == 0 {
		t.Fatal("no AudioSpecificConfig in the file")
	}
	patch := bytes.Clone(b)
	patch[at] = track.Config[0]&0x07 | 5<<3 // HE-AAC's object type

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "he.m4a"), patch, 0o644); err != nil {
		t.Fatal(err)
	}
	s := scanned(t, dir)
	got := songs(t, s)
	if len(got) != 1 {
		t.Fatalf("listed %v", names(got))
	}
	if got[0].Duration <= 0 {
		t.Fatalf("got %q, %d ms", got[0].Name, got[0].Duration)
	}
}
