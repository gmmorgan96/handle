# T008 - Build service drill-down UI

## Description
Clicking a service shows its features with nested tasks and bugs, with breadcrumb navigation.

## Related Feature
- [F003](../feature/feature-003.md)

## Requirements
- Service page listing features and their tasks and bugs
- Status per item derived from acceptance criteria
- Breadcrumb navigation from project → service → item

## Dependencies
- T007

## Acceptance Criteria
- [x] Can navigate project → service → item and back
- [x] A feature shows its nested tasks
- [x] Bug detail shows Steps to Reproduce, Expected/Actual, and Potentially Introduced
- [x] Bugs are listed separately from tasks
- [x] A deep link works after a page reload

## Implementation
- Commit: b9faa0658b8dd7fdd59da16f0ad88ac50e0db337
- Message: "T008: Build service drill-down UI"

## Summary
- Added server-rendered service drill-down routes: `/services/{service}`, `/services/{service}/features/{feature}`, `/services/{service}/tasks/{task}`, and `/services/{service}/bugs/{bug}`.
- The service page lists feature cards with nested tasks and bugs in separate sections; tasks and bugs without a matching feature appear under "Unassigned".
- Every page has breadcrumb navigation back through project → service → feature → item, and overview service names now link to their service pages.
- Feature, task, and bug badges derive from each document's own acceptance criteria (`complete`, `in progress`, `not started`, `no criteria`); the service rollup still matches the overview and API.
- Bug detail renders Steps to Reproduce, Expected/Actual, and Potentially Introduced, including the introduced commit and message when recorded.
- Added `internal/ui/detail.go`, `detail_test.go`, shared template partials, and detail-page styles.
- Tests cover navigation and breadcrumb targets, nested items, separate bug listing, acceptance-criteria statuses, bug fields, empty and broken service states, 404/405 handling, and a repeated deep-link request with fresh handlers.
- Verified with `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `go test -race ./internal/ui`, and live `curl` checks against a temporary project; all passed, including an identical body on deep-link reload.
