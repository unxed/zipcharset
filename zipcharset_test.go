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
	binary.LittleEndian.PutUint16(extra[0:2], unicodePathExtraID)
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
		{"EFS Flag", []byte("Привет"), true, creatorFAT, 20, nil, "Привет"},
		{"NTFS (ANSI)", win1251Raw, false, creatorNTFS, 20, nil, "Привет"},
		{"FAT (OEM)", cp866Raw, false, creatorFAT, 10, nil, "Привет"},
		{"Unicode Extra valid", cp866Raw, false, creatorFAT, 10, buildUnicodeExtra(cp866Raw, "Unicode"), "Unicode"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := decodeBytes(tc.raw, tc.isUTF8, tc.packOS, tc.packVer, tc.extra, false)
			if actual != tc.expected {
				t.Errorf("got %q, want %q", actual, tc.expected)
			}
		})
	}
}

func TestNewNameDecoder(t *testing.T) {
	origOEM := localecp.OEMDecoder
	defer func() { localecp.OEMDecoder = origOEM }()
	localecp.OEMDecoder = charmap.CodePage866.NewDecoder()

	cp866Raw := []byte{0x8f, 0xe0, 0xa8, 0xa2, 0xa5, 0xe2}
	fh := &zip.FileHeader{
		Name:           string(cp866Raw),
		CreatorVersion: creatorFAT << 8,
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