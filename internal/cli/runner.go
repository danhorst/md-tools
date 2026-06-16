package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// TransformFunc is a function that transforms input content to output content.
type TransformFunc func(string) string

// Flags holds the standard md-tools flags. Register them with RegisterFlags
// before flag.Parse(), then pass the populated struct to Run.
type Flags struct {
	WriteInPlace bool
	InPlace      bool
	ShowVersion  bool
}

// RegisterFlags registers -w, -i, -v, and -version on the default flag set
// and returns a Flags whose fields are populated by flag.Parse().
func RegisterFlags() *Flags {
	f := &Flags{}
	flag.BoolVar(&f.WriteInPlace, "w", false, "write result to file instead of stdout")
	flag.BoolVar(&f.InPlace, "i", false, "read stdin and write result to the file argument")
	flag.BoolVar(&f.ShowVersion, "v", false, "print version and exit")
	flag.BoolVar(&f.ShowVersion, "version", false, "print version and exit")
	flag.Usage = alignedUsage
	return f
}

// alignedUsage prints flag descriptions with all flag names padded to the same
// column, so help output stays visually consistent across long and short names.
func alignedUsage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Usage of %s:\n", os.Args[0])
	var widest int
	flag.VisitAll(func(f *flag.Flag) {
		name := "-" + f.Name
		if _, isBool := f.Value.(interface{ IsBoolFlag() bool }); !isBool {
			name += " " + flagTypeName(f)
		}
		if len(name) > widest {
			widest = len(name)
		}
	})
	flag.VisitAll(func(f *flag.Flag) {
		name := "-" + f.Name
		if _, isBool := f.Value.(interface{ IsBoolFlag() bool }); !isBool {
			name += " " + flagTypeName(f)
		}
		fmt.Fprintf(w, "  %-*s  %s", widest, name, f.Usage)
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			fmt.Fprintf(w, " (default %s)", f.DefValue)
		}
		fmt.Fprintln(w)
	})
}

// flagTypeName returns the placeholder name for a non-bool flag, picking up
// backtick-quoted names from the usage string (matching flag.UnquoteUsage).
func flagTypeName(f *flag.Flag) string {
	name, _ := flag.UnquoteUsage(f)
	if name == "" {
		return "value"
	}
	return name
}

// Run executes a CLI tool with the standard md-tools interface. It dispatches
// on the parsed flags: -v prints the version; -w writes the result back to each
// file argument; -i reads stdin and writes the result to the single file
// argument. The default reads from files (or stdin) and writes to stdout.
func Run(toolName string, flags *Flags, args []string, transform TransformFunc) error {
	if flags.ShowVersion {
		fmt.Println(toolName, Version)
		return nil
	}

	if flags.WriteInPlace && flags.InPlace {
		return fmt.Errorf("-w and -i are mutually exclusive")
	}

	if flags.InPlace {
		if len(args) != 1 {
			return fmt.Errorf("-i requires exactly one file argument")
		}
		if isStdinTerminal() {
			return fmt.Errorf("-i requires data on stdin")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		result := transform(string(data))
		return os.WriteFile(args[0], []byte(result), 0644)
	}

	if flags.WriteInPlace {
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
