# T001 - Define and document services.yaml schema

## Description
Specify the project manifest used to register services, their repository paths, and service-level dependencies, including validation rules and a worked example.

## Related Feature
- [F001](../feature/feature-001.md)

## Requirements
- Define fields: `project`, `services[].id`, `name`, `path`, `description`, `depends_on`
- Enumerate validation rules: unique ids, no self-dependency, no cycles, resolvable paths
- Define the `## Dependencies` convention for feature and task files (local ids, `svc/kind-N` for cross-service refs, `- None` when empty)
- Define the required sections for feature, task, and bug files, including `## Tasks`, `## Steps to Reproduce`, `## Expected/Actual`, `## Potentially Introduced`, `## Implementation`, and `## Summary`
- Define `path` semantics: service paths are relative to the project root (the parent of `.project/`), not to the manifest file
- Create the live manifest at `.project/services.yaml`, registering `handle` as the first service
- Add a validated example manifest to the docs

## Dependencies
- None

## Acceptance Criteria
- [ ] Schema is documented
- [ ] Example manifest validates
- [ ] `.project/services.yaml` exists and validates against the documented schema
- [ ] A loader can find every service root from the manifest alone
- [ ] Validation rules are enumerated

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
