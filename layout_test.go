package fixtures_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The stylesheet has no other coverage. Six layout defects surfaced during its
// development — a sidenote counter rendering 0, a doubled footnote divider, a
// code block overhanging the measure, a narrow-screen media query leaking into
// print, a checkbox misaligned with the bullets beside it, and an intra-note gap
// equal to the inter-note gap — and every fixture test passed through all six,
// because they only compare markup.
//
// These assert geometric relationships rather than pixel values, so they survive
// Chrome updates and font substitution. Chrome is driven through --dump-dom
// rather than a CDP library: keeping this repo dependency-light is why the
// browser-dependent tools live elsewhere.

// findChrome locates a browser, or skips. Layout coverage is a local guard, not
// a CI gate.
func findChrome(t *testing.T) string {
	t.Helper()

	if p := os.Getenv("MD_TOOLS_CHROME"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		t.Skipf("MD_TOOLS_CHROME=%s does not exist", p)
	}

	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}

	t.Skip("no Chrome found; set MD_TOOLS_CHROME to run layout tests")
	return ""
}

// box is one element's geometry, in CSS pixels.
type box struct {
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
}

type layout struct {
	Paragraph  box       `json:"paragraph"`
	Pre        box       `json:"pre"`
	Blockquote box       `json:"blockquote"`
	Alert      box       `json:"alert"`
	Table      box       `json:"table"`
	Rule       box       `json:"rule"`
	NoteText   []box     `json:"noteText"`
	NoteMarker []float64 `json:"noteMarker"`
	Sidenotes  []box     `json:"sidenotes"`
	BulletText box       `json:"bulletText"`
	TaskText   box       `json:"taskText"`
	Checkbox   box       `json:"checkbox"`
	ScrollW    float64   `json:"scrollW"`
	ClientW    float64   `json:"clientW"`
}

// probe is spliced into the rendered page and reports geometry as JSON.
const probe = `<script>
(function () {
  const q = s => document.querySelector(s);
  const qa = s => [...document.querySelectorAll(s)];
  const R = el => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return {left: r.left, right: r.right, top: r.top, bottom: r.bottom};
  };
  // The li box does not move when a marker sits inside it, so measure where
  // the text actually starts.
  const textBox = el => {
    if (!el) return null;
    // Walk the subtree: a footnote's body is wrapped in a <p>, a list item's
    // is not, and both need the same measurement.
    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
    let node = null;
    while (walker.nextNode()) {
      if (walker.currentNode.textContent.trim()) { node = walker.currentNode; break; }
    }
    if (!node) return null;
    const r = document.createRange();
    r.selectNodeContents(node);
    const rect = r.getBoundingClientRect();
    return {left: rect.left, right: rect.right, top: rect.top, bottom: rect.bottom};
  };
  const notes = qa('.footnotes li');
  const bulletLi = qa('li').find(l => !l.querySelector('input[type=checkbox]') && l.closest('ul'));
  const taskLi = qa('li').find(l => l.querySelector('input[type=checkbox]'));
  const out = {
    paragraph: R(q('body > p')),
    pre: R(q('pre')),
    blockquote: R(q('blockquote:not(.markdown-alert)')),
    alert: R(q('.markdown-alert')),
    table: R(q('table')),
    rule: R(q('body > hr')),
    noteText: notes.map(textBox),
    // A pseudo-element has no rect; its used 'right' proves the alignment.
    noteMarker: notes.map(n => parseFloat(getComputedStyle(n, '::before').right)),
    sidenotes: qa('.sidenote').map(R),
    bulletText: textBox(bulletLi),
    taskText: textBox(taskLi),
    checkbox: R(q('li > input[type=checkbox]')),
    scrollW: document.documentElement.scrollWidth,
    clientW: document.documentElement.clientWidth
  };
  const d = document.createElement('div');
  d.id = 'md-tools-probe';
  d.textContent = JSON.stringify(out);
  document.body.appendChild(d);
})();
</script></body>`

var probeRe = regexp.MustCompile(`(?s)id="md-tools-probe">(.*?)</div>`)

// measure renders the showcase at a viewport width and reports its geometry.
func measure(t *testing.T, width int) layout {
	t.Helper()

	chrome := findChrome(t)
	mdhtml := buildTool(t, "mdhtml")

	var page bytes.Buffer
	cmd := exec.Command(mdhtml, "-inline-svg", "examples/showcase.md")
	cmd.Stdout = &page
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("rendering showcase: %v\nstderr: %s", err, stderr.String())
	}

	html := strings.Replace(page.String(), "</body>", probe, 1)
	if !strings.Contains(html, "md-tools-probe") {
		t.Fatal("could not splice the probe: no </body> in the rendered page")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "showcase.html")
	if err := os.WriteFile(path, []byte(html), 0644); err != nil {
		t.Fatalf("write page: %v", err)
	}

	var dom bytes.Buffer
	run := exec.Command(chrome,
		"--headless", "--disable-gpu", "--no-sandbox",
		fmt.Sprintf("--window-size=%d,1400", width),
		"--virtual-time-budget=2000",
		"--dump-dom", path,
	)
	run.Stdout = &dom
	if err := run.Run(); err != nil {
		t.Fatalf("chrome: %v", err)
	}

	m := probeRe.FindSubmatch(dom.Bytes())
	if m == nil {
		t.Fatal("probe produced no output; the page may not have loaded")
	}

	var got layout
	if err := json.Unmarshal(m[1], &got); err != nil {
		t.Fatalf("decoding probe output: %v\n%s", err, m[1])
	}
	return got
}

// mustFind fails loudly when a selector matched nothing, rather than letting a
// zero box masquerade as an element sitting at the origin.
func mustFind(t *testing.T, name string, b box) box {
	t.Helper()
	if b == (box{}) {
		t.Fatalf("%s not found in the rendered page", name)
	}
	return b
}

func closeTo(a, b, tol float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= tol
}

// TestLayoutMainColumn asserts every block in the main column shares one right
// edge. A padded block overhanging it is what pushed code blocks into the
// sidenote gutter.
func TestLayoutMainColumn(t *testing.T) {
	l := measure(t, 1400)

	para := mustFind(t, "paragraph", l.Paragraph)
	edges := map[string]box{
		"pre":        l.Pre,
		"blockquote": l.Blockquote,
		"alert":      l.Alert,
		"table":      l.Table,
		"hr":         l.Rule,
	}
	for name, b := range edges {
		got := mustFind(t, name, b)
		if !closeTo(got.Right, para.Right, 1) {
			t.Errorf("%s right edge is %.1f, paragraph is %.1f", name, got.Right, para.Right)
		}
	}
}

// TestLayoutFootnotes asserts note text sits on the prose edge and the hanging
// markers right-align with each other.
func TestLayoutFootnotes(t *testing.T) {
	l := measure(t, 1400)

	if len(l.NoteText) < 10 {
		t.Fatalf("showcase should carry at least ten notes to exercise double digits, found %d", len(l.NoteText))
	}

	for i, n := range l.NoteText {
		if !closeTo(n.Left, l.Paragraph.Left, 1) {
			t.Errorf("note %d text starts at %.1f, paragraph starts at %.1f", i+1, n.Left, l.Paragraph.Left)
		}
	}

	// Every marker is positioned from the same right offset, which is what
	// makes 1. and 10. line up regardless of width.
	for i, r := range l.NoteMarker {
		if !closeTo(r, l.NoteMarker[0], 0.5) {
			t.Errorf("note %d marker offset is %.1f, note 1 is %.1f", i+1, r, l.NoteMarker[0])
		}
	}
}

// TestLayoutSidenotes asserts sidenotes sit in the margin, clear of the column.
func TestLayoutSidenotes(t *testing.T) {
	l := measure(t, 1400)

	if len(l.Sidenotes) == 0 {
		t.Fatal("showcase should carry sidenotes")
	}
	for i, s := range l.Sidenotes {
		if s.Left <= l.Paragraph.Right {
			t.Errorf("sidenote %d starts at %.1f, inside the column ending at %.1f", i+1, s.Left, l.Paragraph.Right)
		}
	}
	// All sidenotes share one left edge, so the margin reads as a column.
	for i, s := range l.Sidenotes {
		if !closeTo(s.Left, l.Sidenotes[0].Left, 1) {
			t.Errorf("sidenote %d starts at %.1f, sidenote 1 at %.1f", i+1, s.Left, l.Sidenotes[0].Left)
		}
	}
}

// TestLayoutListMarkers asserts a task item's label starts where a bulleted
// item's does, with the checkbox out in the marker column rather than in the
// text. Leaving the checkbox in the flow pushed its label 8px right.
func TestLayoutListMarkers(t *testing.T) {
	l := measure(t, 1400)

	mustFind(t, "bullet item", l.BulletText)
	mustFind(t, "task item", l.TaskText)
	mustFind(t, "checkbox", l.Checkbox)

	if !closeTo(l.TaskText.Left, l.BulletText.Left, 1) {
		t.Errorf("task text starts at %.1f, bullet text at %.1f", l.TaskText.Left, l.BulletText.Left)
	}
	if l.Checkbox.Right > l.TaskText.Left+1 {
		t.Errorf("checkbox extends to %.1f, past the text edge at %.1f", l.Checkbox.Right, l.TaskText.Left)
	}
}

// TestLayoutNoHorizontalScroll asserts nothing overflows at any width. Hanging
// markers reach outside the measure, so they are a standing risk here.
func TestLayoutNoHorizontalScroll(t *testing.T) {
	for _, width := range []int{1400, 1100, 900} {
		t.Run(fmt.Sprintf("%dpx", width), func(t *testing.T) {
			l := measure(t, width)
			if l.ScrollW > l.ClientW {
				t.Errorf("document scrolls horizontally: scrollWidth %.0f > clientWidth %.0f", l.ScrollW, l.ClientW)
			}
		})
	}
}

// TestPrintSmoke renders to PDF. It does not assert print geometry, only that a
// document comes out — enough to catch the print stylesheet failing outright,
// as it did when a max-width query matched the page box and undid the layout.
func TestPrintSmoke(t *testing.T) {
	chrome := findChrome(t)
	mdhtml := buildTool(t, "mdhtml")

	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "showcase.html")
	pdfPath := filepath.Join(dir, "showcase.pdf")

	page, err := exec.Command(mdhtml, "-inline-svg", "examples/showcase.md").Output()
	if err != nil {
		t.Fatalf("rendering showcase: %v", err)
	}
	if err := os.WriteFile(htmlPath, page, 0644); err != nil {
		t.Fatalf("write page: %v", err)
	}

	cmd := exec.Command(chrome,
		"--headless", "--disable-gpu", "--no-sandbox",
		"--no-pdf-header-footer",
		"--print-to-pdf="+pdfPath,
		htmlPath,
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("chrome print: %v", err)
	}

	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatalf("read pdf: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
	if pages := bytes.Count(pdf, []byte("/Type /Page")); pages == 0 {
		t.Error("PDF contains no pages")
	}
}

// TestPrintMediaQueryScoped guards a regression the geometry tests cannot see.
// A letter page is about 672px of content, so an unscoped max-width query
// matches in print and silently replaces the print layout with the phone one.
// --dump-dom renders screen media, and asserting print geometry would mean
// parsing a PDF, so this checks the stylesheet directly instead.
func TestPrintMediaQueryScoped(t *testing.T) {
	css, err := os.ReadFile("internal/render/default.css")
	if err != nil {
		t.Fatalf("read stylesheet: %v", err)
	}

	for _, line := range strings.Split(string(css), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "@media") || !strings.Contains(trimmed, "max-width") {
			continue
		}
		if !strings.Contains(trimmed, "screen") {
			t.Errorf("width-based media query is not scoped to screen, so it also matches the print page box:\n  %s", trimmed)
		}
	}
}
