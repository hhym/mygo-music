package local

import (
	"bytes"
	"encoding/binary"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/dhowden/tag"
)

// A WMA is an ASF file: a header object of other objects, among them the
// Content Description, which holds a title and an author, and the Extended
// Content Description, which holds the rest, a picture too. Windows Media
// wrote them in UTF-16, under names of its own.

var (
	asfHeaderGUID           = [16]byte{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}
	asfContentGUID          = [16]byte{0x33, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9, 0x00, 0xAA, 0x00, 0x62, 0xCE, 0x6C}
	asfExtContentGUID       = [16]byte{0x40, 0xA4, 0xD0, 0xD2, 0x07, 0xE3, 0xD2, 0x11, 0x97, 0xF0, 0x00, 0xA0, 0xC9, 0x5E, 0xA8, 0x50}
	asfHeaderExtGUID        = [16]byte{0xB5, 0x03, 0xBF, 0x5F, 0x2E, 0xA9, 0xCF, 0x11, 0x8E, 0xE3, 0x00, 0xC0, 0x0C, 0x20, 0x53, 0x65}
	asfMetadataGUID         = [16]byte{0xEA, 0xCB, 0xF8, 0xC5, 0xAF, 0x5B, 0x77, 0x48, 0x84, 0x67, 0xAA, 0x8C, 0x44, 0xFA, 0x4C, 0xCA}
	asfLibraryGUID          = [16]byte{0x94, 0x1C, 0x23, 0x44, 0x98, 0x94, 0xD1, 0x49, 0xA1, 0x41, 0x1D, 0x13, 0x4E, 0x45, 0x70, 0x54}
	asfMaxHeader      int64 = 32 << 20
)

// asfTags reads a WMA's tags, or empty ones if it holds none, or isn't a WMA.
func asfTags(f *os.File, size int64) (t tags) {
	var top [24]byte
	if _, err := f.ReadAt(top[:], 0); err != nil || !bytes.Equal(top[:16], asfHeaderGUID[:]) {
		return t
	}
	n := int64(binary.LittleEndian.Uint64(top[16:]))
	if n < 30 || n > size || n > asfMaxHeader {
		return t
	}
	b := make([]byte, n)
	if _, err := f.ReadAt(b, 0); err != nil {
		return t
	}
	t.asfObjects(b[30:]) // after the count of objects and two reserved bytes
	return t
}

// asfObjects takes each object of a header, or of its extension, by what its
// GUID says it is.
func (t *tags) asfObjects(b []byte) {
	for pos := 0; pos+24 <= len(b); {
		obj := b[pos:]
		length := int(binary.LittleEndian.Uint64(obj[16:]))
		if length < 24 || pos+length > len(b) {
			return
		}
		body := obj[24:length]
		switch {
		case bytes.Equal(obj[:16], asfContentGUID[:]):
			t.asfContent(body)
		case bytes.Equal(obj[:16], asfExtContentGUID[:]):
			t.asfExtContent(body)
		case bytes.Equal(obj[:16], asfHeaderExtGUID[:]):
			if len(body) >= 22 {
				t.asfObjects(body[22:]) // what it extends the header with
			}
		case bytes.Equal(obj[:16], asfMetadataGUID[:]):
			t.asfMetadata(body)
		case bytes.Equal(obj[:16], asfLibraryGUID[:]):
			t.asfLibrary(body)
		}
		pos += length
	}
}

// asfContent reads a Content Description: five strings, a title, an author, a
// copyright, a description and a rating, one after the other.
func (t *tags) asfContent(b []byte) {
	if len(b) < 10 {
		return
	}
	lengths := [5]int{
		int(binary.LittleEndian.Uint16(b[0:])),
		int(binary.LittleEndian.Uint16(b[2:])),
		int(binary.LittleEndian.Uint16(b[4:])),
		int(binary.LittleEndian.Uint16(b[6:])),
		int(binary.LittleEndian.Uint16(b[8:])),
	}
	off := 10
	for i, n := range lengths {
		if off+n > len(b) {
			return
		}
		text := cleanTag(asfUTF16(b[off : off+n]))
		off += n
		switch i {
		case 0:
			t.title = text
		case 1:
			t.artist = text
		}
	}
}

// asfExtContent reads descriptors of a name, a type and a value.
func (t *tags) asfExtContent(b []byte) {
	if len(b) < 2 {
		return
	}
	count := int(binary.LittleEndian.Uint16(b[0:]))
	off := 2
	for i := 0; i < count && off+2 <= len(b); i++ {
		nameLen := int(binary.LittleEndian.Uint16(b[off:]))
		off += 2
		if off+nameLen > len(b) {
			return
		}
		name := asfUTF16(b[off : off+nameLen])
		off += nameLen
		if off+4 > len(b) {
			return
		}
		kind := binary.LittleEndian.Uint16(b[off:])
		valueLen := int(binary.LittleEndian.Uint16(b[off+2:]))
		off += 4
		if off+valueLen > len(b) {
			return
		}
		t.asfValue(name, kind, b[off:off+valueLen])
		off += valueLen
	}
}

// asfMetadata reads the descriptors of a Metadata Object, which say a stream
// number too.
func (t *tags) asfMetadata(b []byte) {
	t.asfDescriptors(b, 1)
}

// asfLibrary reads the descriptors of a Metadata Library Object, which say a
// language besides.
func (t *tags) asfLibrary(b []byte) {
	t.asfDescriptors(b, 2)
}

// asfDescriptors reads descriptors of a stream, or of a language and a stream,
// a type, a length, and then the name and the value.
func (t *tags) asfDescriptors(b []byte, words int) {
	if len(b) < 2 {
		return
	}
	count := int(binary.LittleEndian.Uint16(b[0:]))
	off := 2
	for i := 0; i < count; i++ {
		off += 2 * words // the stream, and the language of a library
		if off+8 > len(b) {
			return
		}
		nameLen := int(binary.LittleEndian.Uint16(b[off:]))
		kind := binary.LittleEndian.Uint16(b[off+2:])
		valueLen := int(binary.LittleEndian.Uint32(b[off+4:]))
		off += 8
		if nameLen <= 0 || valueLen < 0 || off+nameLen+valueLen > len(b) {
			return
		}
		name := asfUTF16(b[off : off+nameLen])
		off += nameLen
		t.asfValue(name, kind, b[off:off+valueLen])
		off += valueLen
	}
}

// asfValue takes a descriptor by the name Windows Media gave it.
func (t *tags) asfValue(name string, kind uint16, value []byte) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "wm/picture":
		if kind == 1 {
			if pic := wmPicture(value); pic != nil {
				t.picture = pic
			}
		}
	case "wm/albumtitle":
		t.album = cleanTag(asfValueText(kind, value))
	case "wm/albumartist", "author":
		t.albumArtist = cleanTag(asfValueText(kind, value))
	case "wm/year":
		t.year = yearOfText(asfValueText(kind, value))
	case "wm/tracknumber":
		t.track = numberOf(asfValueText(kind, value))
	case "wm/track":
		// This one counts from zero.
		t.track = numberOf(asfValueText(kind, value))
		if kind == 3 && len(value) >= 4 {
			t.track = int(binary.LittleEndian.Uint32(value)) + 1
		}
	case "wm/lyrics", "wm/lyrics_synchronised":
		if text := cleanTag(asfValueText(kind, value)); text != "" {
			t.lyrics = text
		}
	case "wm/partofset":
		t.disc = numberOf(asfValueText(kind, value))
	case "replaygain_track_gain":
		t.gain = parseGain(asfValueText(kind, value))
	case "replaygain_track_peak":
		t.peak = parseGain(asfValueText(kind, value))
	}
}

// asfValueText is a descriptor's value as text, by its type.
func asfValueText(kind uint16, value []byte) string {
	switch kind {
	case 0: // a Unicode string, in UTF-16
		return asfUTF16(value)
	case 2: // a boolean
		if len(value) >= 4 && binary.LittleEndian.Uint32(value) != 0 {
			return "1"
		}
	case 3: // a DWORD
		if len(value) >= 4 {
			return strconv.FormatUint(uint64(binary.LittleEndian.Uint32(value)), 10)
		}
	case 4: // a QWORD
		if len(value) >= 8 {
			return strconv.FormatUint(binary.LittleEndian.Uint64(value), 10)
		}
	case 5: // a WORD
		if len(value) >= 2 {
			return strconv.FormatUint(uint64(binary.LittleEndian.Uint16(value)), 10)
		}
	}
	return ""
}

// wmPicture reads a WM/Picture: its type, its MIME type, a description, and the
// picture itself.
func wmPicture(b []byte) *tag.Picture {
	if len(b) < 5 {
		return nil
	}
	mimeLen := int(binary.LittleEndian.Uint16(b[1:]))
	off := 3
	if off+mimeLen > len(b) {
		return nil
	}
	mime := asfUTF16(b[off : off+mimeLen])
	off += mimeLen
	if off+2 > len(b) {
		return nil
	}
	descLen := int(binary.LittleEndian.Uint16(b[off:]))
	off += 2
	if off+descLen > len(b) {
		return nil
	}
	off += descLen
	if off+4 > len(b) {
		return nil
	}
	dataLen := int(binary.LittleEndian.Uint32(b[off:]))
	off += 4
	if dataLen <= 0 || off+dataLen > len(b) {
		return nil
	}
	return &tag.Picture{MIMEType: strings.TrimSpace(mime), Data: b[off : off+dataLen]}
}

// asfUTF16 reads a UTF-16LE string up to its null, as Windows Media writes them.
func asfUTF16(b []byte) string {
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		v := binary.LittleEndian.Uint16(b[i:])
		if v == 0 {
			break
		}
		u = append(u, v)
	}
	return string(utf16.Decode(u))
}
