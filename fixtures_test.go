package fixtures_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestFixtures discovers and runs all fixture tests.
// Fixtures are organized as fixtures/<tool>/<name>.in.md and fixtures/<tool>/<name>.out.md
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

		// Construct expected output path
		outputPath := filepath.Join(dir, testName+".out.md")

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
			cmd := exec.Command(binary)
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
			cmd := exec.Command(binary)
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
