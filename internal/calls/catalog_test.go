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
	testDataStores(t, cat)
	testChannel(t, cat)
	testFlow(t, cat)

	// The library's own integrity check (duplicate IDs across commands and
	// events, empty required fields, dangling references) replaces the
	// hand-rolled duplicate sweep this suite used to carry.
	if violations := cat.Validate(); len(violations) > 0 {
		t.Fatalf("catalog.Validate(): %d violations", len(violations))
	}

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

	if !slices.Equal(svc.ReadsFrom, []catalog.DataStoreID{"freeswitch", "telnyx"}) {
		t.Errorf("Service.ReadsFrom = %v, want [freeswitch telnyx]", svc.ReadsFrom)
	}

	if !slices.Equal(svc.WritesTo, []catalog.DataStoreID{"call-index"}) {
		t.Errorf("Service.WritesTo = %v, want [call-index]", svc.WritesTo)
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

	for _, tc := range cases {
		if len(tc.msgs) != len(tc.ids) {
			t.Errorf("len(%s) = %d, want %d", tc.name, len(tc.msgs), len(tc.ids))

			continue
		}

		for i, id := range tc.ids {
			testCatalogMessage(t, tc.name, i, tc.msgs[i], id, tc.dir)
		}
	}
}

func testCatalogMessage(t *testing.T, name string, i int, msg catalog.Message, id string, dir catalog.Direction) {
	t.Helper()

	if string(msg.ID) != id {
		t.Errorf("%s[%d].ID = %q, want %q", name, i, msg.ID, id)
	}

	if msg.Direction != dir {
		t.Errorf("%s[%d].Direction = %q, want %q", name, i, msg.Direction, dir)
	}

	if msg.Summary == "" {
		t.Errorf("%s[%d] (%s) has an empty summary", name, i, msg.ID)
	}

	if dir == catalog.Sends && msg.Schema == nil {
		t.Errorf("%s[%d] (%s) has no derived schema", name, i, msg.ID)
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

func testDataStores(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	wantIDs := []string{"call-index", "freeswitch", "telnyx"}
	if len(cat.DataStores) != len(wantIDs) {
		t.Fatalf("len(DataStores) = %d, want %d", len(cat.DataStores), len(wantIDs))
	}

	for i, want := range wantIDs {
		ds := cat.DataStores[i]

		if string(ds.ID) != want {
			t.Errorf("DataStores[%d].ID = %q, want %q", i, ds.ID, want)
		}

		if !slices.Equal(ds.Owners, []string{"lars"}) {
			t.Errorf("DataStores[%d] (%s).Owners = %v, want [lars]", i, ds.ID, ds.Owners)
		}

		if string(ds.Version) != CatalogVersion {
			t.Errorf("DataStores[%d] (%s).Version = %q, want %q", i, ds.ID, ds.Version, CatalogVersion)
		}

		if ds.Summary == "" {
			t.Errorf("DataStores[%d] (%s) has an empty summary", i, ds.ID)
		}
	}
}

func testChannel(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	if len(cat.Channels) != 1 {
		t.Fatalf("len(Channels) = %d, want 1", len(cat.Channels))
	}

	channel := cat.Channels[0]

	if channel.ID != "pbx-receiver-webhook" {
		t.Errorf("Channel.ID = %q, want pbx-receiver-webhook", channel.ID)
	}

	if !slices.Equal(channel.Owners, []string{"lars"}) {
		t.Errorf("Channel.Owners = %v, want [lars]", channel.Owners)
	}

	if string(channel.Version) != CatalogVersion {
		t.Errorf("Channel.Version = %q, want %q", channel.Version, CatalogVersion)
	}

	if channel.Summary == "" {
		t.Error("Channel.Summary is empty")
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
