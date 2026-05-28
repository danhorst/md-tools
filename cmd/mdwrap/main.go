// mdwrap wraps Markdown paragraphs to a specified width (default 60).
//
// Usage:
//
//	mdwrap [file...]
//	cat file.md | mdwrap
//	mdwrap -c 80 file.md  # wrap to 80 columns
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
	writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")
	wrapWidth    = flag.Int("c", 60, "column width to wrap to")
)

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdwrap", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdwrap: %v\n", err)
		os.Exit(1)
	}
}

func transform(content string) string {
	return markdown.Transform(content, markdown.Handlers{
		Paragraph:  wrapParagraph,
		Blockquote: wrapBlockquote,
	})
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
