package calls

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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

func (o Observation) ContentHash() string {
	// SourceEvents is intentionally NOT hashed: it carries the source's
	// non-speech markers (idle notices, tool calls), which are diagnostic
	// annotation rather than observation identity. Hashing it would fork
	// every stored call_views row the moment a marker is persisted, so the
	// transcript stays the only hashed speech surface (see SourceEvent).
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\n%s\n%s\n%d\n%d\n%s\n%s\n%s\n%t\n%s\n%s\n%s",
		o.NativeID, o.Kind, o.Direction, o.StartedAt.UTC().UnixNano(), o.DurationSecs,
		o.FromNumber, o.ToNumber, o.HangupCause, o.IsAI, o.TranscriptExcerpt,
		transcriptHash(o.Transcript), o.RecordingRef)

	// MOS joins the hash only when reported, so a source that never carries
	// it keeps the exact hash it had before the field existed.
	if o.MOS != 0 {
		_, _ = fmt.Fprintf(h, "\nmos=%g", o.MOS)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func transcriptHash(turns []TranscriptTurn) string {
	if len(turns) == 0 {
		return ""
	}

	h := sha256.New()
	for _, turn := range turns {
		_, _ = fmt.Fprintf(h, "%s\x00%s\x00%d\x00", turn.Role, turn.Text, turn.At.UTC().Unix())
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
