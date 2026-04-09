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

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
)

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdtable", transform); err != nil {
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
			if j < numCols && len(cell) > widths[j] {
				widths[j] = len(cell)
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
				sb.WriteString(strings.Repeat(" ", widths[j]-len(cell)))
			}
			sb.WriteString(" ")
		}
		sb.WriteString("|")
		result[i] = sb.String()
	}

	return result
}

// parseRow splits a GFM table row into trimmed cell strings.
func parseRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "|") {
		trimmed = trimmed[1:]
	}
	if strings.HasSuffix(trimmed, "|") {
		trimmed = trimmed[:len(trimmed)-1]
	}
	parts := strings.Split(trimmed, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
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
