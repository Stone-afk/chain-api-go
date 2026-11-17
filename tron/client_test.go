package tron

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

const hexAddress Address = "41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const base58Address Address = "T111111111111111111111111111111111"
const txID TxID = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

type fakePoster struct {
	response string
	err      error
	calls    int
	ctx      context.Context
	path     string
	request  any
	body     []byte
}

func (p *fakePoster) Post(ctx context.Context, path string, request, result any) error {
	p.calls++
	p.ctx, p.path, p.request = ctx, path, request
	var err error
	p.body, err = json.Marshal(request)
	if err != nil {
		return err
	}
	if p.err != nil {
		return p.err
	}
	return json.Unmarshal([]byte(p.response), result)
}
func newClient(t *testing.T, p *fakePoster) *Client {
	t.Helper()
	c, err := New(Config{}, p)
	if err != nil || c == nil {
		t.Fatalf("constructor failed")
	}
	return c
}
func checkJSON(t *testing.T, got []byte, want string) {
	t.Helper()
	var a, b any
	// UseNumber compares wire values without rounding integers through float64.
	decoder := json.NewDecoder(bytes.NewReader(got))
	decoder.UseNumber()
	if err := decoder.Decode(&a); err != nil {
		t.Errorf("invalid request/result JSON: %s", got)
		return
	}
	decoder = json.NewDecoder(bytes.NewReader([]byte(want)))
	decoder.UseNumber()
	if err := decoder.Decode(&b); err != nil {
		t.Fatalf("bad fixture")
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("JSON mismatch: got %s want %s", got, want)
	}
}
func checkWire(t *testing.T, p *fakePoster, ctx context.Context, path, body string) {
	t.Helper()
	if p.calls != 1 {
		t.Errorf("Poster calls = %d, want 1", p.calls)
	}
	if p.ctx != ctx {
		t.Error("context identity lost")
	}
	if p.path != path {
		t.Errorf("path = %q, want %q", p.path, path)
	}
	checkJSON(t, p.body, body)
}
func checkError(t *testing.T, err error, kind apierror.Kind, unknown bool) apierror.Error {
	t.Helper()
	var e apierror.Error
	if !errors.As(err, &e) {
		t.Errorf("missing apierror value, want %s", kind)
		return e
	}
	if e.Kind != kind {
		t.Errorf("kind = %s, want %s", e.Kind, kind)
	}
	if e.SubmissionUnknown != unknown {
		t.Errorf("unknown = %v, want %v", e.SubmissionUnknown, unknown)
	}
	return e
}
func TestConstructor(t *testing.T) {
	c, err := New(Config{}, nil)
	if c != nil {
		t.Error("nil Poster accepted")
	}
	checkError(t, err, apierror.Configuration, false)
	p := &fakePoster{}
	newClient(t, p)
	if p.calls != 0 {
		t.Error("constructor performed I/O")
	}
}

type markerError struct{}

func (markerError) Error() string { return "synthetic dependency marker" }
func TestPosterErrorPreservation(t *testing.T) {
	sentinel := errors.New("synthetic sentinel")
	joined := errors.Join(context.Canceled, context.DeadlineExceeded, sentinel, markerError{})
	original := apierror.Error{Kind: apierror.HTTP, HTTPStatus: 503, RPCCode: 0, HasRPCCode: true, NodeCode: "raw-code", Message: "  synthetic message  ", Data: json.RawMessage(`{"n":9007199254740993}`), ResponseBody: []byte("synthetic partial body"), ResponseBodyTruncated: true, Cause: joined}
	for _, tc := range []struct {
		name  string
		cause error
		kind  apierror.Kind
	}{
		{"wrapped structured", fmt.Errorf("outer: %w", original), apierror.HTTP},
		{"ordinary", sentinel, apierror.Transport},
		{"joined", joined, apierror.Transport},
		{"typed", markerError{}, apierror.Transport},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			p := &fakePoster{err: tc.cause}
			_, err := newClient(t, p).GetNodeInfo(ctx)
			e := checkError(t, err, tc.kind, false)
			if e.Cause != tc.cause {
				t.Error("original dependency error identity lost")
			}
			if e.Method != "/wallet/getnodeinfo" {
				t.Error("actual method missing")
			}
			if !errors.Is(err, tc.cause) {
				t.Error("dependency error not reachable")
			}
			if tc.name == "wrapped structured" {
				if e.HTTPStatus != 503 || !e.HasRPCCode || e.RPCCode != 0 || e.NodeCode != original.NodeCode || e.Message != original.Message || string(e.Data) != string(original.Data) || string(e.ResponseBody) != string(original.ResponseBody) || !e.ResponseBodyTruncated {
					t.Error("structured fields lost")
				}
			}
			if tc.name == "wrapped structured" || tc.name == "joined" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, sentinel) {
					t.Error("joined branches lost")
				}
				var typed markerError
				if !errors.As(err, &typed) {
					t.Error("typed error lost")
				}
			}
			if tc.name == "ordinary" && errors.Is(err, context.Canceled) {
				t.Error("context replaced actual error")
			}
			checkWire(t, p, ctx, "/wallet/getnodeinfo", `{}`)
		})
	}
}
