// Package transport defines the HTTPDoer injection boundary.
//
// The host supplies a valid implementation of Do(*http.Request)
// (*http.Response, error), owns its connection pool and lifecycle, and chooses
// timeouts, redirects and any transport policy. SDK constructors do not create
// a default HTTP client. A typed-nil implementation violates the caller contract.
//
// SDK protocol clients close response bodies, but do not close shared transports.
// The host must not hide replay of transaction submissions in a wrapper.
// Concurrency safety depends on the injected implementation.
package transport
