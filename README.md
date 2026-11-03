# Handle

Handle is a local, read-only web viewer for a project that spans multiple
service repositories. Each service keeps its own `.handle/plan` markdown, and a
project-level manifest registers the services and the dependencies between
them. Handle aggregates the service checkouts, namespaces work-item ids, rolls
progress up to the project, and lists cross-service blockers.

Markdown on disk is the source of truth. Handle has no database and no write
path; git stays the history.

## Requirements

- Go 1.24.11 or newer (matching `go.mod`)
- A project root with `.project/services.yaml` and one checkout per registered
  service

## Quick start

Run the included example project from the repository root:

```bash
go run ./cmd/visualizer -root examples/basic
```

Then open <http://localhost:8080>.

To build a binary:

```bash
go build -o handle ./cmd/visualizer
./handle -root examples/basic
```

### Flags

| Flag | Default | Meaning |
|---|---|---|
| `-root` | `.` | Project root directory containing `.project/services.yaml` |
| `-addr` | `:8080` | Address to listen on |

## Example project

[`examples/basic`](examples/basic) is a self-contained two-service project:

```text
examples/basic/
  .project/
    project.md
    services.yaml
  api/
    .handle/plan/
      plan.md
      feature/feature-001.md
      task/task-001.md
      task/task-002.md
      bug/bug-001.md
  worker/
    .handle/plan/
      plan.md
      feature/feature-001.md
      task/task-001.md
```

It demonstrates:

- Service-level dependencies: `api` depends on `worker`.
- A cross-service work-item blocker: `api/T002` depends on `worker/T001`.
- Mixed completion: the API service has one complete task and one incomplete
  task; the worker has one incomplete task.
- One open bug in the API service.

The manifest is
[`examples/basic/.project/services.yaml`](examples/basic/.project/services.yaml).
After starting the viewer, the overview shows two services, the service page
shows the nested tasks and bug, and **What's blocking me** shows the
cross-service blocker.

## Project layout

```text
my-project/                  # project root
  .project/
    project.md               # goal and phase goals (not read by Handle)
    services.yaml            # manifest: registers the services
  api/
    .handle/plan/
      plan.md
      feature/feature-001.md
      task/task-001.md
      bug/bug-001.md
  worker/
    .handle/plan/
      ...
```

Each service repository remains useful on its own: it keeps its own `.handle`
directory, and the project root only adds the manifest and project-level
documents. Service directories may be separate git repositories.

Within a service's `.handle/plan` directory:

- `plan.md` is required when the directory has any content. It defines phases
  and links features.
- `feature/`, `task/`, and `bug/` are optional directories containing the
  corresponding markdown documents. Every `.md` file directly inside them is
  loaded.
- An empty `.handle/plan` directory is a valid, empty service.

## Manifest: `services.yaml`

```yaml
project: Example Tasks
services:
  - id: api
    name: Task API
    path: ./api
    description: Serves the task list over HTTP
    depends_on: [worker]
  - id: worker
    name: Task Worker
    path: ./worker
    description: Processes queued tasks in the background
    depends_on: []
```

| Field | Required | Meaning |
|---|---|---|
| `project` | yes | Human-readable project name |
| `services` | yes | Non-empty list of service repositories |
| `services[].id` | yes | Stable, unique lowercase slug (`api`, `worker`) |
| `services[].name` | yes | Display name |
| `services[].path` | yes | Repository path relative to the project root |
| `services[].description` | no | One-line description shown in the UI |
| `services[].depends_on` | no | Service ids this service depends on; default `[]` |

`path` is resolved relative to the project root (the directory containing
`.project/`), not relative to `services.yaml`.

Validation has two tiers:

1. **Schema validation:** the project name and service list are non-empty, ids
   match `^[a-z0-9]+(-[a-z0-9]+)*$` and are unique, `depends_on` entries name
   declared services, and there are no self-dependencies or dependency cycles.
2. **Resolution validation:** every `path` resolves to an existing directory
   containing `.handle/plan/`.

A missing or malformed manifest is fatal: the JSON API returns `500` and the UI
shows a project-load error page. Problems with an individual service are
reported per service instead; the other services still load and render.

## Work items and ids

Work-item files are plain markdown with no frontmatter and no status field.
Each document starts with a heading in the form `# <id> - <title>`:

```md
# T001 - Add task list endpoint
```

Local ids are unique only within a service. Handle namespaces them across the
project:

```text
api/feature-001
api/T001
worker/T001
```

Use the namespaced form when referring to work in another service. Use the
short form for items in the same service.

### Feature files

`feature/feature-NNN.md` contains:

```md
# F001 - Task list API

## Description
Expose the example task list over HTTP.

## Requirements
- List tasks over HTTP

## Tasks
- [x] [T001](../task/task-001.md) Add task list endpoint
- [ ] [T002](../task/task-002.md) Add task detail endpoint

## Bugs
- [ ] [B001](../bug/bug-001.md) Task list returns stale results

## Dependencies
- None

## Acceptance Criteria
- [x] Task list endpoint is available
- [ ] Task detail endpoint is available
```

### Task files

`task/task-NNN.md` contains `## Related Feature`, `## Requirements`,
`## Dependencies`, and `## Acceptance Criteria`, for example:

```md
# T002 - Add task detail endpoint

## Related Feature
- [F001](../feature/feature-001.md)

## Requirements
- Serve GET /tasks/{id}

## Dependencies
- worker/T001

## Acceptance Criteria
- [ ] GET /tasks/{id} returns one task
```

### Bug files

`bug/bug-NNN.md` adds the reproduction and diagnosis sections:

```md
# B001 - Task list returns stale results

## Related Feature
- [F001](../feature/feature-001.md)

## Steps to Reproduce
1. Change a task document on disk.
2. Refresh the task list endpoint.

## Expected/Actual
- **Expected**: The response reflects the change.
- **Actual**: The response keeps the old value.

## Potentially Introduced
- Commit: (unknown)
- Message: (unknown)

## Dependencies
- None

## Acceptance Criteria
- [ ] The task list reflects changes immediately
```

### Dependencies

`## Dependencies` is the blocker list:

```md
## Dependencies
- None
```

or a local/namespaced work-item id per line:

```md
## Dependencies
- T004
- worker/T001
```

Dependencies are resolved against every loaded service. References that do not
match a loaded work item are reported as unresolved in the blockers view.

## Progress

Completion is derived only from acceptance-criteria checkboxes:

| Level | Completion |
|---|---|
| Task | checked criteria / total criteria |
| Feature | complete tasks / tasks linked from `## Tasks` or `## Related Feature` |
| Service | complete tasks / all tasks in the service |
| Project | complete tasks / all tasks across every loaded service |

A task is complete only when it has at least one criterion and every criterion
is checked. An empty level is `0%` rather than a division by zero. The feature
checkbox in `plan.md` is informational; the task's own criteria are
authoritative.

## Web UI

| Path | Page |
|---|---|
| `/` | Project overview with service cards, filters, and sorting |
| `/blockers` | Cross-service blocked and blocking work items |
| `/services/{service-id}` | Service drill-down with features and nested items |
| `/services/{service-id}/features/{feature-id}` | Feature detail |
| `/services/{service-id}/tasks/{task-id}` | Task detail |
| `/services/{service-id}/bugs/{bug-id}` | Bug detail |
| `/events` | Server-Sent Events stream used for live reload |
| `/static/...` | CSS assets |

The overview accepts these query parameters:

- `q` filters services by id or name.
- `status` is one of `all`, `active`, `complete`, `empty`, or `error`.
- `sort` is `name` or `status`.

## API

### `GET /api/project`

Returns the aggregated project as JSON. Services keep the order declared in
`.project/services.yaml`.

```json
{
  "name": "Example Tasks",
  "root": "/absolute/path/to/examples/basic",
  "progress": {"done": 1, "total": 3, "percent": 33.33333333333333},
  "services": [
    {
      "id": "api",
      "name": "Task API",
      "description": "Serves the task list over HTTP",
      "path": "./api",
      "depends_on": ["worker"],
      "progress": {"done": 1, "total": 2, "percent": 50},
      "counts": {"features": 1, "tasks": 2, "bugs": 1}
    },
    {
      "id": "worker",
      "name": "Task Worker",
      "description": "Processes queued tasks in the background",
      "path": "./worker",
      "depends_on": [],
      "progress": {"done": 0, "total": 1, "percent": 0},
      "counts": {"features": 1, "tasks": 1, "bugs": 0}
    }
  ]
}
```

Response fields:

| Field | Meaning |
|---|---|
| `name` | Project name from the manifest |
| `root` | Absolute project root |
| `progress` | Project-wide task rollup |
| `services[].progress` | Service-wide task rollup |
| `services[].counts` | Number of parsed features, tasks, and bugs |
| `services[].error` | Per-service load error; omitted when the service loaded |

Status codes:

- `200` — project loaded. Individual service failures appear in
  `services[].error`.
- `405` — method other than `GET`.
- `500` — the manifest could not be read or decoded.

## Live reload

Every HTML page includes a small `EventSource` client that listens to
`/events`. When a planning file changes, the browser reloads the current page:

1. `/events` sends `retry: 1000`, then an initial `event: ready`.
2. After a coalesced burst of edits it sends `event: change`.
3. The page calls `window.location.reload()` and reconnects.

The watcher polls `.project/services.yaml` and every registered service's
`.handle` tree every 500 ms, with a 200 ms debounce. It re-reads the manifest on
each snapshot, so adding a service appears after the next reload without
restarting the server. Because it fingerprints the tree rather than holding
file descriptors, it also survives file and directory deletion/recreation.

Project data is reloaded from disk on every HTTP request; the watcher only tells
the browser when to refresh.

## Architecture

Handle is a small pipeline, with the manifest as the single entry point:

```text
.project/services.yaml
        |
   internal/manifest        parse + validate the manifest
        |
   internal/plan            parse each service's .handle/plan
        |                   and namespace its ids
   internal/project         aggregate services, isolate failures
        |
   +----+-------------------+
   |                        |
internal/progress      internal/blockers
(rollups)              (dependency graph)
   |                        |
   +-----------+------------+
               |
       internal/api   internal/ui
       (JSON)         (HTML + SSE)
               |
       internal/watcher  (change notifications)
```

| Package | Responsibility |
|---|---|
| `cmd/visualizer` | Flag parsing, HTTP server, route wiring |
| `internal/manifest` | Load and validate `.project/services.yaml`, resolve service paths |
| `internal/plan` | Parse `plan.md`, features, tasks, and bugs; namespace local ids |
| `internal/project` | Combine the manifest with each service's parsed plan; record per-service errors |
| `internal/progress` | Task → feature → service → project completion rollup |
| `internal/blockers` | Resolve local and cross-service dependency references |
| `internal/api` | `GET /api/project` JSON handler |
| `internal/ui` | Server-rendered pages, filters, drill-down, and the SSE endpoint |
| `internal/watcher` | Polling file-change detection with debounce |
| `internal/workitem` | Work-item id parsing, validation, and namespacing |

A service that fails to resolve or parse is reported in the UI and API while the
remaining services continue to load. The viewer does not cache project data, so
there is no cache invalidation step after an edit.

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

The only external dependency is `gopkg.in/yaml.v3` for manifest decoding. There
is no database, authentication, or external service to start.

Phase one is intentionally read-only and lists-first: editing from the UI,
dependency graph rendering, and agent orchestration are out of scope.
