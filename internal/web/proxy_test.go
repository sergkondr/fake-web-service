package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sergkondr/fake-web-service/internal/config"
)

func TestProxyForwardsRequestAndResponse(t *testing.T) {
	var received struct {
		method string
		path   string
		query  string
		body   string
		host   string
		header string
		xff    string
	}
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read backend request body: %v", err)
		}
		received.method = r.Method
		received.path = r.URL.Path
		received.query = r.URL.RawQuery
		received.body = string(body)
		received.host = r.Host
		received.header = r.Header.Get("X-Test")
		received.xff = r.Header.Get("X-Forwarded-For")
		w.Header().Set("X-Backend", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("proxied"))
	}))
	defer backend.Close()

	router := newProxyRouter(t, "/service", backend.URL+"/api/v1?fixed=yes", false, config.Chaos{})
	request := httptest.NewRequest(http.MethodPost, "/service/users?id=42", strings.NewReader("request body"))
	request.Header.Set("X-Test", "forwarded")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if body := recorder.Body.String(); body != "proxied" {
		t.Errorf("response body = %q, want %q", body, "proxied")
	}
	if header := recorder.Header().Get("X-Backend"); header != "yes" {
		t.Errorf("X-Backend = %q, want %q", header, "yes")
	}
	if received.method != http.MethodPost || received.path != "/api/v1/users" || received.query != "fixed=yes&id=42" || received.body != "request body" {
		t.Errorf("backend request = %#v, want POST /api/v1/users?fixed=yes&id=42 with request body", received)
	}
	if received.header != "forwarded" || received.xff == "" {
		t.Errorf("backend headers X-Test=%q X-Forwarded-For=%q, want forwarded values", received.header, received.xff)
	}
	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("parse backend URL: %v", err)
	}
	if received.host != backendURL.Host {
		t.Errorf("backend Host = %q, want %q", received.host, backendURL.Host)
	}
}

func TestProxySupportsExactPathAndHTTPMethods(t *testing.T) {
	var methods []string
	var paths []string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	router := newProxyRouter(t, "/service", backend.URL+"/api", false, config.Chaos{})
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, "/service", nil))
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d, want %d", method, recorder.Code, http.StatusNoContent)
		}
	}

	if got := strings.Join(methods, ","); got != "GET,POST,PUT,DELETE" {
		t.Errorf("methods = %q", got)
	}
	if got := strings.Join(paths, ","); got != "/api,/api,/api,/api" {
		t.Errorf("paths = %q", got)
	}
}

func TestProxyPreservesNestedTrailingSlash(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	router := newProxyRouter(t, "/service", backend.URL+"/api", false, config.Chaos{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service/", nil))
	if recorder.Code != http.StatusNoContent || gotPath != "/api/" {
		t.Fatalf("status=%d upstream path=%q, want 204 and /api/", recorder.Code, gotPath)
	}
}

func TestProxyMetricsUseConfiguredEndpoint(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	router := newTestRouter(t, config.Config{
		Metrics: config.Metrics{Enabled: true, Path: "/metrics"},
		Endpoints: []config.Endpoint{{
			Type:    config.EndpointTypeProxy,
			Path:    "/service",
			Backend: &config.BackendConfig{URL: backend.URL, Timeout: time.Second},
		}},
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service/users/123", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("proxy status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	metrics := httptest.NewRecorder()
	router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metrics.Body.String()
	for _, want := range []string{
		`fakesvc_http_requests_total{endpoint="/service",method="GET",status_code="204",type="proxy"} 1`,
		`fakesvc_proxy_upstream_duration_seconds_count{endpoint="/service"} 1`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics do not contain %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "/service/users/123") {
		t.Errorf("metrics contain raw proxy subpath:\n%s", body)
	}
}

func TestProxyPreserveHost(t *testing.T) {
	for _, preserveHost := range []bool{false, true} {
		t.Run(map[bool]string{false: "backend host", true: "original host"}[preserveHost], func(t *testing.T) {
			var gotHost string
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotHost = r.Host
				w.WriteHeader(http.StatusNoContent)
			}))
			defer backend.Close()

			router := newProxyRouter(t, "/service", backend.URL, preserveHost, config.Chaos{})
			request := httptest.NewRequest(http.MethodGet, "/service", nil)
			request.Host = "original.example"
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			wantHost := "original.example"
			if !preserveHost {
				backendURL, err := url.Parse(backend.URL)
				if err != nil {
					t.Fatalf("parse backend URL: %v", err)
				}
				wantHost = backendURL.Host
			}
			if gotHost != wantHost {
				t.Errorf("Host = %q, want %q", gotHost, wantHost)
			}
		})
	}
}

func TestProxyReturnsBadGatewayForUnreachableBackend(t *testing.T) {
	router := newProxyRouter(t, "/service", "http://127.0.0.1:1", false, config.Chaos{})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service", nil))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
	}
}

func TestProxyTimeoutAndChaos(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer backend.Close()

		router := newProxyRouterWithTimeout(t, "/service", backend.URL, 20*time.Millisecond, false, config.Chaos{})
		recorder := httptest.NewRecorder()
		start := time.Now()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service", nil))
		if recorder.Code != http.StatusBadGateway {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadGateway)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("timeout took %s, want less than 1s", elapsed)
		}
	})

	t.Run("injected error does not call backend", func(t *testing.T) {
		var calls atomic.Int32
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		router := newProxyRouter(t, "/service", backend.URL, false, config.Chaos{ErrorRate: 1, ErrorStatus: http.StatusServiceUnavailable})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service", nil))
		if recorder.Code != http.StatusServiceUnavailable || calls.Load() != 0 {
			t.Errorf("status=%d backend calls=%d, want 503 and 0", recorder.Code, calls.Load())
		}
	})

	t.Run("latency runs before backend", func(t *testing.T) {
		arrived := make(chan time.Time, 1)
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			arrived <- time.Now()
			w.WriteHeader(http.StatusNoContent)
		}))
		defer backend.Close()

		delay := 30 * time.Millisecond
		router := newProxyRouter(t, "/service", backend.URL, false, config.Chaos{Latency: &config.Latency{Min: delay, P95: delay, Max: delay}})
		start := time.Now()
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/service", nil))
		if elapsed := (<-arrived).Sub(start); elapsed < delay {
			t.Errorf("backend received request after %s, want at least %s", elapsed, delay)
		}
	})
}

func TestProxyCancelsUpstreamWhenClientDisconnects(t *testing.T) {
	canceled := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		close(canceled)
	}))
	defer backend.Close()

	router := newProxyRouter(t, "/service", backend.URL, false, config.Chaos{})
	requestContext, cancel := context.WithCancel(t.Context())
	request := httptest.NewRequestWithContext(requestContext, http.MethodGet, "/service", nil)
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	router.ServeHTTP(httptest.NewRecorder(), request)

	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("upstream request was not canceled after client context cancellation")
	}
}

func newProxyRouter(t *testing.T, path, backendURL string, preserveHost bool, chaos config.Chaos) http.Handler {
	t.Helper()
	return newProxyRouterWithTimeout(t, path, backendURL, time.Second, preserveHost, chaos)
}

func newProxyRouterWithTimeout(t *testing.T, path, backendURL string, timeout time.Duration, preserveHost bool, chaos config.Chaos) http.Handler {
	t.Helper()
	return newTestRouter(t, config.Config{Endpoints: []config.Endpoint{{
		Type:  config.EndpointTypeProxy,
		Path:  path,
		Chaos: chaos,
		Backend: &config.BackendConfig{
			URL:          backendURL,
			Timeout:      timeout,
			PreserveHost: preserveHost,
		},
	}}})
}
