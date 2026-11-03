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
)

// blockerFeatureDoc is the feature document used by the blocker fixtures. It
// links the first task so the drill-down pages have a feature to render.
const blockerFeatureDoc = `# F001 - First feature

## Description
The first feature.

## Tasks
- [ ] [T001](../task/task-001.md) First task

## Acceptance Criteria
- [x] Done
`

func blockerTaskDoc(id, title, dependencies string) string {
	return fmt.Sprintf(`# %s - %s

## Related Feature
- [F001](../feature/feature-001.md)

## Dependencies
%s

## Acceptance Criteria
- [ ] Done
`, id, title, dependencies)
}

// blockerProject builds a project with cross-service, local, and unresolved
// dependency edges spread across two healthy services, plus a broken service.
func blockerProject(t *testing.T) (string, http.Handler) {
	t.Helper()
	root := t.TempDir()

	writeBlockerService(t, root, "alpha", map[string]string{
		"task-001.md": blockerTaskDoc("T001", "Call beta endpoint", "- beta/T001"),
		"task-002.md": blockerTaskDoc("T002", "Needs missing work", "- beta/T999\n- alpha/T999"),
		"task-003.md": blockerTaskDoc("T003", "Follow up locally", "- alpha/T001"),
		"task-004.md": blockerTaskDoc("T004", "Independent work", "- None"),
		"task-005.md": blockerTaskDoc("T005", "Blocked by broken service", "- broken/T001"),
	})
	writeBlockerService(t, root, "beta", map[string]string{
		"task-001.md": blockerTaskDoc("T001", "Provide beta endpoint", "- None"),
		"task-002.md": blockerTaskDoc("T002", "Consume alpha task", "- alpha/T001"),
	})
	if err := os.MkdirAll(filepath.Join(root, "broken"), 0o755); err != nil {
		t.Fatalf("create broken service directory: %v", err)
	}

	writeFile(t, filepath.Join(root, ".project", "services.yaml"), `project: Example Project
services:
  - id: alpha
    name: Alpha API
    path: ./alpha
  - id: beta
    name: Beta Worker
    path: ./beta
  - id: broken
    name: Broken Service
    path: ./broken
`)

	return root, NewHandler(root)
}

func writeBlockerService(t *testing.T, root, id string, tasks map[string]string) {
	t.Helper()
	planDir := filepath.Join(root, id, ".handle", "plan")
	writeFile(t, filepath.Join(planDir, "plan.md"), planDoc)
	writeFile(t, filepath.Join(planDir, "feature", "feature-001.md"), blockerFeatureDoc)
	for name, content := range tasks {
		writeFile(t, filepath.Join(planDir, "task", name), content)
	}
}

func TestBlockersPageShowsBlockedItemsWithTheirBlockers(t *testing.T) {
	_, handler := blockerProject(t)

	response := get(t, handler, "/blockers")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body := response.Body.String()
	for _, want := range []string{"What's blocking me", "Alpha API", "Beta Worker", "Blocked by", "Blocks"} {
		if !strings.Contains(body, want) {
			t.Errorf("blockers page does not contain %q", want)
		}
	}

	card := blockerCardHTML(t, body, "alpha/T001")
	if !strings.Contains(card, "Call beta endpoint") {
		t.Error("alpha/T001 card does not show its title")
	}
	if got := attributeOf(t, card, "data-kind"); got != "task" {
		t.Errorf("alpha/T001 data-kind = %q, want task", got)
	}

	blockedBy := sectionBetween(t, card, "<h4>Blocked by</h4>", "<h4>Blocks</h4>")
	for _, want := range []string{"beta/T001", "Provide beta endpoint"} {
		if !strings.Contains(blockedBy, want) {
			t.Errorf("Blocked by section does not contain %q", want)
		}
	}

	blocks := card[indexOf(t, card, "<h4>Blocks</h4>"):]
	for _, want := range []string{"beta/T002", "Consume alpha task"} {
		if !strings.Contains(blocks, want) {
			t.Errorf("Blocks section does not contain %q", want)
		}
	}
}

func TestBlockersPageShowsCrossServiceBlockersOnBothItems(t *testing.T) {
	_, handler := blockerProject(t)
	body := get(t, handler, "/blockers").Body.String()

	alphaCard := blockerCardHTML(t, body, "alpha/T001")
	betaCard := blockerCardHTML(t, body, "beta/T001")
	for name, card := range map[string]string{
		"alpha/T001": alphaCard,
		"beta/T001":  betaCard,
	} {
		if !strings.Contains(card, `data-cross-service="true"`) {
			t.Errorf("%s card does not mark the item as cross-service", name)
		}
		if !strings.Contains(card, "cross-service") {
			t.Errorf("%s card does not render a cross-service badge", name)
		}
	}

	alphaBlockedBy := sectionBetween(t, alphaCard, "<h4>Blocked by</h4>", "<h4>Blocks</h4>")
	if !strings.Contains(alphaBlockedBy, "beta/T001") {
		t.Error("alpha/T001 Blocked by section does not contain the beta item")
	}
	if !strings.Contains(alphaBlockedBy, `data-cross-service="true"`) {
		t.Error("alpha/T001 blocker reference is not marked cross-service")
	}

	betaBlocks := betaCard[indexOf(t, betaCard, "<h4>Blocks</h4>"):]
	if !strings.Contains(betaBlocks, "alpha/T001") {
		t.Error("beta/T001 Blocks section does not contain the alpha item")
	}
	if !strings.Contains(betaBlocks, `data-cross-service="true"`) {
		t.Error("beta/T001 dependent reference is not marked cross-service")
	}

	betaConsumer := blockerCardHTML(t, body, "beta/T002")
	betaBlockedBy := sectionBetween(t, betaConsumer, "<h4>Blocked by</h4>", "<h4>Blocks</h4>")
	if !strings.Contains(betaBlockedBy, "alpha/T001") {
		t.Error("beta/T002 Blocked by section does not contain the alpha item")
	}
	if !strings.Contains(betaBlockedBy, `data-cross-service="true"`) {
		t.Error("beta/T002 blocker reference is not marked cross-service")
	}
}

func TestBlockersPageSurfacesUnresolvedReferences(t *testing.T) {
	_, handler := blockerProject(t)
	body := get(t, handler, "/blockers").Body.String()

	unresolved := regexp.MustCompile(`(?s)<section[^>]*data-unresolved-section.*?</section>`).FindString(body)
	if unresolved == "" {
		t.Fatal("unresolved references section not found")
	}
	if !strings.Contains(unresolved, "Unresolved references") {
		t.Error("unresolved section does not have a heading")
	}
	for _, target := range []string{"beta/T999", "alpha/T999", "broken/T001"} {
		want := fmt.Sprintf(`data-unresolved-target="%s"`, target)
		if !strings.Contains(unresolved, want) {
			t.Errorf("unresolved section does not list %q (%q)", target, want)
		}
	}

	missing := blockerCardHTML(t, body, "alpha/T002")
	blockedBy := sectionBetween(t, missing, "<h4>Blocked by</h4>", "<h4>Blocks</h4>")
	for _, want := range []string{
		"beta/T999",
		"alpha/T999",
		`data-resolved="false"`,
		"unresolved",
	} {
		if !strings.Contains(blockedBy, want) {
			t.Errorf("alpha/T002 Blocked by section does not contain %q", want)
		}
	}

	broken := blockerCardHTML(t, body, "alpha/T005")
	if !strings.Contains(broken, "broken/T001") {
		t.Error("alpha/T005 card does not surface the reference into the broken service")
	}
	if !strings.Contains(broken, `data-resolved="false"`) {
		t.Error("alpha/T005 reference into the broken service is not marked unresolved")
	}
}

func TestBlockersPageSummaryCounts(t *testing.T) {
	_, handler := blockerProject(t)
	body := get(t, handler, "/blockers").Body.String()

	want := []string{
		`data-blocked-items="5"`,
		`data-cross-service="4"`,
		`data-unresolved="3"`,
	}
	for _, attribute := range want {
		if !strings.Contains(body, attribute) {
			t.Errorf("blocker summary does not contain %q", attribute)
		}
	}
}

func TestBlockersPageGroupsItemsByServiceInManifestOrder(t *testing.T) {
	_, handler := blockerProject(t)
	body := get(t, handler, "/blockers").Body.String()

	assertOrder(t, body, "Alpha API", "Beta Worker")

	alpha := serviceBlockersHTML(t, body, "alpha")
	for _, id := range []string{"alpha/T001", "alpha/T002", "alpha/T003", "alpha/T005"} {
		want := fmt.Sprintf(`data-item-id="%s"`, id)
		if !strings.Contains(alpha, want) {
			t.Errorf("alpha group does not contain %q (%q)", id, want)
		}
	}
	if strings.Contains(alpha, `data-item-id="alpha/T004"`) {
		t.Error("alpha group includes an item with no dependencies or dependents")
	}

	beta := serviceBlockersHTML(t, body, "beta")
	for _, id := range []string{"beta/T001", "beta/T002"} {
		want := fmt.Sprintf(`data-item-id="%s"`, id)
		if !strings.Contains(beta, want) {
			t.Errorf("beta group does not contain %q (%q)", id, want)
		}
	}

	if strings.Contains(body, `data-service-id="broken"`) {
		t.Error("blockers page rendered a group for a service with no blocker edges")
	}
}

func TestBlockersPageEmptyState(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())

	body := get(t, NewHandler(root), "/blockers").Body.String()

	for _, want := range []string{"No blockers recorded", "## Dependencies"} {
		if !strings.Contains(body, want) {
			t.Errorf("empty blockers page does not contain %q", want)
		}
	}
	if strings.Contains(body, "blocker-card") {
		t.Error("empty blockers page unexpectedly rendered blocker cards")
	}
	if strings.Contains(body, "data-unresolved-section") {
		t.Error("empty blockers page unexpectedly rendered the unresolved section")
	}
}

func TestBlockersNavigationLinkOnEveryPage(t *testing.T) {
	_, handler := blockerProject(t)

	for _, path := range []string{"/", "/blockers", "/services/alpha"} {
		body := get(t, handler, path).Body.String()
		if !strings.Contains(body, `href="/blockers"`) {
			t.Errorf("%s does not link to /blockers", path)
		}
		if !strings.Contains(body, "What's blocking me") {
			t.Errorf("%s does not label the blockers link", path)
		}
	}
}

func TestBlockersRouteRejectsOtherMethods(t *testing.T) {
	_, handler := blockerProject(t)

	request := httptest.NewRequest(http.MethodPost, "/blockers", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if got := response.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow = %q, want %q", got, http.MethodGet)
	}
}

func TestBlockersRouteReturnsNotFound(t *testing.T) {
	_, handler := blockerProject(t)

	for _, path := range []string{"/blockers/extra", "/blocker"} {
		response := get(t, handler, path)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}

func TestBlockersPageFatalManifestError(t *testing.T) {
	response := get(t, NewHandler(t.TempDir()), "/blockers")

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

func blockerCardHTML(t *testing.T, body, itemID string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)<article[^>]*\bdata-item-id="` + regexp.QuoteMeta(itemID) + `"[^>]*>.*?</article>`)
	card := pattern.FindString(body)
	if card == "" {
		t.Fatalf("blocker card %q not found", itemID)
	}
	return card
}

func serviceBlockersHTML(t *testing.T, body, serviceID string) string {
	t.Helper()
	pattern := regexp.MustCompile(`(?s)<section[^>]*\bdata-service-id="` + regexp.QuoteMeta(serviceID) + `"[^>]*>.*?</section>`)
	group := pattern.FindString(body)
	if group == "" {
		t.Fatalf("blocker group %q not found", serviceID)
	}
	return group
}

func attributeOf(t *testing.T, tag, name string) string {
	t.Helper()
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `="([^"]*)"`)
	match := pattern.FindStringSubmatch(tag)
	if match == nil {
		t.Fatalf("attribute %q not found in %q", name, tag)
	}
	return match[1]
}
