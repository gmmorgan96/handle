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
- [x] Unit test covers a valid manifest
- [x] Unit test covers malformed YAML
- [x] Unit tests cover duplicate id, self-dependency, and missing path
- [x] Errors identify the offending service

## Implementation
- Commit: 33379f840c51cfcceaf8b8fa676b79f73df8f73b
- Message: "T002: Implement manifest loader and validation"

## Summary
- Added the `internal/manifest` package with `Project`, `Service`, and `ServiceDependency` types plus `Load` and `LoadProject` entry points.
- Decoded manifests with `gopkg.in/yaml.v3`; the project root is the parent of `.project/`, and service paths resolve against that root rather than the manifest file.
- Validation checks project/services presence, id format and uniqueness, name/path presence, self/unknown dependencies, dependency cycles, missing paths, and missing `.handle/plan` directories.
- Failures are collected as typed `ValidationErrors` and each message names the offending service; malformed YAML returns a contextual parse error.
- Added unit tests for valid manifests, malformed YAML, duplicate ids, self-dependencies, unknown dependencies, cycles, missing paths, missing plan directories, absolute paths, invalid ids, and empty service lists.
- Bootstrapped the repo's Go module (`github.com/griffinmorgan/handle`) and `go.sum` because the current service repo had no Go code or module.
- Verified with `go build ./...`, `go vet ./...`, and `go test ./... -count=1 -v`; all passed.
