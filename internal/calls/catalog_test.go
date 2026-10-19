package calls

import (
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func TestBuildCatalog(t *testing.T) {
	t.Parallel()

	cat := BuildCatalog()

	if cat.Title != CatalogTitle {
		t.Errorf("Title = %q, want %q", cat.Title, CatalogTitle)
	}

	if cat.Version != CatalogVersion {
		t.Errorf("Version = %q, want %q", cat.Version, CatalogVersion)
	}

	if len(cat.Services) != 1 {
		t.Fatalf("len(Services) = %d, want 1", len(cat.Services))
	}

	testService(t, cat.Services[0])
	testDomain(t, cat)
	testDataStore(t, cat)
	testFlow(t, cat)

	if len(cat.Users) != 1 || cat.Users[0].ID != "lars" {
		t.Errorf("Users = %v, want one user lars", cat.Users)
	}
}

func testService(t *testing.T, svc catalog.Service) {
	t.Helper()

	if svc.ID != "pbx-lens" {
		t.Errorf("Service.ID = %q, want %q", svc.ID, "pbx-lens")
	}

	if !slices.Equal(svc.Owners, []string{"lars"}) {
		t.Errorf("Service.Owners = %v, want [lars]", svc.Owners)
	}

	if svc.Summary == "" {
		t.Error("Service.Summary is empty")
	}

	cases := []struct {
		name string
		msgs []catalog.Message
		ids  []string
		dir  catalog.Direction
	}{
		{
			name: "commands",
			msgs: svc.Commands,
			ids:  []string{"call.record", "call.enrich", "call.retire"},
			dir:  catalog.Receives,
		},
		{
			name: "events",
			msgs: svc.Events,
			ids:  []string{"call.observed", "call.corrected", "call.enriched", "call.retired"},
			dir:  catalog.Sends,
		},
	}

	ids := make([]string, 0, len(svc.Commands)+len(svc.Events))

	for _, tc := range cases {
		if len(tc.msgs) != len(tc.ids) {
			t.Errorf("len(%s) = %d, want %d", tc.name, len(tc.msgs), len(tc.ids))

			continue
		}

		for i, id := range tc.ids {
			msg := tc.msgs[i]

			if string(msg.ID) != id {
				t.Errorf("%s[%d].ID = %q, want %q", tc.name, i, msg.ID, id)
			}

			if msg.Direction != tc.dir {
				t.Errorf("%s[%d].Direction = %q, want %q", tc.name, i, msg.Direction, tc.dir)
			}

			if msg.Summary == "" {
				t.Errorf("%s[%d] (%s) has an empty summary", tc.name, i, msg.ID)
			}

			if tc.dir == catalog.Sends && msg.Schema == nil {
				t.Errorf("%s[%d] (%s) has no derived schema", tc.name, i, msg.ID)
			}

			ids = append(ids, string(msg.ID))
		}
	}

	slices.Sort(ids)

	for i := 1; i < len(ids); i++ {
		if ids[i] == ids[i-1] {
			t.Errorf("duplicate message ID in catalog: %s", ids[i])
		}
	}
}

func testDomain(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	if len(cat.Domains) != 1 {
		t.Fatalf("len(Domains) = %d, want 1", len(cat.Domains))
	}

	domain := cat.Domains[0]

	if domain.ID != "telephony" {
		t.Errorf("Domain.ID = %q, want telephony", domain.ID)
	}

	if len(domain.Services) != 1 || domain.Services[0] != "pbx-lens" {
		t.Errorf("Domain.Services = %v, want [pbx-lens]", domain.Services)
	}
}

func testDataStore(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	if len(cat.DataStores) != 1 {
		t.Fatalf("len(DataStores) = %d, want 1", len(cat.DataStores))
	}

	if cat.DataStores[0].ID != "call-index" {
		t.Errorf("DataStores[0].ID = %q, want call-index", cat.DataStores[0].ID)
	}
}

func testFlow(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	if len(cat.Flows) != 1 {
		t.Fatalf("len(Flows) = %d, want 1", len(cat.Flows))
	}

	flow := cat.Flows[0]

	if flow.ID != "call-lifecycle-flow" {
		t.Errorf("Flow.ID = %q, want call-lifecycle-flow", flow.ID)
	}

	if len(flow.Steps) != 7 {
		t.Errorf("len(Flow.Steps) = %d, want 7", len(flow.Steps))
	}

	if !slices.Equal(flow.Owners, []string{"lars"}) {
		t.Errorf("Flow.Owners = %v, want [lars]", flow.Owners)
	}
}
