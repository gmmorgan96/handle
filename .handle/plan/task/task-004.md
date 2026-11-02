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
- [x] Two or more services parse into one project
- [x] One broken service yields an error entry while the others still load
- [x] A missing `.handle` is reported, not fatal

## Implementation
- Commit: 218fc9fa6243a18a6e6c3a8dec5368abdc51c2d1
- Message: "T004: Multi-repo parse orchestration"

## Summary
- Added the `internal/project` package with an aggregate `Project` and `Service` model that combines manifest metadata with each service's parsed `plan.Result`.
- `Load(projectRoot)` reads `.project/services.yaml`, iterates services in manifest order, and parses each `<service>/.handle/plan` directory.
- Per-service validation and parse failures are recorded on `Service.Err`; successful services keep their parsed result, so one broken service does not abort the load.
- An unreadable or malformed manifest is fatal; a decoded manifest returns a nil overall error and represents service problems in the aggregate.
- Added tests for two-service aggregation and namespacing, one broken service among healthy siblings, a missing `.handle` directory, and a malformed manifest.
- Verified with `go build ./...`, `go vet ./...`, and `go test ./... -count=1`; all passed.
