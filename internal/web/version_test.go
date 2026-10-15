package web

import "testing"

// TestFormatMainVersion pins the footer version prefix: Go's Main.Version
// is already semver, so the formatter must not stack a second "v" (the
// vv0.1.1-0.… footer bug), while a bare version and the placeholder/devel
// cases keep their old contract.
func TestFormatMainVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "v0.1.1-0.20261015060056-b943b84583f9+dirty", want: "v0.1.1-0.20261015060056-b943b84583f9+dirty"},
		{in: "v0.1.0", want: "v0.1.0"},
		{in: "0.1.0", want: "v0.1.0"},
		{in: "(devel)", want: ""},
		{in: "", want: ""},
	}

	for _, tc := range cases {
		if got := formatMainVersion(tc.in); got != tc.want {
			t.Errorf("formatMainVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
