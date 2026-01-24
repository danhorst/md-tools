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

	htmltomd "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/dbh/md-tools/internal/cli"
)

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdfootnote", transform); err != nil {
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

// sidenotePattern matches the full sidenote HTML block
var sidenotePattern = regexp.MustCompile(
	`\n<label for="sidenote-(\d+)" class="margin-toggle sidenote-number"></label>\n` +
		`<input type="checkbox" id="sidenote-\d+" class="margin-toggle"/>\n` +
		`<span class="sidenote">([^<]*(?:<[^>]+>[^<]*)*)</span>`,
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
		mdContent, err := htmltomd.ConvertString(htmlContent)
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
