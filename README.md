# zipcharset

`zipcharset` is an advanced Go decoder adapter for ZIP archives that resolves legacy, non-Unicode filenames and comments, preventing Mojibake.

It relies on the zero-dependency library `localecp` for system-locale codepage deduction, while layering ZIP-specific heuristics and extra-fields extraction on top.

## Features

* **ZIP-Specific Heuristics:** Automatically evaluates packer version, flags, and packing OS metadata to determine if ANSI (Windows) or OEM (DOS) translation should apply.
* **Unicode Extra Fields Extraction:** Safely parses and validates Info-ZIP Unicode Path (`0x7075`) and Unicode Comment (`0x6375`) structures.
* **Seamless Integration:** Exposes a simple `zip.ReaderOptions` callback function for the `klauspost/compress/zip` library.

## Dependencies

* `github.com/unxed/localecp` (zero-dependency locale mapping engine)
* `github.com/klauspost/compress`

## Installation

```bash
go get github.com/unxed/zipcharset
```

## Usage

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

	opts := zip.ReaderOptions{
		NameDecoder: zipcharset.NewNameDecoder(),
	}

	zr, _ := zip.NewReaderWithOptions(f, stat.Size(), opts)

	for _, file := range zr.File {
		fmt.Println(file.Name)
	}
}
```

## License

Licensed under the 3-Clause BSD License.