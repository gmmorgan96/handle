# T011 - Add live reload on file change

## Description
Watch `.handle` files and the project manifest, and refresh the UI without restarting the server.

## Related Feature
- [F005](../feature/feature-005.md)

## Requirements
- Watch the manifest and every service `.handle` directory
- Debounce rapid successive changes
- Push or poll updates to the browser

## Dependencies
- T004

## Acceptance Criteria
- [x] Editing a task file updates the UI without a restart
- [x] Adding a service to the manifest appears after refresh
- [x] The watcher survives file deletion and recreation

## Implementation
- Commit: 841d15c9f85d47637ad5e956d7953b6839bbd79d
- Message: "T011: Add live reload on file change"

## Summary
- Added `internal/watcher`, a dependency-free polling watcher that fingerprints the manifest plus every registered service's `.handle` tree and emits debounced change notifications.
- Re-discovers services from the manifest on every snapshot, so a newly registered service is picked up without restarting the watcher or server.
- Added `GET /events`, a Server-Sent Events stream that sends an initial `ready` event and a `change` event per coalesced burst of edits, and wired it into the UI handler.
- Added an inline `EventSource` client to the shared layout that reloads the current page on a `change` event.
- Tests cover edits, manifest service additions, delete/recreate survival, debounce coalescing, context cancellation, SSE headers/events, page reflection without restart, manifest refresh, and method rejection.
- Verified with `go build ./...`, `go vet ./...`, `go test ./... -count=1`, and `go test -race ./internal/watcher ./internal/ui`; all passed.
- Verified live against a temporary project: `/` rendered the service and `curl -N /events` received `ready` followed by `change` after editing a task file.
