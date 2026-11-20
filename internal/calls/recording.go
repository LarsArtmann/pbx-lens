package calls

// RecordingMedia is the consumer-facing recording payload the proxy
// streams (T10, 2026-11-20): the source's media download URLs, carried by
// the calls package so the web read side never imports an ingest client.
// The ingest wire DTO (telnyx.Recording) stays at the boundary; this shape
// is what crosses into the UI. WAV is preferred over MP3 by the proxy —
// the fields are raw URLs, the choice is presentation policy.
type RecordingMedia struct {
	WAVURL string `json:"wav_url"`
	MP3URL string `json:"mp3_url"`
}
