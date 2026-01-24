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
	lines := strings.Split(content, "\n")

	var result []string
	i := 0

	// Handle YAML frontmatter
	if i < len(lines) {
		hasFrontmatter := false
		if strings.TrimSpace(lines[i]) == "---" {
			if i+1 < len(lines) && markdown.LooksLikeFrontmatterProperty(lines[i+1]) {
				hasFrontmatter = true
				result = append(result, lines[i])
				i++
			}
		} else if markdown.LooksLikeFrontmatterProperty(lines[i]) {
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == "---" {
					hasFrontmatter = true
					break
				}
				if strings.TrimSpace(lines[j]) == "" {
					break
				}
			}
		}

		if hasFrontmatter {
			for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
				result = append(result, lines[i])
				i++
			}
			if i < len(lines) {
				result = append(result, lines[i])
				i++
			}
		}
	}

	// Process the rest of the document
	for i < len(lines) {
		line := lines[i]

		// Check for code block
		if strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "~~~") {
			fence := strings.TrimSpace(line)[:3]
			result = append(result, line)
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), fence) {
				result = append(result, lines[i])
				i++
			}
			if i < len(lines) {
				result = append(result, lines[i])
				i++
			}
			continue
		}

		// Check for indented code block
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			result = append(result, line)
			i++
			continue
		}

		// Check for footnote definition
		if markdown.IsFootnoteDefinition(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for link reference definition
		if markdown.IsLinkRefDefinition(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for blank line
		if strings.TrimSpace(line) == "" {
			result = append(result, line)
			i++
			continue
		}

		// Check for header
		if strings.HasPrefix(line, "#") {
			result = append(result, line)
			i++
			continue
		}

		// Check for list item
		if markdown.IsListItem(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for blockquote
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			var bqLines []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				bqLines = append(bqLines, lines[i])
				i++
			}
			split := splitBlockquote(bqLines)
			result = append(result, split...)
			continue
		}

		// Check for horizontal rule
		if markdown.IsHorizontalRule(line) {
			result = append(result, line)
			i++
			continue
		}

		// Regular paragraph - collect all lines until a break
		var paraLines []string
		for i < len(lines) {
			l := lines[i]

			if strings.TrimSpace(l) == "" {
				break
			}

			if strings.HasPrefix(strings.TrimSpace(l), "```") ||
				strings.HasPrefix(strings.TrimSpace(l), "~~~") ||
				strings.HasPrefix(l, "    ") ||
				strings.HasPrefix(l, "\t") ||
				markdown.IsFootnoteDefinition(l) ||
				markdown.IsLinkRefDefinition(l) ||
				strings.HasPrefix(l, "#") ||
				markdown.IsListItem(l) ||
				strings.HasPrefix(strings.TrimSpace(l), ">") ||
				markdown.IsHorizontalRule(l) {
				break
			}

			// Check for explicit line break (two trailing spaces)
			if strings.HasSuffix(l, "  ") {
				paraLines = append(paraLines, l)
				i++
				break
			}

			paraLines = append(paraLines, l)
			i++
		}

		if len(paraLines) > 0 {
			split := splitParagraph(paraLines)
			result = append(result, split...)
		}
	}

	output := strings.Join(result, "\n")
	output = strings.TrimRight(output, "\n") + "\n"

	return output
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
	if len(lines) == 0 {
		return nil
	}

	const prefix = "> "

	var result []string
	var contentLines []string

	for _, line := range lines {
		content := strings.TrimPrefix(line, ">")
		content = strings.TrimPrefix(content, " ")

		if strings.HasPrefix(content, "[!") && strings.Contains(content, "]") {
			if len(contentLines) > 0 {
				sentences := splitToSentences(contentLines)
				for _, s := range sentences {
					result = append(result, prefix+s)
				}
				contentLines = nil
			}
			result = append(result, prefix+content)
			continue
		}

		contentLines = append(contentLines, content)
	}

	if len(contentLines) > 0 {
		sentences := splitToSentences(contentLines)
		for _, s := range sentences {
			result = append(result, prefix+s)
		}
	}

	return result
}

// splitToSentences joins lines and splits into sentences.
func splitToSentences(lines []string) []string {
	text := strings.Join(lines, " ")
	text = strings.Join(strings.Fields(text), " ")
	return splitSentences(text)
}
