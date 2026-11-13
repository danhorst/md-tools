// Package cli provides common I/O utilities for md-tools binaries.
package cli

import (
	"io"
	"os"
)

// Input returns a reader for the tool's input. If file arguments are provided,
// it opens and concatenates them. Otherwise, it returns stdin.
// Per the CLI contract: if both stdin and file arguments are provided,
// file arguments take precedence.
func Input(args []string) (io.ReadCloser, error) {
	if len(args) == 0 {
		return os.Stdin, nil
	}
	return openFiles(args)
}

// openFiles opens multiple files and returns a reader that concatenates them.
func openFiles(paths []string) (io.ReadCloser, error) {
	if len(paths) == 1 {
		return os.Open(paths[0])
	}

	readers := make([]io.Reader, len(paths))
	closers := make([]io.Closer, len(paths))

	for i, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			// Close any files we've already opened
			for j := 0; j < i; j++ {
				closers[j].Close()
			}
			return nil, err
		}
		readers[i] = f
		closers[i] = f
	}

	return &multiReadCloser{
		reader:  io.MultiReader(readers...),
		closers: closers,
	}, nil
}

type multiReadCloser struct {
	reader  io.Reader
	closers []io.Closer
}

func (m *multiReadCloser) Read(p []byte) (int, error) {
	return m.reader.Read(p)
}

func (m *multiReadCloser) Close() error {
	var firstErr error
	for _, c := range m.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
