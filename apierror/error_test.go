package apierror

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Stone-afk/chain-api-go/transport"
)

// Compile-time contracts: the standard library client needs no adapter, and
// Error implements error and Unwrap with a value method set.
var _ transport.HTTPDoer = (*http.Client)(nil)
var _ error = Error{}
var _ interface{ Unwrap() error } = Error{}
var _ func(Kind) string = Kind.ToString
var _ func(Error, error) Error = Error.WithContextCause

func TestKindToString(t *testing.T) {
	for _, tc := range []struct {
		kind Kind
		want string
	}{
		{Configuration, "configuration"}, {InvalidArgument, "invalid_argument"},
		{Transport, "transport"}, {HTTP, "http"}, {Protocol, "protocol"},
		{RPC, "rpc"}, {Unsupported, "unsupported"}, {NetworkMismatch, "network_mismatch"},
		{Kind(""), ""}, {Kind("custom"), "custom"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.kind.ToString(); got != tc.want {
				t.Errorf("ToString() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestErrorContextSentinels(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cause error
	}{
		{"none", nil}, {"canceled", context.Canceled},
		{"deadline", context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Error{Kind: Transport, cause: tc.cause}
			if got := e.Unwrap(); got != tc.cause {
				t.Errorf("Unwrap() = %v, want exact sentinel %v", got, tc.cause)
			}
			wrapped := fmt.Errorf("operation failed: %w", e)
			for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
				if got, want := errors.Is(wrapped, sentinel), tc.cause == sentinel; got != want {
					t.Errorf("errors.Is(_, %v) = %v, want %v", sentinel, got, want)
				}
			}
		})
	}
}

func TestErrorValueAsPreservesCodes(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		has  bool
	}{
		{"absent", 0, false}, {"explicit_zero", 0, true},
		{"negative", -32601, true}, {"positive", 42, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := Error{Kind: RPC, Method: "test_method", HTTPStatus: 400,
				RPCCode: tc.code, HasRPCCode: tc.has, NodeCode: "REJECTED", SubmissionUnknown: true}
			var got Error
			if !errors.As(fmt.Errorf("operation failed: %w", want), &got) {
				t.Fatal("errors.As did not extract Error value")
			}
			if got != want {
				t.Errorf("extracted error = %#v, want %#v", got, want)
			}
		})
	}
}

func TestErrorSafeFieldSet(t *testing.T) {
	// An exact allowlist also rejects sensitive fields under new names and
	// pointer substitutions that lose the approved value semantics.
	want := map[string]reflect.Type{
		"Kind": reflect.TypeOf(Kind("")), "Method": reflect.TypeOf(""),
		"HTTPStatus": reflect.TypeOf(0), "RPCCode": reflect.TypeOf(0),
		"HasRPCCode": reflect.TypeOf(false), "NodeCode": reflect.TypeOf(""),
		"SubmissionUnknown": reflect.TypeOf(false),
		"cause":             reflect.TypeOf((*error)(nil)).Elem(),
	}
	typ := reflect.TypeOf(Error{})
	if typ.NumField() != len(want) {
		t.Errorf("Error has %d fields, want %d safe fields", typ.NumField(), len(want))
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if expected, ok := want[field.Name]; !ok || field.Type != expected {
			t.Errorf("unapproved field or type: %s %v", field.Name, field.Type)
		}
		if field.Name == "cause" && field.IsExported() {
			t.Error("cause must remain private")
		}
	}
}

func TestErrorTextDoesNotExposeCause(t *testing.T) {
	// Deliberately malformed internal state challenges formatting only. This
	// synthetic marker is not a real credential or a supported stored cause.
	const marker = "sensitive-test-marker"
	want := (Error{Kind: Transport}).Error()
	for _, cause := range []error{nil, context.Canceled, context.DeadlineExceeded, errors.New(marker)} {
		e := Error{Kind: Transport, cause: cause}
		for _, text := range []string{e.Error(), fmt.Sprint(e), fmt.Sprintf("%v", e)} {
			if text != want {
				t.Error("error formatting changed when only the private cause changed")
			}
			if strings.Contains(text, marker) {
				t.Error("error formatting exposed the underlying cause text")
			}
		}
	}
}

func TestErrorWithContextCause(t *testing.T) {
	const marker = "sensitive-context-test-marker"
	for _, tc := range []struct {
		name        string
		input, want error
	}{
		{"nil", nil, nil},
		{"canceled", context.Canceled, context.Canceled},
		{"deadline", context.DeadlineExceeded, context.DeadlineExceeded},
		{"wrapped_canceled", fmt.Errorf(marker+": %w", context.Canceled), context.Canceled},
		{"wrapped_deadline", fmt.Errorf(marker+": %w", context.DeadlineExceeded), context.DeadlineExceeded},
		{"both", errors.Join(context.DeadlineExceeded, context.Canceled), context.Canceled},
		{"sensitive", errors.New(marker), nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, previous := range []error{nil, context.Canceled, context.DeadlineExceeded} {
				original := Error{Kind: Transport, Method: "test_method", HTTPStatus: 503,
					RPCCode: -32603, HasRPCCode: true, NodeCode: "REJECTED", SubmissionUnknown: true, cause: previous}
				before := original
				got := original.WithContextCause(tc.input)
				if original != before {
					t.Error("WithContextCause changed the original value")
				}
				want := before
				want.cause = tc.want
				if got != want {
					t.Error("WithContextCause did not preserve safe fields and normalize cause")
				}
				if got.Unwrap() != tc.want {
					t.Error("Unwrap did not return the exact expected sentinel or nil")
				}
				for _, sentinel := range []error{context.Canceled, context.DeadlineExceeded} {
					if errors.Is(got, sentinel) != (tc.want == sentinel) {
						t.Errorf("errors.Is(_, %v) did not match the normalized cause", sentinel)
					}
				}
				if strings.Contains(got.Error(), marker) {
					t.Error("WithContextCause retained sensitive error text")
				}
			}
		})
	}
}
