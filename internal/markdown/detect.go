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

// IsFootnoteDefinition returns true if the line starts a footnote definition.
// Footnote definitions have the form [^label]: ...
func IsFootnoteDefinition(line string) bool {
	return footnoteDefRe.MatchString(line)
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
