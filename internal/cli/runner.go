package cli

import (
	"fmt"
	"io"
	"os"
)

// TransformFunc is a function that transforms input content to output content.
type TransformFunc func(string) string

// Run executes a CLI tool with the standard md-tools interface.
// It handles -w (write in place) and -i (read stdin, write to file) modes,
// stdin/file input, and stdout output. The toolName is used in error messages.
func Run(args []string, writeInPlace bool, inPlaceFile string, toolName string, transform TransformFunc) error {
	if writeInPlace && inPlaceFile != "" {
		return fmt.Errorf("-w and -i are mutually exclusive")
	}

	if inPlaceFile != "" {
		if len(args) > 0 {
			return fmt.Errorf("-i does not take file arguments; it reads from stdin")
		}
		if isStdinTerminal() {
			return fmt.Errorf("-i requires data on stdin")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		result := transform(string(data))
		return os.WriteFile(inPlaceFile, []byte(result), 0644)
	}

	if writeInPlace {
		if len(args) == 0 {
			return fmt.Errorf("-w requires at least one file argument")
		}
		for _, path := range args {
			if err := processFile(path, transform); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		return nil
	}

	// Default: read from files or stdin, write to stdout
	var input io.ReadCloser
	if len(args) == 0 {
		input = os.Stdin
	} else {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		input = f
	}
	defer input.Close()

	data, err := io.ReadAll(input)
	if err != nil {
		return err
	}

	result := transform(string(data))
	_, err = os.Stdout.WriteString(result)
	return err
}

// isStdinTerminal reports whether stdin is attached to a terminal (no piped data).
func isStdinTerminal() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// processFile transforms a file in place, only writing if content changed.
func processFile(path string, transform TransformFunc) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result := transform(string(data))

	// Only write if content changed
	if result == string(data) {
		return nil
	}

	return os.WriteFile(path, []byte(result), 0644)
}
