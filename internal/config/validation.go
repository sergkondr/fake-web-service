package config

import (
	"fmt"
	"net"
	"net/url"
	pathpkg "path"
	"strconv"
	"strings"
)

func validateConfig(cfg Config) error {
	if err := validateListenAddress(cfg.ListenAddr); err != nil {
		return fmt.Errorf("invalid listen address %q: %w", cfg.ListenAddr, err)
	}
	if len(cfg.Endpoints) == 0 {
		return fmt.Errorf("no endpoints defined in the config")
	}

	if cfg.Metrics.Enabled {
		if err := validatePublicPath(cfg.Metrics.Path); err != nil {
			return fmt.Errorf("invalid metrics path %q: %w", cfg.Metrics.Path, err)
		}
		if cfg.Metrics.Path == defaultHealthcheckPath {
			return fmt.Errorf("metrics path %q overlaps with healthcheck path", cfg.Metrics.Path)
		}
	}

	paths := make(map[string]int, len(cfg.Endpoints))
	for i, endpoint := range cfg.Endpoints {
		label := endpointLabel(i, endpoint)

		if err := validatePublicPath(endpoint.Path); err != nil {
			return fmt.Errorf("%s has invalid path %q: %w", label, endpoint.Path, err)
		}
		if endpoint.Path == defaultHealthcheckPath {
			return fmt.Errorf("%s path %q is reserved for healthcheck", label, endpoint.Path)
		}
		if cfg.Metrics.Enabled && endpoint.Path == cfg.Metrics.Path {
			return fmt.Errorf("%s path %q overlaps with metrics path", label, endpoint.Path)
		}
		if previous, ok := paths[endpoint.Path]; ok {
			return fmt.Errorf("%s duplicates endpoint[%d] path %q", label, previous, endpoint.Path)
		}
		paths[endpoint.Path] = i

		if err := validateChaos(endpoint.Chaos); err != nil {
			return fmt.Errorf("%s has invalid chaos config: %w", label, err)
		}
		if err := validateEndpointType(endpoint); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
	}

	if err := validateProxyPathOwnership(cfg.Endpoints); err != nil {
		return err
	}

	return nil
}

func validateListenAddress(address string) error {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("must use host:port format: %w", err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("port must be a number between 1 and 65535")
	}

	return nil
}

func validateEndpointType(endpoint Endpoint) error {
	switch endpoint.Type {
	case EndpointTypeHTTP:
		if endpoint.Backend != nil {
			return fmt.Errorf("type %q does not allow backend config", endpoint.Type)
		}
		if endpoint.Stream != nil {
			return fmt.Errorf("type %q does not allow stream config", endpoint.Type)
		}
		if endpoint.Response != nil {
			if err := validateResponse(*endpoint.Response); err != nil {
				return fmt.Errorf("invalid response config: %w", err)
			}
		}
	case EndpointTypeProxy:
		if endpoint.Backend == nil {
			return fmt.Errorf("type %q requires backend config", endpoint.Type)
		}
		if endpoint.Response != nil {
			return fmt.Errorf("type %q does not allow response config", endpoint.Type)
		}
		if endpoint.Stream != nil {
			return fmt.Errorf("type %q does not allow stream config", endpoint.Type)
		}
		if err := validateBackend(*endpoint.Backend); err != nil {
			return fmt.Errorf("invalid backend config: %w", err)
		}
	case EndpointTypeWSEcho:
		if endpoint.Response != nil || endpoint.Backend != nil || endpoint.Stream != nil {
			return fmt.Errorf("type %q does not allow response, backend, or stream config", endpoint.Type)
		}
	case EndpointTypeWSStream:
		if endpoint.Stream == nil {
			return fmt.Errorf("type %q requires stream config", endpoint.Type)
		}
		if endpoint.Response != nil {
			return fmt.Errorf("type %q does not allow response config", endpoint.Type)
		}
		if endpoint.Backend != nil {
			return fmt.Errorf("type %q does not allow backend config", endpoint.Type)
		}
		if err := validateStream(*endpoint.Stream); err != nil {
			return fmt.Errorf("invalid stream config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported type %q", endpoint.Type)
	}

	return nil
}

func validateChaos(chaos Chaos) error {
	if chaos.ErrorRate < 0 || chaos.ErrorRate > 1 {
		return fmt.Errorf("error_rate must be between 0.0 and 1.0 inclusive")
	}
	if chaos.ErrorStatus < 100 || chaos.ErrorStatus > 599 {
		return fmt.Errorf("error_status must be between 100 and 599")
	}
	if chaos.Latency == nil {
		return nil
	}

	latency := chaos.Latency
	if latency.Min < 0 || latency.P95 < 0 || latency.Max < 0 {
		return fmt.Errorf("latency durations cannot be negative")
	}
	if latency.Min > latency.P95 || latency.P95 > latency.Max {
		return fmt.Errorf("latency must satisfy min <= p95 <= max")
	}

	return nil
}

func validateResponse(response ResponseConfig) error {
	if response.Status < 100 || response.Status > 599 {
		return fmt.Errorf("status must be between 100 and 599")
	}
	for name, value := range response.Headers {
		if !validHeaderName(name) {
			return fmt.Errorf("invalid header name %q", name)
		}
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("header %q contains a newline", name)
		}
	}

	return nil
}

func validateBackend(backend BackendConfig) error {
	parsed, err := url.Parse(backend.URL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("url must contain a host")
	}
	if parsed.User != nil {
		return fmt.Errorf("url must not contain userinfo")
	}
	if parsed.Fragment != "" {
		return fmt.Errorf("url must not contain a fragment")
	}
	if backend.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than zero")
	}

	return nil
}

func validateStream(stream StreamConfig) error {
	if stream.Interval <= 0 {
		return fmt.Errorf("interval must be greater than zero")
	}

	return nil
}

func validatePublicPath(value string) error {
	if value == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if !strings.HasPrefix(value, "/") {
		return fmt.Errorf("path must start with /")
	}
	if value == "/" {
		return fmt.Errorf("path is reserved")
	}
	if strings.ContainsAny(value, "?#") {
		return fmt.Errorf("path must not contain query or fragment")
	}
	if strings.HasSuffix(value, "/") {
		return fmt.Errorf("path must not have a trailing slash")
	}
	if cleaned := pathpkg.Clean(value); cleaned != value {
		return fmt.Errorf("path must be normalized; use %q", cleaned)
	}

	return nil
}

func validateProxyPathOwnership(endpoints []Endpoint) error {
	for i, proxy := range endpoints {
		if proxy.Type != EndpointTypeProxy {
			continue
		}
		for j, endpoint := range endpoints {
			if i == j {
				continue
			}
			if strings.HasPrefix(endpoint.Path, proxy.Path+"/") {
				return fmt.Errorf(
					"%s path %q is owned by proxy %s path %q",
					endpointLabel(j, endpoint), endpoint.Path, endpointLabel(i, proxy), proxy.Path,
				)
			}
		}
	}

	return nil
}

func endpointLabel(index int, endpoint Endpoint) string {
	if endpoint.Name == "" {
		return fmt.Sprintf("endpoint[%d]", index)
	}
	return fmt.Sprintf("endpoint[%d] %q", index, endpoint.Name)
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	const separators = "!#$%&'*+-.^_`|~"
	for _, character := range name {
		if character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' ||
			strings.ContainsRune(separators, character) {
			continue
		}
		return false
	}

	return true
}
