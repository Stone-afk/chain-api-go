// Package httpjson implements injected HTTP JSON POST requests for node APIs.
//
// New(Config, transport.HTTPDoer) requires an explicit HTTP(S) Endpoint and
// positive MaxResponseBytes, validates locally and copies Headers. It performs
// no network I/O. The host owns timeout, redirects and the shared transport.
//
// Poster.Post(context.Context, string, any, any) error is the Tron injection
// boundary. Client.Post and Client.RawPost each make at most one Do call. Paths
// are constrained to the configured endpoint and prefix; absolute URLs and
// traversal are not accepted. Response bodies are bounded and closed; decoding
// into a writable non-nil result pointer is atomic.
//
// RawPost preserves protocol errors and raw failure bodies but does not interpret
// Tron HTTP-200 result/Error fields or infer SubmissionUnknown. Tron performs
// that chain interpretation. Error() is not safe logging output. Standard JSON
// marshalling may change insignificant whitespace or escaping in RawMessage.
package httpjson
