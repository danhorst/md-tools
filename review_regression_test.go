package fixtures_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderFailurePreservesOutput(t *testing.T) {
	binary := buildTool(t, "mdhtml")
	dir := t.TempDir()
	target := filepath.Join(dir, "output.html")
	for _, existing := range []bool{false, true} {
		os.Remove(target)
		if existing {
			if err := os.WriteFile(target, []byte("keep me"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(binary, "-inline-svg", "-base", dir, "-i", target)
		cmd.Stdin = strings.NewReader("![x](missing.svg)\n")
		if err := cmd.Run(); err == nil {
			t.Fatal("expected missing SVG error")
		}
		data, err := os.ReadFile(target)
		if existing {
			if err != nil || string(data) != "keep me" {
				t.Fatalf("output changed: %q, %v", data, err)
			}
		} else if !os.IsNotExist(err) {
			t.Fatalf("failed render created output: %v", err)
		}
	}
	// An output argument is not a source path and must not imply an SVG base.
	cmd := exec.Command(binary, "-inline-svg", "-i", target)
	cmd.Stdin = strings.NewReader("text\n")
	if err := cmd.Run(); err == nil {
		t.Fatal("stdin SVG mode must require -base")
	}
}

func TestMultipleInputFiles(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.md")
	for path, data := range map[string]string{a: "First.\n\n", b: "Second.\n"} {
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tools, err := filepath.Glob("cmd/*")
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		if _, err := os.Stat(filepath.Join(tool, "main.go")); err != nil {
			continue
		}
		t.Run(filepath.Base(tool), func(t *testing.T) {
			binary := buildTool(t, filepath.Base(tool))
			cmd := exec.Command(binary, a, b)
			cmd.Stdin = strings.NewReader("ignored stdin")
			out, err := cmd.Output()
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), "First.") || !strings.Contains(string(out), "Second.") || strings.Contains(string(out), "ignored stdin") {
				t.Fatalf("missing input: %s", out)
			}
			cmd = exec.Command(binary, a, filepath.Join(dir, "missing.md"))
			out, err = cmd.Output()
			if err == nil || len(out) != 0 {
				t.Fatalf("missing later input: output %q, err %v", out, err)
			}
		})
	}
}

func TestSidenoteRenumberDeterministic(t *testing.T) {
	binary := buildTool(t, "mdsidenote")
	input := "[a][2] [b][3]\n\n[2]: /a\n[3]: /b\n"
	want := "[a][1] [b][2]\n\n[1]: /a\n[2]: /b\n"
	for i := 0; i < 40; i++ {
		cmd := exec.Command(binary)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.Output()
		if err != nil || string(out) != want {
			t.Fatalf("run %d: %q, %v", i, out, err)
		}
	}
}
