package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/larsartmann/pbx-lens/internal/cli"
)

func catalogCmd(args []string) {
	fs := flag.NewFlagSet("catalog", flag.ExitOnError)
	output := fs.String("o", "docs/catalog", "directory to write the catalog into")
	plain := fs.Bool("plain", false, "emit plain frontmatter-ID refs (governance lint exports; always skips bootstrap files)")
	skipBootstrap := fs.Bool("skip-bootstrap", false,
		"omit the generated bootstrap files (the merging hub owns its site title, core pin, and dependencies)")
	_ = fs.Parse(args)

	if err := cli.RenderCatalog(*output, *plain, *skipBootstrap); err != nil {
		fatal(err)
	}

	dir := *output + "/eventcatalog"

	//nolint:erraudit // CLI output stream — write failures are not actionable
	fmt.Fprintf(os.Stdout, "catalog: wrote eventcatalog tree to %s\n", dir)
}
