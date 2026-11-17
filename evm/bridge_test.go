package evm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"testing"
)

type nativeDependencyLeaf struct {
	tag   string
	cause error
}

func (e nativeDependencyLeaf) Error() string { return e.tag }
func (e nativeDependencyLeaf) Unwrap() error { return e.cause }

func TestNativeEVMWrappedErrorAndRequestIsolation(t *testing.T) {
	const addressMarker = "0x1234567890123456789012345678901234567890"
	const payloadMarker = "0xdecafbad0123456789abcdef"
	ctx := context.Background() // Caller returns cancellation; the request context remains valid.
	for _, op := range []struct {
		name, method string
		run          func(nativeContract) error
	}{
		{"query", "eth_getBalance", func(c nativeContract) error {
			_, err := c.BalanceAt(ctx, common.HexToAddress(addressMarker), nil)
			return err
		}},
		{"compute", "eth_call", func(c nativeContract) error {
			to := common.HexToAddress(addressMarker)
			_, err := c.CallContract(ctx, geth.CallMsg{To: &to, Data: common.FromHex(payloadMarker)}, nil)
			return err
		}},
		{"submit", "eth_sendRawTransaction", func(c nativeContract) error {
			_, err := c.SendRawTransaction(ctx, payloadMarker)
			return err
		}},
	} {
		for _, tc := range []struct {
			name  string
			kind  apierror.Kind
			cause error
		}{
			{"rpc", apierror.RPC, nil},
			{"http", apierror.HTTP, nil},
			{"plain-typed", "", nil},
			{"canceled", apierror.Transport, context.Canceled},
			{"deadline", apierror.Transport, context.DeadlineExceeded},
		} {
			t.Run(op.name+"/"+tc.name, func(t *testing.T) {
				leaf := nativeDependencyLeaf{tag: "SYNTHETIC_TYPED_DEPENDENCY", cause: tc.cause}
				source := apierror.Error{Kind: tc.kind, Method: op.method, HasRPCCode: true, RPCCode: -32001,
					HTTPStatus: 503, Message: "  SYNTHETIC_NODE_MESSAGE\n", Data: json.RawMessage(` { "n":9007199254740993 } `),
					ResponseBody: []byte("SYNTHETIC_RESPONSE_BODY\n"), ResponseBodyTruncated: true, Cause: leaf}
				var dependency error = source
				if tc.kind == "" {
					dependency = leaf
				}
				wrapper := fmt.Errorf("SYNTHETIC_DEPENDENCY_WRAPPER: %w", dependency)
				f := &rpcFake{err: wrapper}
				c := nativeClient(t, f)
				err := op.run(c)
				if !errors.Is(err, wrapper) || !errors.Is(err, leaf) {
					t.Fatalf("NS-07: complete dependency wrapper/leaf identity lost: %v", err)
				}
				var typed nativeDependencyLeaf
				if !errors.As(err, &typed) || typed != leaf {
					t.Fatalf("NS-07: typed dependency leaf lost: %v", err)
				}
				if tc.cause != nil && !errors.Is(err, tc.cause) {
					t.Fatalf("NS-07: wrapped context cause lost: %v", err)
				}
				equal(t, len(f.calls), 1)
				equal(t, f.calls[0].method, op.method)
				equal(t, f.calls[0].ctx, ctx)
				if tc.kind != "" {
					a := assertKind(t, err, tc.kind)
					equal(t, a.Message, source.Message)
					equal(t, a.Data, source.Data)
					equal(t, a.ResponseBody, source.ResponseBody)
					equal(t, a.ResponseBodyTruncated, true)
					equal(t, a.RPCCode, source.RPCCode)
					equal(t, a.HasRPCCode, true)
					equal(t, a.HTTPStatus, 503)
					equal(t, a.SubmissionUnknown, op.name == "submit" && tc.kind != apierror.RPC)
					if !strings.Contains(err.Error(), source.Message) {
						t.Fatal("NS-07: original node text lost")
					}
				} else {
					if !strings.Contains(err.Error(), wrapper.Error()) {
						t.Fatal("NS-07: original dependency text lost")
					}
					if op.name == "submit" {
						equal(t, assertKind(t, err, apierror.Transport).SubmissionUnknown, true)
					}
				}
				encoded, _ := json.Marshal(err)
				returned := fmt.Sprintf("%v %+v %#v %s", err, err, err, encoded)
				for _, marker := range []string{addressMarker, payloadMarker, strings.TrimPrefix(payloadMarker, "0x")} {
					if strings.Contains(strings.ToLower(returned), strings.ToLower(marker)) {
						t.Fatal("NS-07: SDK appended request-only address/payload to returned error")
					}
				}
			})
		}
	}
}

func TestNativeEVMCallerErrorReachability(t *testing.T) {
	leaf := errors.New("SYNTHETIC_CAUSE")
	for _, kind := range []apierror.Kind{apierror.RPC, apierror.HTTP, apierror.Transport, apierror.Protocol} {
		t.Run(kind.ToString(), func(t *testing.T) {
			source := apierror.Error{Kind: kind, Method: "eth_blockNumber", HasRPCCode: true, RPCCode: 0, HTTPStatus: 503, Message: "  SYNTHETIC_NODE\n", Data: json.RawMessage(` { "n":9007199254740993 } `), ResponseBody: []byte("SYNTHETIC_BODY\n"), ResponseBodyTruncated: true, Cause: leaf}
			f := &rpcFake{err: source}
			c, err := New(f)
			if err != nil {
				t.Fatal(err)
			}
			_, err = c.BlockNumber(context.Background())
			a := assertKind(t, err, kind)
			equal(t, a.Message, source.Message)
			equal(t, a.Data, source.Data)
			equal(t, a.ResponseBody, source.ResponseBody)
			equal(t, a.RPCCode, source.RPCCode)
			equal(t, a.HasRPCCode, true)
			equal(t, a.HTTPStatus, 503)
			equal(t, a.ResponseBodyTruncated, true)
			if !errors.Is(err, leaf) {
				t.Fatal("NS-07: original cause unreachable")
			}
			equal(t, len(f.calls), 1)
		})
	}
}

// The literal header is sufficient for fixed geth decoding. No transaction is constructed.
func nativeHeader(uncleHash string) string {
	return `{"parentHash":"` + h1 + `","sha3Uncles":"` + uncleHash + `","miner":"` + a1 + `","stateRoot":"` + h1 + `","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"` + h1 + `","logsBloom":"0x` + strings.Repeat("00", 256) + `","difficulty":"0x1","number":"0x8","gasLimit":"0x100000","gasUsed":"0x0","timestamp":"0x9","extraData":"0x","mixHash":"` + h1 + `","nonce":"0x0000000000000000"}`
}
func TestNativeEVMCompleteBlockUncles(t *testing.T) {
	for _, failAt := range []int{0, 2} {
		t.Run(map[int]string{0: "three-sequential-calls", 2: "stop-on-first-uncle-error"}[failAt], func(t *testing.T) {
			leaf := errors.New("SYNTHETIC_UNCLE_FAILURE")
			f := &rpcFake{respond: func(n int, method string, p json.RawMessage) (string, error) {
				if n == 1 {
					equal(t, method, "eth_getBlockByNumber")
					equal(t, string(p), `["0x8",true]`)
					return strings.TrimSuffix(nativeHeader(h1), "}") + `,"hash":"` + h1 + `","transactions":[],"uncles":["` + h1 + `","` + h1 + `"]}`, nil
				}
				equal(t, method, "eth_getUncleByBlockHashAndIndex")
				if n == 2 {
					equal(t, string(p), `["`+h1+`","0x0"]`)
				} else {
					equal(t, string(p), `["`+h1+`","0x1"]`)
				}
				if n == failAt {
					return "", leaf
				}
				return nativeHeader(h1), nil
			}}
			c := nativeClient(t, f)
			block, err := c.BlockByNumber(context.Background(), big.NewInt(8))
			if failAt > 0 {
				if !errors.Is(err, leaf) {
					t.Fatalf("uncle Cause lost: %v", err)
				}
				equal(t, len(f.calls), 2)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			equal(t, len(f.calls), 3)
			equal(t, block.NumberU64(), uint64(8))
			equal(t, len(block.Uncles()), 2)
			equal(t, len(block.Transactions()), 0)
		})
	}
}

func TestNativeEVMCompleteBlockTransactions(t *testing.T) {
	header := nativeHeader("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")
	header = strings.Replace(header, "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421", h1, 1)
	f := &rpcFake{reply: strings.TrimSuffix(header, "}") + `,"hash":"` + h1 + `","transactions":[` + nativeTransaction + `],"uncles":[]}`}
	c := nativeClient(t, f)
	b, err := c.BlockByNumber(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, len(b.Transactions()), 1)
	equal(t, b.Transactions()[0].Value().String(), "18446744073709551617")
	equal(t, len(b.Uncles()), 0)
	assertWire(t, f, "eth_getBlockByNumber", `["latest",true]`, context.Background())
}
