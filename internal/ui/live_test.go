package ui

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	watch "github.com/gmmorgan96/handle/internal/watcher"
)

const renamedTaskDoc = `# T001 - Renamed task

## Related Feature
- [F001](../feature/feature-001.md)

## Acceptance Criteria
- [x] Done
`

func TestEventsStreamsReadyAndChange(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())
	handler := liveHandler(t, root)

	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/events", nil)
	if err != nil {
		t.Fatalf("create event request: %v", err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("open event stream: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", contentType)
	}
	if cacheControl := response.Header.Get("Cache-Control"); cacheControl != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cacheControl)
	}
	if buffering := response.Header.Get("X-Accel-Buffering"); buffering != "no" {
		t.Errorf("X-Accel-Buffering = %q, want no", buffering)
	}

	reader := bufio.NewReader(response.Body)
	if event, data := readSSEEvent(t, reader); event != "ready" || data != "connected" {
		t.Fatalf("first event = %q/%q, want ready/connected", event, data)
	}

	writeFile(t, filepath.Join(root, "alpha", ".handle", "plan", "task", "task-001.md"), renamedTaskDoc)

	if event, data := readSSEEvent(t, reader); event != "change" || data != "reload" {
		t.Fatalf("second event = %q/%q, want change/reload", event, data)
	}

	body := get(t, handler, "/services/alpha/tasks/T001").Body.String()
	if !strings.Contains(body, "Renamed task") {
		t.Error("task page does not reflect the edited task after a change event")
	}
}

func TestTaskEditReflectedWithoutRestart(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())
	handler := liveHandler(t, root)

	if body := get(t, handler, "/services/alpha/tasks/T001").Body.String(); !strings.Contains(body, "First task") {
		t.Fatal("task page does not show the original title")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := handler.watcher.Changes(ctx)

	writeFile(t, filepath.Join(root, "alpha", ".handle", "plan", "task", "task-001.md"), renamedTaskDoc)
	waitForUIChange(t, changes)

	body := get(t, handler, "/services/alpha/tasks/T001").Body.String()
	if !strings.Contains(body, "Renamed task") {
		t.Error("task page does not reflect the edited task without restarting the handler")
	}
}

func TestManifestServiceAppearsAfterRefresh(t *testing.T) {
	root := t.TempDir()
	services := exampleServices()
	writeProject(t, root, services[:1])
	handler := liveHandler(t, root)

	if body := get(t, handler, "/").Body.String(); strings.Contains(body, "Beta Worker") {
		t.Fatal("overview unexpectedly shows the service before it is added to the manifest")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := handler.watcher.Changes(ctx)

	writeProject(t, root, services)
	waitForUIChange(t, changes)

	body := get(t, handler, "/").Body.String()
	if !strings.Contains(body, "Beta Worker") {
		t.Error("overview does not show the service added to the manifest")
	}
}

func TestLayoutIncludesLiveReloadClient(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, exampleServices())

	body := get(t, NewHandler(root), "/").Body.String()
	for _, want := range []string{
		`new EventSource("/events")`,
		`source.addEventListener("change"`,
		`window.location.reload()`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("layout does not contain %q", want)
		}
	}
}

func TestEventsRejectsNonGet(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/events", nil)
	response := httptest.NewRecorder()

	NewHandler(t.TempDir()).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow = %q, want %q", allow, http.MethodGet)
	}
}

func liveHandler(t *testing.T, root string) *Handler {
	t.Helper()
	handler := NewHandler(root)
	handler.watcher = watch.NewWithOptions(root, watch.Options{
		Interval: 10 * time.Millisecond,
		Debounce: 20 * time.Millisecond,
	})
	return handler
}

func waitForUIChange(t *testing.T, changes <-chan struct{}) {
	t.Helper()
	select {
	case _, ok := <-changes:
		if !ok {
			t.Fatal("change channel closed while waiting for a change")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a change notification")
	}
}

func readSSEEvent(t *testing.T, reader *bufio.Reader) (event, data string) {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read event stream: %v", err)
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if event != "" || data != "" {
				return event, data
			}
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
}
