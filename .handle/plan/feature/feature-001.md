# F001 - Define project manifest and multi-service data model

## Description
Specify how a project registers its services and their dependencies, and make work-item identity globally unique across services.

## Requirements
- Define the `services.yaml` schema: `project`, `services[].id`, `name`, `path`, `description`, `depends_on`
- Define validation rules: unique ids, no self-dependency, no cycles, resolvable paths
- Introduce namespaced work-item IDs in the form `svc/kind-N` (for example `api-a/feature-001`)
- Add Go types for `Project`, `Service`, and `ServiceDependency`

## Tasks
- [x] [T001](../task/task-001.md) Define and document services.yaml schema
- [ ] [T002](../task/task-002.md) Implement manifest loader and validation
- [ ] [T003](../task/task-003.md) Introduce namespaced work-item IDs

## Dependencies
- None

## Acceptance Criteria
- [x] Schema documented with a validated example manifest
- [x] `.project/services.yaml` exists and validates
- [ ] Loader parses valid manifests and rejects invalid ones with actionable errors
- [ ] Two services can both contain `feature-001` without collision
- [ ] `go build ./...` and `go test ./...` pass

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
