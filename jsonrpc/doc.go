// Package jsonrpc implements injected JSON-RPC 2.0 and Bitcoin dialect clients.
//
// New(Config, transport.HTTPDoer) validates and copies configuration locally.
// Config requires an explicit HTTP(S) Endpoint, positive MaxResponseBytes and
// Version (V2 or Bitcoin); Headers are copied. Version.ToString exposes the wire
// value. The host owns timeouts, redirect policy and transport lifecycle.
//
// Caller.Call(context.Context, string, any, any) error is the chain injection
// boundary. Client.Call and Client.RawCall each perform at most one Do call,
// validate the response envelope and matching ID, limit and close its body, and
// decode atomically into a writable non-nil result pointer. Request IDs are
// concurrency safe. Pass a valid context and a concurrent-safe Doer when sharing.
//
// RawCall is a protocol escape hatch, not a chain dispatcher. It does not infer
// submission semantics, retry safety or SubmissionUnknown. Use json.RawMessage
// to retain the raw result. Node errors and bounded failure bodies are preserved
// in apierror.Error; Error() must not be treated as redacted logging output.
package jsonrpc
