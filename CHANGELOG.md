# Changelog

## 0.2.0 — Unreleased

### Breaking changes

- Configuration now uses one strict `endpoints` list. Legacy `http_endpoints`, `ws_endpoints`, and `slowness` are rejected.
- Every endpoint has an explicit `type`: `http`, `proxy`, `ws/echo`, or `ws/stream`.
- `path` is the final public path. WebSocket routes no longer receive an implicit `/ws` prefix.
- Go 1.25 is now required to build the project.

### Added

- Configurable static HTTP responses, endpoint-specific chaos settings, and validation for route conflicts.
- Reverse-proxy endpoints with timeout, path/query/header forwarding, host controls, and injected chaos.
- WebSocket echo and periodic stream endpoints with connection limits, deadlines, ping/pong handling, and metrics.
- Bounded Prometheus metrics for HTTP, WebSocket, and proxy traffic.
- Graceful shutdown on `SIGINT` and `SIGTERM`, server timeouts, CI lint/race checks, and separate local/multi-architecture Docker targets.
