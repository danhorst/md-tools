// mdwrap wraps Markdown paragraphs to a specified width (default 60).
//
// Usage:
//
//	mdwrap [file...]
//	cat file.md | mdwrap
//	mdwrap -c 80 file.md  # wrap to 80 columns
//	mdwrap -f file.md     # also wrap footnote bodies
//	mdwrap -w file.md     # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
)

var (
	flags         = cli.RegisterFlags()
	wrapWidth     = flag.Int("c", 60, "column width to wrap to")
	wrapFootnotes = flag.Bool("f", false, "wrap footnote bodies, indenting continuation lines 4 spaces")
)

// footnoteIndent prefixes continuation lines of a wrapped footnote. Four spaces
// keeps the continuation parsable as part of the footnote in strict engines.
const footnoteIndent = "    "

func main() {
	flag.Parse()
	if err := cli.Run("mdwrap", flags, flag.Args(), transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdwrap: %v\n", err)
		os.Exit(1)
	}
}

func transform(content string) string {
	h := markdown.Handlers{
		Paragraph:  wrapParagraph,
		Blockquote: wrapBlockquote,
	}
	if *wrapFootnotes {
		h.Footnote = wrapFootnote
	}
	return markdown.Transform(content, h)
}

// wrapFootnote wraps a footnote definition's body to the column width, keeping
// the "[^label]: " marker on the first line and indenting continuation lines.
func wrapFootnote(lines []string) []string {
	idx := strings.Index(lines[0], "]:")
	prefix := lines[0][:idx+2] + " "
	body := append([]string{strings.TrimSpace(lines[0][idx+2:])}, lines[1:]...)

	words := strings.Fields(strings.Join(body, " "))
	if len(words) == 0 {
		return []string{strings.TrimRight(prefix, " ")}
	}

	var result []string
	var cur strings.Builder
	first := true
	width := *wrapWidth - utf8.RuneCountInString(prefix)
	flush := func() {
		if first {
			result = append(result, prefix+cur.String())
			first = false
			width = *wrapWidth - len(footnoteIndent)
		} else {
			result = append(result, footnoteIndent+cur.String())
		}
		cur.Reset()
	}

	for _, word := range words {
		switch {
		case cur.Len() == 0:
			cur.WriteString(word)
		case utf8.RuneCountInString(cur.String())+1+utf8.RuneCountInString(word) <= width:
			cur.WriteString(" ")
			cur.WriteString(word)
		default:
			flush()
			cur.WriteString(word)
		}
	}
	flush()
	return result
}

func wrapParagraph(lines []string) []string {
	// Check if last line has explicit line break (two trailing spaces)
	hasHardBreak := len(lines) > 0 && strings.HasSuffix(lines[len(lines)-1], "  ")

	// Join all lines into one, then wrap
	text := strings.Join(lines, " ")

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var result []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if utf8.RuneCountInString(currentLine.String())+1+utf8.RuneCountInString(word) <= *wrapWidth {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lastLine := currentLine.String()
		if hasHardBreak {
			lastLine += "  "
		}
		result = append(result, lastLine)
	}

	return result
}

// wrapBlockquote wraps blockquote lines, accounting for the "> " prefix in width.
func wrapBlockquote(lines []string) []string {
	const prefix = "> "
	contentWidth := *wrapWidth - len(prefix)
	return markdown.TransformBlockquote(lines, func(content []string) []string {
		var out []string
		for _, w := range wrapToWidth(content, contentWidth) {
			out = append(out, prefix+w)
		}
		return out
	})
}

// wrapToWidth wraps lines to the specified width.
func wrapToWidth(lines []string, width int) []string {
	text := strings.Join(lines, " ")

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var result []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if utf8.RuneCountInString(currentLine.String())+1+utf8.RuneCountInString(word) <= width {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

	return result
}
