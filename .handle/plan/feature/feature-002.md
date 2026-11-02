# F002 - Aggregate repos and roll up progress

## Description
Load every registered service's `.handle` directory into a single project model and compute progress from acceptance criteria.

## Requirements
- Resolve each service path and parse its `.handle` directory
- Aggregate parsed services into one `Project` model
- Degrade gracefully when a service fails to parse or is missing
- Roll progress up task → feature → service → project from checkbox counts

## Tasks
- [ ] [T013](../task/task-013.md) Port .handle plan parser
- [ ] [T004](../task/task-004.md) Multi-repo parse orchestration
- [ ] [T005](../task/task-005.md) Progress rollup

## Dependencies
- F001

## Acceptance Criteria
- [ ] A project with two or more services parses into one model
- [ ] A broken service appears with an error while the others still load
- [ ] Rollup math is covered by unit tests
- [ ] An empty service and an empty project are handled without panic

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
