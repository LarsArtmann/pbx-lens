package cli

import (
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// RenderCatalog exports the calls aggregate's event/command catalog as a
// full EventCatalog project tree into <output>/eventcatalog/. It is a pure
// render: no database, no network. The plain variant emits plain
// frontmatter-ID refs for the hub's per-source governance lint.
func RenderCatalog(output string, plain bool) error {
	dir := filepath.Join(output, "eventcatalog")

	var opts []eventcatalog.Option
	if plain {
		opts = append(opts, eventcatalog.WithPlainRefIDs())
	}

	err := eventcatalog.NewExporter(dir, opts...).Export(calls.BuildCatalog())
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "lens.cli.catalog",
			"export eventcatalog tree to "+dir)
	}

	return nil
}
