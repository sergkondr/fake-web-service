# fakesvc

[![License: MIT](https://img.shields.io/badge/License-MIT%202.0-blue.svg)](https://github.com/sergkondr/fake-web-service/blob/main/LICENSE)
[//]: # ([![GitHub release]&#40;https://img.shields.io/github/release/sergkondr/fake-web-service.svg&#41;]&#40;https://github.com/sergkondr/fake-web-service/releases/latest&#41;)
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
➜ curl fakesvc.my.lab/
Available endpoints:
- /good - Good endpoint: Fast enough, no errors at all
- /bad - Bad endpoint: 30% of requests fails with 500 error
- /slow - Slow endpoint: Sometimes it fails, but it is always slow

➜ curl fakesvc.my.lab/slow -v
* Host fakesvc.my.lab:80 was resolved.
* IPv6: (none)
* IPv4: 127.0.0.1
*   Trying 127.0.0.1:80...
* Connected to fakesvc.my.lab (127.0.0.1) port 80
> GET /slow HTTP/1.1
> Host: fakesvc.my.lab
> User-Agent: curl/8.6.0
> Accept: */*
>
< HTTP/1.1 500 Internal Server Error
< Date: Sat, 27 Jul 2024 19:33:11 GMT
< Content-Type: text/plain; charset=utf-8
< Content-Length: 27
< Connection: keep-alive
<
* Connection #0 to host fakesvc.my.lab left intact
sorry, something went wrong

```
