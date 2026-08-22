package httpendpoint

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
)

// ProxyObserver records bounded metrics for an upstream request.
type ProxyObserver interface {
	UpstreamRequest(endpoint string, duration time.Duration)
	UpstreamError(endpoint string)
}

// Proxy creates a reverse proxy for one configured public path and its subpaths.
func Proxy(endpointPath string, backend config.BackendConfig, observer ProxyObserver) (http.Handler, error) {
	target, err := url.Parse(backend.URL)
	if err != nil {
		return nil, fmt.Errorf("parse backend URL: %w", err)
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default HTTP transport has unexpected type %T", http.DefaultTransport)
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			publicPath := request.In.URL.Path
			publicRawPath := request.In.URL.EscapedPath()
			request.SetURL(target)
			request.Out.URL.Path, request.Out.URL.RawPath = joinTargetPath(target, endpointPath, publicPath, publicRawPath)
			request.Out.URL.RawQuery = joinQuery(target.RawQuery, request.In.URL.RawQuery)
			if backend.PreserveHost {
				request.Out.Host = request.In.Host
			}
			request.SetXForwarded()
		},
		Transport: &observedTransport{
			transport: transport.Clone(),
			timeout:   backend.Timeout,
			endpoint:  endpointPath,
			observer:  observer,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("proxy upstream request failed", "endpoint", endpointPath, "error", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		},
	}

	return proxy, nil
}

type observedTransport struct {
	transport http.RoundTripper
	timeout   time.Duration
	endpoint  string
	observer  ProxyObserver
}

func (transport *observedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	start := time.Now()
	contextWithTimeout, cancel := context.WithTimeout(request.Context(), transport.timeout)
	response, err := transport.transport.RoundTrip(request.WithContext(contextWithTimeout))
	duration := time.Since(start)
	if transport.observer != nil {
		transport.observer.UpstreamRequest(transport.endpoint, duration)
	}
	if err != nil {
		cancel()
		if transport.observer != nil {
			transport.observer.UpstreamError(transport.endpoint)
		}
		return nil, err
	}

	response.Body = &cancelOnClose{ReadCloser: response.Body, cancel: cancel}
	return response, nil
}

type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
	once   sync.Once
}

func (body *cancelOnClose) Close() error {
	err := body.ReadCloser.Close()
	body.once.Do(body.cancel)
	return err
}

func joinTargetPath(target *url.URL, endpointPath, requestPath, escapedRequestPath string) (string, string) {
	suffix := strings.TrimPrefix(requestPath, endpointPath)
	escapedSuffix := strings.TrimPrefix(escapedRequestPath, endpointPath)
	joinedPath := joinPath(target.Path, suffix)
	joinedEscapedPath := joinPath(target.EscapedPath(), escapedSuffix)

	if joinedEscapedPath == (&url.URL{Path: joinedPath}).EscapedPath() {
		return joinedPath, ""
	}
	return joinedPath, joinedEscapedPath
}

func joinPath(base, suffix string) string {
	if base == "" || base == "/" {
		if suffix == "" {
			return base
		}
		return suffix
	}
	if suffix == "" {
		return base
	}
	if suffix == "/" {
		return strings.TrimRight(base, "/") + "/"
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(suffix, "/")
}

func joinQuery(targetQuery, requestQuery string) string {
	if targetQuery == "" || requestQuery == "" {
		return targetQuery + requestQuery
	}
	return targetQuery + "&" + requestQuery
}
