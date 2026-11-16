package apierror

import (
	"context"
	"errors"
)

type Kind string

const (
	Configuration   Kind = "configuration"
	InvalidArgument Kind = "invalid_argument"
	Transport       Kind = "transport"
	HTTP            Kind = "http"
	Protocol        Kind = "protocol"
	RPC             Kind = "rpc"
	Unsupported     Kind = "unsupported"
	NetworkMismatch Kind = "network_mismatch"
)

type Error struct {
	Kind              Kind
	Method            string
	HTTPStatus        int
	RPCCode           int
	HasRPCCode        bool
	NodeCode          string
	SubmissionUnknown bool
	cause             error
}

// ToString returns the underlying error category.
func (k Kind) ToString() string { return string(k) }

// Error reports the category without formatting the private cause.
// Structured details remain available through the value's fields.
func (e Error) Error() string { return e.Kind.ToString() }

// Unwrap returns the stored context sentinel, or nil.
func (e Error) Unwrap() error { return e.cause }

// WithContextCause returns a copy with only a normalized context sentinel.
// Cancellation takes precedence; other errors clear the cause.
func (e Error) WithContextCause(err error) Error {
	e.cause = nil
	if errors.Is(err, context.Canceled) {
		e.cause = context.Canceled
		return e
	}
	if errors.Is(err, context.DeadlineExceeded) {
		e.cause = context.DeadlineExceeded
	}
	return e
}
