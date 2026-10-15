package web

import (
	"fmt"
	"os"
	"testing"
	"time"
	_ "time/tzdata" // goldens render Europe/Berlin; resolve the zone without a system tz database
)

// TestMain pins the package to the operator zone: goldens and staleness
// fixtures render Europe/Berlin (owner decision R2) and CI pins TZ — the pin
// removes the machine clock from the equation for every local run too. The
// v0.2.0 CI red (09:01 UTC vs 11:01 Berlin in a golden) is the class this
// kills; time/tzdata (imported above) keeps the zone resolvable on machines
// without the system tz database.
func TestMain(m *testing.M) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "load Europe/Berlin:", err)

		os.Exit(1)
	}

	time.Local = berlin //nolint:gosmopolitan // the pin IS the point: this package's renders are defined in the operator zone

	os.Exit(m.Run())
}
