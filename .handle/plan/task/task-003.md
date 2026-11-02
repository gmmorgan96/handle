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
- [x] Two services with `feature-001` do not collide
- [x] Ids round-trip through JSON
- [x] Existing single-repo behaviour is unchanged

## Implementation
- Commit: 0cea47c760c2bb8ad57a5831bf5c598bbca89012
- Message: "T003: Introduce namespaced work-item IDs"

## Summary
- Added the `internal/workitem` package with a `WorkItemID` type, `NewID`, `Parse`, `Namespaced`, and service/local accessors.
- IDs may stay local (`feature-001`, `T001`) or be namespaced as `svc/kind-N` (`api-a/feature-001`); two services with the same local id now produce distinct IDs.
- `Parse` and `UnmarshalJSON` validate the service and local-id shapes, while `MarshalJSON` emits the canonical string.
- Tests cover the two-service collision case, JSON round-trip for local and namespaced IDs, valid and invalid parsing, local-id preservation, and already-namespaced IDs.
- No existing files were modified; local IDs remain valid and unchanged, so single-repo behaviour is untouched.
- Verified with `go build ./...`, `go vet ./...`, and `go test ./... -count=1`; all passed.
- This repo has no aggregation, link, or URL code yet; `NewID`/`Namespaced`/`String` provide the namespacing mechanism that T004 and later link/URL work will consume.
