// Package wire holds decoding helpers shared by every source-adapter's wire
// DTOs. Sources speak slightly different dialects of JSON; the lens normalizes
// them once, here.
package wire

import (
	"fmt"
	"strings"
	"time"
)

// FlexTime decodes timestamps that arrive in multiple layouts from the same
// API. Telnyx serves the same conversation with nanosecond precision in one
// response and microsecond precision minutes later, and recordings carry
// offset-less stamps ("2026-10-09T19:51:35"). Decoding through FlexTime makes
// every consumer robust to all three shapes (paid for live, 2026-10-10).
//
// FlexTime promotes time.Time methods on the RECEIVER only:
// ft.UTC() compiles, but x.After(ft) does not accept a FlexTime argument —
// write ft.Time.After(other.Time) explicitly.
type FlexTime struct {
	time.Time
}

func (t *FlexTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "null" || s == "" {
		return nil
	}

	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05"} {
		if pt, err := time.Parse(layout, s); err == nil {
			t.Time = pt

			return nil
		}
	}

	return fmt.Errorf("unparseable timestamp %q", s)
}
