## Ghost Ideas — Multi-Service Project Viewer

A local, browser-based viewer for a project that spans multiple service repositories.
Each service keeps its own `.handle` planning directory; a project-level manifest
registers the services and the dependencies between them.

### Phase One - Multi-Service Project Viewer ([P001](../../../.project/project.md#phase-one-multi-service-project-viewer (P001)))

**Purpose**: Aggregate multiple service repos behind a project manifest, namespace work items, and present progress and cross-service blockers in list form.

**Features**
- [x] ([F001](feature/feature-001.md)) Define project manifest and multi-service data model
- [x] ([F002](feature/feature-002.md)) Aggregate repos and roll up progress
- [x] ([F003](feature/feature-003.md)) Build project overview and drill-down UI
- [ ] ([F004](feature/feature-004.md)) Surface cross-service blockers
- [ ] ([F005](feature/feature-005.md)) Add live reload and documentation

**Dependencies**
- None
