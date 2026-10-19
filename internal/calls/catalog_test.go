package calls

import (
	"slices"
	"testing"
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
	svc := cat.Services[0]
	if svc.ID != "pbx-lens" {
		t.Errorf("Service.ID = %q, want %q", svc.ID, "pbx-lens")
	}
	if !slices.Equal(svc.Owners, []string{"lars"}) {
		t.Errorf("Service.Owners = %v, want [lars]", svc.Owners)
	}
	if svc.Summary == "" {
		t.Error("Service.Summary is empty")
	}

	wantCommands := []string{"call.record", "call.enrich", "call.retire"}
	if len(svc.Commands) != len(wantCommands) {
		t.Fatalf("len(Commands) = %d, want %d", len(svc.Commands), len(wantCommands))
	}
	for i, want := range wantCommands {
		if got := string(svc.Commands[i].ID); got != want {
			t.Errorf("Commands[%d].ID = %q, want %q", i, got, want)
		}
		if svc.Commands[i].Direction != "receives" {
			t.Errorf("Commands[%d].Direction = %q, want receives", i, svc.Commands[i].Direction)
		}
		if svc.Commands[i].Summary == "" {
			t.Errorf("Commands[%d] (%s) has an empty summary", i, svc.Commands[i].ID)
		}
	}

	wantEvents := []string{"call.observed", "call.corrected", "call.enriched", "call.retired"}
	if len(svc.Events) != len(wantEvents) {
		t.Fatalf("len(Events) = %d, want %d", len(svc.Events), len(wantEvents))
	}
	for i, want := range wantEvents {
		if got := string(svc.Events[i].ID); got != want {
			t.Errorf("Events[%d].ID = %q, want %q", i, got, want)
		}
		if svc.Events[i].Direction != "sends" {
			t.Errorf("Events[%d].Direction = %q, want sends", i, svc.Events[i].Direction)
		}
		if svc.Events[i].Summary == "" {
			t.Errorf("Events[%d] (%s) has an empty summary", i, svc.Events[i].ID)
		}
		if svc.Events[i].Schema == nil {
			t.Errorf("Events[%d] (%s) has no derived schema", i, svc.Events[i].ID)
		}
	}

	ids := make([]string, 0, len(svc.Commands)+len(svc.Events))
	for _, msg := range append(slices.Clone(svc.Commands), svc.Events...) {
		ids = append(ids, string(msg.ID))
	}
	slices.Sort(ids)
	for i := 1; i < len(ids); i++ {
		if ids[i] == ids[i-1] {
			t.Errorf("duplicate message ID in catalog: %s", ids[i])
		}
	}

	if len(cat.Users) != 1 || cat.Users[0].ID != "lars" {
		t.Errorf("Users = %v, want one user lars", cat.Users)
	}

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

	if len(cat.DataStores) != 1 || cat.DataStores[0].ID != "call-index" {
		t.Errorf("DataStores = %v, want one store call-index", cat.DataStores)
	}

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
