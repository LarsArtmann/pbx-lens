package calls

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// CatalogTitle and CatalogVersion identify the generated event/command
// catalog (`pbx-lens catalog` export, federation hub).
const (
	CatalogTitle   = "PBX-Lens"
	CatalogVersion = "1.0.0"

	// catalogServiceID is the service id in the catalog and the federation
	// hub union — keep in sync with the repo's .eventcatalog-hub.json name.
	catalogServiceID = "pbx-lens"
)

// BuildCatalog constructs the event/command catalog for the calls aggregate.
// Message schemas are auto-derived from the command and event structs via
// reflection, so the catalog stays in sync with the domain without manual
// maintenance. The catalog is consumed by the `pbx-lens catalog` command,
// which renders the EventCatalog project tree for the architecture-catalog
// hub. No database is required.
func BuildCatalog() *catalog.Catalog {
	builder := catalog.NewBuilder(CatalogTitle, CatalogVersion)

	builder.AddService(
		catalogServiceID, CatalogTitle, CatalogVersion,
		"Unified call-history index for pbx.artmann.tech: read-only ingest over "+
			"FreeSWITCH CDRs, the Telnyx conversations/recordings APIs, and the "+
			"receiver webhook JSONL into one searchable SQLite index.",
		// --- Commands (the write model accepts these) ---
		catalog.Command[RecordCallCmd](
			catalog.MessageID(CommandRecord),
			catalog.WithName("Record Call"),
			catalog.WithSummary("Records one call observation from a source snapshot; content-hash dedup makes re-ingest idempotent, a grown snapshot supersedes via call.corrected."),
		),
		catalog.Command[EnrichCallCmd](
			catalog.MessageID(CommandEnrich),
			catalog.WithName("Enrich Call"),
			catalog.WithSummary("Attaches derived data (Telnyx cost, insight summary) to an observed call."),
		),
		catalog.Command[RetireCallCmd](
			catalog.MessageID(CommandRetire),
			catalog.WithName("Retire Call"),
			catalog.WithSummary("Retention tombstone addressed by stream identity alone: removes the call from the read model while the stream stays append-only."),
		),
		// --- Events (the write model emits these) ---
		catalog.Event[CallObserved](
			catalog.MessageID(EventObserved), catalog.Sends,
			catalog.WithName("Call Observed"),
			catalog.WithSummary("A call snapshot landed in the index (new stream or content-identical replay)."),
		),
		catalog.Event[CallCorrected](
			catalog.MessageID(EventCorrected), catalog.Sends,
			catalog.WithName("Call Corrected"),
			catalog.WithSummary("A later snapshot from the same source superseded the stream's observation; the enrichment rides along so projection recomputes never lose cost/insight data."),
		),
		catalog.Event[CallEnriched](
			catalog.MessageID(EventEnriched), catalog.Sends,
			catalog.WithName("Call Enriched"),
			catalog.WithSummary("Derived data (cost, insight) landed; it refreshes independently of the observation identity and never triggers corrections."),
		),
		catalog.Event[CallRetired](
			catalog.MessageID(EventRetired), catalog.Sends,
			catalog.WithName("Call Retired"),
			catalog.WithSummary("Retention tombstone: the call left every read-model surface; the payload carries only the policy reason, never call data."),
		),
	)

	// Owner-of-record for the federation hub (catalog.home.lan): the service
	// page links to this user; the hub renders owner cards from it.
	builder.AddUser(catalog.User{
		ID:    "lars",
		Name:  "Lars Artmann",
		Email: "git@lars.software",
	})
	builder.ConfigureService(catalogServiceID, catalog.ServiceOwners("lars"))

	builder.AddDomain(
		"telephony", "Telephony", CatalogVersion,
		"Call capture, enrichment, and retention across the pbx.artmann.tech stack.",
		catalogServiceID,
	)

	builder.AddDataStore(catalog.DataStore{
		ID:            "call-index",
		Name:          "Call Index SQLite",
		Version:       CatalogVersion,
		ContainerType: "database",
		Technology:    "sqlite",
		Summary:       "Disposable, always re-derivable call index (event store + read model). Read-only over every source; no write-back path exists.",
	})

	// The call lifecycle as a first-class flow — the aggregate's visible
	// story (ingest → observe → enrich → persist → retire). Telnyx is an
	// EXTERNAL node (not a catalog resource), mirroring the bank-sync
	// provider pattern.
	builder.AddFlow(catalog.Flow{
		ID:      "call-lifecycle-flow",
		Name:    "Call Lifecycle",
		Version: CatalogVersion,
		Summary: "One call from source snapshot to enriched index entry — and out again at retention time",
		Owners:  []string{"lars"},
		Steps:   callLifecycleFlowSteps(),
	})

	return builder.Build()
}

// callLifecycleFlowSteps walks the calls aggregate's story: a source
// snapshot is ingested and observed, Telnyx-derived enrichment lands, the
// result persists to the SQLite index, and retention eventually retires it.
func callLifecycleFlowSteps() []catalog.FlowStep {
	return []catalog.FlowStep{
		{
			ID:       "s1",
			Title:    "Ingest source snapshot",
			Service:  &catalog.FlowStepRef{ID: catalogServiceID},
			NextStep: &catalog.FlowEdge{ID: "s2"},
		},
		{
			ID:       "s2",
			Title:    "Call observed",
			Message:  &catalog.FlowStepRef{ID: catalog.MessageID(EventObserved)},
			NextStep: &catalog.FlowEdge{ID: "s3"},
		},
		{
			ID:    "s3",
			Title: "Pull Telnyx enrichment",
			External: &catalog.FlowActor{
				Name:    "Telnyx conversations/recordings API",
				Summary: "Deliver cost and recording metadata for the observed call",
			},
			NextStep: &catalog.FlowEdge{ID: "s4"},
		},
		{
			ID:       "s4",
			Title:    "Call enriched",
			Message:  &catalog.FlowStepRef{ID: catalog.MessageID(EventEnriched)},
			NextStep: &catalog.FlowEdge{ID: "s5"},
		},
		{
			ID:       "s5",
			Title:    "Persist to call index",
			DataStore: &catalog.FlowStepRef{ID: "call-index"},
			NextStep: &catalog.FlowEdge{ID: "s6"},
		},
		{
			ID:       "s6",
			Title:    "Retire by retention policy",
			Service:  &catalog.FlowStepRef{ID: catalogServiceID},
			NextStep: &catalog.FlowEdge{ID: "s7"},
		},
		{
			ID:      "s7",
			Title:   "Call retired",
			Message: &catalog.FlowStepRef{ID: catalog.MessageID(EventRetired)},
		},
	}
}
