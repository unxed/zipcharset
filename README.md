# zipcharset

`zipcharset` is a lightweight Go micro-library designed to decode legacy, non-Unicode (non-UTF-8) filenames and comments in ZIP archives, preventing Mojibake (garbled text).

It packages a heuristics algorithm adapted from the 7-Zip and far2l projects, matching archive metadata, packing operating systems, and locale-specific codepages.

## Features

* **Unicode Extra Fields Support:** Extracts and validates the Info-ZIP Unicode Path (`0x7075`) and Unicode Comment (`0x6375`) extra fields.
* **Smart Heuristics:** Evaluates packing OS (MS-DOS, NTFS, Unix) and packer version headers to select the optimal codepage (OEM vs. ANSI).
* **Locale-Aware Fallback:** Automatically deduces standard system OEM/ANSI active codepages (e.g. CP866/CP1251 for Russian, CP932 for Japanese) on Unix (via environment variables) and Windows (via API system calls).
* **Klauspost Compress Integration:** Provides a drop-in callback for `klauspost/compress/zip.ReaderOptions`.

## Installation

```bash
go get github.com/unxed/zipcharset
```

## Usage

Integrate `zipcharset` with `github.com/klauspost/compress/zip` during reader initialization:

```go
package main

import (
	"fmt"
	"os"

	"github.com/klauspost/compress/zip"
	"github.com/unxed/zipcharset"
)

func main() {
	f, _ := os.Open("legacy_archive.zip")
	defer f.Close()

	stat, _ := f.Stat()

	// Initialize the custom name decoder
	opts := zip.ReaderOptions{
		NameDecoder: zipcharset.NewNameDecoder(),
	}

	zr, _ := zip.NewReaderWithOptions(f, stat.Size(), opts)

	for _, file := range zr.File {
		// Filenames are now safely decoded to UTF-8
		fmt.Println(file.Name)
	}
}
```

## License

This library is licensed under the 3-Clause BSD License.