// mdwrap wraps Markdown paragraphs to 80 characters.
//
// Usage:
//
//	mdwrap [file...]
//	cat file.md | mdwrap
//	mdwrap -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
)

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

const wrapWidth = 80

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdwrap", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdwrap: %v\n", err)
		os.Exit(1)
	}
}

func transform(content string) string {
	lines := strings.Split(content, "\n")

	var result []string
	i := 0

	// Handle YAML frontmatter
	// Two formats:
	// 1. Starts with --- and ends with ---
	// 2. Starts with a property line and ends with ---
	if i < len(lines) {
		hasFrontmatter := false
		if strings.TrimSpace(lines[i]) == "---" {
			// Check if next line looks like a property
			if i+1 < len(lines) && looksLikeFrontmatterProperty(lines[i+1]) {
				hasFrontmatter = true
				result = append(result, lines[i])
				i++
			}
		} else if looksLikeFrontmatterProperty(lines[i]) {
			// Check if there's a closing --- somewhere
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == "---" {
					hasFrontmatter = true
					break
				}
				if strings.TrimSpace(lines[j]) == "" {
					// Blank line before --- means no frontmatter
					break
				}
			}
		}

		if hasFrontmatter {
			// Copy until closing ---
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

		// Check for indented code block (4 spaces or tab)
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			result = append(result, line)
			i++
			continue
		}

		// Check for footnote definition
		if isFootnoteDefinition(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for link reference definition
		if isLinkRefDefinition(line) {
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

		// Check for list item (preserve as-is for now)
		if isListItem(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for blockquote (preserve as-is for now, will handle in future fixture)
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			result = append(result, line)
			i++
			continue
		}

		// Check for horizontal rule
		if isHorizontalRule(line) {
			result = append(result, line)
			i++
			continue
		}

		// Regular paragraph - collect all lines until a break
		var paraLines []string
		for i < len(lines) {
			l := lines[i]

			// Stop at blank line
			if strings.TrimSpace(l) == "" {
				break
			}

			// Stop at special constructs
			if strings.HasPrefix(strings.TrimSpace(l), "```") ||
				strings.HasPrefix(strings.TrimSpace(l), "~~~") ||
				strings.HasPrefix(l, "    ") ||
				strings.HasPrefix(l, "\t") ||
				isFootnoteDefinition(l) ||
				isLinkRefDefinition(l) ||
				strings.HasPrefix(l, "#") ||
				isListItem(l) ||
				strings.HasPrefix(strings.TrimSpace(l), ">") ||
				isHorizontalRule(l) {
				break
			}

			paraLines = append(paraLines, l)
			i++
		}

		if len(paraLines) > 0 {
			wrapped := wrapParagraph(paraLines)
			result = append(result, wrapped...)
		}
	}

	// Ensure single trailing newline
	output := strings.Join(result, "\n")
	output = strings.TrimRight(output, "\n") + "\n"

	return output
}

func looksLikeFrontmatterProperty(line string) bool {
	// Simple heuristic: contains a colon not at the start
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "---" {
		return false
	}
	idx := strings.Index(trimmed, ":")
	return idx > 0
}

func isFootnoteDefinition(line string) bool {
	matched, _ := regexp.MatchString(`^\[\^[^\]]+\]:`, line)
	return matched
}

func isLinkRefDefinition(line string) bool {
	// [label]: URL
	// Must not be a footnote definition
	if isFootnoteDefinition(line) {
		return false
	}
	matched, _ := regexp.MatchString(`^\[[^\]]+\]:\s*\S`, line)
	return matched
}

func isListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Unordered: -, *, +
	if len(trimmed) > 1 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' ' {
		return true
	}
	// Ordered: 1. 2. etc
	matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed)
	return matched
}

func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	// ---, ***, ___ with optional spaces between
	dashes := strings.ReplaceAll(trimmed, " ", "")
	if len(dashes) >= 3 {
		allSame := true
		ch := dashes[0]
		if ch == '-' || ch == '*' || ch == '_' {
			for _, c := range dashes {
				if byte(c) != ch {
					allSame = false
					break
				}
			}
			return allSame
		}
	}
	return false
}

func wrapParagraph(lines []string) []string {
	// Join all lines into one, then wrap
	text := strings.Join(lines, " ")

	// Tokenize into markdown-aware chunks
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return nil
	}

	var result []string
	var currentLine strings.Builder

	for i, token := range tokens {
		if currentLine.Len() == 0 {
			currentLine.WriteString(token)
		} else {
			newLen := currentLine.Len() + 1 + len(token)
			if newLen <= wrapWidth {
				// Fits within width
				currentLine.WriteString(" ")
				currentLine.WriteString(token)
			} else if containsLink(token) {
				// Token contains a link - allow overflow to keep it together
				currentLine.WriteString(" ")
				currentLine.WriteString(token)
			} else {
				// Check if breaking here would leave the next token orphaned
				// or if there's a better break point
				shouldBreak := true

				// Look ahead: if next token is a link that would fit better
				// on a new line with this token, break before this token
				if i+1 < len(tokens) && containsLink(tokens[i+1]) {
					nextLen := len(token) + 1 + len(tokens[i+1])
					if nextLen <= wrapWidth {
						// Breaking now lets token+nextToken fit on new line
						shouldBreak = true
					}
				}

				if shouldBreak {
					result = append(result, currentLine.String())
					currentLine.Reset()
					currentLine.WriteString(token)
				} else {
					currentLine.WriteString(" ")
					currentLine.WriteString(token)
				}
			}
		}
	}

	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

	return result
}

// containsLink checks if a token contains a markdown link construct
func containsLink(token string) bool {
	// Check for [...](...) or [...][...]
	if !strings.Contains(token, "[") {
		return false
	}
	// Simple heuristic: contains [] followed by () or []
	re := regexp.MustCompile(`\[[^\]]+\](\([^\)]+\)|\[[^\]]*\])`)
	return re.MatchString(token)
}

// tokenize splits text into wrappable tokens, keeping markdown constructs together.
// Links like [text](url) or [text][ref] are kept as single tokens.
// Tokens include any trailing punctuation or links that are attached (no space).
func tokenize(text string) []string {
	var tokens []string
	var current strings.Builder

	i := 0
	for i < len(text) {
		ch := text[i]

		// Skip whitespace, flush current token
		if ch == ' ' || ch == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			i++
			continue
		}

		// Check for markdown link starting with [
		if ch == '[' {
			// Try to parse a complete link construct
			linkEnd := parseLinkConstruct(text, i)
			if linkEnd > i {
				// Append link to current token (keeps word[^1] together)
				current.WriteString(text[i:linkEnd])
				i = linkEnd
				// Continue to pick up any trailing punctuation
				continue
			}
		}

		// Regular character
		current.WriteByte(ch)
		i++
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// parseLinkConstruct tries to parse a markdown link starting at pos.
// Returns the end position if successful, or pos if not a valid link.
// Handles: [text](url), [text][ref], [text][], [ref] (when followed by valid context)
func parseLinkConstruct(text string, pos int) int {
	if pos >= len(text) || text[pos] != '[' {
		return pos
	}

	// Find closing ]
	bracketEnd := findClosingBracket(text, pos)
	if bracketEnd < 0 {
		return pos
	}

	end := bracketEnd + 1

	// Check what follows the ]
	if end < len(text) {
		if text[end] == '(' {
			// Inline link [text](url)
			parenEnd := findClosingParen(text, end)
			if parenEnd > 0 {
				return parenEnd + 1
			}
		} else if text[end] == '[' {
			// Reference link [text][ref] or [text][]
			refEnd := findClosingBracket(text, end)
			if refEnd > 0 {
				return refEnd + 1
			}
		}
	}

	// Could be a shortcut reference [ref] - return just the bracket portion
	// Only if it looks like a standalone reference (not followed by more link syntax)
	return end
}

// findClosingBracket finds the ] that closes the [ at pos
func findClosingBracket(text string, pos int) int {
	if pos >= len(text) || text[pos] != '[' {
		return -1
	}

	depth := 0
	for i := pos; i < len(text); i++ {
		if text[i] == '[' {
			depth++
		} else if text[i] == ']' {
			depth--
			if depth == 0 {
				return i
			}
		} else if text[i] == '\n' {
			// Don't span newlines
			return -1
		}
	}
	return -1
}

// findClosingParen finds the ) that closes the ( at pos
func findClosingParen(text string, pos int) int {
	if pos >= len(text) || text[pos] != '(' {
		return -1
	}

	depth := 0
	for i := pos; i < len(text); i++ {
		if text[i] == '(' {
			depth++
		} else if text[i] == ')' {
			depth--
			if depth == 0 {
				return i
			}
		} else if text[i] == '\n' {
			// Don't span newlines
			return -1
		}
	}
	return -1
}
