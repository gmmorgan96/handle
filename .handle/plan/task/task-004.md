# T004 - Multi-repo parse orchestration

## Description
For each service in the manifest, run the existing parser against `<path>/.handle` and aggregate the results into one project.

## Related Feature
- [F002](../feature/feature-002.md)

## Requirements
- Iterate manifest services and parse each `.handle` directory
- Aggregate results into the `Project` model
- Record per-service errors without aborting the whole load

## Dependencies
- T002
- T003
- T013

## Acceptance Criteria
- [ ] Two or more services parse into one project
- [ ] One broken service yields an error entry while the others still load
- [ ] A missing `.handle` is reported, not fatal

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
