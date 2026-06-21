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
		// Inline span (code, link, emphasis, strikethrough, footnote) —
		// copy verbatim so a sentence boundary inside it never splits.
		if n := spanLen(runes, i); n > 0 {
			end := i + n
			for k := i; k < end; k++ {
				current.WriteRune(runes[k])
			}
			// A sentence may end inside the span, just before its closing
			// delimiter (e.g. "**Done.** Next"). Break after the span.
			if end+1 < len(runes) && runes[end] == ' ' && !unicode.IsLower(runes[end+1]) && spanEndsSentence(runes[i:end]) {
				sentences = append(sentences, current.String())
				current.Reset()
				i = end
				continue
			}
			i = end - 1
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

// spanLen returns the rune length of an inline span beginning at i whose
// interior must not be split, or 0 if no span begins there. Recognized spans
// are code spans, links/images, emphasis, strikethrough, and footnotes.
func spanLen(runes []rune, i int) int {
	switch {
	case runes[i] == '`':
		return codeSpanLen(runes, i)
	case runes[i] == '^':
		return footnoteLen(runes, i) // inline footnote ^[...]
	case runes[i] == '[':
		return bracketSpanLen(runes, i)
	case runes[i] == '*' || runes[i] == '_':
		return emphasisLen(runes, i)
	case runes[i] == '~':
		return strikeLen(runes, i)
	}
	return 0
}

// codeSpanLen returns the length of a backtick code span at i, closed by a run
// of the same number of backticks, or 0 if unterminated.
func codeSpanLen(runes []rune, i int) int {
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

// bracketSpanLen returns the length of a [text] span at i, plus a following
// (target) or [reference] when balanced, or 0 if the brackets are unbalanced.
func bracketSpanLen(runes []rune, i int) int {
	textLen := balancedLen(runes, i, '[', ']')
	if textLen == 0 {
		return 0
	}
	j := i + textLen
	if j < len(runes) {
		if t := balancedLen(runes, j, '(', ')'); t > 0 {
			return textLen + t
		}
		if t := balancedLen(runes, j, '[', ']'); t > 0 {
			return textLen + t
		}
	}
	return textLen
}

// balancedLen returns the length of a balanced open/close run starting at start
// (which must hold open), or 0 if it is never closed.
func balancedLen(runes []rune, start int, open, close rune) int {
	if start >= len(runes) || runes[start] != open {
		return 0
	}
	depth := 0
	for j := start; j < len(runes); j++ {
		switch runes[j] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return j - start + 1
			}
		}
	}
	return 0
}

// emphasisLen returns the length of an emphasis/strong span (*, _, **, ***, …)
// at i, or 0 if no span opens there. A delimiter run opens a span only when it
// is not followed by whitespace (and, for _, not inside a word); it closes at
// the first matching run not preceded by whitespace.
func emphasisLen(runes []rune, i int) int {
	c := runes[i]
	n := 0
	for i+n < len(runes) && runes[i+n] == c {
		n++
	}
	after := i + n
	if after >= len(runes) || isSpace(runes[after]) {
		return 0
	}
	if c == '_' && i > 0 && isWordChar(runes[i-1]) {
		return 0
	}
	for j := after; j < len(runes); j++ {
		if runes[j] != c {
			continue
		}
		m := 0
		for j+m < len(runes) && runes[j+m] == c {
			m++
		}
		if isSpace(runes[j-1]) || (c == '_' && j+m < len(runes) && isWordChar(runes[j+m])) {
			j += m - 1
			continue
		}
		return j + m - i
	}
	return 0
}

// strikeLen returns the length of a GFM strikethrough span (~~…~~) at i, or 0
// if no span opens there.
func strikeLen(runes []rune, i int) int {
	if i+1 >= len(runes) || runes[i+1] != '~' {
		return 0
	}
	after := i + 2
	if after >= len(runes) || isSpace(runes[after]) {
		return 0
	}
	for j := after; j+1 < len(runes); j++ {
		if runes[j] == '~' && runes[j+1] == '~' && !isSpace(runes[j-1]) {
			return j + 2 - i
		}
	}
	return 0
}

func isSpace(r rune) bool    { return unicode.IsSpace(r) }
func isWordChar(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// spanEndsSentence reports whether span (delimiters included) ends with
// terminal punctuation once trailing closing delimiters are removed, e.g.
// "**Done.**" or "`x = 1.`".
func spanEndsSentence(span []rune) bool {
	j := len(span) - 1
	for j >= 0 && isCloser(span[j]) {
		j--
	}
	return j >= 0 && (span[j] == '.' || span[j] == '!' || span[j] == '?')
}

func isCloser(r rune) bool {
	switch r {
	case '*', '_', '~', '`', ')', ']', '"', '\'':
		return true
	}
	return false
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
