package fixtures_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// buildTool builds a single tool and returns its binary path.
func buildTool(t *testing.T, name string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", binary, "./cmd/"+name)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build %s: %v\n%s", name, err, output)
	}
	return binary
}

// TestInPlaceFlag verifies the -i FILE flag across the standard tools.
// -i reads stdin, applies the transform, and writes the result to FILE.
func TestInPlaceFlag(t *testing.T) {
	tools := []string{"mdsplit", "mdtable"}

	for _, tool := range tools {
		tool := tool
		t.Run(tool+"/happy_path", func(t *testing.T) {
			binary := buildTool(t, tool)
			out := filepath.Join(t.TempDir(), "out.md")

			input := "One sentence. Another sentence.\n"
			cmd := exec.Command(binary, "-i", out)
			cmd.Stdin = strings.NewReader(input)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			stdout, err := cmd.Output()
			if err != nil {
				t.Fatalf("unexpected error: %v\nstderr: %s", err, stderr.String())
			}
			if len(stdout) != 0 {
				t.Errorf("expected no stdout output, got %q", stdout)
			}
			got, err := os.ReadFile(out)
			if err != nil {
				t.Fatalf("file not written: %v", err)
			}
			if len(got) == 0 {
				t.Errorf("expected file to be written, got empty file")
			}
		})

		t.Run(tool+"/missing_stdin", func(t *testing.T) {
			binary := buildTool(t, tool)
			out := filepath.Join(t.TempDir(), "out.md")

			// No cmd.Stdin set → inherits /dev/null → not a piped stream.
			cmd := exec.Command(binary, "-i", out)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err == nil {
				t.Fatalf("expected error when stdin is not piped; got nil")
			}
			if !strings.Contains(stderr.String(), "stdin") {
				t.Errorf("expected stderr to mention stdin, got %q", stderr.String())
			}
			if _, err := os.Stat(out); err == nil {
				t.Errorf("expected output file not to be created on error")
			}
		})

		t.Run(tool+"/missing_file_arg", func(t *testing.T) {
			binary := buildTool(t, tool)

			cmd := exec.Command(binary, "-i")
			cmd.Stdin = strings.NewReader("hi\n")
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err == nil {
				t.Fatalf("expected error when -i is missing a file argument")
			}
			if !strings.Contains(stderr.String(), "-i") {
				t.Errorf("expected stderr to mention -i, got %q", stderr.String())
			}
		})

		t.Run(tool+"/conflicts_with_w", func(t *testing.T) {
			binary := buildTool(t, tool)
			src := filepath.Join(t.TempDir(), "src.md")
			if err := os.WriteFile(src, []byte("hi\n"), 0644); err != nil {
				t.Fatal(err)
			}

			cmd := exec.Command(binary, "-i", "-w", src)
			cmd.Stdin = strings.NewReader("hi\n")
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Run(); err == nil {
				t.Fatalf("expected error when -i and -w are combined")
			}
			if !strings.Contains(stderr.String(), "mutually exclusive") {
				t.Errorf("expected stderr to mention mutual exclusion, got %q", stderr.String())
			}
		})
	}
}

// TestVersionFlag verifies that -v and -version both print "name VERSION".
func TestVersionFlag(t *testing.T) {
	for _, tool := range []string{"mdsplit", "mdtable"} {
		tool := tool
		for _, flagName := range []string{"-v", "-version"} {
			flagName := flagName
			t.Run(tool+"/"+strings.TrimLeft(flagName, "-"), func(t *testing.T) {
				binary := buildTool(t, tool)
				cmd := exec.Command(binary, flagName)
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				got := strings.TrimSpace(string(out))
				if !strings.HasPrefix(got, tool+" ") {
					t.Errorf("expected output to start with %q, got %q", tool+" ", got)
				}
				// Version is at least vN.N.N.
				rest := strings.TrimPrefix(got, tool+" ")
				if !regexp.MustCompile(`^\d+\.\d+\.\d+`).MatchString(rest) {
					t.Errorf("expected semver version after tool name, got %q", got)
				}
			})
		}
	}
}

// TestFootnoteWrapFlag verifies mdwrap -f wraps footnote bodies to the column
// width with continuation lines indented four spaces, and leaves them on a
// single line without the flag.
func TestFootnoteWrapFlag(t *testing.T) {
	mdwrap := buildTool(t, "mdwrap")
	input := "[^1]: A footnote whose body is long enough to wrap across several lines once the column width is applied to it.\n"

	t.Run("default_leaves_single_line", func(t *testing.T) {
		cmd := exec.Command(mdwrap)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(strings.TrimSpace(string(out)), "\n") != 0 {
			t.Errorf("expected footnote to stay on one line without -f, got:\n%s", out)
		}
	})

	t.Run("flag_wraps_and_indents", func(t *testing.T) {
		cmd := exec.Command(mdwrap, "-f", "-c", "60")
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
		if len(lines) < 2 {
			t.Fatalf("expected footnote to wrap to multiple lines, got:\n%s", out)
		}
		if !strings.HasPrefix(lines[0], "[^1]: ") {
			t.Errorf("expected first line to keep the marker, got %q", lines[0])
		}
		for _, l := range lines[1:] {
			if !strings.HasPrefix(l, "    ") {
				t.Errorf("expected continuation line indented 4 spaces, got %q", l)
			}
			if utf8.RuneCountInString(l) > 60 {
				t.Errorf("line exceeds width: %q (%d)", l, utf8.RuneCountInString(l))
			}
		}
	})
}

// TestInPlaceFlagChain verifies the canonical mdsplit X | mdtable -i X form:
// stdin from the upstream pipe is captured and written to the target file.
func TestInPlaceFlagChain(t *testing.T) {
	mdsplit := buildTool(t, "mdsplit")
	mdtable := buildTool(t, "mdtable")

	target := filepath.Join(t.TempDir(), "doc.md")
	input := "Heading sentence. Second sentence.\n\n| a | b |\n| - | - |\n| 1 | 2 |\n"
	if err := os.WriteFile(target, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	// Equivalent to: mdsplit target | mdtable -i target
	splitCmd := exec.Command(mdsplit, target)
	tableCmd := exec.Command(mdtable, "-i", target)
	pipe, err := splitCmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	tableCmd.Stdin = pipe
	var tableErr bytes.Buffer
	tableCmd.Stderr = &tableErr

	if err := tableCmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := splitCmd.Run(); err != nil {
		t.Fatalf("mdsplit failed: %v", err)
	}
	if err := tableCmd.Wait(); err != nil {
		t.Fatalf("mdtable -i failed: %v\nstderr: %s", err, tableErr.String())
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "Heading sentence.\nSecond sentence.") {
		t.Errorf("expected mdsplit transform applied; got:\n%s", got)
	}
	if !strings.Contains(string(got), "| a   | b   |") {
		t.Errorf("expected mdtable transform applied (padded columns); got:\n%s", got)
	}
}
