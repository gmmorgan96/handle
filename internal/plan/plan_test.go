package plan

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gmmorgan96/handle/internal/workitem"
)

const samplePlan = `## Sample Project

A sample project used to test plan parsing.

### Phase One - Setup ([P001](../../.project/project.md#phase-one-setup (P001)))

**Purpose**: Establish the foundation.

**Features**
- [ ] ([F001](feature/feature-001.md)) Initialize the module
- [x] ([F002](feature/feature-002.md)) Add configuration

**Dependencies**
- None

### Phase Two - Core ([P002](../../.project/project.md#phase-two-core (P002)))

**Purpose**: Build the core.

**Features**
- [ ] ([F003](feature/feature-003.md)) Implement the core

**Dependencies**
- [ ] P001
`

func TestParsePlan(t *testing.T) {
	plan, err := ParsePlan(strings.NewReader(samplePlan))
	if err != nil {
		t.Fatalf("ParsePlan returned error: %v", err)
	}

	if plan.Title != "Sample Project" {
		t.Errorf("Title = %q, want %q", plan.Title, "Sample Project")
	}
	if plan.Summary != "A sample project used to test plan parsing." {
		t.Errorf("Summary = %q", plan.Summary)
	}
	if len(plan.Phases) != 2 {
		t.Fatalf("len(Phases) = %d, want 2", len(plan.Phases))
	}

	first := plan.Phases[0]
	if first.ID != workitem.WorkItemID("P001") {
		t.Errorf("Phases[0].ID = %q, want P001", first.ID)
	}
	if first.Name != "Phase One - Setup" {
		t.Errorf("Phases[0].Name = %q, want %q", first.Name, "Phase One - Setup")
	}
	if first.Purpose != "Establish the foundation." {
		t.Errorf("Phases[0].Purpose = %q", first.Purpose)
	}
	if len(first.Features) != 2 {
		t.Fatalf("len(Phases[0].Features) = %d, want 2", len(first.Features))
	}
	if first.Features[0] != (FeatureRef{ID: "F001", Title: "Initialize the module", Path: "feature/feature-001.md"}) {
		t.Errorf("Phases[0].Features[0] = %+v", first.Features[0])
	}
	if first.Features[1] != (FeatureRef{ID: "F002", Title: "Add configuration", Path: "feature/feature-002.md"}) {
		t.Errorf("Phases[0].Features[1] = %+v", first.Features[1])
	}
	if len(first.Dependencies) != 0 {
		t.Errorf("Phases[0].Dependencies = %v, want none", first.Dependencies)
	}

	second := plan.Phases[1]
	if second.ID != workitem.WorkItemID("P002") {
		t.Errorf("Phases[1].ID = %q, want P002", second.ID)
	}
	if len(second.Features) != 1 || second.Features[0].ID != "F003" {
		t.Errorf("Phases[1].Features = %+v, want F003", second.Features)
	}
	if !reflect.DeepEqual(second.Dependencies, []workitem.WorkItemID{"P001"}) {
		t.Errorf("Phases[1].Dependencies = %v, want [P001]", second.Dependencies)
	}
}

func TestParsePlanDependencies(t *testing.T) {
	tests := []struct {
		name         string
		dependencies string
		want         []workitem.WorkItemID
	}{
		{name: "none", dependencies: "- None"},
		{name: "single", dependencies: "- [ ] P001", want: []workitem.WorkItemID{"P001"}},
		{name: "multiple", dependencies: "- [ ] P001\n- [ ] P002", want: []workitem.WorkItemID{"P001", "P002"}},
		{name: "plain bullet", dependencies: "- P003", want: []workitem.WorkItemID{"P003"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "## Plan\n\n### Phase One - One ([P001](x))\n\n**Purpose**: Test.\n\n**Features**\n\n**Dependencies**\n" + tt.dependencies + "\n"
			plan, err := ParsePlan(strings.NewReader(input))
			if err != nil {
				t.Fatalf("ParsePlan returned error: %v", err)
			}
			if !reflect.DeepEqual(plan.Phases[0].Dependencies, tt.want) {
				t.Errorf("Dependencies = %v, want %v", plan.Phases[0].Dependencies, tt.want)
			}
		})
	}
}

func TestParsePlanEmpty(t *testing.T) {
	for _, input := range []string{"", "   \n\n", "## Plan\n\nOnly a summary.\n"} {
		plan, err := ParsePlan(strings.NewReader(input))
		if err != nil {
			t.Fatalf("ParsePlan(%q) returned error: %v", input, err)
		}
		if len(plan.Phases) != 0 {
			t.Errorf("ParsePlan(%q).Phases = %+v, want none", input, plan.Phases)
		}
	}
}

func TestParsePlanErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "content without title", input: "Summary only.\n", want: "title"},
		{name: "malformed phase", input: "## Plan\n\n### Phase One\n", want: "phase heading"},
		{name: "malformed phase id", input: "## Plan\n\n### Phase One - One ([P001x](x))\n", want: "phase id"},
		{
			name:  "malformed feature",
			input: "## Plan\n\n### Phase One - One ([P001](x))\n\n**Features**\n- not a feature\n",
			want:  "feature",
		},
		{
			name:  "malformed dependency",
			input: "## Plan\n\n### Phase One - One ([P001](x))\n\n**Dependencies**\n- TBD\n",
			want:  "dependency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePlan(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("ParsePlan returned nil error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}
