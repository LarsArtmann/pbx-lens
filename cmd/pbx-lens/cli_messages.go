package main

import errorfamily "github.com/larsartmann/go-error-family"

// The CLI boundary renders a classified error from its CODE: the error
// library looks a What/Why/Fix template up by code and, finding none,
// prints the bare code over the family's default advice ("Check your input
// and try again") — advice that is simply wrong for a wrong file. The
// templates below are the operator-facing text for the codes the
// identity-export verb can raise, plus the output-path code it shares with
// backup. Every other lens.* code still takes that fallback; TODO_LIST
// tracks registering the rest here.

// errMissingAuthDB rejects an identity-reading verb invoked without -auth-db.
func errMissingAuthDB(verb string) error {
	return errorfamily.NewRejection("lens.cli.missing_auth_db", verb+" needs -auth-db <identity.db>").
		WithContext("command", verb)
}

// errMissingOut rejects a file-writing verb invoked without -out.
func errMissingOut(verb string) error {
	return errorfamily.NewRejection("lens.cli.missing_out", verb+" needs -out <path>").
		WithContext("command", verb)
}

// registerCLITemplates installs the CLI's What/Why/Fix templates (see the
// file comment). main calls it once, before any verb can fail.
func registerCLITemplates() {
	for code, template := range cliTemplates() {
		errorfamily.RegisterTemplate(code, template)
	}
}

// cliTemplates lists the registered templates by error code. Placeholders
// ({dsn}, {cause}, {out}, {command}) are filled from the error's context;
// TestIdentityExportErrorsTellTheOperatorWhatToDo renders every code and
// fails on an unfilled one, so a context-key typo cannot ship.
func cliTemplates() map[string]errorfamily.MessageTemplate {
	return map[string]errorfamily.MessageTemplate{
		"lens.cli.missing_auth_db": {
			What:   "The identity database is missing from the command line.",
			Why:    "{command} reads the identity database — the serve verb's -auth-db file — and will not guess which file that is.",
			Fix:    "Add -auth-db <identity.db>: the file serve was started with, never the call index's -db.",
			WayOut: "pbx-lens identity-export -h lists every flag.",
		},
		"lens.cli.missing_out": {
			What: "The output path is missing from the command line.",
			Why:  "{command} writes a file and needs to know where.",
			Fix:  "Add -out <path> naming the file {command} should write.",
		},
		"lens.web.identity_db_missing": {
			What: "No identity database exists at {dsn}.",
			Why:  "identity-export only reads an existing identity database — it never creates one, so a wrong path cannot pass as an empty export.",
			Fix:  "Pass the serve verb's -auth-db file (identity.db); the -auth-db path serve was started with is in its systemd unit.",
		},
		"lens.web.identity_not_identity_db": {
			What: "{dsn} is not an identity database.",
			Why:  "The file carries none of the identity tables — most likely it is the call index. Nothing was written to it.",
			Fix:  "Pass the serve verb's -auth-db file (identity.db), not the call index's -db (calls.db).",
		},
		"lens.web.identity_db_unreadable": {
			What: "{dsn} cannot be read as a SQLite database: {cause}",
			Why:  "identity-export only reads an existing identity database, and this file is not one it can open.",
			Fix:  "Check the path and the file's permissions; it must be the serve verb's -auth-db file.",
		},
		"lens.identity_export.out": {
			What: "The export could not be written to {out}: {cause}",
			Why:  "The identity database was read, but staging or publishing the artifact failed.",
			Fix:  "Check that the directory exists and is writable, then run the command again — the identity database itself was not changed.",
		},
	}
}
