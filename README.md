# fake-web-service

[![License: MIT](https://img.shields.io/badge/License-MIT%202.0-blue.svg)](https://github.com/sergkondr/fake-web-service/blob/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/sergkondr/fake-web-service)](https://goreportcard.com/report/github.com/sergkondr/fake-web-service)

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
