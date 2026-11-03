// Package watcher reports when a project's planning files change.
//
// A Watcher polls the project manifest and every registered service's .handle
// tree instead of relying on platform file-system notifications. Polling keeps
// working when files or whole directories are deleted and recreated, and it
// avoids the per-directory bookkeeping that recursive notifications require on
// some platforms. Changes are fingerprinted by path, metadata, and content, so
// an edit is detected even when file sizes stay the same.
package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/gmmorgan96/handle/internal/manifest"
)

const (
	// DefaultInterval is how often the watcher samples the file tree.
	DefaultInterval = 500 * time.Millisecond
	// DefaultDebounce is how long the watcher waits for changes to settle
	// before reporting them.
	DefaultDebounce = 200 * time.Millisecond
)

// Options configures a Watcher. Zero values select the defaults.
type Options struct {
	Interval time.Duration
	Debounce time.Duration
}

// Watcher samples a project's manifest and service planning directories and
// reports changes.
type Watcher struct {
	root     string
	interval time.Duration
	debounce time.Duration
}

// New creates a Watcher for the project rooted at root.
func New(root string) *Watcher {
	return NewWithOptions(root, Options{})
}

// NewWithOptions creates a Watcher with explicit polling and debounce
// intervals.
func NewWithOptions(root string, options Options) *Watcher {
	if options.Interval <= 0 {
		options.Interval = DefaultInterval
	}
	if options.Debounce <= 0 {
		options.Debounce = DefaultDebounce
	}
	return &Watcher{
		root:     filepath.Clean(root),
		interval: options.Interval,
		debounce: options.Debounce,
	}
}

// Changes returns a channel that receives one notification after a burst of
// changes settles. Rapid successive writes are coalesced, and at most one
// notification is buffered if the caller falls behind. The channel closes when
// ctx is cancelled.
func (w *Watcher) Changes(ctx context.Context) <-chan struct{} {
	changes := make(chan struct{}, 1)
	last := w.snapshot()

	go func() {
		defer close(changes)

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		var (
			timer  *time.Timer
			timerC <-chan time.Time
		)
		stopTimer := func() {
			if timer == nil {
				return
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer = nil
			timerC = nil
		}
		defer stopTimer()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current := w.snapshot()
				if current == last {
					continue
				}
				last = current
				stopTimer()
				timer = time.NewTimer(w.debounce)
				timerC = timer.C
			case <-timerC:
				timer = nil
				timerC = nil
				select {
				case changes <- struct{}{}:
				default:
				}
			}
		}
	}()

	return changes
}

// snapshot fingerprints the manifest and every registered service's .handle
// tree. Services are re-discovered on every call so a manifest edit that adds
// a service is reflected without restarting the watcher.
func (w *Watcher) snapshot() string {
	digest := sha256.New()
	writeField(digest, "root", w.root)
	hashFile(digest, filepath.Join(w.root, ".project", "services.yaml"))
	for _, serviceDir := range w.serviceDirs() {
		hashHandleTree(digest, serviceDir)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func (w *Watcher) serviceDirs() []string {
	decoded, _ := manifest.LoadProject(w.root)
	if decoded == nil {
		return nil
	}

	dirs := make([]string, 0, len(decoded.Services))
	for _, service := range decoded.Services {
		if service.Dir != "" {
			dirs = append(dirs, service.Dir)
		}
	}
	sort.Strings(dirs)
	return dirs
}

func hashHandleTree(digest hash.Hash, serviceDir string) {
	handleDir := filepath.Join(serviceDir, ".handle")
	writeField(digest, "service", serviceDir)

	info, err := os.Stat(handleDir)
	if err != nil {
		writeField(digest, "handle-stat-error", err.Error())
		return
	}
	if !info.IsDir() {
		writeField(digest, "handle-not-dir", info.Mode().String())
		return
	}
	writeField(digest, "handle-mode", info.Mode().String())
	writeField(digest, "handle-mtime", info.ModTime().UTC().Format(time.RFC3339Nano))

	err = filepath.WalkDir(handleDir, func(path string, entry fs.DirEntry, walkErr error) error {
		relative, relErr := filepath.Rel(handleDir, path)
		if relErr != nil {
			relative = path
		}

		switch {
		case walkErr != nil:
			writeField(digest, "walk-error", relative, walkErr.Error())
		case entry.IsDir():
			writeField(digest, "dir", relative)
		case !entry.Type().IsRegular():
			writeField(digest, "special", relative, entry.Type().String())
		default:
			hashFile(digest, path)
		}
		return nil
	})
	if err != nil {
		writeField(digest, "walk-failed", err.Error())
	}
}

func hashFile(digest hash.Hash, path string) {
	writeField(digest, "file", path)

	info, err := os.Stat(path)
	if err != nil {
		writeField(digest, "stat-error", err.Error())
		return
	}
	writeField(digest, "mode", info.Mode().String())
	writeField(digest, "size", strconv.FormatInt(info.Size(), 10))
	writeField(digest, "mtime", info.ModTime().UTC().Format(time.RFC3339Nano))

	data, err := os.ReadFile(path)
	if err != nil {
		writeField(digest, "read-error", err.Error())
		return
	}
	checksum := sha256.Sum256(data)
	writeField(digest, "content", hex.EncodeToString(checksum[:]))
}

func writeField(digest hash.Hash, values ...string) {
	for _, value := range values {
		_, _ = io.WriteString(digest, value)
		_, _ = io.WriteString(digest, "\x00")
	}
}
