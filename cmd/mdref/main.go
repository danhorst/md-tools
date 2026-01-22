// mdref converts inline Markdown links to reference-style links.
//
// Usage:
//
//	mdref [file...]
//	cat file.md | mdref
package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "mdref: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	input, err := cli.Input(args)
	if err != nil {
		return err
	}
	defer input.Close()

	data, err := io.ReadAll(input)
	if err != nil {
		return err
	}

	result := transform(string(data))
	_, err = os.Stdout.WriteString(result)
	return err
}

// transform converts inline links to reference-style links.
func transform(content string) string {
	// Track seen URLs to their reference numbers
	urlToRef := make(map[string]int)
	// Track references in order for output
	var refs []reference

	// Regex to match inline links: [text](url) or [text](url "title")
	// Negative lookbehind for ! to avoid matching image links
	// This pattern handles:
	// - Basic links: [text](url)
	// - Links with titles: [text](url "title") or [text](url 'title')
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	var result strings.Builder
	lastEnd := 0

	matches := linkPattern.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		fullStart := match[0]
		fullEnd := match[1]
		textStart := match[2]
		textEnd := match[3]
		urlPartStart := match[4]
		urlPartEnd := match[5]

		// Check if this is an image link (preceded by !)
		if fullStart > 0 && content[fullStart-1] == '!' {
			continue
		}

		// Write content before this match
		result.WriteString(content[lastEnd:fullStart])

		linkText := content[textStart:textEnd]
		urlPart := content[urlPartStart:urlPartEnd]

		// Parse URL and optional title from urlPart
		url, title := parseURLAndTitle(urlPart)

		// Create a key that includes both URL and title for deduplication
		refKey := url
		if title != "" {
			refKey = url + "\x00" + title
		}

		// Get or assign reference number
		refNum, exists := urlToRef[refKey]
		if !exists {
			refNum = len(refs) + 1
			urlToRef[refKey] = refNum
			refs = append(refs, reference{url: url, title: title})
		}

		// Write the reference-style link
		result.WriteString(fmt.Sprintf("[%s][%d]", linkText, refNum))

		lastEnd = fullEnd
	}

	// Write remaining content
	result.WriteString(content[lastEnd:])

	// Append reference definitions if any
	if len(refs) > 0 {
		// Ensure there's a blank line before references
		text := result.String()
		if !strings.HasSuffix(text, "\n\n") {
			if strings.HasSuffix(text, "\n") {
				result.Reset()
				result.WriteString(text)
				result.WriteString("\n")
			} else {
				result.Reset()
				result.WriteString(text)
				result.WriteString("\n\n")
			}
		}

		for i, ref := range refs {
			if ref.title != "" {
				fmt.Fprintf(&result, "[%d]: %s %q\n", i+1, ref.url, ref.title)
			} else {
				fmt.Fprintf(&result, "[%d]: %s\n", i+1, ref.url)
			}
		}
	}

	return result.String()
}

type reference struct {
	url   string
	title string
}

// parseURLAndTitle extracts the URL and optional title from a link's URL portion.
// Handles formats like:
//   - url
//   - url "title"
//   - url 'title'
func parseURLAndTitle(s string) (url, title string) {
	s = strings.TrimSpace(s)

	// Check for title in double quotes
	if idx := strings.Index(s, " \""); idx != -1 {
		url = strings.TrimSpace(s[:idx])
		titlePart := s[idx+2:]
		if strings.HasSuffix(titlePart, "\"") {
			title = titlePart[:len(titlePart)-1]
		}
		return
	}

	// Check for title in single quotes
	if idx := strings.Index(s, " '"); idx != -1 {
		url = strings.TrimSpace(s[:idx])
		titlePart := s[idx+2:]
		if strings.HasSuffix(titlePart, "'") {
			title = titlePart[:len(titlePart)-1]
		}
		return
	}

	return s, ""
}
