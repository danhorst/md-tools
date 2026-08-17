package fixtures_test

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// pipe runs input through a chain of built tools and returns the final stdout.
func pipe(t *testing.T, input string, stages ...[]string) string {
	t.Helper()

	data := []byte(input)
	for _, stage := range stages {
		binary := buildTool(t, stage[0])
		cmd := exec.Command(binary, stage[1:]...)
		cmd.Stdin = bytes.NewReader(data)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%v failed: %v\nstderr: %s", stage, err, stderr.String())
		}
		data = stdout.Bytes()
	}
	return string(data)
}

// roundTripDoc exercises a footnote at a sentence end, one mid-sentence, and a
// second note, so reattachment is checked in more than one position.
const roundTripDoc = `Sidenotes belong against the text they annotate.[^1]
A mid-sentence reference[^2] should also survive the trip.

[^1]: The first note.
[^2]: The second note.
`

// TestSidenoteRoundTrip asserts that mdsidenote and mdfootnote invert each
// other, and keep doing so when a reflow tool runs between them. The reflow
// cases are the point: the pairing exists to be used inside a pipeline.
func TestSidenoteRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		relay [][]string
	}{
		{"direct", nil},
		{"through mdjoin", [][]string{{"mdjoin"}}},
		{"through mdsplit", [][]string{{"mdsplit"}}},
		{"through mdwrap", [][]string{{"mdwrap", "-c", "60"}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stages := [][]string{{"mdsidenote"}}
			stages = append(stages, tc.relay...)
			stages = append(stages, []string{"mdfootnote"})

			got := pipe(t, roundTripDoc, stages...)

			// Reflow legitimately changes line breaks, so compare the
			// recovered structure rather than exact bytes.
			for _, want := range []string{"[^1]", "[^2]", "[^1]: The first note.", "[^2]: The second note."} {
				if !strings.Contains(got, want) {
					t.Errorf("recovered document is missing %q\n--- got\n%s", want, got)
				}
			}

			if strings.Contains(got, `class="sidenote"`) {
				t.Errorf("sidenote markup survived recovery\n--- got\n%s", got)
			}

			// The reference must land against its referent, not adrift.
			if !strings.Contains(got, "annotate.[^1]") {
				t.Errorf("reference not reattached to its text\n--- got\n%s", got)
			}
		})
	}
}

// TestSidenoteMarkerAttached pins the emitted layout: the marker follows the
// text the reference followed, with nothing between them.
func TestSidenoteMarkerAttached(t *testing.T) {
	got := pipe(t, roundTripDoc, []string{"mdsidenote"})

	if !strings.Contains(got, "annotate.<label") {
		t.Errorf("marker is not attached to the preceding text\n--- got\n%s", got)
	}
	if strings.Contains(got, "annotate.\n<label") {
		t.Errorf("marker is still on its own line\n--- got\n%s", got)
	}
}

// TestSidenoteMarkerRendersTight guards the reason for all of this: a newline
// inside a paragraph is a space, so a detached marker renders detached.
func TestSidenoteMarkerRendersTight(t *testing.T) {
	got := pipe(t, roundTripDoc, []string{"mdsidenote"}, []string{"mdhtml", "-fragment", "-no-css"})

	if !strings.Contains(got, "annotate.<label") {
		t.Errorf("rendered output has whitespace before the marker\n--- got\n%s", got)
	}
}

// TestWrapKeepsTagsWhole asserts mdwrap treats a tag as one token. Splitting a
// tag on the spaces between its attributes is what defeats mdfootnote.
func TestWrapKeepsTagsWhole(t *testing.T) {
	input := "Text before the tag.<label for=\"sidenote-1\" class=\"margin-toggle sidenote-number\"></label>\n" +
		"<span class=\"sidenote\">A note.</span>\n" +
		"Text after.\n"

	got := pipe(t, input, []string{"mdwrap", "-c", "40"})

	for _, line := range strings.Split(got, "\n") {
		opens := strings.Count(line, "<")
		closes := strings.Count(line, ">")
		if opens != closes {
			t.Errorf("line splits a tag (%d '<' vs %d '>'): %q\n--- got\n%s", opens, closes, line, got)
		}
	}
}

// TestWrapLeavesComparisonsAlone verifies an angle bracket that opens no tag
// is ordinary text, so it cannot absorb the rest of the paragraph.
func TestWrapLeavesComparisonsAlone(t *testing.T) {
	input := "When a < b and 5<10 the wrap should behave exactly as it does for any other prose in this paragraph.\n"

	got := pipe(t, input, []string{"mdwrap", "-c", "30"})

	if lines := strings.Count(strings.TrimRight(got, "\n"), "\n") + 1; lines < 3 {
		t.Errorf("expected the paragraph to wrap across several lines, got %d\n--- got\n%s", lines, got)
	}
	if !strings.Contains(got, "a < b") {
		t.Errorf("comparison text was altered\n--- got\n%s", got)
	}
}

// TestFixturesInvert asserts the pair inverts on the committed fixtures. A
// golden alone cannot catch a recovery bug — mdfootnote's greedy capture was
// swallowing two paragraphs and a whole note into one definition, and the
// golden simply committed that output as correct.
func TestFixturesInvert(t *testing.T) {
	t.Run("simple is byte-identical", func(t *testing.T) {
		source, err := os.ReadFile("fixtures/mdsidenote/simple.in.md")
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}

		got := pipe(t, string(source), []string{"mdsidenote"}, []string{"mdfootnote"})
		if got != string(source) {
			t.Errorf("round trip altered the document\n--- want\n%s\n--- got\n%s", source, got)
		}
	})

	// complex contains reference-style links inside a note. mdsidenote inlines
	// those when it builds the markup, discarding the label, so mdfootnote has
	// nothing to restore the reference form from. That bound is upstream and
	// documented; everything else must still survive.
	t.Run("complex recovers structurally", func(t *testing.T) {
		source, err := os.ReadFile("fixtures/mdsidenote/complex.in.md")
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}

		got := pipe(t, string(source), []string{"mdsidenote"}, []string{"mdfootnote"})

		for _, want := range []string{"[^1]:", "[^2]:"} {
			if !strings.Contains(got, want) {
				t.Errorf("definition %q was lost\n--- got\n%s", want, got)
			}
		}

		// The prose that the greedy capture used to absorb must stay prose.
		const absorbed = "In lieu of a site builder"
		if !strings.Contains(got, absorbed) {
			t.Fatalf("body text %q vanished\n--- got\n%s", absorbed, got)
		}
		for _, line := range strings.Split(got, "\n") {
			if strings.HasPrefix(line, "[^") && strings.Contains(line, absorbed) {
				t.Errorf("body text was absorbed into a footnote definition:\n%s", line)
			}
		}

		if strings.Contains(got, `class="sidenote"`) {
			t.Errorf("sidenote markup survived recovery\n--- got\n%s", got)
		}
	})
}
