package markdown

import "strings"

// ByteRange represents a range of bytes in source content.
type ByteRange struct {
	Start int
	End   int
}

// ExcludeRanges returns content with any overlapping ranges removed.
// contentStart is the byte offset where content begins in the original source.
// Ranges are specified in terms of the original source byte positions.
func ExcludeRanges(content string, contentStart int, ranges []ByteRange) string {
	contentEnd := contentStart + len(content)
	var result strings.Builder

	pos := 0
	for _, r := range ranges {
		// Convert range to be relative to content
		relStart := r.Start - contentStart
		relEnd := r.End - contentStart

		// Skip ranges that don't overlap with content
		if r.End <= contentStart || r.Start >= contentEnd {
			continue
		}

		// Clamp to content bounds
		if relStart < 0 {
			relStart = 0
		}
		if relEnd > len(content) {
			relEnd = len(content)
		}

		// Write content before this range
		if relStart > pos {
			result.WriteString(content[pos:relStart])
		}
		pos = relEnd
	}

	// Write remaining content
	if pos < len(content) {
		result.WriteString(content[pos:])
	}

	return result.String()
}
