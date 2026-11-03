package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gmmorgan96/handle/internal/api"
)

const planDoc = `## Example Project

An example project.

### Phase One - Setup ([P001](x))

**Purpose**: Set up the project.

**Features**
- [ ] ([F001](feature/feature-001.md)) First feature

**Dependencies**
- None
`

const featureWithBugDoc = `# F001 - First feature

## Description
The first feature.

## Requirements
- Do the thing

## Tasks
- [x] [T001](../task/task-001.md) First task

## Bugs
- [ ] [B001](../bug/bug-001.md) First bug

## Acceptance Criteria
- [x] Done
`

const featureWithoutBugDoc = `# F001 - First feature

## Description
The first feature.

## Requirements
- Do the thing

## Tasks
- [x] [T001](../task/task-001.md) First task

## Acceptance Criteria
- [x] Done
`

const completeTaskDoc = `# T001 - First task

## Related Feature
- [F001](../feature/feature-001.md)

## Acceptance Criteria
- [x] Done
`

const partialTaskDoc = `# T001 - First task

## Related Feature
- [F001](../feature/feature-001.md)

## Acceptance Criteria
- [ ] Not done
`

const bugDoc = `# B001 - First bug

## Description
Something is broken.

## Related Feature
- [F001](../feature/feature-001.md)

## Steps to Reproduce
1. Do the thing

## Expected/Actual
- **Expected**: It works
- **Actual**: It fails

## Potentially Introduced
- Commit: (unknown)
- Message: (unknown)

## Acceptance Criteria
- [ ] Fixed
`

type fixtureService struct {
	id          string
	name        string
	description string
	taskDoc     string
	withBug     bool
	broken      bool
}

func exampleServices() []fixtureService {
	return []fixtureService{
		{id: "alpha", name: "Alpha API", description: "Public-facing API", taskDoc: completeTaskDoc, withBug: true},
		{id: "beta", name: "Beta Worker", description: "Background worker", taskDoc: partialTaskDoc},
		{id: "broken", name: "Broken Service", broken: true},
	}
}

func TestOverviewShowsEveryService(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())

	response := get(t, NewHandler(root), "/")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body := response.Body.String()
	for _, want := range []string{"Alpha API", "Beta Worker", "Broken Service", "Showing 3 of 3 services"} {
		if !strings.Contains(body, want) {
			t.Errorf("overview does not contain %q", want)
		}
	}
	if got := strings.Count(body, "data-service-id="); got != 3 {
		t.Errorf("service cards = %d, want 3", got)
	}

	alpha := findServiceCard(t, body, "alpha")
	for _, want := range []string{"Public-facing API", `class="progress"`, "badge-complete"} {
		if !strings.Contains(alpha, want) {
			t.Errorf("alpha service card does not contain %q", want)
		}
	}
}

func TestOverviewProgressAndCountsMatchAPI(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())

	apiResponse := get(t, api.NewHandler(root), "/api/project")
	if apiResponse.Code != http.StatusOK {
		t.Fatalf("API status = %d, want %d", apiResponse.Code, http.StatusOK)
	}
	var payload api.ProjectResponse
	if err := json.NewDecoder(apiResponse.Body).Decode(&payload); err != nil {
		t.Fatalf("decode API response: %v", err)
	}

	body := get(t, NewHandler(root), "/").Body.String()
	for _, service := range payload.Services {
		card := findServiceCard(t, body, service.ID)
		want := map[string]string{
			"data-status":           expectedStatus(service),
			"data-progress-done":    strconv.Itoa(service.Progress.Done),
			"data-progress-total":   strconv.Itoa(service.Progress.Total),
			"data-progress-percent": fmt.Sprintf("%.0f", service.Progress.Percent),
			"data-bug-count":        strconv.Itoa(service.Counts.Bugs),
		}
		for attributeName, wantValue := range want {
			if got := attribute(t, card, attributeName); got != wantValue {
				t.Errorf("service %q %s = %q, want %q", service.ID, attributeName, got, wantValue)
			}
		}
	}

	wantSummary := fmt.Sprintf("%.0f%% · %d/%d tasks complete", payload.Progress.Percent, payload.Progress.Done, payload.Progress.Total)
	if !strings.Contains(body, wantSummary) {
		t.Errorf("overall progress does not contain %q", wantSummary)
	}
}

func TestOverviewShowsErrorBadgeForBrokenService(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())

	body := get(t, NewHandler(root), "/").Body.String()

	broken := findServiceCard(t, body, "broken")
	if got := attribute(t, broken, "data-status"); got != "error" {
		t.Errorf("broken service status = %q, want error", got)
	}
	for _, want := range []string{"badge-error", "Failed to load:", "missing .handle/plan"} {
		if !strings.Contains(broken, want) {
			t.Errorf("broken service card does not contain %q", want)
		}
	}

	healthy := findServiceCard(t, body, "alpha")
	if got := attribute(t, healthy, "data-status"); got == "error" {
		t.Errorf("healthy service status = %q, want not error", got)
	}
}

func TestOverviewEmptyProjectShowsGuidance(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".project", "services.yaml"), "project: Empty Project\nservices: []\n")

	response := get(t, NewHandler(root), "/")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, want := range []string{"Empty Project", "No services yet", ".project/services.yaml"} {
		if !strings.Contains(body, want) {
			t.Errorf("empty overview does not contain %q", want)
		}
	}
	if strings.Contains(body, "data-service-id=") {
		t.Error("empty overview unexpectedly rendered service cards")
	}
}

func TestOverviewFiltersAndSorts(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())
	handler := NewHandler(root)

	t.Run("filter by name", func(t *testing.T) {
		body := get(t, handler, "/?q=beta").Body.String()
		if !strings.Contains(body, "Beta Worker") {
			t.Error("name filter did not include Beta Worker")
		}
		if strings.Contains(body, "Alpha API") || strings.Contains(body, "Broken Service") {
			t.Error("name filter included a non-matching service")
		}
		if !strings.Contains(body, "Showing 1 of 3 services") {
			t.Error("name filter did not report the visible count")
		}
		if !strings.Contains(body, `value="beta"`) {
			t.Error("name filter did not preserve the query value")
		}
	})

	t.Run("filter by active status", func(t *testing.T) {
		body := get(t, handler, "/?status=active").Body.String()
		if !strings.Contains(body, "Beta Worker") {
			t.Error("status filter did not include the active service")
		}
		if strings.Contains(body, "Alpha API") || strings.Contains(body, "Broken Service") {
			t.Error("status filter included a service with another status")
		}
	})

	t.Run("filter by error status", func(t *testing.T) {
		body := get(t, handler, "/?status=error").Body.String()
		if !strings.Contains(body, "Broken Service") {
			t.Error("error filter did not include the broken service")
		}
		if strings.Contains(body, "Alpha API") || strings.Contains(body, "Beta Worker") {
			t.Error("error filter included a healthy service")
		}
	})

	t.Run("sort by name", func(t *testing.T) {
		body := get(t, handler, "/?sort=name").Body.String()
		assertOrder(t, body, "Alpha API", "Beta Worker", "Broken Service")
	})

	t.Run("sort by status", func(t *testing.T) {
		body := get(t, handler, "/?sort=status").Body.String()
		assertOrder(t, body, "Broken Service", "Beta Worker", "Alpha API")
	})

	t.Run("no matches", func(t *testing.T) {
		body := get(t, handler, "/?q=missing").Body.String()
		for _, want := range []string{"No matching services", "Clear filters"} {
			if !strings.Contains(body, want) {
				t.Errorf("no-match overview does not contain %q", want)
			}
		}
		if strings.Contains(body, "data-service-id=") {
			t.Error("no-match overview unexpectedly rendered service cards")
		}
	})

	t.Run("invalid filters fall back", func(t *testing.T) {
		body := get(t, handler, "/?status=bogus&sort=bogus").Body.String()
		if !strings.Contains(body, "Showing 3 of 3 services") {
			t.Error("invalid filters did not fall back to all services")
		}
		assertOrder(t, body, "Alpha API", "Beta Worker", "Broken Service")
	})
}

func TestOverviewMethodNotAllowed(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	response := httptest.NewRecorder()

	NewHandler(t.TempDir()).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want %q", got, http.MethodGet)
	}
}

func TestOverviewUnknownPathReturnsNotFound(t *testing.T) {
	response := get(t, NewHandler(t.TempDir()), "/services/alpha")

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestStaticStylesheetServed(t *testing.T) {
	response := get(t, NewHandler(t.TempDir()), "/static/style.css")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "text/css") {
		t.Errorf("Content-Type = %q, want text/css", contentType)
	}
	if !strings.Contains(response.Body.String(), ".service-card") {
		t.Error("stylesheet does not contain the service card styles")
	}
}

func TestOverviewFatalManifestError(t *testing.T) {
	response := get(t, NewHandler(t.TempDir()), "/")

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	body := response.Body.String()
	for _, want := range []string{"Could not load project", "services.yaml"} {
		if !strings.Contains(body, want) {
			t.Errorf("error page does not contain %q", want)
		}
	}
}

func expectedStatus(service api.ServiceSummary) string {
	switch {
	case service.Error != "":
		return statusError
	case service.Progress.Total == 0:
		return statusEmpty
	case service.Progress.Done == service.Progress.Total:
		return statusComplete
	default:
		return statusActive
	}
}

func findServiceCard(t *testing.T, body, id string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)<article[^>]*\bdata-service-id="` + regexp.QuoteMeta(id) + `"[^>]*>.*?</article>`)
	card := pattern.FindString(body)
	if card == "" {
		t.Fatalf("service card %q not found", id)
	}
	return card
}

func attribute(t *testing.T, tag, name string) string {
	t.Helper()
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `="([^"]*)"`)
	match := pattern.FindStringSubmatch(tag)
	if match == nil {
		t.Fatalf("attribute %q not found in %q", name, tag)
	}
	return match[1]
}

func assertOrder(t *testing.T, body string, names ...string) {
	t.Helper()
	last := -1
	for _, name := range names {
		index := strings.Index(body, name)
		if index < 0 {
			t.Fatalf("%q not found in overview", name)
		}
		if index <= last {
			t.Fatalf("%q appears out of order", name)
		}
		last = index
	}
}

func get(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, target, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func writeProject(t *testing.T, root string, services []fixtureService) {
	t.Helper()
	var manifest strings.Builder
	manifest.WriteString("project: Example Project\nservices:\n")
	for _, service := range services {
		fmt.Fprintf(&manifest, "  - id: %s\n    name: %s\n", service.id, service.name)
		if service.description != "" {
			fmt.Fprintf(&manifest, "    description: %s\n", service.description)
		}
		fmt.Fprintf(&manifest, "    path: ./%s\n", service.id)
		if service.broken {
			if err := os.MkdirAll(filepath.Join(root, service.id), 0o755); err != nil {
				t.Fatalf("create broken service directory: %v", err)
			}
			continue
		}
		writeService(t, root, service)
	}
	writeFile(t, filepath.Join(root, ".project", "services.yaml"), manifest.String())
}

func writeService(t *testing.T, root string, service fixtureService) {
	t.Helper()
	planDir := filepath.Join(root, service.id, ".handle", "plan")
	writeFile(t, filepath.Join(planDir, "plan.md"), planDoc)
	feature := featureWithoutBugDoc
	if service.withBug {
		feature = featureWithBugDoc
		writeFile(t, filepath.Join(planDir, "bug", "bug-001.md"), bugDoc)
	}
	writeFile(t, filepath.Join(planDir, "feature", "feature-001.md"), feature)
	writeFile(t, filepath.Join(planDir, "task", "task-001.md"), service.taskDoc)
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
