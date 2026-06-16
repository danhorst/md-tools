// mdtable normalizes GFM table column widths so every cell in a column
// is padded to equal width, making tables visually aligned in plain text.
//
// Usage:
//
//	mdtable [file...]
//	cat file.md | mdtable
//	mdtable -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
)

var flags = cli.RegisterFlags()

func main() {
	flag.Parse()
	if err := cli.Run("mdtable", flags, flag.Args(), transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdtable: %v\n", err)
		os.Exit(1)
	}
}

// separatorCellRe matches a GFM table separator cell: optional colon,
// one or more dashes, optional colon.
var separatorCellRe = regexp.MustCompile(`^:?-+:?$`)

func transform(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	i := 0

	// Handle YAML frontmatter (same pattern as other md-tools).
	if i < len(lines) {
		hasFrontmatter := false
		if strings.TrimSpace(lines[i]) == "---" {
			if i+1 < len(lines) && markdown.LooksLikeFrontmatterProperty(lines[i+1]) {
				hasFrontmatter = true
				result = append(result, lines[i])
				i++
			}
		} else if markdown.LooksLikeFrontmatterProperty(lines[i]) {
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == "---" {
					hasFrontmatter = true
					break
				}
				if strings.TrimSpace(lines[j]) == "" {
					break
				}
			}
		}
		if hasFrontmatter {
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

	for i < len(lines) {
		line := lines[i]

		// Fenced code block: pass through unchanged.
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

		// Table: collect consecutive pipe-delimited rows and normalize.
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			var tableLines []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
				tableLines = append(tableLines, lines[i])
				i++
			}
			result = append(result, normalizeTable(tableLines)...)
			continue
		}

		result = append(result, line)
		i++
	}

	output := strings.Join(result, "\n")
	output = strings.TrimRight(output, "\n") + "\n"
	return output
}

// normalizeTable pads all cells in each column to equal width.
func normalizeTable(rows []string) []string {
	if len(rows) == 0 {
		return rows
	}

	// Parse every row into trimmed cell slices.
	parsed := make([][]string, len(rows))
	numCols := 0
	for i, row := range rows {
		cells := parseRow(row)
		parsed[i] = cells
		if len(cells) > numCols {
			numCols = len(cells)
		}
	}
	if numCols == 0 {
		return rows
	}

	// Identify the separator row (all cells match :?-+:?).
	sepIdx := -1
	for i, cells := range parsed {
		if len(cells) == 0 {
			continue
		}
		allSep := true
		for _, cell := range cells {
			if !separatorCellRe.MatchString(cell) {
				allSep = false
				break
			}
		}
		if allSep {
			sepIdx = i
			break
		}
	}

	// Compute column widths from header and data rows; minimum 3 keeps
	// separator cells valid (at least one dash).
	widths := make([]int, numCols)
	for j := range widths {
		widths[j] = 3
	}
	for i, cells := range parsed {
		if i == sepIdx {
			continue
		}
		for j, cell := range cells {
			if j < numCols && utf8.RuneCountInString(cell) > widths[j] {
				widths[j] = utf8.RuneCountInString(cell)
			}
		}
	}

	// Reconstruct each row with uniform column widths.
	result := make([]string, len(rows))
	for i, cells := range parsed {
		var sb strings.Builder
		for j := 0; j < numCols; j++ {
			cell := ""
			if j < len(cells) {
				cell = cells[j]
			}
			sb.WriteString("| ")
			if i == sepIdx {
				sb.WriteString(padSeparator(cell, widths[j]))
			} else {
				sb.WriteString(cell)
				sb.WriteString(strings.Repeat(" ", widths[j]-utf8.RuneCountInString(cell)))
			}
			sb.WriteString(" ")
		}
		sb.WriteString("|")
		result[i] = sb.String()
	}

	return result
}

// parseRow splits a GFM table row into trimmed cell strings.
// | inside backtick code spans and \| escapes are treated as literal, not delimiters.
func parseRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "|") {
		trimmed = trimmed[1:]
	}
	if strings.HasSuffix(trimmed, "|") {
		trimmed = trimmed[:len(trimmed)-1]
	}

	var cells []string
	var current strings.Builder
	runes := []rune(trimmed)
	i := 0
	for i < len(runes) {
		r := runes[i]
		switch {
		case r == '\\' && i+1 < len(runes) && runes[i+1] == '|':
			current.WriteRune('\\')
			current.WriteRune('|')
			i += 2
		case r == '`':
			j := i
			for j < len(runes) && runes[j] == '`' {
				j++
			}
			openerLen := j - i
			for k := i; k < j; k++ {
				current.WriteRune(runes[k])
			}
			i = j
			for i < len(runes) {
				if runes[i] == '`' {
					k := i
					for k < len(runes) && runes[k] == '`' {
						k++
					}
					closerLen := k - i
					for m := i; m < k; m++ {
						current.WriteRune(runes[m])
					}
					i = k
					if closerLen == openerLen {
						break
					}
				} else {
					current.WriteRune(runes[i])
					i++
				}
			}
		case r == '|':
			cells = append(cells, strings.TrimSpace(current.String()))
			current.Reset()
			i++
		default:
			current.WriteRune(r)
			i++
		}
	}
	cells = append(cells, strings.TrimSpace(current.String()))
	return cells
}

// padSeparator pads a separator cell to exactly width bytes using dashes,
// preserving any leading/trailing colon alignment markers.
func padSeparator(cell string, width int) string {
	leftColon := strings.HasPrefix(cell, ":")
	rightColon := strings.HasSuffix(cell, ":") && len(cell) > 1
	dashCount := width
	if leftColon {
		dashCount--
	}
	if rightColon {
		dashCount--
	}
	if dashCount < 1 {
		dashCount = 1
	}
	dashes := strings.Repeat("-", dashCount)
	switch {
	case leftColon && rightColon:
		return ":" + dashes + ":"
	case leftColon:
		return ":" + dashes
	case rightColon:
		return dashes + ":"
	default:
		return dashes
	}
}
