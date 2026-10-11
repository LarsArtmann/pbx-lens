package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveKeyFromEnv(t *testing.T) {
	t.Setenv("TELNYX_API_KEY", "  key0123  ")

	key, err := ResolveKey("")
	if err != nil {
		t.Fatalf("ResolveKey(env) error: %v", err)
	}

	if key != "key0123" {
		t.Fatalf("ResolveKey(env) = %q, want trimmed key0123", key)
	}
}

func TestResolveKeyEnvMissing(t *testing.T) {
	t.Setenv("TELNYX_API_KEY", "")

	if _, err := ResolveKey(""); err == nil {
		t.Fatal("ResolveKey(missing env) should reject")
	}
}

func TestResolveKeyFileWins(t *testing.T) {
	t.Setenv("TELNYX_API_KEY", "from-env")

	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, []byte(" from-file\n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	key, err := ResolveKey(path)
	if err != nil {
		t.Fatalf("ResolveKey(file) error: %v", err)
	}

	if key != "from-file" {
		t.Fatalf("ResolveKey(file) = %q, want from-file", key)
	}
}

func TestResolveKeyFileEmptyRejects(t *testing.T) {
	t.Setenv("TELNYX_API_KEY", "from-env")

	path := filepath.Join(t.TempDir(), "empty-key")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	if key, err := ResolveKey(path); err == nil {
		t.Fatalf("ResolveKey(empty file) = %q, want rejection", key)
	}
}

func TestParseSince(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{name: "rfc3339", input: "2026-10-09T19:51:25Z", want: time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)},
		{name: "duration", input: "0s"},
		{name: "garbage", input: "yesterday", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseSince(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseSince(%q) should reject", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseSince(%q) error: %v", tc.input, err)
			}

			if tc.name == "rfc3339" && !got.Equal(tc.want) {
				t.Fatalf("ParseSince(%q) = %v, want %v", tc.input, got, tc.want)
			}

			if tc.name == "duration" && got.After(time.Now().UTC()) {
				t.Fatalf("ParseSince(%q) = %v, should be in the past", tc.input, got)
			}
		})
	}
}

func TestParseDirection(t *testing.T) {
	for _, in := range []string{"inbound", "outbound"} {
		if _, err := ParseDirection(in); err != nil {
			t.Fatalf("ParseDirection(%q) error: %v", in, err)
		}
	}

	if _, err := ParseDirection("sideways"); err == nil {
		t.Fatal("ParseDirection(sideways) should reject")
	}
}
