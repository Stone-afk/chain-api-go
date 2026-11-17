package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/transport"
)

type Version string

const (
	V2      Version = "2.0"
	Bitcoin Version = "1.0"
)

func (v Version) ToString() string { return string(v) }

type Config struct {
	Endpoint         string
	Headers          http.Header
	MaxResponseBytes int64
	Version          Version
}

type Caller interface {
	Call(context.Context, string, any, any) error
}

// Client shares immutable configuration and a concurrent request counter.
// The caller owns the injected transport and its concurrency guarantees.
type Client struct {
	cfg  Config
	doer transport.HTTPDoer
	id   atomic.Uint64
}

// New validates and copies configuration without performing I/O.
func New(cfg Config, doer transport.HTTPDoer) (*Client, error) {
	invalid := apierror.Error{Kind: apierror.Configuration}
	if isNil(doer) || cfg.MaxResponseBytes <= 0 || (cfg.Version != V2 && cfg.Version != Bitcoin) {
		return nil, invalid
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Fragment != "" {
		return nil, invalid
	}
	for name, values := range cfg.Headers {
		if !headerNameValid(name) {
			return nil, invalid
		}
		for _, value := range values {
			for i := 0; i < len(value); i++ {
				if (value[i] < 32 && value[i] != '\t') || value[i] == 127 {
					return nil, invalid
				}
			}
		}
	}
	cfg.Headers = cfg.Headers.Clone()
	return &Client{cfg: cfg, doer: doer}, nil
}

// Call sends one JSON-RPC request and decodes its result into a non-nil pointer.
func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	return c.RawCall(ctx, method, params, result)
}

// RawCall uses the same validation and resource limits as Call. It does not
// infer method semantics or retry; redirect policy belongs to the injected Doer.
func (c *Client) RawCall(ctx context.Context, method string, params any, result any) error {
	e := apierror.Error{Kind: apierror.InvalidArgument, Method: method}
	rv := reflect.ValueOf(result)
	if isNil(ctx) || strings.TrimSpace(method) == "" || !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() || !rv.Elem().CanSet() {
		return e
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		e.Cause = err
		return e
	}
	e.Kind = apierror.Transport
	if err := ctx.Err(); err != nil {
		return e.WithContextCause(err)
	}
	id := c.id.Add(1)
	payload, err := json.Marshal(request{Version: c.cfg.Version.ToString(), ID: id, Method: method, Params: encoded})
	if err != nil {
		e.Kind = apierror.InvalidArgument
		e.Cause = err
		return e
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint, bytes.NewReader(payload))
	if err != nil {
		e.Kind = apierror.Configuration
		e.Cause = err
		return e
	}
	req.Header = c.cfg.Headers.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.doer.Do(req)
	if resp != nil {
		e.HTTPStatus = resp.StatusCode
		if !isNil(resp.Body) {
			defer resp.Body.Close()
		}
	}
	if err != nil {
		e.Cause = err
		return e
	}
	if resp == nil {
		return e
	}
	e.Kind = apierror.Protocol
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e.Kind = apierror.HTTP
	}
	if isNil(resp.Body) {
		return e
	}
	// Read at most the limit plus one byte, without overflowing an int64 limit.
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxResponseBytes))
	e.ResponseBody = body
	if err != nil {
		e.Kind = apierror.Transport
		e.Cause = err
		return e
	}
	if int64(len(body)) == c.cfg.MaxResponseBytes {
		extra, readErr := io.ReadAll(io.LimitReader(resp.Body, 1))
		e.ResponseBodyTruncated = len(extra) != 0
		if readErr != nil {
			e.Kind = apierror.Transport
			e.Cause = readErr
			return e
		}
		if len(extra) != 0 {
			return e
		}
	}
	if err := ctx.Err(); err != nil {
		e.Kind = apierror.Transport
		return e.WithContextCause(err)
	}
	decoded, nodeError, valid := parseResponse(body, c.cfg.Version, id)
	e.Cause = nodeError.Cause
	if !valid {
		return e
	}
	if nodeError.HasRPCCode {
		e.Kind = apierror.RPC
		e.RPCCode = nodeError.RPCCode
		e.HasRPCCode = true
		e.Message = nodeError.Message
		e.Data = nodeError.Data
		return e
	}
	if e.Kind == apierror.HTTP {
		return e
	}
	temporary := reflect.New(rv.Elem().Type())
	if err := json.Unmarshal(decoded, temporary.Interface()); err != nil {
		e.Cause = err
		return e
	}
	rv.Elem().Set(temporary.Elem())
	return nil
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func headerNameValid(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(ch)) {
			continue
		}
		return false
	}
	return true
}
