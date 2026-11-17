// Package apierror preserves node errors and underlying causes as Error values.
//
// Use errors.As with a variable of type Error, and errors.Is/As to traverse Cause.
// Unwrap returns the original Cause. WithContextCause returns a value copy with
// Cause replaced verbatim, including nil; it does not filter context errors.
// Kind.ToString returns the underlying category. HasRPCCode distinguishes an
// absent numeric code from zero. NodeCode retains native string codes.
//
// Message is the decoded node string; Data preserves raw JSON, including absent
// (nil) versus explicit null. ResponseBody holds bytes already read on failure,
// limited by protocol MaxResponseBytes; ResponseBodyTruncated flags detected
// excess bytes. A partial read is not a complete response.
//
// Error() is NOT a safe logging view: it prefers Message, then Cause text.
// Data and body are not automatically concatenated, but may contain secrets.
// The SDK does not append configuration or payloads; it also does not remove
// sensitive data already supplied by a node or underlying error. The caller owns
// logging redaction, business mapping and retry policy. Error is not comparable.
//
// SubmissionUnknown means a typed submission cannot establish its result after
// entering the dependency. It is not proof of rejection or permission to resend.
package apierror
