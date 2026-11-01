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
- [ ] Unit tests cover the rollup math
- [ ] An empty service or project returns 0 without panic
- [ ] Partially complete items are counted correctly

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
