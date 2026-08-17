// mdhtml renders a Markdown document to standalone HTML.
//
// Usage:
//
//	mdhtml [file]
//	cat file.md | mdhtml
//	mdhtml -fragment file.md         # body content only
//	mdhtml -no-css file.md           # omit the stylesheet
//	mdhtml -css tufte.css file.md    # use a different stylesheet
//	mdhtml -title "Notes" file.md    # set the document title
//	mdhtml -inline-svg file.md       # inline referenced .svg files
//	mdhtml -i out.html < file.md     # write the result to a file
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
	"github.com/dbh/md-tools/internal/render"
)

var (
	flags     = cli.RegisterFlags()
	fragment  = flag.Bool("fragment", false, "emit body content only, without the document wrapper")
	title     = flag.String("title", "", "document `title` (defaults to the input file's base name)")
	cssPath   = flag.String("css", "", "use the stylesheet in `FILE` instead of the built-in default")
	noCSS     = flag.Bool("no-css", false, "omit the stylesheet entirely")
	inlineSVG = flag.Bool("inline-svg", false, "replace .svg image references with the file contents inline")
	baseDir   = flag.String("base", "", "`DIR` to resolve relative SVG references against (required with -inline-svg on stdin)")
	unsafeSVG = flag.Bool("unsafe-svg", false, "inline SVG without checking it for scripts and other active content")
)

// renderErr records a failure inside the transform, which cannot report one
// through cli.TransformFunc's signature.
var renderErr error

func main() {
	// -w rewrites the input file with the result elsewhere in the toolkit.
	// Here that would replace Markdown source with HTML, so it is refused.
	flag.Lookup("w").Usage = "not supported by mdhtml; use -i FILE or shell redirection"

	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdhtml: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if flags.ShowVersion {
		fmt.Println("mdhtml", cli.Version)
		return nil
	}

	if flags.WriteInPlace {
		return fmt.Errorf("-w would overwrite the Markdown source with HTML; use -i FILE or shell redirection")
	}
	if *cssPath != "" && *noCSS {
		return fmt.Errorf("-css and -no-css are mutually exclusive")
	}

	css, err := stylesheet()
	if err != nil {
		return err
	}

	args := flag.Args()
	var srcPath string
	if len(args) > 0 {
		srcPath = args[0]
	}

	if *inlineSVG && srcPath == "" && *baseDir == "" {
		return fmt.Errorf("-inline-svg requires -base DIR when reading from stdin")
	}

	base := *baseDir
	if base == "" && srcPath != "" {
		base = filepath.Dir(srcPath)
	}

	docTitle := *title
	if docTitle == "" && srcPath != "" {
		docTitle = strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))
	}

	transform := func(content string) string {
		out, err := renderDoc(content, docTitle, css, base)
		if err != nil {
			renderErr = err
			return ""
		}
		return out
	}

	if err := cli.Run("mdhtml", flags, args, transform); err != nil {
		return err
	}
	return renderErr
}

// stylesheet resolves the CSS to embed. An explicit -css file is read even in
// fragment mode, so that naming a missing file is always an error.
func stylesheet() (string, error) {
	if *cssPath != "" {
		data, err := os.ReadFile(*cssPath)
		if err != nil {
			return "", fmt.Errorf("reading stylesheet: %w", err)
		}
		return string(data), nil
	}
	if *noCSS {
		return "", nil
	}
	return render.DefaultCSS, nil
}

func renderDoc(content, docTitle, css, base string) (string, error) {
	frontmatter, body := markdown.SplitFrontmatter(content)
	if frontmatter != "" {
		body = strings.TrimLeft(body, "\n")
	}

	var buf bytes.Buffer
	if err := render.New().Convert([]byte(body), &buf); err != nil {
		return "", err
	}
	out := buf.String()

	if *inlineSVG {
		var err error
		if out, err = inlineSVGRefs(out, base); err != nil {
			return "", err
		}
	}

	if *fragment {
		return strings.TrimRight(out, "\n") + "\n", nil
	}
	return render.Document(out, docTitle, css), nil
}

var (
	imgRe      = regexp.MustCompile(`<img\s[^>]*>`)
	srcRe      = regexp.MustCompile(`\ssrc="([^"]*)"`)
	svgIDRe    = regexp.MustCompile(`\sid="([^"]+)"`)
	xmlDeclRe  = regexp.MustCompile(`(?s)<\?xml.*?\?>`)
	docTypeRe  = regexp.MustCompile(`(?s)<!DOCTYPE[^>]*>`)
	schemeOnRe = regexp.MustCompile(`^(?:[a-zA-Z][a-zA-Z0-9+.-]*:|//)`)
)

// warnf reports something worth knowing without failing. Warnings go to stderr
// so stdout stays a clean document for the next stage of a pipeline.
func warnf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "mdhtml: "+format+"\n", args...)
}

// inlineSVGRefs replaces local .svg image references with the file's contents.
// Remote references are left alone; there is no file to read.
func inlineSVGRefs(htmlDoc, base string) (string, error) {
	var outer error
	count := 0

	result := imgRe.ReplaceAllStringFunc(htmlDoc, func(tag string) string {
		if outer != nil {
			return tag
		}
		m := srcRe.FindStringSubmatch(tag)
		if m == nil {
			return tag
		}
		src := m[1]
		if !strings.EqualFold(filepath.Ext(src), ".svg") {
			return tag
		}
		if schemeOnRe.MatchString(src) {
			// Fetching it would break determinism and invite SSRF. Say so,
			// rather than quietly handing back a document that isn't whole.
			warnf("%s: remote reference left as <img>; output is not self-contained", src)
			return tag
		}

		path := src
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			outer = fmt.Errorf("inlining %s: %w", src, err)
			return tag
		}

		if !*unsafeSVG {
			warnings, verr := vetSVG(data)
			if verr != nil {
				outer = fmt.Errorf("inlining %s: %v; inlining would make it live in the document. Pass -unsafe-svg if you trust this file", src, verr)
				return tag
			}
			for _, w := range warnings {
				warnf("%s: %s", src, w)
			}
		}

		count++
		return namespaceIDs(cleanSVG(string(data)), count)
	})

	if outer != nil {
		return "", outer
	}
	return result, nil
}

// cleanSVG strips the prolog so the element can sit inside an HTML document.
func cleanSVG(svg string) string {
	svg = xmlDeclRe.ReplaceAllString(svg, "")
	svg = docTypeRe.ReplaceAllString(svg, "")
	return strings.TrimSpace(svg)
}

// namespaceIDs prefixes every identifier declared in an SVG, along with the
// references to it, so that inlining two documents cannot collide. Mermaid
// scopes its internal stylesheet under the root element's id, and generates
// identical ids for identical diagrams.
func namespaceIDs(svg string, n int) string {
	matches := svgIDRe.FindAllStringSubmatch(svg, -1)
	if len(matches) == 0 {
		return svg
	}

	seen := make(map[string]bool)
	var ids []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}

	// Longest first, so rewriting "a" cannot corrupt a reference to "ab".
	sort.Slice(ids, func(i, j int) bool { return len(ids[i]) > len(ids[j]) })

	prefix := fmt.Sprintf("p%d-", n)
	for _, id := range ids {
		svg = strings.ReplaceAll(svg, `id="`+id+`"`, `id="`+prefix+id+`"`)
		svg = strings.ReplaceAll(svg, "#"+id, "#"+prefix+id)
	}
	return svg
}
