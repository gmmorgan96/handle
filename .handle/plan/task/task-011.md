# T011 - Add live reload on file change

## Description
Watch `.handle` files and the project manifest, and refresh the UI without restarting the server.

## Related Feature
- [F005](../feature/feature-005.md)

## Requirements
- Watch the manifest and every service `.handle` directory
- Debounce rapid successive changes
- Push or poll updates to the browser

## Dependencies
- T004

## Acceptance Criteria
- [ ] Editing a task file updates the UI without a restart
- [ ] Adding a service to the manifest appears after refresh
- [ ] The watcher survives file deletion and recreation

## Implementation
- Commit: (pending)
- Message: (pending)

## Summary
(pending)
