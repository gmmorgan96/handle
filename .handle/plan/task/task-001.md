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
- Define the `## Tasks` entry format as `- [ ] [T001](../task/task-001.md) Title`, with the checkbox mirroring the task's completion
- Define `path` semantics: service paths are relative to the project root (the parent of `.project/`), not to the manifest file
- Create the live manifest at `.project/services.yaml`, registering `handle` as the first service
- Add a validated example manifest to the docs

## Dependencies
- None

## Acceptance Criteria
- [x] Schema is documented
- [x] Example manifest validates
- [x] `.project/services.yaml` exists and validates against the documented schema
- [x] A loader can find every service root from the manifest alone
- [x] Validation rules are enumerated

## Implementation
- Commit: 22209363fcad2b8833ec0c207ee82fd4468142e9
- Message: "T001: Define and document services.yaml schema"

## Summary
- Added `.project/schema.md` documenting the on-disk contract: project layout, `services.yaml` fields, project-root-relative path semantics, two-tier validation, work-item namespacing, and required sections for feature/task/bug files.
- Defined the `## Dependencies`, `## Tasks`, `## Implementation`, and `## Summary` conventions, including the `(pending)` placeholder.
- Documented progress as checkbox-derived and deferred the exact rollup to T005.
- Verified the live `.project/services.yaml` and the doc's example against schema validation; the live manifest also passes resolution validation (`./handle` exists and contains `.handle/plan/`).
- Verified a loader resolves every service root from the manifest alone, with no hardcoded paths.
- Linked the schema from `.project/project.md`.
- The implementation commit lives in the project repo (`ghost-ideas`), because the schema is project-level, not service-level.
