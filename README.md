# fake-web-service

[![License: MIT](https://img.shields.io/badge/License-MIT%202.0-blue.svg)](https://github.com/sergkondr/fake-web-service/blob/main/LICENSE)
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

[Here](./examples/config.yaml) you can find a config file I use for developer purposes. I suppose it the most detailed config possible.

For now, config implements the following options:
```yaml
listen: 127.0.0.1:8080  # optional, default value = 0.0.0.0:8080

ws_endpoints:                     # only 1 ws endpoint is supported now
  - name: echo                    # optional
    description: WebSocket echo   # optional
    path: /echo                   # required, but will be rewrited to /ws/{{ path }} 
    type: echo                    # required, only "echo" is supported now

http_endpoints:
  - name: Some endpoint             # optional, used in endpoint list on /
    description: Simple description # optional, used in endpoint list on /
    path: /path                     # required
    error_rate: 0.0                 # optional, in range [0.0, 1.0]
    hidden: true                    # optional, do not display on request to /
    do_not_log: true                # optional, do not write access logs
    slowness:                       # required
      min: 10ms                     # required, time duration, should be less than p95
      p95: 50ms                     # required, time duration, should be less than max
      max: 100ms                    # required, time duration
```
