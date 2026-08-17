package main

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// Inlining moves an SVG out of an image context, where a browser runs no
// scripts and loads no external resources, into the live document where both
// happen. vetSVG refuses the constructs that become dangerous in that move and
// reports the ones that merely leak.
//
// This inspects the token stream rather than pattern-matching the text, so it
// sees names after entity decoding, is not fooled by CDATA, and handles
// namespace prefixes. Go's decoder does not resolve external entities, so a
// DOCTYPE cannot pull in outside content.
//
// It is a guard against accident and casual hostility, not a guarantee. mXSS
// turns on how browsers re-parse markup, not on how we parse it. The real
// assurance is only inlining SVG you generated.

// refusedElements become executable or embed arbitrary HTML once inlined.
var refusedElements = map[string]string{
	"script":        "a <script> element",
	"iframe":        "an <iframe> element",
	"embed":         "an <embed> element",
	"object":        "an <object> element",
	"foreignobject": "a <foreignObject> element, which embeds arbitrary HTML (generate the diagram with htmlLabels disabled to avoid it)",
}

// urlAttrs carry references that can name a script or an external document.
var urlAttrs = map[string]bool{"href": true, "src": true}

// vetSVG reports whether an SVG is safe to inline. Warnings describe things
// worth knowing that do not justify refusal.
func vetSVG(data []byte) (warnings []string, err error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	// Strict, so that malformed markup is refused rather than guessed at — a
	// file we cannot parse is a file we cannot vouch for. The HTML entity
	// table keeps that from rejecting the &nbsp; and friends that real SVG
	// exporters emit, which XML does not predefine.
	dec.Strict = true
	dec.Entity = xml.HTMLEntity

	var inStyle bool

	for {
		tok, terr := dec.Token()
		if terr != nil {
			if terr.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("cannot parse as XML (%v); refusing to inline what cannot be inspected", terr)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			name := strings.ToLower(t.Name.Local)
			if reason, bad := refusedElements[name]; bad {
				return nil, fmt.Errorf("contains %s", reason)
			}
			if name == "style" {
				inStyle = true
			}
			w, aerr := vetAttrs(name, t.Attr)
			if aerr != nil {
				return nil, aerr
			}
			warnings = append(warnings, w...)

		case xml.EndElement:
			if strings.ToLower(t.Name.Local) == "style" {
				inStyle = false
			}

		case xml.CharData:
			if !inStyle {
				continue
			}
			css := string(t)
			if strings.Contains(strings.ToLower(css), "@import") {
				return nil, fmt.Errorf("contains an @import in a <style> block, which loads an external stylesheet")
			}
			if u, found := externalCSSURL(css); found {
				warnings = append(warnings, fmt.Sprintf("loads an external resource in CSS (%s)", u))
			}
		}
	}

	return warnings, nil
}

func vetAttrs(element string, attrs []xml.Attr) (warnings []string, err error) {
	for _, a := range attrs {
		name := strings.ToLower(a.Name.Local)
		value := strings.TrimSpace(a.Value)

		// Event handlers run on load, on interaction, or on animation events.
		if strings.HasPrefix(name, "on") && len(name) > 2 {
			return nil, fmt.Errorf("carries an %s= event handler", name)
		}

		if !urlAttrs[name] {
			// SMIL can rewrite a link target into a script URL.
			if name == "attributename" && (strings.EqualFold(value, "href") || strings.EqualFold(value, "xlink:href")) {
				return nil, fmt.Errorf("uses <%s> to rewrite a link target, which can inject a script URL", element)
			}
			continue
		}

		if isScriptURL(value) {
			return nil, fmt.Errorf("contains a javascript: URL in %s=", a.Name.Local)
		}

		// A <use> pulling from another document imports markup we never saw.
		if element == "use" && value != "" && !strings.HasPrefix(value, "#") {
			return nil, fmt.Errorf("contains a <use> referencing another document (%s)", value)
		}

		if isExternalURL(value) {
			warnings = append(warnings, fmt.Sprintf("references an external resource (%s)", value))
		}
	}
	return warnings, nil
}

// isScriptURL reports a javascript: or data:text/html target, ignoring the
// whitespace and casing a browser also ignores.
func isScriptURL(v string) bool {
	s := strings.ToLower(v)
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == 0 {
			return -1
		}
		return r
	}, s)
	return strings.HasPrefix(s, "javascript:") || strings.HasPrefix(s, "data:text/html")
}

func isExternalURL(v string) bool {
	s := strings.ToLower(v)
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "//")
}

// externalCSSURL finds the first url(...) pointing off-host.
func externalCSSURL(css string) (string, bool) {
	rest := css
	for {
		i := strings.Index(strings.ToLower(rest), "url(")
		if i < 0 {
			return "", false
		}
		rest = rest[i+4:]
		end := strings.Index(rest, ")")
		if end < 0 {
			return "", false
		}
		u := strings.Trim(strings.TrimSpace(rest[:end]), `"'`)
		if isExternalURL(u) {
			return u, true
		}
		rest = rest[end:]
	}
}
