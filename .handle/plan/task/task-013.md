# T013 - Port .handle plan parser

## Description
Bring the existing single-repo `.handle` parser into this module so the multi-service loader has something to run against each service. The parser reads `<service>/.handle/plan/` and returns the phases, features, tasks, and bugs that aggregation, progress, the API, and the UI need. The original parser lives in the existing Handle codebase (`internal/viz`); this task ports its behavior rather than inventing a new file format.

## Related Feature
- [F002](../feature/feature-002.md)

## Requirements
- Port the existing parsing behavior for `plan/plan.md`, `feature/*.md`, `task/*.md`, and `bug/*.md` into this module.
- Parse the fields required by the documented schema: ids, titles, descriptions, requirements, `## Tasks`, `## Dependencies`, acceptance-criteria checkboxes, `## Implementation`, `## Summary`, and the bug-specific sections.
- Return a per-service result that T004 can aggregate; keep cross-service loading and error isolation in T004.
- Use `workitem.WorkItemID` for item ids and namespace every item with its service id.
- Return actionable errors for missing or malformed plan files instead of panicking.
- Preserve the existing single-repo output for a valid `.handle/plan/` tree.
- Live inside this module so T004 can import the parser without a local `replace` directive.
- Add tests covering plan, feature, task, and bug documents, an empty plan, and missing or malformed files.

## Dependencies
- T001
- T003

## Acceptance Criteria
- [ ] A valid `.handle/plan/` tree parses into phases, features, tasks, and bugs
- [ ] Item ids are namespaced with the service id and do not collide across services
- [ ] Missing and malformed plan files return errors without panicking
- [ ] An empty `.handle/plan/` parses to an empty result without error
- [ ] `go build ./...` and `go test ./...` pass

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
