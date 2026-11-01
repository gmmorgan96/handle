# T003 - Introduce namespaced work-item IDs

## Description
Make work-item identity globally unique across services by namespacing ids as `svc/kind-N` and update the internal model to use them.

## Related Feature
- [F001](../feature/feature-001.md)

## Requirements
- Add a `WorkItemID` type or equivalent
- Namespace ids during aggregation
- Update links and URLs to use the namespaced form

## Dependencies
- None

## Acceptance Criteria
- [ ] Two services with `feature-001` do not collide
- [ ] Ids round-trip through JSON
- [ ] Existing single-repo behaviour is unchanged

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
