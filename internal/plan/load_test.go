package plan

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gmmorgan96/handle/internal/workitem"
)

const loadPlan = `## Load Project

A load fixture.

### Phase One - Setup ([P001](x))

**Purpose**: Set up the project.

**Features**
- [ ] ([F001](feature/feature-001.md)) First feature

**Dependencies**
- None
`

const loadFeature = `# F001 - First feature

## Description
The first feature.

## Tasks
- [ ] [T001](../task/task-001.md) First task

## Dependencies
- F002
- api-b/feature-004

## Acceptance Criteria
- [x] Done
- [ ] Pending

## Implementation
- Commit: (pending)
- Message: (pending)
`

const loadTask = `# T001 - First task

## Related Feature
- [F001](../feature/feature-001.md)

## Dependencies
- T002
- api-b/T004

## Acceptance Criteria
- [x] Done
`

const loadBug = `# B001 - First bug

## Related Feature
- [F001](../feature/feature-001.md)

## Phase
- P001

## Dependencies
- T003

## Acceptance Criteria
- [ ] Pending
`

func TestLoad(t *testing.T) {
	root := t.TempDir()
	planDir := filepath.Join(root, ".handle", "plan")
	writeFile(t, filepath.Join(planDir, "plan.md"), loadPlan)
	writeFile(t, filepath.Join(planDir, "feature", "feature-001.md"), loadFeature)
	writeFile(t, filepath.Join(planDir, "task", "task-001.md"), loadTask)
	writeFile(t, filepath.Join(planDir, "bug", "bug-001.md"), loadBug)

	result, err := Load("api-a", root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve temp root: %v", err)
	}
	if result.ServiceID != "api-a" {
		t.Errorf("ServiceID = %q, want api-a", result.ServiceID)
	}
	if result.Root != absoluteRoot {
		t.Errorf("Root = %q, want %q", result.Root, absoluteRoot)
	}
	if want := filepath.Join(absoluteRoot, ".handle", "plan"); result.PlanDir != want {
		t.Errorf("PlanDir = %q, want %q", result.PlanDir, want)
	}

	if result.Plan == nil {
		t.Fatal("Plan = nil")
	}
	if len(result.Plan.Phases) != 1 || result.Plan.Phases[0].ID != workitem.WorkItemID("api-a/P001") {
		t.Fatalf("Phases = %+v, want api-a/P001", result.Plan.Phases)
	}
	if len(result.Plan.Phases[0].Features) != 1 {
		t.Fatalf("Phase features = %+v, want one", result.Plan.Phases[0].Features)
	}
	if result.Plan.Phases[0].Features[0].ID != workitem.WorkItemID("api-a/F001") {
		t.Errorf("Phase feature = %q, want api-a/F001", result.Plan.Phases[0].Features[0].ID)
	}

	if len(result.Features) != 1 {
		t.Fatalf("len(Features) = %d, want 1", len(result.Features))
	}
	feature := result.Features[0]
	if feature.ID != workitem.WorkItemID("api-a/F001") {
		t.Errorf("Feature.ID = %q, want api-a/F001", feature.ID)
	}
	if len(feature.Tasks) != 1 || feature.Tasks[0].ID != workitem.WorkItemID("api-a/T001") {
		t.Errorf("Feature.Tasks = %+v, want api-a/T001", feature.Tasks)
	}
	wantDependencies := []workitem.WorkItemID{"api-a/F002", "api-b/feature-004"}
	if !reflect.DeepEqual(feature.Dependencies, wantDependencies) {
		t.Errorf("Feature.Dependencies = %v, want %v", feature.Dependencies, wantDependencies)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("len(Tasks) = %d, want 1", len(result.Tasks))
	}
	task := result.Tasks[0]
	if task.ID != workitem.WorkItemID("api-a/T001") {
		t.Errorf("Task.ID = %q, want api-a/T001", task.ID)
	}
	if task.Feature == nil || task.Feature.ID != workitem.WorkItemID("api-a/F001") {
		t.Errorf("Task.Feature = %+v, want api-a/F001", task.Feature)
	}
	wantTaskDependencies := []workitem.WorkItemID{"api-a/T002", "api-b/T004"}
	if !reflect.DeepEqual(task.Dependencies, wantTaskDependencies) {
		t.Errorf("Task.Dependencies = %v, want %v", task.Dependencies, wantTaskDependencies)
	}

	if len(result.Bugs) != 1 {
		t.Fatalf("len(Bugs) = %d, want 1", len(result.Bugs))
	}
	bug := result.Bugs[0]
	if bug.ID != workitem.WorkItemID("api-a/B001") {
		t.Errorf("Bug.ID = %q, want api-a/B001", bug.ID)
	}
	if bug.Feature == nil || bug.Feature.ID != workitem.WorkItemID("api-a/F001") {
		t.Errorf("Bug.Feature = %+v, want api-a/F001", bug.Feature)
	}
	if bug.Phase != workitem.WorkItemID("api-a/P001") {
		t.Errorf("Bug.Phase = %q, want api-a/P001", bug.Phase)
	}
	if !reflect.DeepEqual(bug.Dependencies, []workitem.WorkItemID{"api-a/T003"}) {
		t.Errorf("Bug.Dependencies = %v, want [api-a/T003]", bug.Dependencies)
	}
}

func TestLoadNamespacesIDsPerService(t *testing.T) {
	root := t.TempDir()
	planDir := filepath.Join(root, ".handle", "plan")
	writeFile(t, filepath.Join(planDir, "plan.md"), loadPlan)
	writeFile(t, filepath.Join(planDir, "feature", "feature-001.md"), loadFeature)
	writeFile(t, filepath.Join(planDir, "task", "task-001.md"), loadTask)

	a, err := Load("api-a", root)
	if err != nil {
		t.Fatalf("Load(api-a) returned error: %v", err)
	}
	b, err := Load("api-b", root)
	if err != nil {
		t.Fatalf("Load(api-b) returned error: %v", err)
	}

	if a.Features[0].ID == b.Features[0].ID {
		t.Errorf("feature ids collided: %q", a.Features[0].ID)
	}
	if a.Tasks[0].ID == b.Tasks[0].ID {
		t.Errorf("task ids collided: %q", a.Tasks[0].ID)
	}
	if a.Plan.Phases[0].ID == b.Plan.Phases[0].ID {
		t.Errorf("phase ids collided: %q", a.Plan.Phases[0].ID)
	}
	if a.Features[0].ID.Service() != "api-a" || b.Features[0].ID.Service() != "api-b" {
		t.Errorf("feature service prefixes = %q, %q", a.Features[0].ID.Service(), b.Features[0].ID.Service())
	}
}

func TestLoadEmptyPlanDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".handle", "plan"), 0o755); err != nil {
		t.Fatalf("create plan directory: %v", err)
	}

	result, err := Load("api-a", root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if result.Plan == nil {
		t.Fatal("Plan = nil, want empty plan")
	}
	if len(result.Plan.Phases) != 0 {
		t.Errorf("Phases = %+v, want none", result.Plan.Phases)
	}
	if len(result.Features) != 0 || len(result.Tasks) != 0 || len(result.Bugs) != 0 {
		t.Errorf("Features/Tasks/Bugs = %d/%d/%d, want 0/0/0", len(result.Features), len(result.Tasks), len(result.Bugs))
	}
}

func TestLoadWithoutWorkItemDirectories(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".handle", "plan", "plan.md"), loadPlan)

	result, err := Load("api-a", root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(result.Plan.Phases) != 1 {
		t.Errorf("Phases = %+v, want one", result.Plan.Phases)
	}
	if len(result.Features) != 0 || len(result.Tasks) != 0 || len(result.Bugs) != 0 {
		t.Errorf("Features/Tasks/Bugs = %d/%d/%d, want 0/0/0", len(result.Features), len(result.Tasks), len(result.Bugs))
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string
		id    string
		want  string
	}{
		{
			name:  "empty service id",
			setup: func(t *testing.T) string { return t.TempDir() },
			want:  "service id is empty",
		},
		{
			name:  "invalid service id",
			setup: func(t *testing.T) string { return t.TempDir() },
			id:    "API-A",
			want:  "invalid service id",
		},
		{
			name:  "empty root",
			setup: func(t *testing.T) string { return "" },
			id:    "api-a",
			want:  "service root is empty",
		},
		{
			name:  "missing plan directory",
			setup: func(t *testing.T) string { return t.TempDir() },
			id:    "api-a",
			want:  ".handle/plan",
		},
		{
			name: "plan path is a file",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				writeFile(t, filepath.Join(root, ".handle", "plan"), "not a directory")
				return root
			},
			id:   "api-a",
			want: "not a directory",
		},
		{
			name: "missing plan.md",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				writeFile(t, filepath.Join(root, ".handle", "plan", "feature", "feature-001.md"), loadFeature)
				return root
			},
			id:   "api-a",
			want: "plan.md",
		},
		{
			name: "malformed plan",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				writeFile(t, filepath.Join(root, ".handle", "plan", "plan.md"), "not a plan\n")
				return root
			},
			id:   "api-a",
			want: "plan.md",
		},
		{
			name: "malformed feature",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				planDir := filepath.Join(root, ".handle", "plan")
				writeFile(t, filepath.Join(planDir, "plan.md"), loadPlan)
				writeFile(t, filepath.Join(planDir, "feature", "feature-001.md"), "# Not an ID\n")
				return root
			},
			id:   "api-a",
			want: "feature-001.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(tt.id, tt.setup(t))
			if err == nil {
				t.Fatal("Load returned nil error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

// TestLoadRepositoryPlan parses this repository's real .handle tree. It proves
// that in-progress documents (with "(pending)" implementation fields) parse and
// that every id is namespaced to the registered service.
func TestLoadRepositoryPlan(t *testing.T) {
	result, err := Load("handle", filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("Load repository plan returned error: %v", err)
	}
	if len(result.Plan.Phases) == 0 {
		t.Error("Phases = none, want at least one")
	}
	if len(result.Features) == 0 {
		t.Error("Features = none, want at least one")
	}
	if len(result.Tasks) == 0 {
		t.Error("Tasks = none, want at least one")
	}
	for _, feature := range result.Features {
		if feature.ID.Service() != "handle" {
			t.Errorf("Feature.ID = %q, want handle prefix", feature.ID)
		}
	}
	for _, task := range result.Tasks {
		if task.ID.Service() != "handle" {
			t.Errorf("Task.ID = %q, want handle prefix", task.ID)
		}
	}
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
