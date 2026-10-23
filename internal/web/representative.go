package web

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// One real call can be described by more than one source row: FreeSWITCH's
// CDR and the voice agent's transcript share the call uuid, so every AI call
// indexes twice. The read model deliberately keeps both (no source's truth is
// merged at ingest — see DetailPage), but the operator-facing surfaces must
// not count one call twice.
//
// The representative row is the CDR: it is the telephony truth (direction,
// duration, hangup cause, MOS, cost, AI destination) and the transcript only
// adds the conversation. A transcript whose CDR is absent stays its own row —
// it is then the only evidence the call exists. Aggregates (dashboard) and
// the list collapse duplicates; the export deliberately does not, because a
// forensic dump must carry every source row under its own identity.

// representativeOf names the source kind whose row stands for view's call
// when both are indexed. The bool is false when view is already that row (or
// when no sibling source joins the kind).
func representativeOf(kind calls.SourceKind) (calls.SourceKind, bool) {
	if kind == calls.SourceAgentTranscript {
		return calls.SourceFreeSwitchCDR, true
	}

	return "", false
}

// supersedingStreamID resolves the stream id of the row that represents
// view's call, and false when view itself is the representative.
func supersedingStreamID(view calls.CallView) (id.StreamID, bool) {
	kind, ok := representativeOf(view.Kind)
	if !ok {
		return id.StreamID{}, false
	}

	obs := calls.Observation{NativeID: view.NativeID, Kind: kind}

	sid, err := obs.StreamID()
	if err != nil {
		// A row with an unusable identity cannot have a sibling.
		return id.StreamID{}, false
	}

	return sid, true
}

// supersededBySibling reports whether view is the redundant half of a call
// the index also holds as its representative row. Presence goes through
// LookupView, not Exists: sqliteengine's SetBackend reads a different table
// than the map-backed rows this index writes, so Exists answers false for a
// row Get finds (verified 2026-10-23 — see docs/gotchas.md).
func supersededBySibling(
	ctx context.Context,
	reader *metaengine.TypedReader[calls.CallView],
	view calls.CallView,
) (bool, error) {
	sid, ok := supersedingStreamID(view)
	if !ok {
		return false, nil
	}

	_, found, err := calls.LookupView(ctx, reader, sid)

	return found, err
}

// listRepresentative resolves one scanned row into the row the list shows and
// reports whether it is shown at all. A representative row absorbs the facts
// only its sibling source recorded (the CDR cannot know the opening line);
// a transcript superseded by its CDR is dropped, because the walk emits the
// CDR row on its own pass.
//
// An unreadable sibling is an error, not a silent drop: masking it would show
// a call with missing evidence and no hint that anything went wrong.
func listRepresentative(
	ctx context.Context,
	reader *metaengine.TypedReader[calls.CallView],
	view calls.CallView,
) (calls.CallView, bool, error) {
	superseded, err := supersededBySibling(ctx, reader, view)
	if err != nil {
		return view, false, err
	}

	if superseded {
		return view, false, nil
	}

	return mergeSibling(ctx, reader, view)
}

// mergeSibling fills the representative row's gaps from the other sources
// that describe the same call (the read-time join the detail page performs
// for one call, applied per list row).
func mergeSibling(
	ctx context.Context,
	reader *metaengine.TypedReader[calls.CallView],
	view calls.CallView,
) (calls.CallView, bool, error) {
	for _, kind := range siblingKinds(view.Kind) {
		obs := calls.Observation{NativeID: view.NativeID, Kind: kind}

		sid, err := obs.StreamID()
		if err != nil {
			continue
		}

		sibling, found, err := calls.LookupView(ctx, reader, sid)
		if err != nil {
			return view, false, err
		}

		if !found {
			continue
		}

		merged, _ := mergeCall(view, sibling)
		view = merged
	}

	return view, true, nil
}
