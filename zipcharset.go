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
	unicodePathExtraID    = 0x7075
	unicodeCommentExtraID = 0x6375

	creatorFAT    = 0
	creatorHPFS   = 6
	creatorNTFS   = 11
	creatorUnix   = 3
	creatorMacOSX = 19
)

// NewNameDecoder returns an opt-in callback function to rewrite legacy,
// non-Unicode filenames and comments into clean UTF-8.
func NewNameDecoder() func(f *zip.FileHeader) error {
	return func(f *zip.FileHeader) error {
		isUTF8 := f.Flags&0x800 != 0
		packOS := byte(f.CreatorVersion >> 8)
		packVer := f.CreatorVersion & 0xFF

		f.Name = decodeBytes([]byte(f.Name), isUTF8, packOS, packVer, f.Extra, false)
		f.Comment = decodeBytes([]byte(f.Comment), isUTF8, packOS, packVer, f.Extra, true)

		// Set flag 11 so further stdlib parser checks assume UTF-8.
		f.Flags |= 0x800
		f.NonUTF8 = false
		return nil
	}
}

func decodeBytes(raw []byte, isUTF8 bool, packOS byte, packVer uint16, extra []byte, isComment bool) string {
	if len(raw) == 0 {
		return ""
	}

	targetID := uint16(unicodePathExtraID)
	if isComment {
		targetID = unicodeCommentExtraID
	}

	if unicodeText := parseUnicodeExtraField(extra, targetID, raw); unicodeText != "" {
		return unicodeText
	}

	if isUTF8 || packOS == creatorUnix || packOS == creatorMacOSX {
		return string(raw)
	}

	var dec *encoding.Decoder

	// Delegate to the pure localecp decoders
	if packOS == creatorNTFS && packVer >= 20 {
		dec = localecp.ANSIDecoder
	} else if packOS == creatorFAT && packVer >= 25 && packVer <= 40 {
		dec = localecp.OEMDecoder
	} else if packOS == creatorFAT || packOS == creatorHPFS || packOS == creatorNTFS {
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

func parseUnicodeExtraField(extra []byte, targetID uint16, rawData []byte) string {
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