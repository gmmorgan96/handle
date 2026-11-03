# T007 - Build project overview UI

## Description
Render every service as a card or list row with progress, open bug count, and error state.

## Related Feature
- [F003](../feature/feature-003.md)

## Requirements
- Service cards showing name, description, and progress bar
- Open bug count per service
- Error badge for services that failed to load
- Sort and filter by name and status

## Dependencies
- T006

## Acceptance Criteria
- [x] All services are visible
- [x] Progress and bug counts match the API
- [x] A broken service shows an error badge
- [x] An empty project shows guidance

## Implementation
- Commit: 830286698019d35ed3d7ecc3ba80bd7fddbb8c0f
- Message: "T007: Build project overview UI"

## Summary
- Added `internal/ui`, a server-rendered overview at `/` with embedded templates and CSS; no JavaScript or frontend build step is required.
- Service cards show name, description, progress bar, task completion, open bug count, dependencies, and a status badge.
- Failed services render an error badge plus their parse/validation error; an empty project shows registration guidance, while a fatal manifest error renders a 500 error page.
- Name and status filters plus name/status sorting are GET query parameters, so filter state survives reload.
- Added `cmd/visualizer` to serve the UI and the existing `/api/project` endpoint on one address.
- Tests compare card progress and bug counts against the JSON API and cover all services, broken services, empty projects, filters, sorting, status codes, static assets, and fatal errors.
- Verified with `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `go test -race ./internal/ui`, and a live server/curl check; all passed.
