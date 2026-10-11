package ingest

import (
	"strings"
	"testing"
)

func TestEachLineTrimsAndSkipsBlanks(t *testing.T) {
	var lines []string

	err := EachLine(strings.NewReader("  a  \n\n\t\nb\n   \nc"), func(line string) {
		lines = append(lines, line)
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"a", "b", "c"}; strings.Join(lines, ",") != strings.Join(want, ",") {
		t.Errorf("lines = %q, want %q", lines, want)
	}
}

func TestEachLineGrowsToBoundedCeiling(t *testing.T) {
	big := strings.Repeat("x", 100*1024) // above the 64 KiB initial buffer

	var sizes []int

	err := EachLine(strings.NewReader(big), func(line string) {
		sizes = append(sizes, len(line))
	})
	if err != nil {
		t.Fatalf("a 100 KiB line must pass (initial buffer grows): %v", err)
	}

	if len(sizes) != 1 || sizes[0] != len(big) {
		t.Fatalf("sizes = %v, want one line of %d bytes", sizes, len(big))
	}
}

func TestEachLineRefusesOverCeiling(t *testing.T) {
	tooBig := strings.Repeat("x", 1024*1024+1) // above the 1 MiB ceiling

	err := EachLine(strings.NewReader(tooBig), func(string) {})
	if err == nil {
		t.Fatal("a line over the 1 MiB ceiling must error, not be silently truncated")
	}
}

func TestNewScannerBufferPolicyConstants(t *testing.T) {
	if scannerInitialBuf != 64*1024 || scannerMaxLine != 1024*1024 {
		t.Fatalf("buffer policy drifted: initial=%d max=%d", scannerInitialBuf, scannerMaxLine)
	}
}
