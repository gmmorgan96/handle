# T010 - Add "what's blocking me" list view

## Description
Provide a global view of blocked work items and their blockers across all services.

## Related Feature
- [F004](../feature/feature-004.md)

## Requirements
- List blocked items together with their blockers
- Group or sort by service
- Surface unresolved references

## Dependencies
- T009

## Acceptance Criteria
- [x] Blocked items and their blockers are shown together
- [x] Cross-service blockers are visible
- [x] Unresolved references are surfaced

## Implementation
- Commit: 94e764235113035c0bd08429d7bfb2662a2115fa
- Message: "T010: Add "what's blocking me" list view"

## Summary
- Added a server-rendered `/blockers` page, linked as “What’s blocking me” from the shared site header on every page.
- Added `internal/ui/blockers.go`, which builds the view from `blockers.Resolve` and groups items by service in manifest order.
- Each item card shows both **Blocked by** and **Blocks**, so a cross-service blocker appears on the blocked and blocking item. Cards and references carry cross-service badges, and phases link without a detail page.
- Unresolved references are marked inline on the declaring item and listed in a dedicated unresolved-references section, with summary counts for blocked items, cross-service references, and unresolved references.
- Added `internal/ui/templates/blockers.html` plus blocker styles in `style.css`, registered the template and route in `ui.go`, and added the header navigation link in `layout.html`.
- Added `internal/ui/blockers_test.go` covering both dependency directions, cross-service markers, unresolved references (including a broken service), service grouping and omission of unrelated items, summary counts, empty state, navigation, 404, 405, and fatal manifest errors.
- Verified with `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `go test -race ./internal/ui`, and a live server/curl check of `/blockers`; all passed.
