package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// ResolveKey finds the Telnyx API key: an explicit key file wins, then the
// TELNYX_API_KEY environment variable. Both are trimmed; an empty key file
// rejects rather than silently falling through to the environment.
func ResolveKey(keyFile string) (string, error) {
	if keyFile != "" {
		raw, err := os.ReadFile(keyFile)
		if err != nil {
			return "", err
		}

		key := strings.TrimSpace(string(raw))
		if key == "" {
			return "", errorfamily.NewRejection(
				"lens.cli.empty_key_file",
				"key file is empty: "+keyFile,
			)
		}

		return key, nil
	}

	key := strings.TrimSpace(os.Getenv("TELNYX_API_KEY"))
	if key == "" {
		return "", errorfamily.NewRejection("lens.cli.no_api_key", "no API key: pass -key-file or set TELNYX_API_KEY")
	}

	return key, nil
}

// ParseSince accepts either an RFC3339 timestamp (an absolute cutoff) or a Go
// duration (relative to now, UTC) for look-back windows.
func ParseSince(s string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, s); err == nil {
		return ts, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, errorfamily.NewRejection(
			"lens.cli.bad_since",
			fmt.Sprintf("-since must be a duration (48h) or RFC3339 timestamp, got %q", s),
		)
	}

	return time.Now().UTC().Add(-d), nil
}

// ParseDirection validates the caller-declared conversation direction flag.
func ParseDirection(s string) (calls.Direction, error) {
	dir := calls.Direction(s)
	if !dir.Valid() {
		return "", errorfamily.NewRejection(
			"lens.cli.bad_direction",
			fmt.Sprintf("invalid -direction %q: must be outbound or inbound", s),
		)
	}

	return dir, nil
}
