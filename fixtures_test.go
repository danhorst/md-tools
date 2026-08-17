package fixtures_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// findExpectedOutput locates the golden file for a fixture. Markdown filters
// pair <name>.in.md with <name>.out.md; renderers pair it with another
// extension, so the sibling is discovered rather than assumed.
func findExpectedOutput(dir, testName string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, testName+".out.*"))
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no .out.* file for fixture %s", testName)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple .out.* files for fixture %s: %v", testName, matches)
	}
}

// readFixtureFlags returns the arguments in an optional <name>.flags file,
// which lets a fixture exercise a tool's flags. Fixtures without one run the
// tool bare.
func readFixtureFlags(dir, testName string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(dir, testName+".flags"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(data)), nil
}

// TestFixtures discovers and runs all fixture tests.
// Fixtures are organized as fixtures/<tool>/<name>.in.md and a sibling
// fixtures/<tool>/<name>.out.<ext> golden file.
func TestFixtures(t *testing.T) {
	// Find all .in.md files
	inputs, err := filepath.Glob("fixtures/*/*.in.md")
	if err != nil {
		t.Fatalf("failed to glob fixtures: %v", err)
	}

	if len(inputs) == 0 {
		t.Fatal("no fixtures found")
	}

	// Build all tools first
	tools := make(map[string]string) // tool name -> binary path
	toolDirs, err := filepath.Glob("cmd/*")
	if err != nil {
		t.Fatalf("failed to glob cmd: %v", err)
	}

	for _, toolDir := range toolDirs {
		toolName := filepath.Base(toolDir)
		mainFile := filepath.Join(toolDir, "main.go")
		if _, err := os.Stat(mainFile); err != nil {
			continue
		}

		// Build to temp location
		binary := filepath.Join(t.TempDir(), toolName)
		cmd := exec.Command("go", "build", "-o", binary, "./"+toolDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("failed to build %s: %v\n%s", toolName, err, output)
		}
		tools[toolName] = binary
	}

	for _, inputPath := range inputs {
		// Extract tool name and test name
		dir := filepath.Dir(inputPath)
		toolName := filepath.Base(dir)
		baseName := filepath.Base(inputPath)
		testName := strings.TrimSuffix(baseName, ".in.md")

		// Locate the expected output. The extension varies: markdown filters
		// use .out.md, renderers emit another format (e.g. .out.html).
		outputPath, err := findExpectedOutput(dir, testName)
		if err != nil {
			t.Fatalf("%s: %v", inputPath, err)
		}

		// Optional per-fixture flags.
		fixtureArgs, err := readFixtureFlags(dir, testName)
		if err != nil {
			t.Fatalf("%s: %v", inputPath, err)
		}

		t.Run(toolName+"/"+testName, func(t *testing.T) {
			binary, ok := tools[toolName]
			if !ok {
				t.Skipf("no binary for tool %s", toolName)
			}

			// Read input
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("failed to read input: %v", err)
			}

			// Read expected output
			expected, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read expected output: %v", err)
			}

			// Run tool
			cmd := exec.Command(binary, fixtureArgs...)
			cmd.Stdin = bytes.NewReader(input)
			actual, err := cmd.Output()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					t.Fatalf("tool failed: %v\nstderr: %s", err, exitErr.Stderr)
				}
				t.Fatalf("tool failed: %v", err)
			}

			// Compare
			if !bytes.Equal(actual, expected) {
				t.Errorf("output mismatch\n--- expected\n%s\n--- actual\n%s", expected, actual)
			}
		})

		// Idempotency only applies when the output is markdown. T(T(x)) == T(x)
		// is undefined for a tool that renders into another format.
		if filepath.Ext(outputPath) != ".md" {
			continue
		}

		// Also test idempotency: T(T(input)) == T(input)
		t.Run(toolName+"/"+testName+"/idempotent", func(t *testing.T) {
			binary, ok := tools[toolName]
			if !ok {
				t.Skipf("no binary for tool %s", toolName)
			}

			// Read expected output (which is T(input))
			firstPass, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read expected output: %v", err)
			}

			// Run tool on expected output
			cmd := exec.Command(binary, fixtureArgs...)
			cmd.Stdin = bytes.NewReader(firstPass)
			secondPass, err := cmd.Output()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					t.Fatalf("tool failed on second pass: %v\nstderr: %s", err, exitErr.Stderr)
				}
				t.Fatalf("tool failed on second pass: %v", err)
			}

			// T(T(input)) should equal T(input)
			if !bytes.Equal(secondPass, firstPass) {
				t.Errorf("not idempotent\n--- first pass\n%s\n--- second pass\n%s", firstPass, secondPass)
			}
		})
	}
}
