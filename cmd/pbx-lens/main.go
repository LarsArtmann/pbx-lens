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
	if len(os.Args) < minArgs {
		usage()
		os.Exit(exitUsage)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("pbx-lens " + version)
	case "smoke":
		smoke(os.Args[minArgs:])
	case "ingest-cdr":
		ingestCDR(os.Args[minArgs:])
	case "ingest-telnyx":
		ingestTelnyx(os.Args[minArgs:])
	case "ingest-webhook":
		ingestWebhook(os.Args[minArgs:])
	case "ingest-agent":
		ingestAgent(os.Args[minArgs:])
	case "serve":
		serve(os.Args[minArgs:])
	case "backup":
		backupCmd(os.Args[minArgs:])
	case "purge":
		purgeCmd(os.Args[minArgs:])
	default:
		usage()
		os.Exit(exitUsage)
	}
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
			" pbx-lens purge -db <dsn> -max-age <duration> [-dry-run]",
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
