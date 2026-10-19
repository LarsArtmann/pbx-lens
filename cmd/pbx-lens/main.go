package main

import (
	"flag"
	"fmt"
	"os"

	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	version = "0.2.0"

	minArgs   = 2
	exitUsage = 2
)

func main() {
	subcommands := map[string]func([]string){
		"smoke":          smoke,
		"ingest-cdr":     ingestCDR,
		"ingest-telnyx":  ingestTelnyx,
		"ingest-webhook": ingestWebhook,
		"ingest-agent":   ingestAgent,
		"serve":          serve,
		"backup":         backupCmd,
		"purge":          purgeCmd,
		"catalog":        catalogCmd,
	}

	if len(os.Args) < minArgs {
		usage()
		os.Exit(exitUsage)
	}

	if os.Args[1] == "version" {
		fmt.Println("pbx-lens " + version)

		return
	}

	run, ok := subcommands[os.Args[1]]
	if !ok {
		usage()
		os.Exit(exitUsage)
	}

	run(os.Args[minArgs:])
}

func usage() {
	fmt.Fprintln(
		os.Stderr,
		"usage: pbx-lens version | pbx-lens smoke [-db path] | pbx-lens ingest-cdr -db <dsn> <Master.csv> |"+
			" pbx-lens ingest-telnyx -db <dsn> [-key-file path] [-since 48h] [-direction outbound] [-limit N] |"+
			" pbx-lens ingest-webhook -db <dsn> <inbound.jsonl> |"+
			" pbx-lens ingest-agent -db <dsn> -dir /var/lib/telephony/recordings/transcripts |"+
			" pbx-lens serve -db <dsn> [-addr 127.0.0.1:8300] |"+
			" pbx-lens backup -db <dsn> -out <snapshot.db> |"+
			" pbx-lens purge -db <dsn> -max-age <duration> [-dry-run] |"+
			" pbx-lens catalog -o <dir> [-plain]",
	)
}

// requireDB enforces the mandatory -db flag every index-opening subcommand
// shares.
func requireDB(fs *flag.FlagSet, db string) {
	if db == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_db", fs.Name()+" needs -db <dsn>"))
	}
}

// requireOneArg enforces the single positional argument of the file-ingesting
// subcommands.
func requireOneArg(fs *flag.FlagSet, what string) {
	if fs.NArg() != 1 {
		fatal(errorfamily.NewRejection("lens.cli.bad_args", fs.Name()+" needs exactly one "+what))
	}
}

func printRejected(rejected []string) {
	for _, reason := range rejected {
		fmt.Println("  rejected:", reason)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "pbx-lens:", err)
	os.Exit(1)
}
