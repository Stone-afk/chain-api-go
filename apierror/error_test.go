package apierror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
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

type syntheticTypedError struct{ marker string }

func (e syntheticTypedError) Error() string { return e.marker }

func TestErrorUnwrapPreservesOriginalCauseAndChains(t *testing.T) {
	sentinel := errors.New("synthetic-sentinel")
	typed := syntheticTypedError{"synthetic-typed"}
	for _, tc := range []struct {
		name    string
		cause   error
		targets []error
		typed   bool
	}{
		{"none", nil, nil, false},
		{"ordinary", sentinel, []error{sentinel}, false},
		{"wrapped", fmt.Errorf("synthetic-wrapper: %w", sentinel), []error{sentinel}, false},
		{"typed", typed, nil, true},
		{"wrapped_typed", fmt.Errorf("synthetic-wrapper: %w", typed), nil, true},
		{"canceled", context.Canceled, []error{context.Canceled}, false},
		{"deadline", context.DeadlineExceeded, []error{context.DeadlineExceeded}, false},
		{"wrapped_context", fmt.Errorf("synthetic-wrapper: %w", context.Canceled), []error{context.Canceled}, false},
		{"joined_context", errors.Join(fmt.Errorf("synthetic-wrapper: %w", context.DeadlineExceeded), context.Canceled, sentinel, typed), []error{context.DeadlineExceeded, context.Canceled, sentinel}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := Error{Kind: Transport, Message: "synthetic-message", Cause: tc.cause}
			if got := e.Unwrap(); got != tc.cause {
				t.Errorf("Unwrap() = %v, want exact original Cause %v", got, tc.cause)
			}
			wrapped := fmt.Errorf("operation failed: %w", e)
			for _, target := range tc.targets {
				if !errors.Is(wrapped, target) {
					t.Errorf("errors.Is cannot reach %v", target)
				}
			}
			if tc.typed {
				var got syntheticTypedError
				if !errors.As(wrapped, &got) || got != typed {
					t.Error("errors.As did not preserve typed Cause")
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
				RPCCode: tc.code, HasRPCCode: tc.has, NodeCode: "synthetic-code", SubmissionUnknown: true,
				Message: " synthetic-message ", Data: json.RawMessage(`{"n":9007199254740993}`),
				ResponseBody: []byte("synthetic-body"), ResponseBodyTruncated: true, Cause: errors.New("synthetic-cause")}
			var got Error
			if !errors.As(fmt.Errorf("operation failed: %w", want), &got) {
				t.Fatal("errors.As did not extract Error value")
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("extracted error = %#v, want %#v", got, want)
			}
		})
	}
}

func TestErrorApprovedPublicFieldTypes(t *testing.T) {
	want := []struct {
		name string
		typ  reflect.Type
	}{
		{"Kind", reflect.TypeOf(Kind(""))},
		{"Method", reflect.TypeOf("")},
		{"HTTPStatus", reflect.TypeOf(0)},
		{"RPCCode", reflect.TypeOf(0)},
		{"HasRPCCode", reflect.TypeOf(false)},
		{"NodeCode", reflect.TypeOf("")},
		{"Message", reflect.TypeOf("")},
		{"Data", reflect.TypeOf(json.RawMessage(nil))},
		{"ResponseBody", reflect.TypeOf([]byte(nil))},
		{"ResponseBodyTruncated", reflect.TypeOf(false)},
		{"SubmissionUnknown", reflect.TypeOf(false)},
		{"Cause", reflect.TypeOf((*error)(nil)).Elem()},
	}
	typ := reflect.TypeOf(Error{})
	if typ.NumField() != len(want) {
		t.Errorf("Error has %d fields, want %d approved public fields", typ.NumField(), len(want))
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if i >= len(want) {
			t.Errorf("unexpected field at index %d: %s %v", i, field.Name, field.Type)
			continue
		}
		if expected := want[i]; field.Name != expected.name || field.Type != expected.typ || !field.IsExported() {
			t.Errorf("field at index %d = %s %v (exported=%t), want %s %v (exported=true)", i, field.Name, field.Type, field.IsExported(), expected.name, expected.typ)
		}
	}
}

func TestErrorTextSelectsMessageBeforeCause(t *testing.T) {
	cause := errors.New("synthetic-cause")
	for _, tc := range []struct {
		name  string
		input Error
		want  string
	}{
		{"kind_only", Error{Kind: RPC}, "rpc"},
		{"message", Error{Kind: RPC, Message: " synthetic-message "}, "rpc:  synthetic-message "},
		{"cause", Error{Kind: Transport, Cause: cause}, "transport: synthetic-cause"},
		{"message_wins", Error{Kind: RPC, Message: "synthetic-message", Cause: cause}, "rpc: synthetic-message"},
		{"empty", Error{}, ""},
		{"empty_kind_message", Error{Message: "synthetic-message", Cause: cause}, "synthetic-message"},
		{"empty_kind_cause", Error{Cause: cause}, "synthetic-cause"},
		{"structured_only", Error{Kind: HTTP, Data: json.RawMessage(`"synthetic-data"`), ResponseBody: []byte("synthetic-body")}, "http"},
		{"structured_with_message", Error{Kind: RPC, Message: "synthetic-message", Data: json.RawMessage(`null`), ResponseBody: []byte("synthetic-body")}, "rpc: synthetic-message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.input.Error(); got != tc.want {
				t.Errorf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestErrorWithContextCauseReturnsUnchangedInputInValueCopy(t *testing.T) {
	const marker = "synthetic-context-marker"
	for _, tc := range []struct {
		name  string
		input error
	}{
		{"nil", nil},
		{"canceled", context.Canceled},
		{"deadline", context.DeadlineExceeded},
		{"wrapped_canceled", fmt.Errorf(marker+": %w", context.Canceled)},
		{"wrapped_deadline", fmt.Errorf(marker+": %w", context.DeadlineExceeded)},
		{"both", errors.Join(context.DeadlineExceeded, context.Canceled)},
		{"ordinary", errors.New(marker)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, previous := range []error{nil, errors.New("synthetic-previous")} {
				original := Error{Kind: Transport, Method: "test_method", HTTPStatus: 503,
					RPCCode: 0, HasRPCCode: true, NodeCode: "synthetic-code", SubmissionUnknown: true, Cause: previous,
					Message: "synthetic-message", Data: json.RawMessage(`{"n":9007199254740993}`),
					ResponseBody: []byte("synthetic-body"), ResponseBodyTruncated: true}
				before := original
				before.Data = append(json.RawMessage(nil), original.Data...)
				before.ResponseBody = append([]byte(nil), original.ResponseBody...)
				got := original.WithContextCause(tc.input)
				if !reflect.DeepEqual(original, before) || original.Cause != previous {
					t.Error("WithContextCause changed the original value")
				}
				want := before
				want.Cause = tc.input
				if !reflect.DeepEqual(got, want) {
					t.Error("WithContextCause changed fields other than Cause or failed to set Cause")
				}
				if got.Cause != tc.input {
					t.Error("WithContextCause did not preserve the exact input Cause")
				}
			}
		})
	}
}
