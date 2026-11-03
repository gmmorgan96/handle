package blockers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gmmorgan96/handle/internal/plan"
	"github.com/gmmorgan96/handle/internal/project"
	"github.com/gmmorgan96/handle/internal/workitem"
)

func TestResolveCrossServiceReferenceOnBothItems(t *testing.T) {
	caller := &plan.TaskDoc{
		ID:           "api-a/T001",
		Title:        "Call remote service",
		Dependencies: []workitem.WorkItemID{"api-b/T004"},
	}
	callee := &plan.TaskDoc{ID: "api-b/T004", Title: "Provide remote endpoint"}

	resolution := Resolve(&project.Project{Services: []project.Service{
		{ID: "api-a", Result: &plan.Result{ServiceID: "api-a", Tasks: []*plan.TaskDoc{caller}}},
		{ID: "api-b", Result: &plan.Result{ServiceID: "api-b", Tasks: []*plan.TaskDoc{callee}}},
	}})

	source := resolution.Lookup("api-a/T001")
	target := resolution.Lookup("api-b/T004")
	if source == nil || target == nil {
		t.Fatalf("resolved items = %v / %v, want both found", source, target)
	}
	if len(source.Dependencies) != 1 {
		t.Fatalf("source Dependencies = %d, want 1", len(source.Dependencies))
	}

	reference := source.Dependencies[0]
	if !reference.Resolved {
		t.Error("cross-service reference was not resolved")
	}
	if reference.Item != target {
		t.Errorf("reference.Item = %+v, want the api-b/T004 item", reference.Item)
	}
	if !reference.CrossService {
		t.Error("reference.CrossService = false, want true")
	}
	if reference.ID != "api-b/T004" {
		t.Errorf("reference.ID = %q, want api-b/T004", reference.ID)
	}

	if len(target.Dependents) != 1 || target.Dependents[0] != source {
		t.Errorf("target Dependents = %+v, want the source item", target.Dependents)
	}
	if len(resolution.Unresolved) != 0 {
		t.Errorf("Unresolved = %+v, want none", resolution.Unresolved)
	}
}

func TestResolveUnresolvedReferenceIsFlagged(t *testing.T) {
	source := &plan.TaskDoc{
		ID:    "api-a/T001",
		Title: "Needs missing work",
		Dependencies: []workitem.WorkItemID{
			"api-b/T999",
			"api-a/T999",
		},
	}
	existing := &plan.TaskDoc{ID: "api-b/T004", Title: "Existing work"}

	resolution := Resolve(&project.Project{Services: []project.Service{
		{ID: "api-a", Result: &plan.Result{ServiceID: "api-a", Tasks: []*plan.TaskDoc{source}}},
		{ID: "api-b", Result: &plan.Result{ServiceID: "api-b", Tasks: []*plan.TaskDoc{existing}}},
	}})

	resolvedSource := resolution.Lookup("api-a/T001")
	if len(resolvedSource.Dependencies) != 2 {
		t.Fatalf("Dependencies = %d, want 2", len(resolvedSource.Dependencies))
	}
	for _, reference := range resolvedSource.Dependencies {
		if reference.Resolved {
			t.Errorf("reference %q resolved, want unresolved", reference.ID)
		}
		if reference.Item != nil {
			t.Errorf("reference %q Item = %+v, want nil", reference.ID, reference.Item)
		}
	}

	if len(resolution.Unresolved) != 2 {
		t.Fatalf("Unresolved = %d, want 2", len(resolution.Unresolved))
	}
	crossService := unresolvedByTarget(t, resolution, "api-b/T999")
	if crossService.Source != resolvedSource {
		t.Errorf("unresolved Source = %+v, want source item", crossService.Source)
	}
	if !crossService.CrossService {
		t.Error("unresolved api-b/T999 CrossService = false, want true")
	}

	local := unresolvedByTarget(t, resolution, "api-a/T999")
	if local.CrossService {
		t.Error("unresolved api-a/T999 CrossService = true, want false")
	}

	if resolution.Lookup("api-b/T999") != nil {
		t.Error("Lookup returned an item for an unresolved reference")
	}
}

func TestResolveLocalReference(t *testing.T) {
	feature := &plan.FeatureDoc{
		ID:           "api-a/F001",
		Title:        "Feature",
		Dependencies: []workitem.WorkItemID{"api-a/T001"},
	}
	task := &plan.TaskDoc{ID: "api-a/T001", Title: "Task"}

	resolution := Resolve(&project.Project{Services: []project.Service{
		{ID: "api-a", Result: &plan.Result{
			ServiceID: "api-a",
			Features:  []*plan.FeatureDoc{feature},
			Tasks:     []*plan.TaskDoc{task},
		}},
	}})

	reference := resolution.Lookup("api-a/F001").Dependencies[0]
	if !reference.Resolved {
		t.Error("local reference was not resolved")
	}
	if reference.CrossService {
		t.Error("local reference CrossService = true, want false")
	}
	if reference.Item != resolution.Lookup("api-a/T001") {
		t.Error("local reference points at the wrong item")
	}
}

func TestResolveAllWorkItemKinds(t *testing.T) {
	source := &plan.Result{
		ServiceID: "api-a",
		Plan: &plan.Plan{Phases: []plan.Phase{
			{ID: "api-a/P001", Name: "Source phase", Dependencies: []workitem.WorkItemID{"api-b/P001"}},
		}},
		Features: []*plan.FeatureDoc{
			{ID: "api-a/F001", Title: "Source feature", Dependencies: []workitem.WorkItemID{"api-b/F001"}},
		},
		Tasks: []*plan.TaskDoc{
			{ID: "api-a/T001", Title: "Source task", Dependencies: []workitem.WorkItemID{"api-b/T001"}},
		},
		Bugs: []*plan.BugDoc{
			{ID: "api-a/B001", Title: "Source bug", Dependencies: []workitem.WorkItemID{"api-b/B001"}},
		},
	}
	target := &plan.Result{
		ServiceID: "api-b",
		Plan: &plan.Plan{Phases: []plan.Phase{
			{ID: "api-b/P001", Name: "Target phase"},
		}},
		Features: []*plan.FeatureDoc{
			{ID: "api-b/F001", Title: "Target feature"},
		},
		Tasks: []*plan.TaskDoc{
			{ID: "api-b/T001", Title: "Target task"},
		},
		Bugs: []*plan.BugDoc{
			{ID: "api-b/B001", Title: "Target bug"},
		},
	}

	resolution := Resolve(&project.Project{Services: []project.Service{
		{ID: "api-a", Result: source},
		{ID: "api-b", Result: target},
	}})

	tests := []struct {
		sourceID workitem.WorkItemID
		targetID workitem.WorkItemID
		kind     Kind
	}{
		{"api-a/P001", "api-b/P001", KindPhase},
		{"api-a/F001", "api-b/F001", KindFeature},
		{"api-a/T001", "api-b/T001", KindTask},
		{"api-a/B001", "api-b/B001", KindBug},
	}

	for _, test := range tests {
		sourceItem := resolution.Lookup(test.sourceID)
		targetItem := resolution.Lookup(test.targetID)
		if sourceItem == nil || targetItem == nil {
			t.Fatalf("%s / %s not resolved", test.sourceID, test.targetID)
		}
		if targetItem.Kind != test.kind {
			t.Errorf("%s Kind = %q, want %q", test.targetID, targetItem.Kind, test.kind)
		}
		if len(sourceItem.Dependencies) != 1 || !sourceItem.Dependencies[0].Resolved {
			t.Errorf("%s dependency = %+v, want one resolved reference", test.sourceID, sourceItem.Dependencies)
		}
		if len(targetItem.Dependents) != 1 || targetItem.Dependents[0] != sourceItem {
			t.Errorf("%s Dependents = %+v, want the source item", test.targetID, targetItem.Dependents)
		}
	}
}

func TestResolveBrokenServiceReferencesStayUnresolved(t *testing.T) {
	source := &plan.TaskDoc{
		ID:           "api-a/T001",
		Title:        "Blocked by broken service",
		Dependencies: []workitem.WorkItemID{"broken/T001"},
	}
	broken := project.Service{ID: "broken", Result: nil, Err: os.ErrInvalid}

	resolution := Resolve(&project.Project{Services: []project.Service{
		{ID: "api-a", Result: &plan.Result{ServiceID: "api-a", Tasks: []*plan.TaskDoc{source}}},
		broken,
	}})

	if len(resolution.Items) != 1 {
		t.Errorf("Items = %d, want only the loaded item", len(resolution.Items))
	}
	if len(resolution.Unresolved) != 1 {
		t.Fatalf("Unresolved = %+v, want one entry", resolution.Unresolved)
	}
	if resolution.Unresolved[0].Target != "broken/T001" {
		t.Errorf("unresolved Target = %q, want broken/T001", resolution.Unresolved[0].Target)
	}
	if !resolution.Unresolved[0].CrossService {
		t.Error("unresolved broken/T001 CrossService = false, want true")
	}
	if resolution.Unresolved[0].Source != resolution.Lookup("api-a/T001") {
		t.Error("unresolved Source does not point at the declaring item")
	}
}

func TestResolveNilAndEmptyProject(t *testing.T) {
	for name, prj := range map[string]*project.Project{
		"nil":   nil,
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			resolution := Resolve(prj)
			if len(resolution.Items) != 0 {
				t.Errorf("Items = %d, want 0", len(resolution.Items))
			}
			if len(resolution.Unresolved) != 0 {
				t.Errorf("Unresolved = %d, want 0", len(resolution.Unresolved))
			}
			if resolution.Lookup("api-a/T001") != nil {
				t.Error("Lookup returned an item for an empty project")
			}
		})
	}
}

func TestResolveLoadedCrossServiceProject(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "api-a", "Source task", "- api-b/T001")
	writeService(t, root, "api-b", "Target task", "- None")
	writeFile(t, filepath.Join(root, ".project", "services.yaml"), `project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
  - id: api-b
    name: API B
    path: ./api-b
`)

	prj, err := project.Load(root)
	if err != nil {
		t.Fatalf("project.Load: %v", err)
	}

	resolution := Resolve(prj)
	source := resolution.Lookup("api-a/T001")
	target := resolution.Lookup("api-b/T001")
	if source == nil {
		t.Fatal("api-a/T001 not resolved from the parsed project")
	}
	if target == nil {
		t.Fatal("api-b/T001 not resolved from the parsed project")
	}
	if len(source.Dependencies) != 1 {
		t.Fatalf("api-a/T001 Dependencies = %d, want 1", len(source.Dependencies))
	}
	reference := source.Dependencies[0]
	if !reference.Resolved || reference.Item != target {
		t.Errorf("parsed cross-service reference = %+v, want resolved target", reference)
	}
	if !reference.CrossService {
		t.Error("parsed cross-service reference CrossService = false, want true")
	}
	if len(target.Dependents) != 1 || target.Dependents[0] != source {
		t.Errorf("target Dependents = %+v, want the parsed source item", target.Dependents)
	}
}

func unresolvedByTarget(t *testing.T, resolution *Resolution, target workitem.WorkItemID) Unresolved {
	t.Helper()
	for _, unresolved := range resolution.Unresolved {
		if unresolved.Target == target {
			return unresolved
		}
	}
	t.Fatalf("no unresolved reference for %q", target)
	return Unresolved{}
}

func writeService(t *testing.T, root, id, taskTitle, dependencies string) {
	t.Helper()
	planDir := filepath.Join(root, id, ".handle", "plan")
	writeFile(t, filepath.Join(planDir, "plan.md"), `## Example Project

An example project.

### Phase One - Setup ([P001](x))

**Purpose**: Set up the project.

**Features**
- [ ] ([F001](feature/feature-001.md)) First feature

**Dependencies**
- None
`)
	writeFile(t, filepath.Join(planDir, "feature", "feature-001.md"), `# F001 - First feature

## Description
The first feature.

## Tasks
- [ ] [T001](../task/task-001.md) The first task

## Acceptance Criteria
- [x] Done
`)
	writeFile(t, filepath.Join(planDir, "task", "task-001.md"), `# T001 - `+taskTitle+`

## Related Feature
- [F001](../feature/feature-001.md)

## Dependencies
`+dependencies+`

## Acceptance Criteria
- [ ] Done
`)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
