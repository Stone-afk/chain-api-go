package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/transport"
)

type Config struct {
	Endpoint         string
	Headers          http.Header
	MaxResponseBytes int64
}

type Poster interface {
	Post(context.Context, string, any, any) error
}

// Client shares copied immutable configuration.
// The caller owns the injected transport and its concurrency guarantees.
type Client struct {
	cfg      Config
	doer     transport.HTTPDoer
	endpoint url.URL
}

// New validates and copies configuration without performing I/O.
func New(cfg Config, doer transport.HTTPDoer) (*Client, error) {
	invalid := apierror.Error{Kind: apierror.Configuration}
	if doer == nil || cfg.MaxResponseBytes <= 0 {
		return nil, invalid
	}
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, invalid.WithContextCause(err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || strings.Contains(cfg.Endpoint, "#") {
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
	return &Client{cfg: cfg, doer: doer, endpoint: *u}, nil
}

// Post sends one JSON request and publishes a fully decoded result.
func (c *Client) Post(ctx context.Context, path string, request any, result any) error {
	return c.RawPost(ctx, path, request, result)
}

// RawPost uses the same validation and limits as Post. Redirect policy belongs
// to the injected Doer. The SDK does not retry or infer chain semantics.
func (c *Client) RawPost(ctx context.Context, path string, request any, result any) error {
	e := apierror.Error{Kind: apierror.InvalidArgument}
	if !safePath(path) {
		return e
	}
	e.Method = path
	rv := reflect.ValueOf(result)
	if ctx == nil || !rv.IsValid() || rv.Kind() != reflect.Pointer || rv.IsNil() || !rv.Elem().CanSet() {
		return e
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return e.WithContextCause(err)
	}
	e.Kind = apierror.Transport
	if err := ctx.Err(); err != nil {
		return e.WithContextCause(err)
	}
	u := c.endpoint
	// Join in escaped form first so an encoded trailing slash remains data.
	suffix := url.URL{Path: strings.TrimPrefix(path, "/")}
	u.RawPath = strings.TrimRight(c.endpoint.EscapedPath(), "/") + "/" + suffix.EscapedPath()
	u.Path, err = url.PathUnescape(u.RawPath)
	if err != nil {
		return apierror.Error{Kind: apierror.Configuration, Cause: err}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return apierror.Error{Kind: apierror.Configuration, Cause: err}
	}
	req.Header = c.cfg.Headers.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.doer.Do(req)
	if resp != nil {
		e.HTTPStatus = resp.StatusCode
		if resp.Body != nil {
			defer resp.Body.Close()
		}
	}
	if err != nil {
		return e.WithContextCause(err)
	}
	if resp == nil {
		return e
	}
	e.Kind = apierror.Protocol
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e.Kind = apierror.HTTP
	}
	if resp.Body == nil {
		return e
	}
	// A separate detection byte avoids overflow when the limit is MaxInt64.
	body, err := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxResponseBytes))
	e.ResponseBody = body
	if err != nil {
		e.Kind = apierror.Transport
		return e.WithContextCause(err)
	}
	if int64(len(body)) == c.cfg.MaxResponseBytes {
		extra, err := io.ReadAll(io.LimitReader(resp.Body, 1))
		e.ResponseBodyTruncated = len(extra) != 0
		if err != nil {
			e.Kind = apierror.Transport
			return e.WithContextCause(err)
		}
		if e.ResponseBodyTruncated {
			return e
		}
	}
	if err := ctx.Err(); err != nil {
		e.Kind = apierror.Transport
		return e.WithContextCause(err)
	}
	if e.Kind == apierror.HTTP {
		return e
	}
	temporary := reflect.New(rv.Elem().Type())
	if err := json.Unmarshal(body, temporary.Interface()); err != nil {
		return e.WithContextCause(err)
	}
	rv.Elem().Set(temporary.Elem())
	return nil
}

// Reject percent escapes outright so arbitrary decoding depth or mixed encoding
// cannot turn accepted input into traversal, separators or a different origin.
func safePath(path string) bool {
	if strings.TrimSpace(path) == "" || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "%\\?#:") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	for _, ch := range path {
		if ch <= 32 || ch == 127 {
			return false
		}
	}
	return true
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
