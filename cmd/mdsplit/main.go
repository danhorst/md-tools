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

var flags = cli.RegisterFlags()

func main() {
	flag.Parse()
	if err := cli.Run("mdsplit", flags, flag.Args(), transform); err != nil {
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
		// Inline footnote (^[...]) — copy verbatim, never split inside.
		if runes[i] == '^' && i+1 < len(runes) && runes[i+1] == '[' {
			depth := 0
			for ; i < len(runes); i++ {
				current.WriteRune(runes[i])
				if runes[i] == '[' {
					depth++
				} else if runes[i] == ']' {
					depth--
					if depth == 0 {
						break
					}
				}
			}
			continue
		}

		current.WriteRune(runes[i])

		if runes[i] == '.' || runes[i] == '!' || runes[i] == '?' {
			// A footnote (reference or inline) may follow the terminal
			// punctuation, e.g. "end.[^1] Next" or "end.^[note] Next".
			// Skip past any such markup before testing the boundary.
			j := i + 1
			for {
				n := footnoteLen(runes, j)
				if n == 0 {
					break
				}
				j += n
			}
			if j+1 < len(runes) && runes[j] == ' ' && !unicode.IsLower(runes[j+1]) {
				for k := i + 1; k < j; k++ {
					current.WriteRune(runes[k])
				}
				sentences = append(sentences, current.String())
				current.Reset()
				i = j
			}
		}
	}

	if current.Len() > 0 {
		sentences = append(sentences, current.String())
	}

	return sentences
}

// footnoteLen returns the rune length of a footnote beginning at start, or 0
// if none is present there. It recognizes both reference footnotes ([^label])
// and inline footnotes (^[...], which may contain nested brackets).
func footnoteLen(runes []rune, start int) int {
	if start+1 >= len(runes) {
		return 0
	}
	switch {
	case runes[start] == '[' && runes[start+1] == '^':
		for k := start + 2; k < len(runes); k++ {
			if runes[k] == ']' {
				return k - start + 1
			}
			if runes[k] == '[' {
				return 0
			}
		}
	case runes[start] == '^' && runes[start+1] == '[':
		depth := 0
		for k := start + 1; k < len(runes); k++ {
			if runes[k] == '[' {
				depth++
			} else if runes[k] == ']' {
				depth--
				if depth == 0 {
					return k - start + 1
				}
			}
		}
	}
	return 0
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
