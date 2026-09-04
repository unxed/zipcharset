package zipcharset

import (
	"encoding/binary"
	"hash/crc32"
	"testing"

	"github.com/klauspost/compress/zip"
	"github.com/unxed/localecp"
	"golang.org/x/text/encoding/charmap"
)

func buildUnicodeExtra(raw []byte, utf8Str string) []byte {
	crc := crc32.ChecksumIEEE(raw)
	payload := make([]byte, 5+len(utf8Str))
	payload[0] = 1
	binary.LittleEndian.PutUint32(payload[1:5], crc)
	copy(payload[5:], utf8Str)

	extra := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint16(extra[0:2], UnicodePathExtraID)
	binary.LittleEndian.PutUint16(extra[2:4], uint16(len(payload)))
	copy(extra[4:], payload)
	return extra
}

func TestDecodeBytes(t *testing.T) {
	origOEM := localecp.OEMDecoder
	origANSI := localecp.ANSIDecoder
	defer func() {
		localecp.OEMDecoder = origOEM
		localecp.ANSIDecoder = origANSI
	}()

	localecp.OEMDecoder = charmap.CodePage866.NewDecoder()
	localecp.ANSIDecoder = charmap.Windows1251.NewDecoder()

	cp866Raw := []byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2}
	win1251Raw := []byte{0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2}

	testCases := []struct {
		name     string
		raw      []byte
		isUTF8   bool
		packOS   byte
		packVer  uint16
		extra    []byte
		expected string
	}{
		{"EFS Flag", []byte("Привет"), true, CreatorFAT, 20, nil, "Привет"},
		{"NTFS (ANSI)", win1251Raw, false, CreatorNTFS, 20, nil, "Привет"},
		{"FAT (OEM)", cp866Raw, false, CreatorFAT, 10, nil, "Привет"},
		{"Unicode Extra valid", cp866Raw, false, CreatorFAT, 10, buildUnicodeExtra(cp866Raw, "Unicode"), "Unicode"},
		{"Unix OS (Always UTF-8)", []byte("Привет"), false, CreatorUnix, 20, nil, "Привет"},
		{"Fallback System Decoder", []byte("hello"), false, 99, 10, nil, "hello"},
		{"Empty Input", []byte{}, false, CreatorFAT, 10, nil, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := DecodeText(tc.raw, tc.isUTF8, tc.packOS, tc.packVer, tc.extra, false)
			if actual != tc.expected {
				t.Errorf("got %q, want %q", actual, tc.expected)
			}
		})
	}
}
func TestDecodeBytes_Comment(t *testing.T) {
	origOEM := localecp.OEMDecoder
	defer func() { localecp.OEMDecoder = origOEM }()
	localecp.OEMDecoder = charmap.CodePage866.NewDecoder()

	cp866Raw := []byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2}

	// Create Unicode Extra for Comment (0x6375)
	crc := crc32.ChecksumIEEE(cp866Raw)
	payload := make([]byte, 5+len("CommentUnicode"))
	payload[0] = 1
	binary.LittleEndian.PutUint32(payload[1:5], crc)
	copy(payload[5:], "CommentUnicode")

	extra := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint16(extra[0:2], UnicodeCommentExtraID)
	binary.LittleEndian.PutUint16(extra[2:4], uint16(len(payload)))
	copy(extra[4:], payload)

	actual := DecodeText(cp866Raw, false, CreatorFAT, 10, extra, true)
	if actual != "CommentUnicode" {
		t.Errorf("got %q, want 'CommentUnicode'", actual)
	}
}

func TestNewNameDecoder(t *testing.T) {
	origOEM := localecp.OEMDecoder
	defer func() { localecp.OEMDecoder = origOEM }()
	localecp.OEMDecoder = charmap.CodePage866.NewDecoder()

	cp866Raw := []byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2}
	fh := &zip.FileHeader{
		Name:           string(cp866Raw),
		CreatorVersion: CreatorFAT << 8,
	}

	decoder := NewNameDecoder()
	err := decoder(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fh.Name != "Привет" {
		t.Errorf("expected decoded name 'Привет', got %q", fh.Name)
	}
	if fh.Flags&0x800 == 0 {
		t.Error("expected flag 11 to be set")
	}
}

func TestParseUnicodeExtraField_Malformed(t *testing.T) {
	// A truncated extra field that should be safely skipped without panicking
	malformedExtra := []byte{0x75, 0x70, 0x01, 0x00}
	res := ParseUnicodeExtraField(malformedExtra, UnicodePathExtraID, []byte("raw"))
	if res != "" {
		t.Errorf("expected empty string for malformed input, got %q", res)
	}
}

func TestParseUnicodeExtraField_SizeOutOfBounds(t *testing.T) {
	// Extra field array contains a valid first block, but the second block declares an out-of-bounds size
	extra := []byte{
		0x00, 0x00, 0x01, 0x00, 0x00, // Dummy extra block (ID 0, Size 1, Data 1 byte)
		0x75, 0x70, 0xFF, 0xFF, // Target Extra block (ID 0x7075, Size 65535) - out of bounds
	}
	res := ParseUnicodeExtraField(extra, UnicodePathExtraID, []byte("raw"))
	if res != "" {
		t.Errorf("expected empty string for out-of-bounds size, got %q", res)
	}
}

func TestParseUnicodeExtraField_BadCRC(t *testing.T) {
	utf8Str := "MismatchedCRC"
	payload := make([]byte, 5+len(utf8Str))
	payload[0] = 1
	binary.LittleEndian.PutUint32(payload[1:5], 0xDEADBEEF) // Deliberately bad CRC
	copy(payload[5:], utf8Str)

	extra := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint16(extra[0:2], UnicodePathExtraID)
	binary.LittleEndian.PutUint16(extra[2:4], uint16(len(payload)))
	copy(extra[4:], payload)

	res := ParseUnicodeExtraField(extra, UnicodePathExtraID, []byte("some raw data"))
	if res != "" {
		t.Errorf("expected empty string for bad CRC, got %q", res)
	}
}
