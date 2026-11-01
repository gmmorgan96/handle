package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, root, body string) string {
	t.Helper()

	dir := filepath.Join(root, ".project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("create .project: %v", err)
	}
	path := filepath.Join(dir, "services.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return path
}

func makeService(t *testing.T, root, id string) string {
	t.Helper()

	dir := filepath.Join(root, id)
	if err := os.MkdirAll(filepath.Join(dir, ".handle", "plan"), 0o755); err != nil {
		t.Fatalf("create service %s: %v", id, err)
	}
	return dir
}

func requireErrorContaining(t *testing.T, err error, substrings ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	for _, substring := range substrings {
		if !strings.Contains(err.Error(), substring) {
			t.Errorf("error %q does not contain %q", err.Error(), substring)
		}
	}
}

func TestLoadValidManifest(t *testing.T) {
	root := t.TempDir()
	apiA := makeService(t, root, "api-a")
	apiB := makeService(t, root, "api-b")
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
    description: Public-facing API
    depends_on: [api-b]
  - id: api-b
    name: API B
    path: ./api-b
    depends_on: []
`)

	project, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if project.Name != "Example Project" {
		t.Errorf("project name = %q, want %q", project.Name, "Example Project")
	}
	if project.Root != filepath.Clean(root) {
		t.Errorf("project root = %q, want %q", project.Root, filepath.Clean(root))
	}
	if len(project.Services) != 2 {
		t.Fatalf("services = %d, want 2", len(project.Services))
	}

	a := project.Services[0]
	if a.ID != "api-a" {
		t.Errorf("service id = %q, want %q", a.ID, "api-a")
	}
	if a.Name != "API A" {
		t.Errorf("service name = %q, want %q", a.Name, "API A")
	}
	if a.Description != "Public-facing API" {
		t.Errorf("service description = %q, want %q", a.Description, "Public-facing API")
	}
	if a.Dir != apiA {
		t.Errorf("service dir = %q, want %q", a.Dir, apiA)
	}
	if len(a.DependsOn) != 1 || string(a.DependsOn[0]) != "api-b" {
		t.Errorf("service dependencies = %v, want [api-b]", a.DependsOn)
	}
	if project.Services[1].Dir != apiB {
		t.Errorf("service dir = %q, want %q", project.Services[1].Dir, apiB)
	}
}

func TestLoadProjectResolvesPathsAgainstProjectRoot(t *testing.T) {
	root := t.TempDir()
	makeService(t, root, "api-a")
	writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
`)

	project, err := LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject() error = %v", err)
	}

	want := filepath.Join(root, "api-a")
	if got := project.Services[0].Dir; got != want {
		t.Errorf("service dir = %q, want %q (resolved against project root, not .project/)", got, want)
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, "project: Example\nservices:\n  - id: [unterminated\n")

	project, err := Load(path)
	requireErrorContaining(t, err, "parse manifest")
	if project != nil {
		t.Errorf("project = %+v, want nil", project)
	}
}

func TestLoadDuplicateID(t *testing.T) {
	root := t.TempDir()
	makeService(t, root, "api-a")
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
  - id: api-a
    name: API A Again
    path: ./api-a
`)

	_, err := Load(path)
	requireErrorContaining(t, err, `service "api-a"`, "duplicate id")

	var validation ValidationErrors
	if !errors.As(err, &validation) {
		t.Errorf("error type = %T, want ValidationErrors", err)
	}
}

func TestLoadSelfDependency(t *testing.T) {
	root := t.TempDir()
	makeService(t, root, "api-a")
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
    depends_on: [api-a]
`)

	_, err := Load(path)
	requireErrorContaining(t, err, `service "api-a"`, "must not depend on itself")
}

func TestLoadUnknownDependency(t *testing.T) {
	root := t.TempDir()
	makeService(t, root, "api-a")
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
    depends_on: [api-b]
`)

	_, err := Load(path)
	requireErrorContaining(t, err, `service "api-a"`, `depends on unknown service "api-b"`)
}

func TestLoadDependencyCycle(t *testing.T) {
	root := t.TempDir()
	makeService(t, root, "api-a")
	makeService(t, root, "api-b")
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
    depends_on: [api-b]
  - id: api-b
    name: API B
    path: ./api-b
    depends_on: [api-a]
`)

	_, err := Load(path)
	requireErrorContaining(t, err, "dependency cycle", "api-a -> api-b -> api-a")
}

func TestLoadMissingPath(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./missing
`)

	project, err := Load(path)
	requireErrorContaining(t, err, `service "api-a"`, `path "./missing" does not exist`)
	if project == nil {
		t.Fatal("project = nil, want decoded project")
	}
	if project.Services[0].Dir != "" {
		t.Errorf("service dir = %q, want empty", project.Services[0].Dir)
	}
}

func TestLoadPathMissingPlanDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api-a"), 0o755); err != nil {
		t.Fatalf("create service directory: %v", err)
	}
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
`)

	_, err := Load(path)
	requireErrorContaining(t, err, `service "api-a"`, "missing .handle/plan")
}

func TestLoadAbsolutePath(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: /tmp/api-a
`)

	_, err := Load(path)
	requireErrorContaining(t, err, `service "api-a"`, "must be relative to the project root")
}

func TestLoadInvalidID(t *testing.T) {
	root := t.TempDir()
	makeService(t, root, "api-a")
	path := writeManifest(t, root, `
project: Example Project
services:
  - id: API_A
    name: API A
    path: ./api-a
`)

	_, err := Load(path)
	requireErrorContaining(t, err, `service "API_A"`, "id must match")
}

func TestLoadEmptyServices(t *testing.T) {
	root := t.TempDir()
	path := writeManifest(t, root, "project: Example Project\n")

	_, err := Load(path)
	requireErrorContaining(t, err, "at least one service")
}
