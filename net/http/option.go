package http

import (
	"crypto/tls"
	"time"
)

// Option allows to configure unexported handler fields
type Option func(*Handler)

// OptTLS changes the handler TLS configuration.
func OptTLS(config *tls.Config) Option {
	return func(h *Handler) {
		h.server.TLSConfig = config
	}
}

// OptReadTimeout configures the maximum duration for reading the entire
// request, including the body.
func OptReadTimeout(d time.Duration) Option {
	return func(h *Handler) {
		h.server.ReadHeaderTimeout = d
	}
}

// OptReadHeaderTimeout configures the amount of time allowed to read
// request headers
func OptReadHeaderTimeout(d time.Duration) Option {
	return func(h *Handler) {
		h.server.ReadHeaderTimeout = d
	}
}

// OptWriteTimeout configures the maximum duration before timing out
// writes of the response
func OptWriteTimeout(d time.Duration) Option {
	return func(h *Handler) {
		h.server.WriteTimeout = d
	}
}

// OptIdleTimeout configures the maximum amount of time to wait for the
// next request when keep-alives are enabled.
func OptIdleTimeout(d time.Duration) Option {
	return func(h *Handler) {
		h.server.IdleTimeout = d
	}
}
