// mdref converts inline Markdown links to reference-style links.
//
// Usage:
//
//	mdref [file...]
//	cat file.md | mdref
//	mdref -w file.md    # modify file in place
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
	if err := cli.Run("mdref", flags, flag.Args(), transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdref: %v\n", err)
		os.Exit(1)
	}
}
func transform(content string) string { return markdown.RewriteLinks(content, true) }
