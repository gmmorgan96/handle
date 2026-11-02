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
- [x] A valid `.handle/plan/` tree parses into phases, features, tasks, and bugs
- [x] Item ids are namespaced with the service id and do not collide across services
- [x] Missing and malformed plan files return errors without panicking
- [x] An empty `.handle/plan/` parses to an empty result without error
- [x] `go build ./...` and `go test ./...` pass

## Implementation
- Commit: 5b4dda4b4005243c37da4248bce91372e8d429c2
- Message: "T013: Port .handle plan parser"

## Summary
- Added `internal/plan`, a self-contained parser for one service's `.handle/plan` tree, so T004 can import it without a local `replace` directive.
- Ported plan, feature, task, and bug parsing, including descriptions, requirements, `## Tasks`, `## Dependencies`, acceptance checkboxes, `## Implementation`, `## Summary`, and the bug-specific reproduction, expected/actual, and potentially-introduced sections.
- Stored all item identifiers as `workitem.WorkItemID` and namespaced them with the service id in `Load`; cross-service references are left unchanged.
- Added `plan.Load(serviceID, root)` returning a per-service `Result`. An empty `.handle/plan` directory yields an empty result, while missing or malformed files return path-qualified errors.
- Treats `(pending)` implementation fields as "not yet committed" and accepts `(unknown)` in a bug's potentially-introduced section.
- Added tests covering plan/feature/task/bug documents, dependency and namespacing behavior, empty plans and directories, malformed files, and this repository's real `.handle` tree.
- Follow-up commit `355486535e64430d5c3e9eb548e5c1fd40db7bc6` made the optional bug `## Phase` section explicitly singular and added a regression test that rejects multiple entries.
- Verified with `go build ./...`, `go vet ./...`, and `go test ./...`; all pass.
- No feature acceptance criterion was marked complete because F002 still needs T004 aggregation and T005 rollup.
