package ui

import (
	"fmt"
	"net/http"
)

// serveEvents streams Server-Sent Events so the browser can refresh when the
// manifest or any service's .handle files change.
//
// The stream starts with a ready event and then emits one change event per
// coalesced burst of edits. Each browser tab opens its own watcher, which keeps
// the connection independent and means a disconnecting client cannot affect
// other tabs.
func (h *Handler) serveEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.watcher == nil {
		http.Error(w, "live reload is unavailable", http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is unsupported", http.StatusInternalServerError)
		return
	}

	// Take the initial snapshot before announcing readiness. That way any edit
	// made after the browser receives the ready event is guaranteed to appear
	// as a change event.
	changes := h.watcher.Changes(r.Context())

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprint(w, "retry: 1000\n\n")
	fmt.Fprint(w, "event: ready\ndata: connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-changes:
			if !ok {
				return
			}
			if _, err := fmt.Fprint(w, "event: change\ndata: reload\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
