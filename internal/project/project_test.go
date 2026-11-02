package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validPlan = `## Example Project

An example project.

### Phase One - Setup ([P001](x))

**Purpose**: Set up the project.

**Features**
- [ ] ([F001](feature/feature-001.md)) First feature

**Dependencies**
- None
`

const validFeature = `# F001 - First feature

## Description
The first feature.

## Tasks
- [ ] [T001](../task/task-001.md) First task

## Acceptance Criteria
- [x] Done
`

const validTask = `# T001 - First task

## Related Feature
- [F001](../feature/feature-001.md)

## Acceptance Criteria
- [x] Done
`

func TestLoadAggregatesMultipleServices(t *testing.T) {
	root := t.TempDir()
	writeValidService(t, root, "api-a")
	writeValidService(t, root, "api-b")
	writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
    description: Public-facing API
  - id: api-b
    name: API B
    path: ./api-b
`)

	aggregated, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if aggregated.Name != "Example Project" {
		t.Errorf("Name = %q, want Example Project", aggregated.Name)
	}
	if aggregated.Root != root {
		t.Errorf("Root = %q, want %q", aggregated.Root, root)
	}
	if len(aggregated.Services) != 2 {
		t.Fatalf("Services = %d, want 2", len(aggregated.Services))
	}

	a, b := aggregated.Services[0], aggregated.Services[1]
	if a.ID != "api-a" || a.Name != "API A" || a.Description != "Public-facing API" {
		t.Errorf("first service = %+v", a)
	}
	if a.Dir != filepath.Join(root, "api-a") {
		t.Errorf("first service Dir = %q", a.Dir)
	}
	if a.Result == nil || a.Err != nil {
		t.Fatalf("first service = %+v, want parsed result without error", a)
	}
	if b.Result == nil || b.Err != nil {
		t.Fatalf("second service = %+v, want parsed result without error", b)
	}

	if got := a.Result.Features[0].ID; got != "api-a/F001" {
		t.Errorf("first feature ID = %q, want api-a/F001", got)
	}
	if got := b.Result.Features[0].ID; got != "api-b/F001" {
		t.Errorf("second feature ID = %q, want api-b/F001", got)
	}
	if a.Result.Features[0].ID == b.Result.Features[0].ID {
		t.Errorf("feature ids collided: %q", a.Result.Features[0].ID)
	}
	if got := a.Result.Tasks[0].ID; got != "api-a/T001" {
		t.Errorf("first task ID = %q, want api-a/T001", got)
	}
}

func TestLoadIsolatesBrokenService(t *testing.T) {
	root := t.TempDir()
	writeValidService(t, root, "api-a")

	brokenDir := filepath.Join(root, "api-b")
	writeFile(t, filepath.Join(brokenDir, ".handle", "plan", "plan.md"), validPlan)
	writeFile(t, filepath.Join(brokenDir, ".handle", "plan", "feature", "feature-001.md"), "# Not an ID\n")

	writeValidService(t, root, "api-c")
	writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
  - id: api-b
    name: API B
    path: ./api-b
  - id: api-c
    name: API C
    path: ./api-c
`)

	aggregated, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned fatal error: %v", err)
	}
	if len(aggregated.Services) != 3 {
		t.Fatalf("Services = %d, want 3", len(aggregated.Services))
	}

	broken := aggregated.Services[1]
	if broken.Result != nil {
		t.Errorf("broken service Result = %+v, want nil", broken.Result)
	}
	if broken.Err == nil {
		t.Fatal("broken service Err = nil, want parse error")
	}
	if !strings.Contains(broken.Err.Error(), "feature-001.md") {
		t.Errorf("broken service error = %q, want offending path", broken.Err)
	}

	for _, i := range []int{0, 2} {
		service := aggregated.Services[i]
		if service.Result == nil || service.Err != nil {
			t.Errorf("service %q = %+v, want parsed result without error", service.ID, service)
		}
	}
}

func TestLoadReportsMissingHandleWithoutFailing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api-a"), 0o755); err != nil {
		t.Fatalf("create api-a: %v", err)
	}
	writeValidService(t, root, "api-b")
	writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
  - id: api-b
    name: API B
    path: ./api-b
`)

	aggregated, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned fatal error: %v", err)
	}
	if len(aggregated.Services) != 2 {
		t.Fatalf("Services = %d, want 2", len(aggregated.Services))
	}

	missing := aggregated.Services[0]
	if missing.Result != nil {
		t.Errorf("missing-handle service Result = %+v, want nil", missing.Result)
	}
	if missing.Err == nil {
		t.Fatal("missing-handle service Err = nil, want reported error")
	}
	if !strings.Contains(missing.Err.Error(), "missing .handle/plan") {
		t.Errorf("missing-handle service error = %q, want missing .handle/plan", missing.Err)
	}

	healthy := aggregated.Services[1]
	if healthy.Result == nil || healthy.Err != nil {
		t.Errorf("healthy service = %+v, want parsed result without error", healthy)
	}
}

func TestLoadMalformedManifestIsFatal(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "project: Example\nservices:\n  - id: [unterminated\n")

	aggregated, err := Load(root)
	if err == nil {
		t.Fatal("Load returned nil error for malformed manifest")
	}
	if aggregated != nil {
		t.Errorf("Project = %+v, want nil", aggregated)
	}
	if !strings.Contains(err.Error(), "parse manifest") {
		t.Errorf("error = %q, want parse manifest", err)
	}
}

func writeValidService(t *testing.T, root, id string) {
	t.Helper()

	planDir := filepath.Join(root, id, ".handle", "plan")
	writeFile(t, filepath.Join(planDir, "plan.md"), validPlan)
	writeFile(t, filepath.Join(planDir, "feature", "feature-001.md"), validFeature)
	writeFile(t, filepath.Join(planDir, "task", "task-001.md"), validTask)
}

func writeManifest(t *testing.T, root, body string) {
	t.Helper()
	writeFile(t, filepath.Join(root, ".project", "services.yaml"), body)
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
