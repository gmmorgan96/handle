package ui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gmmorgan96/handle/internal/plan"
)

const secondTaskDoc = `# T002 - Second task

## Related Feature
- [F001](../feature/feature-001.md)

## Acceptance Criteria
- [ ] Half done
- [ ] Other half
`

const secondBugDoc = `# B002 - Second bug

## Description
Another bug.

## Related Feature
- [F001](../feature/feature-001.md)

## Steps to Reproduce
1. Repeat the thing
2. Observe the result

## Expected/Actual
- **Expected**: Stable
- **Actual**: Unstable

## Potentially Introduced
- Commit: deadbee
- Message: "Introduce regression"

## Acceptance Criteria
- [x] Fixed
`

// drilldownProject extends the shared overview fixtures with a second task and
// bug so the drill-down pages can show more than one nested item.
func drilldownProject(t *testing.T) (root string, handler http.Handler) {
	t.Helper()
	root = t.TempDir()
	writeProject(t, root, exampleServices())

	planDir := filepath.Join(root, "alpha", ".handle", "plan")
	writeFile(t, filepath.Join(planDir, "task", "task-002.md"), secondTaskDoc)
	writeFile(t, filepath.Join(planDir, "bug", "bug-002.md"), secondBugDoc)

	return root, NewHandler(root)
}

func TestOverviewLinksToServicePages(t *testing.T) {
	_, handler := drilldownProject(t)

	body := get(t, handler, "/").Body.String()

	for _, id := range []string{"alpha", "beta", "broken"} {
		want := fmt.Sprintf(`href="/services/%s"`, id)
		if !strings.Contains(body, want) {
			t.Errorf("overview does not link service %q (%q)", id, want)
		}
	}
}

func TestServicePageListsFeaturesWithNestedItems(t *testing.T) {
	_, handler := drilldownProject(t)

	response := get(t, handler, "/services/alpha")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	feature := featureCardHTML(t, body, "F001")
	for _, want := range []string{
		"First feature",
		`href="/services/alpha/features/F001"`,
		`href="/services/alpha/tasks/T001"`,
		`href="/services/alpha/tasks/T002"`,
		`href="/services/alpha/bugs/B001"`,
		`href="/services/alpha/bugs/B002"`,
	} {
		if !strings.Contains(feature, want) {
			t.Errorf("feature card does not contain %q", want)
		}
	}
}

func TestServicePageListsBugsSeparatelyFromTasks(t *testing.T) {
	_, handler := drilldownProject(t)
	body := get(t, handler, "/services/alpha").Body.String()
	feature := featureCardHTML(t, body, "F001")

	tasksSection := sectionBetween(t, feature, "<h4>Tasks</h4>", "<h4>Bugs</h4>")
	if !strings.Contains(tasksSection, "/services/alpha/tasks/T001") {
		t.Error("Tasks section does not contain the task link")
	}
	if strings.Contains(tasksSection, "/services/alpha/bugs/") {
		t.Error("Tasks section unexpectedly contains bug links")
	}

	bugsSection := feature[len("<h4>Bugs</h4>")+indexOf(t, feature, "<h4>Bugs</h4>"):]
	if !strings.Contains(bugsSection, "/services/alpha/bugs/B001") {
		t.Error("Bugs section does not contain the bug link")
	}
	if strings.Contains(bugsSection, "/services/alpha/tasks/") {
		t.Error("Bugs section unexpectedly contains task links")
	}
}

func TestFeaturePageShowsNestedTasksAndBugs(t *testing.T) {
	_, handler := drilldownProject(t)

	response := get(t, handler, "/services/alpha/features/F001")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, want := range []string{
		"First feature",
		`href="/services/alpha/tasks/T001"`,
		`href="/services/alpha/tasks/T002"`,
		`href="/services/alpha/bugs/B001"`,
		`href="/services/alpha/bugs/B002"`,
		`data-item-id="T002"`,
		`data-item-id="B002"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("feature page does not contain %q", want)
		}
	}
}

func TestTaskPageShowsStatusFromAcceptanceCriteria(t *testing.T) {
	_, handler := drilldownProject(t)

	response := get(t, handler, "/services/alpha/tasks/T002")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, want := range []string{
		"Second task",
		"Not started",
		"badge-not-started",
		"Half done",
		`href="/services/alpha/features/F001"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("task page does not contain %q", want)
		}
	}

	complete := get(t, handler, "/services/alpha/tasks/T001").Body.String()
	if !strings.Contains(complete, "Complete") || !strings.Contains(complete, "badge-complete") {
		t.Error("complete task does not render a complete badge")
	}
}

func TestBugDetailShowsReproductionDetails(t *testing.T) {
	_, handler := drilldownProject(t)

	response := get(t, handler, "/services/alpha/bugs/B002")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	body := response.Body.String()
	for _, want := range []string{
		"Second bug",
		"Steps to Reproduce",
		"Repeat the thing",
		"Observe the result",
		"Expected/Actual",
		"Stable",
		"Unstable",
		"Potentially Introduced",
		"deadbee",
		"Introduce regression",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("bug page does not contain %q", want)
		}
	}

	unknown := get(t, handler, "/services/alpha/bugs/B001").Body.String()
	for _, want := range []string{
		"Steps to Reproduce",
		"Do the thing",
		"Expected/Actual",
		"It works",
		"It fails",
		"Potentially Introduced",
		"No commit recorded.",
	} {
		if !strings.Contains(unknown, want) {
			t.Errorf("bug page without an introduced commit does not contain %q", want)
		}
	}
}

func TestDrilldownBreadcrumbsNavigateBack(t *testing.T) {
	_, handler := drilldownProject(t)

	body := get(t, handler, "/services/alpha/bugs/B002").Body.String()

	crumbs := breadcrumbLinks(t, body)
	want := []struct {
		label string
		href  string
	}{
		{"Project overview", "/"},
		{"Alpha API", "/services/alpha"},
		{"F001", "/services/alpha/features/F001"},
	}
	for _, crumb := range want {
		if crumbs[crumb.label] != crumb.href {
			t.Errorf("breadcrumb %q = %q, want %q", crumb.label, crumbs[crumb.label], crumb.href)
		}
	}
	if _, ok := crumbs["B002"]; ok {
		t.Error("the current item should not be a breadcrumb link")
	}
	if !strings.Contains(body, `<span aria-current="page">B002</span>`) {
		t.Error("the current item should be marked as the current breadcrumb")
	}

	// Walking every breadcrumb link must reach the page it names.
	for label, href := range crumbs {
		response := get(t, handler, href)
		if response.Code != http.StatusOK {
			t.Errorf("breadcrumb %q (%s) status = %d, want %d", label, href, response.Code, http.StatusOK)
		}
	}
}

func TestDeepLinkSurvivesReload(t *testing.T) {
	root, _ := drilldownProject(t)

	first := get(t, NewHandler(root), "/services/alpha/bugs/B002")
	second := get(t, NewHandler(root), "/services/alpha/bugs/B002")

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d, %d, want 200, 200", first.Code, second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Error("bug deep link rendered differently after reload")
	}
	if !strings.Contains(second.Body.String(), "Second bug") {
		t.Error("reloaded deep link lost its content")
	}
}

func TestServicePageEmptyService(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".project", "services.yaml"), "project: Empty Service Project\nservices:\n  - id: empty\n    name: Empty Service\n    path: ./empty\n")
	if err := os.MkdirAll(filepath.Join(root, "empty", ".handle", "plan"), 0o755); err != nil {
		t.Fatalf("create empty plan directory: %v", err)
	}

	response := get(t, NewHandler(root), "/services/empty")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "No features, tasks, or bugs yet") {
		t.Error("empty service page does not show guidance")
	}
}

func TestServicePageBrokenService(t *testing.T) {
	_, handler := drilldownProject(t)

	response := get(t, handler, "/services/broken")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	body := response.Body.String()
	for _, want := range []string{"Broken Service", "Failed to load:", "badge-error"} {
		if !strings.Contains(body, want) {
			t.Errorf("broken service page does not contain %q", want)
		}
	}
}

func TestServiceRoutesReturnNotFound(t *testing.T) {
	_, handler := drilldownProject(t)

	for _, path := range []string{
		"/services",
		"/services/missing",
		"/services/alpha/features/F999",
		"/services/alpha/tasks/T999",
		"/services/alpha/bugs/B999",
		"/services/alpha/unknown/T001",
		"/services/alpha/tasks/T001/extra",
	} {
		response := get(t, handler, path)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func TestServiceRoutesRejectOtherMethods(t *testing.T) {
	_, handler := drilldownProject(t)

	request := httptest.NewRequest(http.MethodPost, "/services/alpha", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want %q", got, http.MethodGet)
	}
}

func TestStatusFromChecks(t *testing.T) {
	tests := []struct {
		name  string
		check []plan.Check
		key   string
		label string
	}{
		{"no criteria", nil, statusEmpty, "No criteria"},
		{"complete", []plan.Check{{Done: true}, {Done: true}}, statusComplete, "Complete"},
		{"not started", []plan.Check{{Done: false}}, statusNotStarted, "Not started"},
		{"in progress", []plan.Check{{Done: true}, {Done: false}}, statusActive, "In progress"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := statusFromChecks(test.check)
			if status.Key != test.key || status.Label != test.label {
				t.Errorf("status = (%q, %q), want (%q, %q)", status.Key, status.Label, test.key, test.label)
			}
		})
	}
}

func featureCardHTML(t *testing.T, body, featureID string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)<article[^>]*\bdata-feature-id="` + regexp.QuoteMeta(featureID) + `"[^>]*>.*?</article>`)
	card := pattern.FindString(body)
	if card == "" {
		t.Fatalf("feature card %q not found", featureID)
	}
	return card
}

func sectionBetween(t *testing.T, body, start, end string) string {
	t.Helper()
	startIndex := indexOf(t, body, start)
	endIndex := indexOf(t, body, end)
	if endIndex < startIndex {
		t.Fatalf("%q appears before %q", end, start)
	}
	return body[startIndex:endIndex]
}

func indexOf(t *testing.T, body, needle string) int {
	t.Helper()
	index := strings.Index(body, needle)
	if index < 0 {
		t.Fatalf("%q not found", needle)
	}
	return index
}

func breadcrumbLinks(t *testing.T, body string) map[string]string {
	t.Helper()
	nav := regexp.MustCompile(`(?s)<nav class="breadcrumbs".*?</nav>`).FindString(body)
	if nav == "" {
		t.Fatal("breadcrumbs not found")
	}

	links := make(map[string]string)
	linkPattern := regexp.MustCompile(`<a href="([^"]+)">([^<]+)</a>`)
	for _, match := range linkPattern.FindAllStringSubmatch(nav, -1) {
		links[match[2]] = match[1]
	}
	return links
}
