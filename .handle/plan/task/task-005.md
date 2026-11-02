# T005 - Progress rollup

## Description
Compute completion from acceptance-criteria checkboxes and roll it up task → feature → service → project.

## Related Feature
- [F002](../feature/feature-002.md)

## Requirements
- Parse checked and unchecked acceptance criteria
- Compute per-task, per-feature, per-service, and project percentages
- Handle division by zero for empty sets

## Dependencies
- T004

## Acceptance Criteria
- [x] Unit tests cover the rollup math
- [x] An empty service or project returns 0 without panic
- [x] Partially complete items are counted correctly

## Implementation
- Commit: b23a178fad5765941fd7d4e1991fc48dbb98d5c5
- Message: "T005: Progress rollup"

## Summary
- Added the `internal/progress` package with `Stats` and `Task`, `Feature`, `Service`, and `Project` rollup functions.
- Task percentage is checked acceptance criteria over total criteria; a task is complete only when it has criteria and all are checked.
- Feature percentage counts complete tasks linked from either the feature's `## Tasks` list or the task's `## Related Feature`.
- Service and project percentages count completed tasks over all parsed tasks, with failed services contributing zero.
- Empty, nil, and zero-denominator inputs return 0% without panicking; percentages are 0-100.
- Added unit tests for partial completion, feature association, service and project rollups, and empty/failed-service cases.
- Verified with `go build ./...`, `go vet ./...`, and `go test ./... -count=1`; all passed.
