// mdfootnote converts Tufte CSS sidenotes back to Markdown footnotes.
//
// Usage:
//
//	mdfootnote [file...]
//	cat file.md | mdfootnote
//	mdfootnote -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/danhorst/md-tools/internal/cli"
)

var flags = cli.RegisterFlags()

func main() {
	flag.Parse()
	if err := cli.Run("mdfootnote", flags, flag.Args(), transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdfootnote: %v\n", err)
		os.Exit(1)
	}
}

// sidenote represents a found sidenote in the document
type sidenote struct {
	start   int    // start position of the full sidenote HTML
	end     int    // end position
	number  int    // sidenote number
	content string // HTML content (will be converted to markdown)
}

// gap matches whatever whitespace separates the marker's elements. mdsidenote
// writes newlines, but mdjoin and mdsplit rewrite them as spaces and mdwrap may
// leave none, so recovery must not depend on the layout. A blank line is
// deliberately not allowed: that would mean the elements sit in different
// paragraphs and are not one marker.
const gap = `[ \t]*\n?[ \t]*`

// sidenotePattern matches the full sidenote HTML block.
//
// The content capture ends at this note's own closing parenthesis span. Matching
// tags generically instead would run to the last </span> in the document, since
// a closing tag looks like any other — which swallowed every following note and
// paragraph.
var sidenotePattern = regexp.MustCompile(
	gap + `<label for="sidenote-(\d+)" class="margin-toggle sidenote-number"></label>` +
		gap + `<input type="checkbox" id="sidenote-\d+" class="margin-toggle"/>` +
		gap + `<span class="sidenote">((?s:.*?)<span class="hidden">\)[^<]*</span>)</span>`,
)

// markdownConverter renders a sidenote's HTML back to markdown. The package's
// ConvertString helper hardcodes its plugins, so the converter is built here to
// pin the emphasis delimiter to the underscore the note was written with —
// otherwise a round trip silently renormalizes _like this_ to *like this*.
var markdownConverter = converter.NewConverter(
	converter.WithPlugins(
		base.NewBasePlugin(),
		commonmark.NewCommonmarkPlugin(
			commonmark.WithEmDelimiter("_"),
		),
	),
)

// hiddenSpanPattern matches the hidden paren spans
var hiddenSpanPattern = regexp.MustCompile(`<span class="hidden">\([^<]*</span>|<span class="hidden">\)[^<]*</span>`)

func transform(content string) string {
	// Find all sidenotes
	matches := sidenotePattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content
	}

	var sidenotes []sidenote
	for _, match := range matches {
		// match[0], match[1] = full match start/end
		// match[2], match[3] = sidenote number
		// match[4], match[5] = span content

		numStr := content[match[2]:match[3]]
		var num int
		fmt.Sscanf(numStr, "%d", &num)

		spanContent := content[match[4]:match[5]]

		sidenotes = append(sidenotes, sidenote{
			start:   match[0],
			end:     match[1],
			number:  num,
			content: spanContent,
		})
	}

	// Sort by position (should already be in order, but be safe)
	sort.Slice(sidenotes, func(i, j int) bool {
		return sidenotes[i].start < sidenotes[j].start
	})

	// Build result
	var result strings.Builder
	var footnotes []string
	lastEnd := 0

	for _, sn := range sidenotes {
		// Write content before this sidenote
		result.WriteString(content[lastEnd:sn.start])

		// Write footnote reference
		result.WriteString(fmt.Sprintf("[^%d]", sn.number))

		// Convert sidenote content to markdown
		htmlContent := sn.content
		// Remove hidden paren spans
		htmlContent = hiddenSpanPattern.ReplaceAllString(htmlContent, "")
		// Trim whitespace
		htmlContent = strings.TrimSpace(htmlContent)

		// Convert HTML to markdown
		mdContent, err := markdownConverter.ConvertString(htmlContent)
		if err != nil {
			// Fallback: use content as-is
			mdContent = htmlContent
		}
		mdContent = strings.TrimSpace(mdContent)

		// Store footnote definition
		for len(footnotes) < sn.number {
			footnotes = append(footnotes, "")
		}
		footnotes[sn.number-1] = mdContent

		lastEnd = sn.end
	}

	// Write remaining content
	remaining := content[lastEnd:]
	remaining = strings.TrimRight(remaining, "\n")
	result.WriteString(remaining)

	// Append footnote definitions
	result.WriteString("\n")
	for i, fn := range footnotes {
		if fn != "" {
			result.WriteString(fmt.Sprintf("\n[^%d]: %s", i+1, fn))
		}
	}
	result.WriteString("\n")

	return result.String()
}
