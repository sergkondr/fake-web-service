# fake-web-service

[![License: MIT](https://img.shields.io/badge/License-MIT%202.0-blue.svg)](https://github.com/sergkondr/fake-web-service/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/sergkondr/fake-web-service.svg)](https://github.com/sergkondr/fake-web-service/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sergkondr/fake-web-service)](https://goreportcard.com/report/github.com/sergkondr/fake-web-service)
[![Pulls](https://img.shields.io/docker/pulls/sergkondr/fakesvc.svg)](https://hub.docker.com/r/sergkondr/fakesvc)
[![Go](https://github.com/sergkondr/fake-web-service/actions/workflows/go.yml/badge.svg)](https://github.com/sergkondr/fake-web-service/actions/workflows/go.yml)

This simple web service is made for testing purposes. 
It has different endpoints that return various results, either a successful response or an error, 
with different delays. You can configure the endpoints, the delays, and the error rate for each endpoint independently.  

### Deploy

```
kubectl apply -f deployments/manifests/kubernetes-deploy.yaml
```

###### Pay attention

For development and testing purposes, I use docker images with the `dev` tag. 
But I also publish images with tag matching the release version, like `0.1.0`. You can find the full list of tags [on Docker Hub](https://hub.docker.com/r/sergkondr/fakesvc/tags)

### Usage

```shell
➜ curl localhost:8080/
Available endpoints:
- /good - Good endpoint: Fast enough, no errors at all
- /bad - Bad endpoint: 30% of requests fails with 500 error
- /slow - Slow endpoint: Sometimes it fails, but it is always slow

➜ time curl localhost:8080/good
success: /good
curl localhost:8080/good  0.00s user 0.01s system 4% cpu 0.230 total

➜ time curl localhost:8080/slow
success: /slow
curl localhost:8080/slow  0.01s user 0.01s system 0% cpu 2.822 total
```

### Configuration

[Here](./examples/config.yaml) you can find the config file that I use for development purposes. I believe it is the most detailed configuration possible.

For now, config implements the following options:
```yaml
listen: 127.0.0.1:8080  # optional, default value = 0.0.0.0:8080

endpoints:
  - name: echo                    # optional
    type: ws/echo                 # required
    description: WebSocket echo   # optional
    path: /ws/echo                # required, final public path

  - name: time                    # optional
    type: ws/stream               # required
    description: Time stream      # optional
    path: /ws/time                # required, final public path
    stream:
      interval: 1s                # required, must be greater than zero
      format: json                # optional, json is the only supported format

  - name: Some endpoint             # optional, used in endpoint list on /
    type: http                       # required
    description: Simple description # optional, used in endpoint list on /
    path: /path                     # required
    hidden: true                    # optional, do not display on request to /
    do_not_log: true                # optional, do not write access logs
    chaos:
      error_rate: 0.0               # optional, in range [0.0, 1.0]
      latency:                      # optional
        min: 10ms                   # required when latency is configured
        p95: 50ms                   # min <= p95 <= max
        max: 100ms
```

The configuration parser is strict. Legacy `http_endpoints`, `ws_endpoints`, and `slowness` fields are not supported.

`ws/echo` accepts text frames and returns a JSON text frame. Binary frames are rejected with WebSocket close code `1003`.
`ws/stream` sends a JSON message immediately after the handshake and then every configured interval. Each message contains `backend`, `endpoint`, an increasing `sequence`, and a UTC `timestamp` in RFC3339Nano format.

### Metrics

When `metrics.enabled` is set, HTTP metrics use only bounded labels: the configured `endpoint`, endpoint `type`, request `method`, and `status_code`. Request URL, query string, client address, and endpoint name are never labels.

WebSocket endpoints expose active/opened/closed connection metrics, sent/received message counters, and read/write error counters. WebSocket connections are intentionally excluded from HTTP request-duration metrics.
