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
	"unicode"
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

	words := tokenize(strings.Join(body, " "))
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

	words := tokenize(text)
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

// tagEnd returns the index just past the HTML tag starting at i, or -1 if no
// tag starts there. A '<' opens a tag only when followed by '/' or an ASCII
// letter, and the tag ends at the next '>'. Requiring both is what keeps
// "a < b" and "5<10" from absorbing the rest of the paragraph.
func tagEnd(text string, i int) int {
	if i+1 >= len(text) {
		return -1
	}
	c := text[i+1]
	isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	if c != '/' && !isLetter {
		return -1
	}
	j := strings.IndexByte(text[i:], '>')
	if j < 0 {
		return -1
	}
	return i + j + 1
}

// tokenize splits text on whitespace like strings.Fields, except that an HTML
// tag stays one token. Breaking between a tag's name and its attributes leaves
// markup that mdfootnote can no longer recognize, so the sidenote round-trip
// would not survive wrapping.
func tokenize(text string) []string {
	var tokens []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}

	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		switch {
		case unicode.IsSpace(r):
			flush()
			i += size
		case r == '<' && tagEnd(text, i) > 0:
			end := tagEnd(text, i)
			cur.WriteString(text[i:end])
			i = end
		default:
			cur.WriteRune(r)
			i += size
		}
	}
	flush()

	return tokens
}

// wrapToWidth wraps lines to the specified width.
func wrapToWidth(lines []string, width int) []string {
	text := strings.Join(lines, " ")

	// A tag is one token, so an over-long one overflows its line rather than
	// being split — the usual treatment for any unbreakable token.
	words := tokenize(text)
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
