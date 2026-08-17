package render

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// GitHub alerts are not part of the GFM specification — that covers tables,
// task lists, strikethrough, autolinks, and tagfilter — so goldmark's GFM
// extension does not produce them. They are a GitHub feature layered on top,
// and this repository's own README uses them, so mdhtml renders them too.
//
// The work splits across two hooks because of when goldmark runs them.
// Stripping the marker has to happen during block parsing, while a paragraph's
// Lines() still drive rendering: by the time AST transformers run, inlines are
// parsed and "[!NOTE]" has fragmented into three Text nodes, because the link
// parser tries "[" and backtracks. Inserting the title has to happen after
// that inline pass instead, so nothing re-parses the synthetic node.
//
// The class attribute carries the alert kind between the two phases.
//
// The container stays a blockquote rather than becoming GitHub's div:
// stylesheets key off the class, and a custom node type would be considerably
// more machinery for a difference nothing observes.

const alertClassPrefix = "markdown-alert markdown-alert-"

var alertMarkerRe = regexp.MustCompile(`^\[!([A-Za-z]+)\]\s*$`)

// alertTitles maps each recognized alert to the label GitHub displays.
var alertTitles = map[string]string{
	"note":      "Note",
	"tip":       "Tip",
	"important": "Important",
	"warning":   "Warning",
	"caution":   "Caution",
}

// alertMarkerStripper removes the marker line and tags the enclosing
// blockquote. It runs as a paragraph transformer, during block parsing.
type alertMarkerStripper struct{}

func (alertMarkerStripper) Transform(para *ast.Paragraph, reader text.Reader, _ parser.Context) {
	bq, ok := para.Parent().(*ast.Blockquote)
	if !ok || bq.FirstChild() != ast.Node(para) {
		return
	}

	lines := para.Lines()
	if lines.Len() == 0 {
		return
	}

	first := lines.At(0)
	marker := alertMarkerRe.FindStringSubmatch(strings.TrimSpace(string(first.Value(reader.Source()))))
	if marker == nil {
		return
	}
	kind := strings.ToLower(marker[1])
	if _, known := alertTitles[kind]; !known {
		return
	}

	// Drop the marker line, and the paragraph with it when that is all it held.
	if lines.Len() == 1 {
		bq.RemoveChild(bq, para)
	} else {
		lines.SetSliced(1, lines.Len())
	}

	bq.SetAttributeString("class", []byte(alertClassPrefix+kind))
}

// alertTitler inserts the title paragraph into blockquotes the stripper
// tagged. It runs as an AST transformer, after inlines are parsed.
type alertTitler struct{}

func (alertTitler) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	var quotes []*ast.Blockquote
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if bq, ok := n.(*ast.Blockquote); ok {
				quotes = append(quotes, bq)
			}
		}
		return ast.WalkContinue, nil
	})

	for _, bq := range quotes {
		label, ok := alertLabel(bq)
		if !ok {
			continue
		}

		title := ast.NewParagraph()
		title.SetAttributeString("class", []byte("markdown-alert-title"))
		title.AppendChild(title, ast.NewString([]byte(label)))

		if first := bq.FirstChild(); first != nil {
			bq.InsertBefore(bq, first, title)
		} else {
			bq.AppendChild(bq, title)
		}
	}
}

// alertLabel recovers the display label from the class the stripper set.
func alertLabel(bq *ast.Blockquote) (string, bool) {
	v, ok := bq.AttributeString("class")
	if !ok {
		return "", false
	}
	class, ok := v.([]byte)
	if !ok {
		return "", false
	}
	kind, found := strings.CutPrefix(string(class), alertClassPrefix)
	if !found {
		return "", false
	}
	label, known := alertTitles[kind]
	return label, known
}
