# T002 - Implement manifest loader and validation

## Description
Parse `services.yaml` into Go types, resolve relative paths against the manifest directory, and reject invalid manifests with actionable errors.

## Related Feature
- [F001](../feature/feature-001.md)

## Requirements
- Parse YAML into `Project` and `Service` types
- Resolve service paths relative to the project root (parent of `.project/`), not the manifest file
- Validate unique ids, self-dependencies, cycles, and missing paths
- Report errors that name the offending service

## Dependencies
- T001

## Acceptance Criteria
- [ ] Unit test covers a valid manifest
- [ ] Unit test covers malformed YAML
- [ ] Unit tests cover duplicate id, self-dependency, and missing path
- [ ] Errors identify the offending service

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
