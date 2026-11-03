package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestProjectHandlerReturnsDocumentedShape(t *testing.T) {
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

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ProjectResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Verify documented shape
	if resp.Name != "Example Project" {
		t.Errorf("Name = %q, want Example Project", resp.Name)
	}
	if resp.Root != root {
		t.Errorf("Root = %q, want %q", resp.Root, root)
	}
	if len(resp.Services) != 2 {
		t.Fatalf("Services = %d, want 2", len(resp.Services))
	}

	// Verify first service shape
	a := resp.Services[0]
	if a.ID != "api-a" || a.Name != "API A" || a.Description != "Public-facing API" {
		t.Errorf("first service = %+v", a)
	}
	if a.Path != "./api-a" {
		t.Errorf("first service Path = %q, want ./api-a", a.Path)
	}
	if len(a.DependsOn) != 0 {
		t.Errorf("first service DependsOn = %v, want empty", a.DependsOn)
	}
	if a.Error != "" {
		t.Errorf("first service Error = %q, want empty", a.Error)
	}
	if a.Counts.Features != 1 || a.Counts.Tasks != 1 || a.Counts.Bugs != 0 {
		t.Errorf("first service Counts = %+v, want {Features:1 Tasks:1 Bugs:0}", a.Counts)
	}
	if a.Progress.Total == 0 {
		t.Errorf("first service Progress.Total = 0, want > 0")
	}
	if a.Progress.Percent < 0 || a.Progress.Percent > 100 {
		t.Errorf("first service Progress.Percent = %v, want 0-100", a.Progress.Percent)
	}

	// Verify second service shape
	b := resp.Services[1]
	if b.ID != "api-b" || b.Name != "API B" {
		t.Errorf("second service = %+v", b)
	}
	if b.Error != "" {
		t.Errorf("second service Error = %q, want empty", b.Error)
	}

	// Verify project-level progress
	if resp.Progress.Total == 0 {
		t.Errorf("project Progress.Total = 0, want > 0")
	}
	if resp.Progress.Percent < 0 || resp.Progress.Percent > 100 {
		t.Errorf("project Progress.Percent = %v, want 0-100", resp.Progress.Percent)
	}
}

func TestProjectHandlerBrokenServiceRepresentedInPayload(t *testing.T) {
	root := t.TempDir()
	writeValidService(t, root, "api-a")

	// Broken service: malformed feature file
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

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ProjectResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Services) != 3 {
		t.Fatalf("Services = %d, want 3", len(resp.Services))
	}

	// Broken service should have error and nil progress/counts
	broken := resp.Services[1]
	if broken.ID != "api-b" {
		t.Errorf("broken service ID = %q, want api-b", broken.ID)
	}
	if broken.Error == "" {
		t.Fatal("broken service Error = empty, want parse error")
	}
	if !strings.Contains(broken.Error, "feature-001.md") {
		t.Errorf("broken service error = %q, want offending path", broken.Error)
	}
	if broken.Progress.Total != 0 || broken.Counts.Features != 0 {
		t.Errorf("broken service Progress/Counts = %+v / %+v, want zeros", broken.Progress, broken.Counts)
	}

	// Other services should still load successfully
	for _, i := range []int{0, 2} {
		svc := resp.Services[i]
		if svc.Error != "" {
			t.Errorf("service %q Error = %q, want empty", svc.ID, svc.Error)
		}
		if svc.Progress.Total == 0 {
			t.Errorf("service %q Progress.Total = 0, want > 0", svc.ID)
		}
	}
}

func TestProjectHandlerMissingHandleDir(t *testing.T) {
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

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp ProjectResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Services) != 2 {
		t.Fatalf("Services = %d, want 2", len(resp.Services))
	}

	missing := resp.Services[0]
	if missing.ID != "api-a" {
		t.Errorf("missing-handle service ID = %q, want api-a", missing.ID)
	}
	if missing.Error == "" {
		t.Fatal("missing-handle service Error = empty, want reported error")
	}
	if !strings.Contains(missing.Error, "missing .handle/plan") {
		t.Errorf("missing-handle service error = %q, want missing .handle/plan", missing.Error)
	}
	if missing.Progress.Total != 0 {
		t.Errorf("missing-handle service Progress.Total = %d, want 0", missing.Progress.Total)
	}

	healthy := resp.Services[1]
	if healthy.Error != "" {
		t.Errorf("healthy service Error = %q, want empty", healthy.Error)
	}
	if healthy.Progress.Total == 0 {
		t.Errorf("healthy service Progress.Total = 0, want > 0 for %q", healthy.ID)
	}
}

func TestProjectHandlerMalformedManifestReturns500(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "project: Example\nservices:\n  - id: [unterminated\n")

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestProjectHandlerMethodNotAllowed(t *testing.T) {
	root := t.TempDir()
	writeValidService(t, root, "api-a")
	writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
`)

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodPost, "/api/project", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
	if got := w.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want %q", got, http.MethodGet)
	}
}

func TestProjectHandlerNotFound(t *testing.T) {
	root := t.TempDir()
	writeValidService(t, root, "api-a")
	writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
`)

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestProgressStatsCompleteFlag(t *testing.T) {
	// Test that Complete flag is correctly set based on Done == Total > 0
	tests := []struct {
		name     string
		stats    ProgressStats
		wantComp bool
	}{
		{"empty", ProgressStats{}, false},
		{"zero total", ProgressStats{Done: 0, Total: 0}, false},
		{"partial", ProgressStats{Done: 1, Total: 3}, false},
		{"complete", ProgressStats{Done: 3, Total: 3}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.stats.Complete() != tt.wantComp {
				t.Errorf("Complete() = %t, want %t", tt.stats.Complete(), tt.wantComp)
			}
		})
	}
}

func TestServiceSummaryDependsOn(t *testing.T) {
	root := t.TempDir()
	writeValidService(t, root, "api-a")
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
    depends_on: [api-a]
`)

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var resp ProjectResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if len(resp.Services) != 2 {
		t.Fatalf("Services = %d, want 2", len(resp.Services))
	}

	a := resp.Services[0]
	if len(a.DependsOn) != 0 {
		t.Errorf("api-a DependsOn = %v, want empty", a.DependsOn)
	}

	b := resp.Services[1]
	if len(b.DependsOn) != 1 || b.DependsOn[0] != "api-a" {
		t.Errorf("api-b DependsOn = %v, want [api-a]", b.DependsOn)
	}
}

func TestResponseJSONSerialization(t *testing.T) {
	root := t.TempDir()
	writeValidService(t, root, "api-a")
	writeManifest(t, root, `
project: Example Project
services:
  - id: api-a
    name: API A
    path: ./api-a
`)

	handler := NewHandler(root)
	req := httptest.NewRequest(http.MethodGet, "/api/project", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify Content-Type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	// Verify valid JSON
	var resp ProjectResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Re-encode to verify no cycles or marshal issues
	reencoded, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("re-encode failed: %v", err)
	}
	if len(reencoded) == 0 {
		t.Error("re-encoded JSON is empty")
	}
}

// Test helpers
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
