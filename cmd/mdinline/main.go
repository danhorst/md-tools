// mdinline converts reference-style Markdown links to inline links.
//
// Usage:
//
//	mdinline [file...]
//	cat file.md | mdinline
//	mdinline -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"github.com/danhorst/md-tools/internal/cli"
	"github.com/danhorst/md-tools/internal/markdown"
	"os"
)

var flags = cli.RegisterFlags()

func main() {
	flag.Parse()
	if err := cli.Run("mdinline", flags, flag.Args(), transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdinline: %v\n", err)
		os.Exit(1)
	}
}
func transform(content string) string { return markdown.RewriteLinks(content, false) }
