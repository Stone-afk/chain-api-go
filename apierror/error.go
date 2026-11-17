package apierror

import "encoding/json"

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
	Kind                  Kind
	Method                string
	HTTPStatus            int
	RPCCode               int
	HasRPCCode            bool
	NodeCode              string
	Message               string
	Data                  json.RawMessage
	ResponseBody          []byte
	ResponseBodyTruncated bool
	SubmissionUnknown     bool
	Cause                 error
}

// ToString returns the underlying error category.
func (k Kind) ToString() string { return string(k) }

// Error reports the category and original Message, or Cause text when Message is empty.
// Data and ResponseBody remain available through the value's fields.
func (e Error) Error() string {
	base := e.Kind.ToString()
	detail := e.Message
	if detail == "" && e.Cause != nil {
		detail = e.Cause.Error()
	}
	if base == "" {
		return detail
	}
	if detail == "" {
		return base
	}
	return base + ": " + detail
}

// Unwrap returns the original Cause.
func (e Error) Unwrap() error { return e.Cause }

// WithContextCause returns a copy with Cause set to err unchanged; nil clears Cause.
func (e Error) WithContextCause(err error) Error {
	e.Cause = err
	return e
}
