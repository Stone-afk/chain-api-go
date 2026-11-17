package tron

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

type queryCase struct {
	name, path, body, response string
	want                       any
	nullable                   bool
	call                       func(*Client, context.Context) (any, bool, error)
}

func queryCases() []queryCase {
	req := GetAccountRequest{Address: hexAddress}
	account := func(c *Client, ctx context.Context) (any, bool, error) { return c.GetAccount(ctx, req) }
	return []queryCase{
		{"node", "/wallet/getnodeinfo", `{}`, `{"block":"head","solidityBlock":"solid","currentConnectCount":3,"configNodeInfo":{"n":9007199254740993},"peerList":null,"future":true}`, NodeInfo{Block: "head", SolidityBlock: "solid", CurrentConnectCount: 3, ConfigNodeInfo: json.RawMessage(`{"n":9007199254740993}`), PeerList: json.RawMessage(`null`)}, false, func(c *Client, ctx context.Context) (any, bool, error) { v, e := c.GetNodeInfo(ctx); return v, true, e }},
		{"now", "/wallet/getnowblock", `{}`, `{"blockID":"opaque","block_header":null,"transactions":[{"txID":"child","ret":[{"contractRet":"REVERT"}]}]}`, Block{BlockID: "opaque", BlockHeader: json.RawMessage(`null`), Transactions: []Transaction{{TxID: "child", Ret: json.RawMessage(`[{"contractRet":"REVERT"}]`)}}}, true, func(c *Client, ctx context.Context) (any, bool, error) { return c.GetNowBlock(ctx) }},
		{"height", "/wallet/getblockbynum", `{"num":0}`, `{"blockID":"genesis"}`, Block{BlockID: "genesis"}, true, func(c *Client, ctx context.Context) (any, bool, error) { return c.GetBlockByNum(ctx, 0) }},
		{"account", "/wallet/getaccount", `{"address":"41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","visible":false}`, `{"address":"native","balance":123,"account_resource":null,"assetV2":[{"key":"x","value":1}]}`, Account{Address: "native", Balance: 123, AccountResource: json.RawMessage(`null`), AssetV2: json.RawMessage(`[{"key":"x","value":1}]`)}, true, account},
		{"resources", "/wallet/getaccountresource", `{"address":"41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","visible":false}`, `{"freeNetUsed":1,"freeNetLimit":2,"NetUsed":3,"NetLimit":4,"EnergyUsed":5,"EnergyLimit":6,"TotalNetLimit":7,"TotalNetWeight":8,"TotalEnergyLimit":9,"TotalEnergyWeight":10}`, AccountResource{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, false, func(c *Client, ctx context.Context) (any, bool, error) {
			v, e := c.GetAccountResource(ctx, req)
			return v, true, e
		}},
		{"parameters", "/wallet/getchainparameters", `{}`, `{"chainParameter":[{"key":"zero"},{"key":"cost","value":420}]}`, ChainParameters{[]ChainParameter{{Key: "zero"}, {Key: "cost", Value: 420}}}, false, func(c *Client, ctx context.Context) (any, bool, error) {
			v, e := c.GetChainParameters(ctx)
			return v, true, e
		}},
		{"transaction", "/wallet/gettransactionbyid", `{"value":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`, `{"txID":"native","raw_data":{"n":9007199254740993},"raw_data_hex":"not-validated","signature":["s2","s1"],"ret":[{"contractRet":"REVERT"}]}`, Transaction{TxID: "native", RawData: json.RawMessage(`{"n":9007199254740993}`), RawDataHex: "not-validated", Signature: []string{"s2", "s1"}, Ret: json.RawMessage(`[{"contractRet":"REVERT"}]`)}, true, func(c *Client, ctx context.Context) (any, bool, error) { return c.GetTransactionByID(ctx, txID) }},
		{"receipt", "/wallet/gettransactioninfobyid", `{"value":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`, `{"id":"native","fee":100,"blockNumber":8,"blockTimeStamp":9,"contractResult":["00"],"contract_address":"raw","receipt":{"result":"REVERT"},"log":null,"internal_transactions":[],"result":"FAILED","resMessage":"raw message"}`, TransactionInfo{ID: "native", Fee: 100, BlockNumber: 8, BlockTimeStamp: 9, ContractResult: []string{"00"}, ContractAddress: "raw", Receipt: json.RawMessage(`{"result":"REVERT"}`), Log: json.RawMessage(`null`), InternalTransactions: json.RawMessage(`[]`), Result: json.RawMessage(`"FAILED"`), ResMessage: json.RawMessage(`"raw message"`)}, true, func(c *Client, ctx context.Context) (any, bool, error) { return c.GetTransactionInfoByID(ctx, txID) }},
		{"solid account", "/walletsolidity/getaccount", `{"address":"41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","visible":false}`, `{"address":"solid"}`, Account{Address: "solid"}, true, func(c *Client, ctx context.Context) (any, bool, error) { return c.GetSolidifiedAccount(ctx, req) }},
		{"solid now", "/walletsolidity/getnowblock", `{}`, `{"blockID":"solid"}`, Block{BlockID: "solid"}, true, func(c *Client, ctx context.Context) (any, bool, error) { return c.GetSolidifiedNowBlock(ctx) }},
		{"solid height", "/walletsolidity/getblockbynum", `{"num":7}`, `{"blockID":"solid-height"}`, Block{BlockID: "solid-height"}, true, func(c *Client, ctx context.Context) (any, bool, error) { return c.GetSolidifiedBlockByNum(ctx, 7) }},
		{"solid tx", "/walletsolidity/gettransactionbyid", `{"value":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`, `{"txID":"solid-tx"}`, Transaction{TxID: "solid-tx"}, true, func(c *Client, ctx context.Context) (any, bool, error) {
			return c.GetSolidifiedTransactionByID(ctx, txID)
		}},
		{"solid receipt", "/walletsolidity/gettransactioninfobyid", `{"value":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}`, `{"id":"solid-receipt"}`, TransactionInfo{ID: "solid-receipt"}, true, func(c *Client, ctx context.Context) (any, bool, error) {
			return c.GetSolidifiedTransactionInfoByID(ctx, txID)
		}},
	}
}
func TestQueriesWireAndDTO(t *testing.T) {
	for _, tc := range queryCases() {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), struct{}{}, "identity")
			p := &fakePoster{response: tc.response}
			got, present, err := tc.call(newClient(t, p), ctx)
			if err != nil {
				t.Error("successful fixture rejected")
			}
			if !present {
				t.Error("result absent")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("DTO mismatch: got %#v want %#v", got, tc.want)
			}
			checkWire(t, p, ctx, tc.path, tc.body)
		})
	}
}
func TestQueryPresenceAndProtocol(t *testing.T) {
	for _, method := range queryCases() {
		for _, response := range []string{`{}`, `null`, `[]`, `"value"`, `{"unrelated":1}`, `{"Error":7}`} {
			t.Run(method.name+response, func(t *testing.T) {
				p := &fakePoster{response: response}
				got, present, err := method.call(newClient(t, p), context.Background())
				if method.nullable && response == `{}` {
					if err != nil || present {
						t.Error("empty object must be absent without error")
					}
					if !reflect.ValueOf(got).IsZero() {
						t.Error("absent DTO is nonzero")
					}
				} else if response == `{}` || response == `{"unrelated":1}` && !method.nullable {
					if err != nil {
						t.Error("standard zero DTO rejected")
					}
				} else {
					checkError(t, err, apierror.Protocol, false)
					if method.nullable && present {
						t.Error("invalid result is present")
					}
				}
				if p.calls != 1 {
					t.Errorf("calls = %d", p.calls)
				}
			})
		}
	}
}
func TestQueryTopLevelError(t *testing.T) {
	for _, method := range queryCases() {
		t.Run(method.name, func(t *testing.T) {
			response := `{"Error":"  unsupported not found synthetic  ","detail":{"n":9007199254740993}}`
			p := &fakePoster{response: response}
			_, present, err := method.call(newClient(t, p), context.Background())
			e := checkError(t, err, apierror.RPC, false)
			if e.Message != "  unsupported not found synthetic  " || string(e.Data) != response {
				t.Error("original Error response lost")
			}
			if e.HTTPStatus != 0 || len(e.ResponseBody) != 0 || e.Cause != nil {
				t.Error("fabricated transport metadata")
			}
			if method.nullable && present {
				t.Error("error marked present")
			}
			if e.Method != method.path {
				t.Error("error method mismatch")
			}
			if p.calls != 1 {
				t.Error("error request count mismatch")
			}
		})
	}
}

func TestQueryStandardDecodeErrorCause(t *testing.T) {
	p := &fakePoster{response: `{"address":"native","balance":"wrong type"}`}
	got, present, err := newClient(t, p).GetAccount(context.Background(), GetAccountRequest{Address: hexAddress})
	e := checkError(t, err, apierror.Protocol, false)
	var typeErr *json.UnmarshalTypeError
	if !errors.As(err, &typeErr) || e.Cause == nil {
		t.Error("standard decode error not reachable")
	}
	if present || !reflect.ValueOf(got).IsZero() {
		t.Error("partially decoded DTO leaked")
	}
	if p.calls != 1 {
		t.Error("decode failure call count mismatch")
	}
}
