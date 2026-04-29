// mdsplit splits Markdown paragraphs into one sentence per line.
//
// Usage:
//
//	mdsplit [file...]
//	cat file.md | mdsplit
//	mdsplit -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
)

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdsplit", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdsplit: %v\n", err)
		os.Exit(1)
	}
}

func transform(content string) string {
	return markdown.Transform(content, markdown.Handlers{
		Paragraph:  splitParagraph,
		Blockquote: splitBlockquote,
	})
}

// splitParagraph joins lines and splits into sentences.
func splitParagraph(lines []string) []string {
	hasHardBreak := len(lines) > 0 && strings.HasSuffix(lines[len(lines)-1], "  ")

	text := strings.Join(lines, " ")
	text = strings.Join(strings.Fields(text), " ")

	sentences := splitSentences(text)

	if hasHardBreak && len(sentences) > 0 {
		sentences[len(sentences)-1] += "  "
	}

	return sentences
}

// splitSentences splits text into sentences.
func splitSentences(text string) []string {
	if text == "" {
		return nil
	}

	var sentences []string
	var current strings.Builder
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			if i+2 < len(runes) && runes[i+1] == ' ' && unicode.IsUpper(runes[i+2]) {
				sentences = append(sentences, current.String())
				current.Reset()
				i++
			}
		}
	}

	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}

	return sentences
}

// splitBlockquote splits blockquote lines into one sentence per line.
func splitBlockquote(lines []string) []string {
	return markdown.TransformBlockquote(lines, func(content []string) []string {
		var out []string
		for _, s := range splitToSentences(content) {
			out = append(out, "> "+s)
		}
		return out
	})
}

// splitToSentences joins lines and splits into sentences.
func splitToSentences(lines []string) []string {
	text := strings.Join(lines, " ")
	text = strings.Join(strings.Fields(text), " ")
	return splitSentences(text)
}
