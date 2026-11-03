# F003 - Build project overview and drill-down UI

## Description
Expose the aggregated project over HTTP and give the browser a usable overview plus per-service drill-down.

## Requirements
- Add a `GET /api/project` JSON endpoint
- Project overview listing services with progress, open bug count, and error state
- Drill-down from project → service → features/tasks/bugs
- Breadcrumb navigation and deep links

## Tasks
- [x] [T006](../task/task-006.md) Add GET /api/project endpoint
- [x] [T007](../task/task-007.md) Build project overview UI
- [x] [T008](../task/task-008.md) Build service drill-down UI

## Dependencies
- F002

## Acceptance Criteria
- [x] `/api/project` returns the documented shape
- [x] Overview shows every service with progress and error badges
- [x] Can navigate three levels deep and back
- [x] Empty and broken states render sensibly

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
