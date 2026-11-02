package workitem

import (
	"encoding/json"
	"testing"
)

func TestNamespacedIDsDoNotCollide(t *testing.T) {
	a := NewID("api-a", "feature-001")
	b := NewID("api-b", "feature-001")

	if a == b {
		t.Fatalf("namespaced ids collided: %q", a)
	}
	if a.String() != "api-a/feature-001" {
		t.Errorf("api-a id = %q, want %q", a, "api-a/feature-001")
	}
	if b.String() != "api-b/feature-001" {
		t.Errorf("api-b id = %q, want %q", b, "api-b/feature-001")
	}
	if a.Service() != "api-a" || b.Service() != "api-b" {
		t.Errorf("service prefixes = %q, %q; want api-a, api-b", a.Service(), b.Service())
	}
	if a.Local() != "feature-001" || b.Local() != "feature-001" {
		t.Errorf("local ids = %q, %q; want feature-001, feature-001", a.Local(), b.Local())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	ids := []WorkItemID{
		NewID("api-a", "feature-001"),
		NewID("api-b", "feature-001"),
		WorkItemID("feature-001"),
		WorkItemID("T001"),
	}

	for _, id := range ids {
		t.Run(string(id), func(t *testing.T) {
			data, err := json.Marshal(id)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if got, want := string(data), `"`+string(id)+`"`; got != want {
				t.Errorf("Marshal() = %s, want %s", got, want)
			}

			var roundTripped WorkItemID
			if err := json.Unmarshal(data, &roundTripped); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if roundTripped != id {
				t.Errorf("round trip = %q, want %q", roundTripped, id)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		raw        string
		service    string
		local      string
		namespaced bool
	}{
		{raw: "api-a/feature-001", service: "api-a", local: "feature-001", namespaced: true},
		{raw: "api-b/T001", service: "api-b", local: "T001", namespaced: true},
		{raw: "feature-001", local: "feature-001"},
		{raw: "bug-012", local: "bug-012"},
		{raw: "T001", local: "T001"},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			id, err := Parse(tt.raw)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if id.String() != tt.raw {
				t.Errorf("String() = %q, want %q", id, tt.raw)
			}
			if id.Service() != tt.service {
				t.Errorf("Service() = %q, want %q", id.Service(), tt.service)
			}
			if id.Local() != tt.local {
				t.Errorf("Local() = %q, want %q", id.Local(), tt.local)
			}
			if id.IsNamespaced() != tt.namespaced {
				t.Errorf("IsNamespaced() = %v, want %v", id.IsNamespaced(), tt.namespaced)
			}
		})
	}
}

func TestParseRejectsInvalidIDs(t *testing.T) {
	for _, raw := range []string{
		"",
		"api-a",
		"api_a/feature-001",
		"API-A/feature-001",
		"api-a/feature",
		"api-a/",
		"/feature-001",
		"api-a/feature-001/extra",
		"feature-001 ",
		"api-a/F001x",
	} {
		t.Run(raw, func(t *testing.T) {
			if id, err := Parse(raw); err == nil {
				t.Errorf("Parse(%q) = %q, nil; want error", raw, id)
			}
		})
	}
}

func TestLocalIDsRemainLocal(t *testing.T) {
	id, err := Parse("feature-001")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if id.IsNamespaced() {
		t.Error("IsNamespaced() = true, want false")
	}
	if !id.IsLocal() {
		t.Error("IsLocal() = false, want true")
	}
	if id.Service() != "" {
		t.Errorf("Service() = %q, want empty", id.Service())
	}
	if id.String() != "feature-001" {
		t.Errorf("String() = %q, want unchanged %q", id, "feature-001")
	}
	if got := id.Namespaced("api-a"); got != NewID("api-a", "feature-001") {
		t.Errorf("Namespaced() = %q, want %q", got, "api-a/feature-001")
	}
}

func TestNamespacedLeavesNamespacedIDUnchanged(t *testing.T) {
	id := NewID("api-a", "feature-001")
	if got := id.Namespaced("api-b"); got != id {
		t.Errorf("Namespaced() = %q, want unchanged %q", got, id)
	}
}

func TestUnmarshalJSONRejectsInvalidIDs(t *testing.T) {
	for _, data := range []string{
		`"not a valid id"`,
		`""`,
		`42`,
		`null`,
	} {
		t.Run(data, func(t *testing.T) {
			var id WorkItemID
			if err := json.Unmarshal([]byte(data), &id); err == nil {
				t.Errorf("Unmarshal(%s) = %q, nil; want error", data, id)
			}
		})
	}
}
