# T006 - Add GET /api/project endpoint

## Description
Expose the aggregated project as JSON for the frontend.

## Related Feature
- [F003](../feature/feature-003.md)

## Requirements
- Serialize services, counts, progress, and per-service errors
- Document the response shape
- Return sensible status codes

## Dependencies
- T005

## Acceptance Criteria
- [x] Endpoint returns the documented shape
- [x] A broken service is represented in the payload
- [x] The response is covered by a test

## Implementation
- Commit: 6f97bc1925bc4723a9348f380246a5ba0ea6ef4e
- Message: "T006: Add GET /api/project endpoint"

## Summary
- Added the `internal/api` package with a `Handler` that serves `GET /api/project` and returns the aggregated project as JSON.
- The response documents and serializes project name/root/progress plus each service's id, name, description, path, dependencies, progress, item counts, and optional load error.
- Healthy services omit `error`; failed services report the parse/validation error with zeroed progress and counts while the remaining services still load.
- Status codes are 200 for success, 500 for a fatal manifest error, 405 (with `Allow: GET`) for other methods, and 404 for unknown paths.
- Tests cover the documented shape, a broken and a missing-`.handle` service, malformed manifests, method handling, dependency serialization, and JSON serialization.
- Verified with `go build ./...`, `go vet ./...`, and `go test ./... -count=1`; all passed.
