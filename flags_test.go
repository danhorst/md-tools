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

// writeSVGDoc writes an SVG and a markdown file referencing it into dir,
// returning the markdown path.
func writeSVGDoc(t *testing.T, dir, name, svg string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name+".svg"), []byte(svg), 0644); err != nil {
		t.Fatalf("write svg: %v", err)
	}
	mdPath := filepath.Join(dir, name+".md")
	if err := os.WriteFile(mdPath, []byte("![x]("+name+".svg)\n"), 0644); err != nil {
		t.Fatalf("write md: %v", err)
	}
	return mdPath
}

// TestInlineSVGRefusesActiveContent covers the trust boundary that -inline-svg
// crosses: a browser runs no scripts in an <img>, but does once the same file
// is inlined into the document.
func TestInlineSVGRefusesActiveContent(t *testing.T) {
	binary := buildTool(t, "mdhtml")

	cases := []struct {
		name    string
		svg     string
		wantMsg string
	}{
		{"script", `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`, "<script>"},
		{"handler", `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect/></svg>`, "onload="},
		{"foreignobject", `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><div/></foreignObject></svg>`, "htmlLabels"},
		{"jsurl", `<svg xmlns="http://www.w3.org/2000/svg"><a href="javascript:alert(1)"><rect/></a></svg>`, "javascript:"},
		{"entity_jsurl", `<svg xmlns="http://www.w3.org/2000/svg"><a href="&#106;avascript:alert(1)"><rect/></a></svg>`, "javascript:"},
		{"external_use", `<svg xmlns="http://www.w3.org/2000/svg"><use href="https://e.test/x.svg#a"/></svg>`, "another document"},
		{"smil", `<svg xmlns="http://www.w3.org/2000/svg"><a><animate attributeName="href" to="javascript:x"/></a></svg>`, "link target"},
		{"css_import", `<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://e.test/x.css);</style></svg>`, "@import"},
		{"malformed", `<svg xmlns="http://www.w3.org/2000/svg"><rect></svg>`, "cannot parse"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mdPath := writeSVGDoc(t, dir, tc.name, tc.svg)

			var stdout, stderr bytes.Buffer
			cmd := exec.Command(binary, "-inline-svg", "-base", dir, mdPath)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()

			if err == nil {
				t.Fatalf("expected refusal, got exit 0\nstdout: %s", stdout.String())
			}
			msg := stderr.String()
			if !strings.Contains(msg, tc.wantMsg) {
				t.Errorf("error should mention %q, got: %s", tc.wantMsg, msg)
			}
			if !strings.Contains(msg, tc.name+".svg") {
				t.Errorf("error should name the file, got: %s", msg)
			}
			if !strings.Contains(msg, "-unsafe-svg") {
				t.Errorf("error should offer the override, got: %s", msg)
			}
		})
	}
}

// TestInlineSVGUnsafeOverride verifies -unsafe-svg skips inspection.
func TestInlineSVGUnsafeOverride(t *testing.T) {
	binary := buildTool(t, "mdhtml")
	dir := t.TempDir()
	mdPath := writeSVGDoc(t, dir, "bad", `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)

	out, err := exec.Command(binary, "-fragment", "-inline-svg", "-unsafe-svg", "-base", dir, mdPath).Output()
	if err != nil {
		t.Fatalf("-unsafe-svg should inline anyway: %v", err)
	}
	if !strings.Contains(string(out), "<script>") {
		t.Errorf("expected the SVG inlined verbatim, got: %s", out)
	}
}

// TestInlineSVGAcceptsOrdinaryDiagram guards against the vetting becoming so
// strict it rejects the diagrams this feature exists to render.
func TestInlineSVGAcceptsOrdinaryDiagram(t *testing.T) {
	binary := buildTool(t, "mdhtml")
	dir := t.TempDir()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" id="c"><style>#c .b{fill:#000}</style>` +
		`<defs><marker id="a"><path d="M0,0"/></marker></defs>` +
		`<rect class="b"/><line marker-end="url(#a)"/><text>a&nbsp;b &amp; c</text></svg>`
	mdPath := writeSVGDoc(t, dir, "diagram", svg)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, "-fragment", "-inline-svg", "-base", dir, mdPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("ordinary diagram refused: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "<svg") {
		t.Errorf("expected inlined svg, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("expected no warnings, got: %s", stderr.String())
	}
}

// TestInlineSVGRemoteReference verifies a remote reference is left as an <img>
// and reported, without polluting stdout.
func TestInlineSVGRemoteReference(t *testing.T) {
	binary := buildTool(t, "mdhtml")
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "remote.md")
	if err := os.WriteFile(mdPath, []byte("![r](https://e.test/a.svg)\n"), 0644); err != nil {
		t.Fatalf("write md: %v", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, "-fragment", "-inline-svg", "-base", dir, mdPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("remote reference should not fail: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), `<img src="https://e.test/a.svg"`) {
		t.Errorf("remote reference should stay an img, got: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "mdhtml:") {
		t.Errorf("warning leaked into stdout: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "not self-contained") {
		t.Errorf("expected a warning on stderr, got: %s", stderr.String())
	}
}

// TestInlineSVGExternalResourceWarns verifies an external resource is reported
// but does not block inlining: it leaks the reader's address without executing.
func TestInlineSVGExternalResourceWarns(t *testing.T) {
	binary := buildTool(t, "mdhtml")
	dir := t.TempDir()
	mdPath := writeSVGDoc(t, dir, "ext",
		`<svg xmlns="http://www.w3.org/2000/svg"><image href="https://e.test/x.png"/></svg>`)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binary, "-fragment", "-inline-svg", "-base", dir, mdPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("external resource should warn, not refuse: %v\nstderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "<svg") {
		t.Errorf("expected the svg inlined, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "external resource") {
		t.Errorf("expected a warning, got: %s", stderr.String())
	}
}

// TestWriteInPlaceRefused verifies mdhtml refuses -w, which elsewhere in the
// toolkit rewrites the input file and here would replace Markdown with HTML.
func TestWriteInPlaceRefused(t *testing.T) {
	binary := buildTool(t, "mdhtml")
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	original := "# Title\n"
	if err := os.WriteFile(mdPath, []byte(original), 0644); err != nil {
		t.Fatalf("write md: %v", err)
	}

	var stderr bytes.Buffer
	cmd := exec.Command(binary, "-w", mdPath)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		t.Fatal("expected -w to be refused")
	}
	if !strings.Contains(stderr.String(), "-i FILE") {
		t.Errorf("error should suggest the alternative, got: %s", stderr.String())
	}

	after, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(after) != original {
		t.Errorf("source file was modified: %q", after)
	}
}
