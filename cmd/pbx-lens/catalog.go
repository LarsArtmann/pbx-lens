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
	plain := fs.Bool("plain", false, "emit plain frontmatter-ID refs (governance lint exports)")
	_ = fs.Parse(args)

	if err := cli.RenderCatalog(*output, *plain); err != nil {
		fatal(err)
	}

	dir := *output + "/eventcatalog"

	//nolint:erraudit // CLI output stream — write failures are not actionable
	fmt.Fprintf(os.Stdout, "catalog: wrote eventcatalog tree to %s\n", dir)
}
