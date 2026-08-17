package markdown

import (
	"regexp"
	"strings"
)

var (
	footnoteDefRe = regexp.MustCompile(`^\[\^[^\]]+\]:`)
	linkRefDefRe  = regexp.MustCompile(`^\[[^\]]+\]:\s*\S`)
	orderedListRe = regexp.MustCompile(`^\d+\.\s`)
)

// LooksLikeFrontmatterProperty returns true if the line appears to be
// a YAML frontmatter property (contains a colon not at the start).
func LooksLikeFrontmatterProperty(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "---" {
		return false
	}
	idx := strings.Index(trimmed, ":")
	return idx > 0
}

// frontmatterLineCount returns the number of leading lines forming a YAML
// frontmatter block, or 0 if the content does not begin with one. Two formats
// are recognized: a --- delimited block, and a bare property line terminated
// by ---.
func frontmatterLineCount(lines []string) int {
	if len(lines) == 0 {
		return 0
	}

	start := 0
	switch {
	case strings.TrimSpace(lines[0]) == "---":
		if len(lines) < 2 || !LooksLikeFrontmatterProperty(lines[1]) {
			return 0
		}
		start = 1
	case LooksLikeFrontmatterProperty(lines[0]):
		terminated := false
		for j := 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "---" {
				terminated = true
				break
			}
			if strings.TrimSpace(lines[j]) == "" {
				break
			}
		}
		if !terminated {
			return 0
		}
	default:
		return 0
	}

	i := start
	for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
		i++
	}
	if i < len(lines) {
		i++
	}
	return i
}

// SplitFrontmatter separates a leading YAML frontmatter block from the document
// body. When there is no frontmatter, frontmatter is empty and body is content;
// otherwise the two rejoin as frontmatter + "\n" + body.
func SplitFrontmatter(content string) (frontmatter, body string) {
	lines := strings.Split(content, "\n")
	n := frontmatterLineCount(lines)
	if n == 0 {
		return "", content
	}
	return strings.Join(lines[:n], "\n"), strings.Join(lines[n:], "\n")
}

// IsFootnoteDefinition returns true if the line starts a footnote definition.
// Footnote definitions have the form [^label]: ...
func IsFootnoteDefinition(line string) bool {
	return footnoteDefRe.MatchString(line)
}

// IsFootnoteContinuation returns true if the line continues the body of a
// preceding footnote definition: a non-blank line that does not itself start
// another block-level construct.
func IsFootnoteContinuation(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if IsFootnoteDefinition(line) || IsLinkRefDefinition(line) {
		return false
	}
	if strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, ">") ||
		strings.HasPrefix(trimmed, "```") ||
		strings.HasPrefix(trimmed, "~~~") {
		return false
	}
	return !IsListItem(line) && !IsTableRow(line) && !IsHorizontalRule(line)
}

// IsLinkRefDefinition returns true if the line is a link reference definition.
// Link reference definitions have the form [label]: URL
// This excludes footnote definitions.
func IsLinkRefDefinition(line string) bool {
	if IsFootnoteDefinition(line) {
		return false
	}
	return linkRefDefRe.MatchString(line)
}

// IsListItem returns true if the line is a list item.
// Supports unordered lists (-, *, +) and ordered lists (1., 2., etc).
func IsListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Unordered: -, *, +
	if len(trimmed) > 1 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' ' {
		return true
	}
	// Ordered: 1. 2. etc
	return orderedListRe.MatchString(trimmed)
}

// IsTableRow returns true if the line is a GFM table row.
// GFM table rows start with a pipe character.
func IsTableRow(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

// IsHorizontalRule returns true if the line is a horizontal rule.
// Horizontal rules are three or more -, *, or _ characters with optional spaces.
func IsHorizontalRule(line string) bool {
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
