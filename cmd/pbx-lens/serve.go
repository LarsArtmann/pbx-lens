package main

import (
	"flag"

	"github.com/larsartmann/pbx-lens/internal/serve"
)

// serveCmd is the thin verb shell: parse the flag surface the serve package
// declares, require the DSN, and hand the resolved flags to the
// composition — internal/serve owns booting the stack, the poller, and the
// signal-owned shutdown; this shell owns the exit policy.
func serveCmd(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	flags := serve.RegisterFlags(fs)

	parseAndRequireDB(fs, flags.Db, args)

	if err := serve.Run(flags, version); err != nil {
		fatal(err)
	}
}
