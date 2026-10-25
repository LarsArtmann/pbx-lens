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

	// catalogOwner is the owner-of-record user id in the catalog and the
	// federation hub (catalog.home.lan): the service page links to this
	// user; the hub renders owner cards from it.
	catalogOwner = "lars"
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
			catalog.Owners(catalogOwner),
			catalog.WithName("Record Call"),
			catalog.WithSummary(
				"Records one call observation from a source snapshot; content-hash dedup "+
					"makes re-ingest idempotent, a grown snapshot supersedes via call.corrected."),
		),
		catalog.Command[EnrichCallCmd](
			catalog.MessageID(CommandEnrich),
			catalog.Owners(catalogOwner),
			catalog.WithName("Enrich Call"),
			catalog.WithSummary("Attaches derived data (Telnyx cost, insight summary) to an observed call."),
		),
		catalog.Command[RetireCallCmd](
			catalog.MessageID(CommandRetire),
			catalog.Owners(catalogOwner),
			catalog.WithName("Retire Call"),
			catalog.WithSummary(
				"Retention tombstone addressed by stream identity alone: removes the call "+
					"from the read model while the stream stays append-only."),
		),
		// --- Events (the write model emits these) ---
		catalog.Event[CallObserved](
			catalog.MessageID(EventObserved), catalog.Sends,
			catalog.Owners(catalogOwner),
			catalog.WithName("Call Observed"),
			catalog.WithSummary("A call snapshot landed in the index (new stream or content-identical replay)."),
		),
		catalog.Event[CallCorrected](
			catalog.MessageID(EventCorrected), catalog.Sends,
			catalog.Owners(catalogOwner),
			catalog.WithName("Call Corrected"),
			catalog.WithSummary(
				"A later snapshot from the same source superseded the stream's observation; the "+
					"enrichment rides along so projection recomputes never lose cost/insight data."),
		),
		catalog.Event[CallEnriched](
			catalog.MessageID(EventEnriched), catalog.Sends,
			catalog.Owners(catalogOwner),
			catalog.WithName("Call Enriched"),
			catalog.WithSummary(
				"Derived data (cost, insight) landed; it refreshes independently of the "+
					"observation identity and never triggers corrections."),
		),
		catalog.Event[CallRetired](
			catalog.MessageID(EventRetired), catalog.Sends,
			catalog.Owners(catalogOwner),
			catalog.WithName("Call Retired"),
			catalog.WithSummary(
				"Retention tombstone: the call left every read-model surface; the payload "+
					"carries only the policy reason, never call data."),
		),
		// --- Queries (the read surface answers these; ids mirror the
		// internal/web read functions so hub consumers see parity between
		// declared queries and the actual HTTP surface) ---
		catalog.Query[CallView](
			"call.search",
			catalog.Owners(catalogOwner),
			catalog.WithName("Call Search"),
			catalog.WithSummary(
				"Full-text and filtered call search (direction, date bucket, AI flag, "+
					"hangup cause) over the call_views index; keyset-paginated."),
		),
		catalog.Query[CallView](
			"call.dashboard",
			catalog.Owners(catalogOwner),
			catalog.WithName("Call Dashboard"),
			catalog.WithSummary(
				"Day-bucketed aggregates (volume, duration, cost, MOS, segments) behind "+
					"the dashboard; resolves days in Europe/Berlin."),
		),
		catalog.Query[CallView](
			"call.detail",
			catalog.Owners(catalogOwner),
			catalog.WithName("Call Detail"),
			catalog.WithSummary(
				"One call looked up by its stream identity (source kind + native id) with "+
					"transcript and enrichment attached."),
		),
	)

	// Owner-of-record for the federation hub (catalog.home.lan): the service
	// page links to this user; the hub renders owner cards from it.
	builder.AddUser(catalog.User{
		ID:    catalogOwner,
		Name:  "Lars Artmann",
		Email: "git@lars.software",
	})

	// Read-only over every source system; the disposable index is the
	// only write target. These container links are the visualiser edges
	// that make the pbx.artmann.tech lineage visible (FreeSWITCH and
	// Telnyx are modeled as external containers, not services — the
	// infra repos themselves export nothing).
	builder.ConfigureService(catalogServiceID,
		catalog.ServiceOwners(catalogOwner),
		catalog.ServiceReadsFrom("freeswitch", "telnyx"),
		catalog.ServiceWritesTo("call-index"),
	)

	builder.AddDomain(
		"telephony", "Telephony", CatalogVersion,
		"Call capture, enrichment, and retention across the pbx.artmann.tech stack.",
		catalogServiceID,
	)
	builder.ConfigureDomain("telephony", catalog.DomainOwners(catalogOwner))

	builder.AddDataStore(catalog.DataStore{
		ID:            "call-index",
		Owners:        []string{catalogOwner},
		Name:          "Call Index SQLite",
		Version:       CatalogVersion,
		ContainerType: "database",
		Technology:    "sqlite",
		Summary: "Disposable, always re-derivable call index (event store + read model). " +
			"Read-only over every source; no write-back path exists.",
	})

	// The source systems the lens reads. External containers, so the
	// visualiser shows the five-project lineage without the infra repos
	// exporting anything: nix-international-telephony runs FreeSWITCH,
	// Telnyx is the carrier, and the webhook feed below is bridged by
	// pbx-artmann's receiver.
	builder.AddDataStore(catalog.DataStore{
		ID:            "freeswitch",
		Owners:        []string{catalogOwner},
		Name:          "FreeSWITCH",
		Version:       CatalogVersion,
		ContainerType: "other",
		Technology:    "freeswitch",
		AccessMode:    "read",
		Authoritative: true,
		Summary: "The pbx.artmann.tech softswitch (nix-international-telephony): SIP " +
			"registrations, call routing, and Master.csv CDRs — the system of record " +
			"for which calls happened.",
	})
	builder.AddDataStore(catalog.DataStore{
		ID:            "telnyx",
		Owners:        []string{catalogOwner},
		Name:          "Telnyx",
		Version:       CatalogVersion,
		ContainerType: "externalSaaS",
		Technology:    "telnyx-api",
		AccessMode:    "read",
		Summary: "Carrier APIs (conversations, recordings, messaging): thread and media " +
			"metadata pulled read-only by the lens; inbound events arrive through the " +
			"receiver webhook feed.",
	})

	// The receiver webhook feed: the stack's webhook bridge writes
	// inbound call/message notifications as JSONL and the lens ingests
	// them as observations. A channel — transport, not an actor — so it
	// documents the feed without pretending to be a service.
	builder.AddChannel(catalog.Channel{
		ID:      "pbx-receiver-webhook",
		Name:    "PBX Receiver Webhook Feed",
		Version: CatalogVersion,
		Summary: "Inbound call and message notifications from the pbx.artmann.tech " +
			"webhook receiver, landed as JSONL and ingested read-only by the lens.",
		Protocols: []catalog.Protocol{"http"},
		Owners:    []string{catalogOwner},
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
		Owners:  []string{catalogOwner},
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
			ID:        "s5",
			Title:     "Persist to call index",
			DataStore: &catalog.FlowStepRef{ID: "call-index"},
			NextStep:  &catalog.FlowEdge{ID: "s6"},
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
