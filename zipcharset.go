package zipcharset

import (
	"encoding/binary"
	"hash/crc32"
	"unicode/utf8"

	"github.com/klauspost/compress/zip"
	"github.com/unxed/localecp"
	"golang.org/x/text/encoding"
)

const (
	UnicodePathExtraID    = 0x7075
	UnicodeCommentExtraID = 0x6375

	CreatorFAT    = 0
	CreatorHPFS   = 6
	CreatorNTFS   = 11
	CreatorUnix   = 3
	CreatorMacOSX = 19
)

// NewNameDecoder returns an opt-in callback function to rewrite legacy,
// non-Unicode filenames and comments into clean UTF-8.
func NewNameDecoder() func(f *zip.FileHeader) error {
	return func(f *zip.FileHeader) error {
		isUTF8 := f.Flags&0x800 != 0
		packOS := byte(f.CreatorVersion >> 8)
		packVer := f.CreatorVersion & 0xFF

		f.Name = DecodeText([]byte(f.Name), isUTF8, packOS, packVer, f.Extra, false)
		f.Comment = DecodeText([]byte(f.Comment), isUTF8, packOS, packVer, f.Extra, true)

		// Set flag 11 so further stdlib parser checks assume UTF-8.
		f.Flags |= 0x800
		f.NonUTF8 = false
		return nil
	}
}

func DecodeText(raw []byte, isUTF8Flag bool, packOS byte, packVer uint16, extra []byte, isComment bool) string {
	if len(raw) == 0 {
		return ""
	}

	targetID := uint16(UnicodePathExtraID)
	if isComment {
		targetID = UnicodeCommentExtraID
	}

	if unicodeText := ParseUnicodeExtraField(extra, targetID, raw); unicodeText != "" {
		return unicodeText
	}

	if isUTF8Flag || packOS == CreatorUnix || packOS == CreatorMacOSX {
		return string(raw)
	}

	var dec *encoding.Decoder

	if packOS == CreatorNTFS && packVer >= 20 {
		dec = localecp.ANSIDecoder
	} else if packOS == CreatorFAT && packVer >= 25 && packVer <= 40 {
		dec = localecp.OEMDecoder
	} else if packOS == CreatorFAT || packOS == CreatorHPFS || packOS == CreatorNTFS {
		dec = localecp.OEMDecoder
	} else {
		dec = localecp.SystemDecoder
	}

	if dec != nil {
		if res, err := dec.Bytes(raw); err == nil {
			return string(res)
		}
	}

	return string(raw)
}

func ParseUnicodeExtraField(extra []byte, targetID uint16, rawData []byte) string {
	for len(extra) >= 4 {
		tag := binary.LittleEndian.Uint16(extra[:2])
		size := binary.LittleEndian.Uint16(extra[2:4])
		extra = extra[4:]

		if int(size) > len(extra) {
			break
		}

		if tag == targetID && size >= 5 {
			version := extra[0]
			if version == 1 {
				expectedCRC := binary.LittleEndian.Uint32(extra[1:5])
				actualCRC := crc32.ChecksumIEEE(rawData)

				if expectedCRC == actualCRC {
					utf8str := string(extra[5:size])
					if utf8.ValidString(utf8str) {
						return utf8str
					}
				}
			}
		}

		extra = extra[size:]
	}
	return ""
}