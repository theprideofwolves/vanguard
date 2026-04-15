package reader

import (
	"fmt"
	"io"
	"os"
)

// Open returns an io.ReadCloser for the given source.
// Pass "-" to read from standard input.
// Pass any other string to open a file from disk.
func Open(source string) (io.ReadCloser, error) {
	if source == "-" {
		return os.Stdin, nil
	}

	file, err := os.Open(source)
	if err != nil {
		return nil, fmt.Errorf("reader: unable to open %q: %w", source, err)
	}
	return file, nil
}
