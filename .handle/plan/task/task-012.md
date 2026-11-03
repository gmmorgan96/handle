# T012 - Update README and architecture notes

## Description
Document setup, the manifest schema, the API, and the multi-service model.

## Related Feature
- [F005](../feature/feature-005.md)

## Requirements
- Update the README with multi-service usage
- Document `services.yaml` and namespaced ids
- Document the API endpoints

## Dependencies
- T010

## Acceptance Criteria
- [x] README covers setup and the manifest
- [x] Endpoints are documented
- [x] An example project is referenced

## Implementation
- Commit: 5c77495d32cf46c1e84ba6dc9c4a7af52ce6bc25
- Message: "T012: Update README and architecture notes"

## Summary
- Added `README.md` covering requirements, build/run flags, project layout, the `services.yaml` schema and validation rules, work-item format and namespaced ids, dependency references, progress rollup, UI routes, the JSON API, live reload, package architecture, and development commands.
- Added the runnable `examples/basic` two-service project, which demonstrates a service-level dependency, a cross-service work-item blocker, mixed task completion, and an open bug.
- Documented `GET /api/project` with the response shape, field meanings, and status codes, plus the UI and `/events` SSE routes.
- Verified the example against the running server: the API reports 1/3 project progress for the two services, the overview shows both services, the blockers page shows `api/T002` blocked by `worker/T001`, the task detail page returns 200, and `/events` sends its `ready` event.
- Verified with `go build ./...`, `go vet ./...`, and `go test ./... -count=1`; all passed. The README's JSON example and referenced example paths were validated separately.
