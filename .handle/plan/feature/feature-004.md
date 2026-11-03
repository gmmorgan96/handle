# F004 - Surface cross-service blockers

## Description
Resolve work-item dependencies that cross service boundaries and make them visible in list form.

## Requirements
- Parse `## Dependencies` entries of the form `svc/kind-N`
- Resolve references across all services
- Flag unresolved references rather than silently dropping them
- Provide a global "what's blocking me" view

## Tasks
- [x] [T009](../task/task-009.md) Resolve cross-service blockers
- [x] [T010](../task/task-010.md) Add "what's blocking me" list view

## Dependencies
- F001
- F003

## Acceptance Criteria
- [x] A cross-service blocker appears on both the blocked and the blocking item
- [x] An unresolved reference is flagged
- [x] The global view lists blocked items together with their blockers

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
