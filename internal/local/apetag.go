package local

import (
	"bytes"
	"encoding/binary"
	"os"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
)

// An APEv2 tag sits at the end of an APE file, in items of a key and a value,
// with a footer that says how many there are and how long they run; a header
// before them may say the same. An ID3v1 tag may come after it, which older
// taggers wrote for players that read nothing else.

// apeTags reads an APE file's APEv2 tag, or an empty one if it has none.
func apeTags(f *os.File, size int64) (t tags) {
	at, tagSize, count, ok := apeFooter(f, size)
	if !ok {
		return t
	}
	n := tagSize - 32 // the items, the footer aside
	if n <= 0 || n > at {
		return t
	}
	b := make([]byte, n)
	got, _ := f.ReadAt(b, at-n)
	return apeItems(b[:got], int(count))
}

// apeFooter finds where an APEv2 tag's footer is, and what it says: how long
// the tag runs, header aside, and how many items it holds.
func apeFooter(f *os.File, size int64) (at, tagSize, count int64, ok bool) {
	offs := []int64{size - 32}
	if size >= 160 {
		var tail [3]byte
		if _, err := f.ReadAt(tail[:], size-128); err == nil && string(tail[:]) == "TAG" {
			offs = append(offs, size-160) // an ID3v1 tag comes after the APE one
		}
	}
	for _, off := range offs {
		if off < 0 {
			continue
		}
		var b [32]byte
		if _, err := f.ReadAt(b[:], off); err != nil || string(b[:8]) != "APETAGEX" {
			continue
		}
		if v := binary.LittleEndian.Uint32(b[8:]); v != 2000 && v != 1000 {
			continue
		}
		flags := binary.LittleEndian.Uint32(b[20:])
		if flags&(1<<29) != 0 {
			continue // a header, which a footer never is at the end
		}
		tagLen := int64(binary.LittleEndian.Uint32(b[12:]))
		count := int64(binary.LittleEndian.Uint32(b[16:]))
		if tagLen < 32 || tagLen > off+32 || count < 0 || count > 1<<20 {
			continue
		}
		return off, tagLen, count, true
	}
	return 0, 0, 0, false
}

// apeItems reads the items of a tag, in the order they are kept: a length and
// flags, a key up to its null, and the value.
func apeItems(b []byte, count int) (t tags) {
	pos := 0
	for i := 0; i < count; i++ {
		if pos+8 > len(b) {
			break
		}
		length := int(binary.LittleEndian.Uint32(b[pos:]))
		flags := binary.LittleEndian.Uint32(b[pos+4:])
		pos += 8
		key, next, ok := apeKey(b, pos)
		if !ok {
			break
		}
		pos = next
		if length < 0 || pos+length > len(b) {
			break
		}
		t.apeItem(key, b[pos:pos+length], (flags>>1)&3)
		pos += length
	}
	return t
}

// apeKey reads a key up to its null, at or after off.
func apeKey(b []byte, off int) (string, int, bool) {
	limit := min(off+256, len(b))
	i := bytes.IndexByte(b[off:limit], 0)
	if i < 0 {
		return "", 0, false
	}
	return string(b[off : off+i]), off + i + 1, true
}

// apeItem takes an item by its key, its value, and the kind of it: 0 is text,
// 1 a picture, and 2 a locator elsewhere.
func (t *tags) apeItem(key string, value []byte, kind uint32) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "title", "name":
		t.title = apeText(value, kind)
	case "artist":
		t.artist = apeText(value, kind)
	case "album":
		t.album = apeText(value, kind)
	case "album artist", "albumartist":
		t.albumArtist = apeText(value, kind)
	case "year":
		t.year = yearOfText(apeText(value, kind))
	case "track":
		t.track = numberOf(apeText(value, kind))
	case "disc", "discnumber":
		t.disc = numberOf(apeText(value, kind))
	case "lyrics", "unsyncedlyrics":
		if text := apeText(value, kind); text != "" {
			t.lyrics = text
		}
	case "cover art (front)", "cover art (back)":
		if pic := apePicture(value, kind); pic != nil {
			t.picture = pic
		}
	case "replaygain_track_gain":
		t.gain = parseGain(apeText(value, kind))
	case "replaygain_track_peak":
		t.peak = parseGain(apeText(value, kind))
	}
}

// apeText is an item's text, which is UTF-8, and else what a tagger wrote in
// its place.
func apeText(value []byte, kind uint32) string {
	if kind != 0 {
		return ""
	}
	return cleanTag(string(bytes.TrimRight(value, "\x00")))
}

// apePicture is a cover: the picture's name up to a null, and then the picture.
func apePicture(value []byte, kind uint32) *tag.Picture {
	if kind != 1 || len(value) == 0 {
		return nil
	}
	if i := bytes.IndexByte(value, 0); i >= 0 {
		value = value[i+1:]
	}
	if len(value) == 0 {
		return nil
	}
	return &tag.Picture{Data: value}
}

// yearOfText reads the year a tag says, which may be a date.
func yearOfText(s string) int {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return 0
	}
	if y, err := strconv.Atoi(s[:4]); err == nil && y >= 1000 {
		return y
	}
	return 0
}

// numberOf reads the number a tag says, as "3" or "3/12".
func numberOf(s string) int {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}
