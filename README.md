# fake-web-service

[![License: MIT](https://img.shields.io/badge/License-MIT%202.0-blue.svg)](https://github.com/sergkondr/fake-web-service/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/sergkondr/fake-web-service.svg)](https://github.com/sergkondr/fake-web-service/releases/latest)
[![Go](https://github.com/sergkondr/fake-web-service/actions/workflows/go.yml/badge.svg)](https://github.com/sergkondr/fake-web-service/actions/workflows/go.yml)

`fake-web-service` is a configurable service for HTTP and WebSocket testing. It can return static HTTP responses, echo and stream WebSocket messages, or proxy an upstream service while injecting latency and errors.

## Run

Go 1.25 or newer is required.

```shell
go run ./cmd --config examples/config.yaml
curl http://localhost:8080/
```

The process exits with a non-zero status if the configuration cannot be loaded or the server cannot listen on the configured address. `SIGINT` and `SIGTERM` stop it gracefully; in-flight requests have up to 10 seconds to finish.

Run the supplied development configuration in Docker:

```shell
docker compose up --build
```

Deploy the Kubernetes example after replacing its image tag with the version you want to run:

```shell
kubectl apply -f deployments/manifests/kubernetes-deploy.yaml
```

## Configuration

The parser is strict: unknown fields, legacy `http_endpoints`, `ws_endpoints`, and `slowness` are rejected. `path` is always the final public path; no endpoint type adds a hidden prefix.

```yaml
listen: 0.0.0.0:8080 # optional; default: 0.0.0.0:8080

metrics:
  enabled: true
  path: /metrics # optional; default: /metrics when enabled

endpoints:
  - name: Good endpoint
    type: http
    description: Static HTTP response
    path: /good
    response: # optional; status defaults to 200
      status: 201
      headers:
        Content-Type: text/plain
        X-Example: configured
      body: created
    chaos: # optional; applies before the handler
      error_rate: 0.3 # range: 0.0–1.0
      error_status: 503 # optional; default: 500
      latency:
        min: 10ms
        p95: 50ms
        max: 100ms

  - name: Upstream service
    type: proxy
    path: /service
    backend:
      url: https://service.example.com/api/v1
      timeout: 10s # optional; default: 10s
      preserve_host: false
    chaos:
      error_rate: 0.1

  - name: Echo
    type: ws/echo
    path: /ws/echo

  - name: Clock stream
    type: ws/stream
    path: /ws/time
    stream:
      interval: 1s
```

All endpoint types support `name`, `description`, `hidden`, `do_not_log`, and `chaos`. `hidden` excludes an endpoint from `/`; `do_not_log` disables its access log.

`http` supports `GET` and `HEAD`. `proxy` supports every HTTP method and owns its configured path plus all nested paths. With `path: /service` and `backend.url: https://service.example.com/api/v1`, a request to `/service/users?id=42` reaches `/api/v1/users?id=42` upstream. Network and timeout errors return `502 Bad Gateway`.

`ws/echo` accepts text frames and returns a JSON text frame. Binary frames are closed with code `1003`. `ws/stream` sends a JSON message immediately after the handshake and then at the configured interval. It includes the backend hostname, endpoint, sequence, and an RFC3339Nano UTC timestamp.

## Metrics and operations

Set `metrics.enabled: true` to expose Prometheus metrics. HTTP labels are bounded to configured `endpoint`, endpoint `type`, request `method`, and `status_code`; URL subpaths, query parameters, client addresses, and endpoint names are never labels. Proxy upstream duration and errors use only the configured public endpoint label. WebSocket endpoints export connection, message, and read/write-error metrics.

The HTTP server has 5-second header, 30-second read/write, and 2-minute idle timeouts. WebSocket handlers maintain their own read/write deadlines and ping/pong policy after upgrade.

## Development and release

```shell
make fmt
make lint
make test
make test-race
make build APP_VERSION=0.2.0
make docker APP_VERSION=0.2.0
make docker-multiarch APP_VERSION=0.2.0
```

`make docker` builds a local image. `make docker-multiarch` builds and pushes `linux/amd64` and `linux/arm64`; set `IMAGE` or `PLATFORMS` to override their defaults.

See [CHANGELOG.md](CHANGELOG.md) for release notes and [0.2.0.md](0.2.0.md) for the 0.2.0 design and implementation checklist.
