package main

import (
	"io"
	"strings"
	"testing"
	"time"
)

func TestTmpBodyDriftProbe(t *testing.T) {
	base, client := serveTestStack(t)

	fetch := func() string {
		resp, err := client.Get(base + "/frag/dashboard")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		return string(b)
	}

	first := fetch()
	time.Sleep(1250 * time.Millisecond)
	second := fetch()

	if first == second {
		t.Log("BODIES IDENTICAL across 1.25s")
		return
	}

	t.Log("BODIES DIFFER across 1.25s")
	for i := 0; i < len(first) && i < len(second); i++ {
		if first[i] != second[i] {
			lo := max(i-120, 0)
			t.Logf("first diff at byte %d:\n...%q\nvs\n...%q", i, first[lo:min(i+120, len(first))], second[lo:min(i+120, len(second))])

			return
		}
	}

	t.Logf("length differs: %d vs %d; tails:\n%q\nvs\n%q", len(first), len(second), first[max(0, len(first)-160):], second[max(0, len(second)-160):])
	_ = strings.TrimSpace
}
