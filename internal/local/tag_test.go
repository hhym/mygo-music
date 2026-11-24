package local

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"mygo-music/internal/music"
)

// apeItem builds an item of an APEv2 tag: its length, its flags, its key, and
// its value.
func apeItem(key string, kind uint32, value []byte) []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint32(len(value)))
	binary.Write(&b, binary.LittleEndian, kind<<1)
	b.WriteString(key)
	b.Write([]byte{0})
	b.Write(value)
	return b.Bytes()
}

// apeTag builds an APEv2 tag of the items, and the footer that says how long it
// is and how many it holds.
func apeTag(items ...[]byte) []byte {
	var b bytes.Buffer
	for _, item := range items {
		b.Write(item)
	}
	var footer bytes.Buffer
	footer.WriteString("APETAGEX")
	binary.Write(&footer, binary.LittleEndian, uint32(2000)) // version
	binary.Write(&footer, binary.LittleEndian, uint32(b.Len()+32))
	binary.Write(&footer, binary.LittleEndian, uint32(len(items)))
	binary.Write(&footer, binary.LittleEndian, uint32(0)) // no header
	footer.Write(make([]byte, 8))
	b.Write(footer.Bytes())
	return b.Bytes()
}

// apeItemsSay is the items of a tag of what a song's tags say.
func apeItemsSay() [][]byte {
	return [][]byte{
		apeItem("Title", 0, []byte("夜曲")),
		apeItem("Artist", 0, []byte("周杰伦")),
		apeItem("Album", 0, []byte("十一月的萧邦")),
		apeItem("Album Artist", 0, []byte("周杰伦")),
		apeItem("Year", 0, []byte("2005-11-01")),
		apeItem("Track", 0, []byte("3/12")),
		apeItem("Disc", 0, []byte("1/2")),
		apeItem("Lyrics", 0, []byte("一群嗜血的蚂蚁")),
		apeItem("ReplayGain_Track_Gain", 0, []byte("-6.50 dB")),
		apeItem("ReplayGain_Track_Peak", 0, []byte("0.985")),
		apeItem("Cover Art (Front)", 1, append([]byte("cover.jpg\x00"), jpegData...)),
	}
}

// apeFile is an APE file of audio that isn't, its tag, and, if wanted, an
// ID3v1 tag after it, as older taggers left them.
func apeFile(tag []byte, id3v1 bool) []byte {
	b := append([]byte("MAC \x96\x0f\x00\x00"), bytes.Repeat([]byte{0x5a}, 4096)...)
	b = append(b, tag...)
	if id3v1 {
		b = append(b, []byte("TAG")...)
		b = append(b, bytes.Repeat([]byte{' '}, 125)...)
	}
	return b
}

// TestAPEItems reads a tag's items by their keys.
func TestAPEItems(t *testing.T) {
	items := apeItemsSay()
	tg := apeItems(bytes.Join(items, nil), len(items))
	if tg.title != "夜曲" || tg.artist != "周杰伦" || tg.album != "十一月的萧邦" || tg.albumArtist != "周杰伦" {
		t.Fatalf("got %q, %q, %q, %q", tg.title, tg.artist, tg.album, tg.albumArtist)
	}
	if tg.year != 2005 || tg.track != 3 || tg.disc != 1 {
		t.Fatalf("got year %d, track %d, disc %d", tg.year, tg.track, tg.disc)
	}
	if tg.lyrics != "一群嗜血的蚂蚁" {
		t.Fatalf("got lyrics %q", tg.lyrics)
	}
	if tg.gain != -6.5 || tg.peak != 0.985 {
		t.Fatalf("got gain %v, peak %v", tg.gain, tg.peak)
	}
	if tg.picture == nil || !bytes.Equal(tg.picture.Data, jpegData) {
		t.Fatal("no cover")
	}
}

// An APE file's tag sits at the end, and an ID3v1 tag older taggers wrote for
// older players may come after it.
func TestAPETags(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "夜曲.ape"), apeFile(apeTag(apeItemsSay()...), true), 0o644); err != nil {
		t.Fatal(err)
	}
	s := scanned(t, dir)
	tr := songNamed(t, s, "夜曲")
	if tr.ArtistNames() != "周杰伦" || tr.Album.Name != "十一月的萧邦" || tr.Album.Released != "2005" {
		t.Fatalf("got %q, %q, %q", tr.ArtistNames(), tr.Album.Name, tr.Album.Released)
	}
	if tr.Album.ImageURL == "" {
		t.Fatal("no cover")
	}
	p, _ := music.PathOf(tr.Album.ImageURL)
	if !strings.HasSuffix(p, ".jpg") || filepath.Dir(p) != s.coverDir() {
		t.Fatalf("cover at %q", p)
	}
	if b, err := os.ReadFile(p); err != nil || !bytes.Equal(b, jpegData) {
		t.Fatalf("cover: %v", err)
	}
}

// asfObject builds an ASF object of a GUID and a body, its length included.
func asfObject(guid [16]byte, body []byte) []byte {
	var b bytes.Buffer
	b.Write(guid[:])
	binary.Write(&b, binary.LittleEndian, uint64(24+len(body)))
	b.Write(body)
	return b.Bytes()
}

// utf16le is a string as ASF keeps them, a null at the end.
func utf16le(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*(len(u)+1))
	for i, v := range u {
		binary.LittleEndian.PutUint16(b[2*i:], v)
	}
	return b
}

// asfText builds a descriptor of a name and a string.
func asfText(name, value string) []byte {
	n := utf16le(name)
	v := utf16le(value)
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint16(len(n)))
	b.Write(n)
	binary.Write(&b, binary.LittleEndian, uint16(0)) // a Unicode string
	binary.Write(&b, binary.LittleEndian, uint16(len(v)))
	b.Write(v)
	return b.Bytes()
}

// asfBytes builds a descriptor of a name and bytes, a picture among them.
func asfBytes(name string, value []byte) []byte {
	n := utf16le(name)
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint16(len(n)))
	b.Write(n)
	binary.Write(&b, binary.LittleEndian, uint16(1)) // bytes
	binary.Write(&b, binary.LittleEndian, uint16(len(value)))
	b.Write(value)
	return b.Bytes()
}

// asfNumber builds a descriptor of a name and a DWORD.
func asfNumber(name string, value uint32) []byte {
	n := utf16le(name)
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint16(len(n)))
	b.Write(n)
	binary.Write(&b, binary.LittleEndian, uint16(3)) // a DWORD
	binary.Write(&b, binary.LittleEndian, uint16(4))
	binary.Write(&b, binary.LittleEndian, value)
	return b.Bytes()
}

// asfPicture builds a WM/Picture of a MIME type and a picture.
func asfPicture(mime string, data []byte) []byte {
	var b bytes.Buffer
	b.Write([]byte{3}) // a cover, front
	m := utf16le(mime)
	binary.Write(&b, binary.LittleEndian, uint16(len(m)))
	b.Write(m)
	d := utf16le("")
	binary.Write(&b, binary.LittleEndian, uint16(len(d)))
	b.Write(d)
	binary.Write(&b, binary.LittleEndian, uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}

// wmaFile is a WMA of a header that keeps what its tags say, and audio that
// isn't.
func wmaFile(objects ...[]byte) []byte {
	var objs bytes.Buffer
	for _, obj := range objects {
		objs.Write(obj)
	}
	var head bytes.Buffer
	head.Write(asfHeaderGUID[:])
	binary.Write(&head, binary.LittleEndian, uint64(30+objs.Len()))
	binary.Write(&head, binary.LittleEndian, uint32(len(objects))) // and a count of them
	binary.Write(&head, binary.LittleEndian, uint16(0))            // reserved
	head.Write(objs.Bytes())
	return append(head.Bytes(), bytes.Repeat([]byte{0x5a}, 64)...)
}

// wmaOf is a WMA of what a song's tags say: a Content Description of a title
// and an author, and an Extended Content Description of the rest, a picture
// among them.
func wmaOf() []byte {
	title, author := utf16le("夜曲"), utf16le("周杰伦")
	s := [][]byte{title, author, utf16le(""), utf16le(""), utf16le("")}
	var content bytes.Buffer
	for _, v := range s {
		binary.Write(&content, binary.LittleEndian, uint16(len(v)))
	}
	for _, v := range s {
		content.Write(v)
	}
	descs := [][]byte{
		asfText("WM/AlbumTitle", "十一月的萧邦"),
		asfText("WM/AlbumArtist", "周杰伦"),
		asfText("WM/Year", "2005"),
		asfNumber("WM/Track", 2), // counts from zero
		asfText("WM/Lyrics", "一群嗜血的蚂蚁"),
		asfText("replaygain_track_gain", "-6.50 dB"),
		asfBytes("WM/Picture", asfPicture("image/jpeg", jpegData)),
	}
	var ext bytes.Buffer
	binary.Write(&ext, binary.LittleEndian, uint16(len(descs)))
	for _, d := range descs {
		ext.Write(d)
	}
	return wmaFile(
		asfObject(asfContentGUID, content.Bytes()),
		asfObject(asfExtContentGUID, ext.Bytes()),
	)
}

// checkWMATags is what a WMA of the tags should say.
func checkWMATags(t *testing.T, tg tags) {
	t.Helper()
	if tg.title != "夜曲" || tg.artist != "周杰伦" || tg.album != "十一月的萧邦" || tg.albumArtist != "周杰伦" {
		t.Fatalf("got %q, %q, %q, %q", tg.title, tg.artist, tg.album, tg.albumArtist)
	}
	if tg.year != 2005 || tg.track != 3 {
		t.Fatalf("got year %d, track %d", tg.year, tg.track)
	}
	if tg.lyrics != "一群嗜血的蚂蚁" {
		t.Fatalf("got lyrics %q", tg.lyrics)
	}
	if tg.gain != -6.5 {
		t.Fatalf("got gain %v", tg.gain)
	}
	if tg.picture == nil || !bytes.Equal(tg.picture.Data, jpegData) || tg.picture.MIMEType != "image/jpeg" {
		t.Fatal("no cover")
	}
}

// TestASFTags reads a WMA's tags from its header.
func TestASFTags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "夜曲.wma")
	if err := os.WriteFile(path, wmaOf(), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	checkWMATags(t, asfTags(f, st.Size()))
}

// A WMA's tags make it into the library like any other file's, its cover too.
func TestWMAFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "夜曲.wma"), wmaOf(), 0o644); err != nil {
		t.Fatal(err)
	}
	s := scanned(t, dir)
	tr := songNamed(t, s, "夜曲")
	if tr.ArtistNames() != "周杰伦" || tr.Album.Name != "十一月的萧邦" || tr.Album.Released != "2005" {
		t.Fatalf("got %q, %q, %q", tr.ArtistNames(), tr.Album.Name, tr.Album.Released)
	}
	p, _ := music.PathOf(tr.Album.ImageURL)
	if !strings.HasSuffix(p, ".jpg") {
		t.Fatalf("cover at %q", p)
	}
}

// A Metadata Object is where a newer tagger keeps a track number, in a header
// extension it extends the header with.
func TestASFMetadata(t *testing.T) {
	var meta bytes.Buffer
	binary.Write(&meta, binary.LittleEndian, uint16(1)) // one descriptor
	n := utf16le("WM/TrackNumber")
	binary.Write(&meta, binary.LittleEndian, uint16(0)) // the stream
	binary.Write(&meta, binary.LittleEndian, uint16(len(n)))
	binary.Write(&meta, binary.LittleEndian, uint16(0)) // a Unicode string
	v := utf16le("5")
	binary.Write(&meta, binary.LittleEndian, uint32(len(v)))
	meta.Write(n)
	meta.Write(v)

	var ext bytes.Buffer
	ext.Write(make([]byte, 16))                        // a reserved GUID
	binary.Write(&ext, binary.LittleEndian, uint16(6)) // reserved
	inner := asfObject(asfMetadataGUID, meta.Bytes())
	binary.Write(&ext, binary.LittleEndian, uint32(len(inner)))
	ext.Write(inner)

	dir := t.TempDir()
	path := filepath.Join(dir, "track.wma")
	if err := os.WriteFile(path, wmaFile(asfObject(asfHeaderExtGUID, ext.Bytes())), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if tg := asfTags(f, st.Size()); tg.track != 5 {
		t.Fatalf("got track %d", tg.track)
	}
}

// A file that isn't a WMA says nothing, though its name says it is one.
func TestWMABroken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.wma")
	if err := os.WriteFile(path, []byte("not a WMA at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	var tg tags
	if tg = asfTags(f, st.Size()); tg != (tags{}) {
		t.Fatalf("got %+v", tg)
	}
}

// A file that isn't an APE file says nothing either.
func TestAPEBroken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.ape")
	if err := os.WriteFile(path, []byte("APETAGEX"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if tg := apeTags(f, st.Size()); tg != (tags{}) {
		t.Fatalf("got %+v", tg)
	}
}
