package markdown

import (
	"strings"
	"unicode"
)

// CodeSpanLen returns the rune length of a backtick code span beginning at i,
// closed by a run of the same number of backticks, or 0 if none opens there or
// it is never closed. GFM preserves the interior of a code span verbatim, so
// callers copy runes[i : i+CodeSpanLen(runes, i)] without modification.
func CodeSpanLen(runes []rune, i int) int {
	if i >= len(runes) || runes[i] != '`' {
		return 0
	}
	n := 0
	for i+n < len(runes) && runes[i+n] == '`' {
		n++
	}
	for j := i + n; j < len(runes); {
		if runes[j] != '`' {
			j++
			continue
		}
		m := 0
		for j+m < len(runes) && runes[j+m] == '`' {
			m++
		}
		if m == n {
			return j + m - i
		}
		j += m
	}
	return 0
}

// CollapseSpaces collapses every run of whitespace to a single space and trims
// leading and trailing whitespace — the paragraph-reflow normalization the
// tools share — except that the interior of an inline code span is copied
// verbatim, since GFM treats the spaces between backticks as significant.
func CollapseSpaces(text string) string {
	runes := []rune(text)
	var b strings.Builder
	pendingSpace, wrote := false, false

	sep := func() {
		if pendingSpace && wrote {
			b.WriteByte(' ')
		}
		pendingSpace = false
	}

	for i := 0; i < len(runes); {
		if n := CodeSpanLen(runes, i); n > 0 {
			sep()
			b.WriteString(string(runes[i : i+n]))
			wrote = true
			i += n
			continue
		}
		if unicode.IsSpace(runes[i]) {
			pendingSpace = true
			i++
			continue
		}
		sep()
		b.WriteRune(runes[i])
		wrote = true
		i++
	}
	return b.String()
}

// LeadingIndent returns the run of leading spaces and tabs on the first line, or
// "" when there are no lines or it has no indent. It lets a paragraph handler
// re-apply an indent that joining would otherwise drop, keeping an indented
// continuation paragraph attached to its list item.
func LeadingIndent(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[0][:len(lines[0])-len(strings.TrimLeft(lines[0], " \t"))]
}
