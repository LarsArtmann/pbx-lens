package web

import (
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
)

func renderToString(t *testing.T, c templ.Component) string {
	t.Helper()

	html, err := templ.RenderString(c)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	return html
}

// TestRelativeTimeDayPrecisionContract pins the pinned library's relative-time
// behavior the operator relies on (TODO_LIST "RelativeTime tooltip" row,
// closed 2026-10-17): every rendering carries the exact timestamp as the
// title tooltip, and very old calls fall back to day-precision absolute
// text instead of an unhelpful "N months ago". Ages 7-30 days stay
// week-granular in the visible text — the tooltip is the precision.
func TestRelativeTimeDayPrecisionContract(t *testing.T) {
	now := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		age      time.Duration
		wantText string
	}{
		{"recent stays relative", 2 * time.Hour, "2 hours ago"},
		{"a week stays week-granular", 10 * 24 * time.Hour, "1 week ago"},
		{"past a month falls back to the absolute date", 40 * 24 * time.Hour, "Oct 7, 2026"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := renderToString(t, display.RelativeTime(display.RelativeTimeProps{
				Time: now.Add(-tc.age),
				Now:  now,
			}))

			if !strings.Contains(html, ">"+tc.wantText+"</time>") {
				t.Errorf("visible text for age %s = want %q in %q", tc.age, tc.wantText, html)
			}

			if !strings.Contains(html, `title="`) {
				t.Errorf("age %s rendering must carry the exact-time title tooltip: %q", tc.age, html)
			}

			if !strings.Contains(html, `datetime="`) {
				t.Errorf("age %s rendering must keep the machine-readable datetime: %q", tc.age, html)
			}
		})
	}
}
