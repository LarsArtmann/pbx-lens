package wire

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestFlexTimeDecodesEverySourceDialect(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    time.Time
		wantErr error
	}{
		{
			name: "json null decodes to zero",
			json: `null`,
		},
		{
			name: "empty string decodes to zero",
			json: `""`,
		},
		{
			name: "rfc3339 nano keeps sub-second precision",
			json: `"2026-10-09T19:51:35.123456789Z"`,
			want: time.Date(2026, 10, 9, 19, 51, 35, 123456789, time.UTC),
		},
		{
			name: "offset-less stamp reads as utc",
			json: `"2026-10-09T19:51:35"`,
			want: time.Date(2026, 10, 9, 19, 51, 35, 0, time.UTC),
		},
		{
			name: "explicit offset keeps its zone",
			json: `"2026-10-09T21:51:35+02:00"`,
			want: time.Date(2026, 10, 9, 21, 51, 35, 0, time.FixedZone("", 2*3600)),
		},
		{
			name:    "prose is unparseable",
			json:    `"not-a-timestamp"`,
			wantErr: ErrUnparseableTimestamp,
		},
		{
			name:    "a bare unix number is unparseable",
			json:    `1760041895`,
			wantErr: ErrUnparseableTimestamp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got FlexTime

			err := got.UnmarshalJSON([]byte(tt.json))

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("UnmarshalJSON(%s) error = %v, want wrapping %v", tt.json, err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("UnmarshalJSON(%s) unexpected error: %v", tt.json, err)
			}

			if !got.Equal(tt.want) {
				t.Errorf("UnmarshalJSON(%s) = %v, want %v", tt.json, got.Time, tt.want)
			}
		})
	}
}

// A FlexTime must survive a marshal/unmarshal round trip at the same
// instant: every wire DTO stamps its fields with one, and the mappers hash
// truncations of the result.
func TestFlexTimeJSONRoundTripKeepsInstant(t *testing.T) {
	in := FlexTime{Time: time.Date(2026, 10, 9, 19, 51, 35, 123456789, time.UTC)}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var out FlexTime

	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal(%s): %v", raw, err)
	}

	if !out.Equal(in.Time) {
		t.Errorf("round trip = %v, want %v", out.Time, in.Time)
	}
}
