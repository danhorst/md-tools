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

// linkMatch represents a link found in the document with its position
type linkMatch struct {
	start    int    // start position in content
	end      int    // end position in content
	text     string // link text
	url      string // resolved URL
	title    string // optional title
	isInline bool   // true for inline links, false for reference-style
}

// transform converts inline links to reference-style links.
func transform(content string) string {
	// First, extract and remove any existing reference definitions
	existingRefs, contentWithoutRefs := extractExistingRefs(content)

	// Regex to match inline links: [text](url) or [text](url "title")
	inlineLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	// Regex to match existing reference-style links: [text][ref]
	refLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\[([^\]]+)\]`)

	// Collect all links with their positions for unified processing
	var allLinks []linkMatch

	// Find inline links
	for _, match := range inlineLinkPattern.FindAllStringSubmatchIndex(contentWithoutRefs, -1) {
		fullStart := match[0]
		fullEnd := match[1]
		textStart := match[2]
		textEnd := match[3]
		urlPartStart := match[4]
		urlPartEnd := match[5]

		// Skip image links (preceded by !)
		if fullStart > 0 && contentWithoutRefs[fullStart-1] == '!' {
			continue
		}

		linkText := contentWithoutRefs[textStart:textEnd]
		urlPart := contentWithoutRefs[urlPartStart:urlPartEnd]
		url, title := parseURLAndTitle(urlPart)

		allLinks = append(allLinks, linkMatch{
			start:    fullStart,
			end:      fullEnd,
			text:     linkText,
			url:      url,
			title:    title,
			isInline: true,
		})
	}

	// Find reference-style links
	for _, match := range refLinkPattern.FindAllStringSubmatchIndex(contentWithoutRefs, -1) {
		fullStart := match[0]
		fullEnd := match[1]
		textStart := match[2]
		textEnd := match[3]
		refStart := match[4]
		refEnd := match[5]

		// Skip image links (preceded by !)
		if fullStart > 0 && contentWithoutRefs[fullStart-1] == '!' {
			continue
		}

		refID := contentWithoutRefs[refStart:refEnd]

		// Skip footnotes (start with ^)
		if strings.HasPrefix(refID, "^") {
			continue
		}

		// Look up the reference definition
		oldRef, exists := existingRefs[refID]
		if !exists {
			// Reference not found, skip (leave as-is by not adding to allLinks)
			continue
		}

		linkText := contentWithoutRefs[textStart:textEnd]

		allLinks = append(allLinks, linkMatch{
			start:    fullStart,
			end:      fullEnd,
			text:     linkText,
			url:      oldRef.url,
			title:    oldRef.title,
			isInline: false,
		})
	}

	// Sort links by position in document
	sortLinksByPosition(allLinks)

	// Process links in document order, assigning reference numbers
	urlToRef := make(map[string]int)
	var refs []reference

	var result strings.Builder
	lastEnd := 0

	for _, link := range allLinks {
		// Write content before this match
		result.WriteString(contentWithoutRefs[lastEnd:link.start])

		// Create a key that includes both URL and title for deduplication
		refKey := link.url
		if link.title != "" {
			refKey = link.url + "\x00" + link.title
		}

		// Get or assign reference number
		refNum, exists := urlToRef[refKey]
		if !exists {
			refNum = len(refs) + 1
			urlToRef[refKey] = refNum
			refs = append(refs, reference{url: link.url, title: link.title})
		}

		// Write the reference-style link
		result.WriteString(fmt.Sprintf("[%s][%d]", link.text, refNum))

		lastEnd = link.end
	}

	// Write remaining content
	result.WriteString(contentWithoutRefs[lastEnd:])

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

// sortLinksByPosition sorts links by their start position in the document
func sortLinksByPosition(links []linkMatch) {
	for i := 1; i < len(links); i++ {
		for j := i; j > 0 && links[j].start < links[j-1].start; j-- {
			links[j], links[j-1] = links[j-1], links[j]
		}
	}
}

// extractExistingRefs parses and removes existing reference definitions from content.
// Returns a map of reference ID to reference, and the content with definitions removed.
func extractExistingRefs(content string) (map[string]reference, string) {
	refs := make(map[string]reference)

	// Pattern to match reference definitions: [id]: url or [id]: url "title"
	// Must be at start of line
	refDefPattern := regexp.MustCompile(`(?m)^\[([^\]]+)\]:\s+(\S+)(?:\s+"([^"]*)")?[ \t]*\n?`)

	// Find all reference definitions
	matches := refDefPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		id := match[1]
		url := match[2]
		title := ""
		if len(match) > 3 {
			title = match[3]
		}
		// Skip footnote definitions (start with ^)
		if strings.HasPrefix(id, "^") {
			continue
		}
		refs[id] = reference{url: url, title: title}
	}

	// Remove reference definitions from content (but not footnote definitions)
	cleaned := refDefPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Check if this is a footnote definition
		if strings.HasPrefix(match, "[^") {
			return match
		}
		return ""
	})

	// Clean up any extra blank lines that may have been left
	// But preserve the structure - just remove trailing blank lines before we add refs
	cleaned = strings.TrimRight(cleaned, "\n") + "\n"

	return refs, cleaned
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
