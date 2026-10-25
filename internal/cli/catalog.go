package cli

import (
	"path/filepath"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4/architecture"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/eventcatalog"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// RenderCatalog exports the calls aggregate's event/command catalog as a
// full EventCatalog project tree into <output>/eventcatalog/. It is a pure
// render: no database, no network. The plain variant emits plain
// frontmatter-ID refs for the hub's per-source governance lint and never
// carries bootstrap files (a lint tree is parsed, not served); the normal
// export keeps them unless skipBootstrap opts out — the merging hub owns
// its own site title, core pin, and dependencies.
func RenderCatalog(output string, plain, skipBootstrap bool) error {
	dir := filepath.Join(output, "eventcatalog")

	var opts []eventcatalog.Option
	if plain {
		opts = append(opts, eventcatalog.WithPlainRefIDs(), eventcatalog.WithSkipBootstrapFiles())
	} else if skipBootstrap {
		opts = append(opts, eventcatalog.WithSkipBootstrapFiles())
	}

	cat := calls.BuildCatalog()

	err := eventcatalog.NewExporter(dir, opts...).Export(cat)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "lens.cli.catalog",
			"export eventcatalog tree to "+dir)
	}

	return nil
}

// RenderArchitecture writes the self-contained architecture reference HTML
// (catalog v4.10.0's exporter, inlined assets) next to the eventcatalog
// tree. Same pure render as RenderCatalog: no database, no network.
func RenderArchitecture(output string) error {
	path := filepath.Join(output, "architecture.html")

	err := architecture.NewExporter(architecture.WithTitle(calls.CatalogTitle)).
		ExportFile(calls.BuildCatalog(), path)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "lens.cli.catalog",
			"export architecture reference to "+path)
	}

	return nil
}
