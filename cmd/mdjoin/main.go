// mdjoin joins Markdown sentences into single-line paragraphs.
//
// Usage:
//
//	mdjoin [file...]
//	cat file.md | mdjoin
//	mdjoin -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
)

var flags = cli.RegisterFlags()

func main() {
	flag.Parse()
	if err := cli.Run("mdjoin", flags, flag.Args(), transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdjoin: %v\n", err)
		os.Exit(1)
	}
}

func transform(content string) string {
	return markdown.Transform(content, markdown.Handlers{
		Paragraph:  unwrapParagraph,
		Blockquote: unwrapBlockquote,
	})
}

// unwrapParagraph joins lines into a single line.
func unwrapParagraph(lines []string) []string {
	// Check if last line has explicit line break (two trailing spaces)
	hasHardBreak := len(lines) > 0 && strings.HasSuffix(lines[len(lines)-1], "  ")

	// Join all lines into one
	text := strings.Join(lines, " ")
	// Normalize multiple spaces
	text = strings.Join(strings.Fields(text), " ")

	if hasHardBreak {
		text += "  "
	}

	return []string{text}
}

// unwrapBlockquote unwraps blockquote lines into single lines per paragraph.
func unwrapBlockquote(lines []string) []string {
	return markdown.TransformBlockquote(lines, func(content []string) []string {
		return []string{"> " + joinLines(content)}
	})
}

// joinLines joins lines into a single line.
func joinLines(lines []string) string {
	text := strings.Join(lines, " ")
	return strings.Join(strings.Fields(text), " ")
}
