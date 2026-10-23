package calls

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// RecordKey is the read model's row key for a call stream: the BARE stream
// value. It is the form projectionadapter's EventWithID.ID has carried since
// go-cqrs-lite v4.6.0, the form the library's remove folds derive from the
// record (a retirement therefore deletes the row it names), and the form
// ParseStreamID accepts and MarshalText emits. Every reader in this repo goes
// through it so the key format is never re-typed at a call site.
func RecordKey(sid id.StreamID) string { return sid.Get() }

// LegacyRecordKey is RecordKey's predecessor: the marker-prefixed display form
// StreamID.String() that projectionadapter v4.5.x used as the row key, and
// therefore the key of every row this index wrote before 2026-10-23. Nothing
// writes it anymore; readers accept it so historical calls stay reachable.
func LegacyRecordKey(sid id.StreamID) string { return sid.String() }

// LookupView reads one call view by stream id, accepting both row-key
// generations: the canonical bare key first, then the pre-v4.6.0 marker form.
// A fallback hit needs no rewrite — the row is immutable history.
func LookupView(
	ctx context.Context,
	reader *metaengine.TypedReader[CallView],
	sid id.StreamID,
) (CallView, bool, error) {
	view, found, err := reader.Get(ctx, RecordKey(sid))
	if err != nil || found {
		return view, found, err
	}

	return reader.Get(ctx, LegacyRecordKey(sid))
}

// StreamRowExists reports whether the read model holds any row for the
// stream, across both row-key generations (see LookupView). It is the cheap
// form of LookupView for callers that only need presence, never the row.
func StreamRowExists(
	ctx context.Context,
	reader *metaengine.TypedReader[CallView],
	sid id.StreamID,
) (bool, error) {
	found, err := reader.Exists(ctx, RecordKey(sid))
	if err != nil || found {
		return found, err
	}

	return reader.Exists(ctx, LegacyRecordKey(sid))
}
