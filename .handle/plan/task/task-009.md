# T009 - Resolve cross-service blockers

## Description
Resolve `## Dependencies` references of the form `svc/kind-N` across services and flag any that cannot be resolved.

## Related Feature
- [F004](../feature/feature-004.md)

## Requirements
- Parse `## Dependencies` entries from work items
- Resolve references against namespaced ids
- Flag unresolved references

## Dependencies
- T003

## Acceptance Criteria
- [x] A cross-service reference resolves
- [x] An unresolved reference is flagged
- [x] Resolution is covered by tests

## Implementation
- Commit: 628cf8cfc468565ec73394475bbbefa61699cbc7
- Message: "T009: Resolve cross-service blockers"

## Summary
- Added `internal/blockers`, which builds a project-wide dependency graph from every successfully loaded service.
- Collects phases, features, tasks, and bugs, then resolves each document's `## Dependencies` entries against namespaced work-item IDs.
- Resolved references link to the target item and are recorded in both directions: the blocked item's `Dependencies` and the blocking item's `Dependents`.
- Cross-service edges are flagged, unresolved references are preserved on the source item and listed in `Resolution.Unresolved`, and references into broken services stay unresolved rather than being dropped.
- Added a `Lookup` method for retrieving items by namespaced ID and defensive handling for nil or empty projects.
- Tests cover resolved cross-service references on both items, local references, all four work-item kinds, unresolved cross-service and local references, broken services, nil/empty projects, and an end-to-end parsed project fixture.
- Verified with `go build ./...`, `go vet ./...`, `go test ./... -count=1`, and `go test -race ./internal/blockers`; all passed.
