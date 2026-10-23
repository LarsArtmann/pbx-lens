package calls

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	EventObserved  = event.Type("call.observed")
	EventCorrected = event.Type("call.corrected")
	EventEnriched  = event.Type("call.enriched")
	EventRetired   = event.Type("call.retired")
)

const StreamType = "Call"

// ObservationSchemaVersion is the revision of the observation's DERIVED
// content — the shape ContentHash hashes, not the shape it stores.
//
// It is the answer to the additive-field drift class (2026-10-23 forensic
// report, TODO additive-refresh-mechanism): an observation field that is
// persisted but deliberately NOT hashed (SourceEvents) reaches a stored row
// only when the hash moves, and the decider answers an equal hash with a
// no-op (decider.go Record). Adding such a field therefore needs one
// deliberate, whole-corpus correction, and this constant is how it is
// requested: bump it when an additive field must reach history, then run the
// backfill (`pbx-lens refresh`) once. Bump it ONLY for that reason — every
// bump re-derives every re-ingested stream exactly once.
//
// History: 1 = the newline-joined, delimiter-ambiguous pre-image used until
// 2026-10-23; 2 = length-prefixed fields, named, plus this version token.
const ObservationSchemaVersion = 2

func (o Observation) ContentHash() string {
	return o.contentHash(ObservationSchemaVersion)
}

// contentHash is the pre-image builder for one schema revision. The version
// is a parameter (not read from the constant) so tests can prove a bump
// actually moves the hash — the property a silent rename would otherwise
// break.
func (o Observation) contentHash(schema int) string {
	// SourceEvents is intentionally NOT hashed: it carries the source's
	// non-speech markers (idle notices, tool calls), which are diagnostic
	// annotation rather than observation identity. Hashing it would fork
	// every stored call_views row the moment a marker is persisted, so the
	// transcript stays the only hashed speech surface (see SourceEvent).
	// The cost of that choice is the schema-version mechanism above.
	h := sha256.New()
	writeHashedField(h, "schema", strconv.Itoa(schema))
	writeHashedField(h, "native_id", string(o.NativeID))
	writeHashedField(h, "kind", string(o.Kind))
	writeHashedField(h, "direction", string(o.Direction))
	writeHashedField(h, "started_at_nanos", strconv.FormatInt(o.StartedAt.UTC().UnixNano(), 10))
	writeHashedField(h, "duration_secs", strconv.Itoa(o.DurationSecs))
	writeHashedField(h, "from_number", o.FromNumber)
	writeHashedField(h, "to_number", o.ToNumber)
	writeHashedField(h, "hangup_cause", o.HangupCause)
	writeHashedField(h, "is_ai", strconv.FormatBool(o.IsAI))
	writeHashedField(h, "transcript_excerpt", o.TranscriptExcerpt)
	writeHashedField(h, "transcript", transcriptHash(o.Transcript))
	writeHashedField(h, "recording_ref", o.RecordingRef)

	// MOS joins the hash only when reported: a source that never carries a
	// score contributes no bytes at all, so an unscored call cannot be
	// mistaken for a scored one that happens to score zero.
	if o.MOS != 0 {
		writeHashedField(h, "mos", strconv.FormatFloat(o.MOS, 'g', -1, 64))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// writeHashedField writes one named, length-prefixed field into a hash
// pre-image. The length prefix is what makes field boundaries unambiguous:
// under the previous newline-joined encoding, moving a byte from one field to
// the next ("ab"+"c" vs "a"+"bc") could produce the same digest, so two
// different observations could claim one identity.
func writeHashedField(h hash.Hash, name, value string) {
	_, _ = fmt.Fprintf(h, "%s=%d:%s\n", name, len(value), value)
}

func transcriptHash(turns []TranscriptTurn) string {
	if len(turns) == 0 {
		return ""
	}

	h := sha256.New()
	for _, turn := range turns {
		writeHashedField(h, "turn.role", string(turn.Role))
		writeHashedField(h, "turn.text", turn.Text)
		writeHashedField(h, "turn.at", strconv.FormatInt(turn.At.UTC().Unix(), 10))
	}

	return hex.EncodeToString(h.Sum(nil))
}

func (o Observation) StreamID() (id.StreamID, error) {
	if o.NativeID == "" || !o.Kind.Valid() {
		return id.StreamID{}, errorfamily.NewRejection(
			"call.record.stream_id_unbuildable",
			"stream id needs a valid kind and native id",
		)
	}

	return id.ParseStreamID(string(o.Kind) + ":" + string(o.NativeID))
}

type CallObserved struct {
	Observation
}

// CallCorrected supersedes the stream's observation with a later snapshot
// from the same source (a Telnyx conversation that grew between polls). The
// enrichment rides along so the projection's full-recompute folds never lose
// cost/insight data a correction would otherwise clobber.
type CallCorrected struct {
	Observation

	Enrichment Enrichment
}

// CallEnriched carries derived data (cost, insight) that is NOT part of the
// observation identity: it refreshes independently of the source snapshot
// and never triggers corrections. The observation rides along for the same
// full-recompute reason as in CallCorrected.
type CallEnriched struct {
	Observation
	Enrichment
}

// CallRetired is the retention tombstone: it removes the call from the read
// model (every list, search, export, and dashboard surface) while the stream
// itself stays append-only in the event store. The payload carries no call
// data, only why: the audit answer lives in the stream id, and a tombstone
// must not re-introduce the PII it is retiring.
type CallRetired struct {
	Reason string `json:"reason"`
}
