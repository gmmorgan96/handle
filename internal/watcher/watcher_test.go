package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testTimeout = 5 * time.Second

func TestChangesAfterFileEdit(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "alpha")
	writeManifest(t, root, "alpha")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changes := testWatcher(root).Changes(ctx)

	taskFile := filepath.Join(root, "alpha", ".handle", "plan", "task", "task-001.md")
	writeFile(t, taskFile, "# T001 - Renamed task\n")

	waitForChange(t, changes)
}

func TestChangesWhenServiceAddedToManifest(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "alpha")
	writeManifest(t, root, "alpha")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changes := testWatcher(root).Changes(ctx)

	writeService(t, root, "beta")
	writeManifest(t, root, "alpha", "beta")

	waitForChange(t, changes)
}

func TestChangesSurviveDeleteAndRecreate(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "alpha")
	writeManifest(t, root, "alpha")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	changes := testWatcher(root).Changes(ctx)

	taskFile := filepath.Join(root, "alpha", ".handle", "plan", "task", "task-001.md")
	if err := os.Remove(taskFile); err != nil {
		t.Fatalf("remove task file: %v", err)
	}
	waitForChange(t, changes)

	writeFile(t, taskFile, "# T001 - Recreated task\n")
	waitForChange(t, changes)
}

func TestChangesDebounceRapidWrites(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "alpha")
	writeManifest(t, root, "alpha")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcher := NewWithOptions(root, Options{Interval: 5 * time.Millisecond, Debounce: 150 * time.Millisecond})
	changes := watcher.Changes(ctx)

	taskFile := filepath.Join(root, "alpha", ".handle", "plan", "task", "task-001.md")
	for i := 0; i < 5; i++ {
		writeFile(t, taskFile, fmt.Sprintf("# T001 - Revision %d\n", i))
		time.Sleep(5 * time.Millisecond)
	}

	waitForChange(t, changes)

	select {
	case <-changes:
		t.Fatal("rapid writes produced more than one change notification")
	case <-time.After(3 * 150 * time.Millisecond):
	}
}

func TestChangesCloseOnContextCancel(t *testing.T) {
	root := t.TempDir()
	writeService(t, root, "alpha")
	writeManifest(t, root, "alpha")

	ctx, cancel := context.WithCancel(context.Background())
	changes := testWatcher(root).Changes(ctx)

	cancel()

	select {
	case _, ok := <-changes:
		if ok {
			t.Fatal("changes channel received a value after cancellation")
		}
	case <-time.After(testTimeout):
		t.Fatal("changes channel did not close after cancellation")
	}
}

func testWatcher(root string) *Watcher {
	return NewWithOptions(root, Options{Interval: 10 * time.Millisecond, Debounce: 20 * time.Millisecond})
}

func waitForChange(t *testing.T, changes <-chan struct{}) {
	t.Helper()
	select {
	case _, ok := <-changes:
		if !ok {
			t.Fatal("changes channel closed while waiting for a change")
		}
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for a change notification")
	}
}

func writeService(t *testing.T, root, id string) {
	t.Helper()
	writeFile(t, filepath.Join(root, id, ".handle", "plan", "task", "task-001.md"), "# T001 - First task\n")
}

func writeManifest(t *testing.T, root string, ids ...string) {
	t.Helper()

	var manifest strings.Builder
	manifest.WriteString("project: Example Project\nservices:\n")
	for _, id := range ids {
		fmt.Fprintf(&manifest, "  - id: %s\n    name: %s\n    path: ./%s\n", id, id, id)
	}
	writeFile(t, filepath.Join(root, ".project", "services.yaml"), manifest.String())
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
