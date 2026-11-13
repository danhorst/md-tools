package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var svgAttrRe = regexp.MustCompile(`([\w:-]+)(\s*=\s*)("[^"]*"|'[^']*')`)
var localURLRe = regexp.MustCompile(`url\(\s*(["']?)#([^\s)'"\x00]+)["']?\s*\)`)
var selectorIDRe = regexp.MustCompile(`#[a-zA-Z_][a-zA-Z0-9_-]*`)

// namespaceIDs changes identifiers and references, never arbitrary hash values
// such as fill colors. Replacements use the original lookup in a single pass.
func namespaceIDs(svg string, n int) string {
	ids := map[string]string{}
	walkSVGTags(svg, func(tag string, start bool) string {
		if start {
			for _, a := range svgAttrRe.FindAllStringSubmatch(tag, -1) {
				if a[1] == "id" {
					id := a[3][1 : len(a[3])-1]
					ids[id] = fmt.Sprintf("p%d-%s", n, id)
				}
			}
		}
		return tag
	})
	urls := func(s string) string {
		return localURLRe.ReplaceAllStringFunc(s, func(m string) string {
			parts := localURLRe.FindStringSubmatch(m)
			if id, ok := ids[parts[2]]; ok {
				return strings.Replace(m, "#"+parts[2], "#"+id, 1)
			}
			return m
		})
	}
	inStyle := false
	return walkSVGParts(svg, func(part string, kind byte) string {
		if kind == 's' {
			if strings.HasPrefix(part, "<style") {
				inStyle = true
			}
			return svgAttrRe.ReplaceAllStringFunc(part, func(a string) string {
				m := svgAttrRe.FindStringSubmatch(a)
				value := m[3][1 : len(m[3])-1]
				switch m[1] {
				case "id":
					if id, ok := ids[value]; ok {
						value = id
					}
				case "href", "xlink:href":
					if id, ok := ids[strings.TrimPrefix(value, "#")]; ok && strings.HasPrefix(value, "#") {
						value = "#" + id
					}
				case "aria-labelledby", "aria-describedby":
					fields := strings.Fields(value)
					for i, f := range fields {
						if id, ok := ids[f]; ok {
							fields[i] = id
						}
					}
					value = strings.Join(fields, " ")
				default:
					value = urls(value)
				}
				return m[1] + m[2] + m[3][:1] + value + m[3][:1]
			})
		}
		if kind == 'e' {
			if strings.HasPrefix(part, "</style") {
				inStyle = false
			}
			return part
		}
		if !inStyle {
			return part
		}
		// Hashes in selector preludes denote IDs; hashes in declarations may be colors.
		var out strings.Builder
		start := 0
		for i, c := range part {
			if c == '{' {
				selector := selectorIDRe.ReplaceAllStringFunc(part[start:i], func(hash string) string {
					if id, ok := ids[hash[1:]]; ok {
						return "#" + id
					}
					return hash
				})
				out.WriteString(selector)
				out.WriteByte('{')
				start = i + 1
			} else if c == '}' {
				out.WriteString(urls(part[start : i+1]))
				start = i + 1
			}
		}
		out.WriteString(urls(part[start:]))
		return out.String()
	})
}

func walkSVGTags(svg string, visit func(string, bool) string) string {
	return walkSVGParts(svg, func(s string, k byte) string {
		if k == 's' || k == 'e' {
			return visit(s, k == 's')
		}
		return s
	})
}

// XML offsets let us preserve the original spelling, quotes and whitespace.
func walkSVGParts(svg string, visit func(string, byte) string) string {
	dec := xml.NewDecoder(strings.NewReader(svg))
	dec.Entity = xml.HTMLEntity
	var out strings.Builder
	start := 0
	for {
		token, err := dec.Token()
		end := int(dec.InputOffset())
		if end > start {
			kind := byte('t')
			switch token.(type) {
			case xml.StartElement:
				kind = 's'
			case xml.EndElement:
				kind = 'e'
			}
			if err != nil && err != io.EOF {
				out.WriteString(svg[start:])
				return out.String()
			}
			out.WriteString(visit(svg[start:end], kind))
			start = end
		}
		if err != nil {
			out.WriteString(svg[start:])
			return out.String()
		}
	}
}
