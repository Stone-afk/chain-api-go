package tron

import (
	"context"
	"strings"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

func TestScalarConversions(t *testing.T) {
	if hexAddress.ToString() != string(hexAddress) || txID.ToString() != string(txID) {
		t.Error("named string conversion changed data")
	}
}
func TestAddressValidation(t *testing.T) {
	for _, method := range []struct {
		name string
		call func(*Client, context.Context, GetAccountRequest) error
	}{
		{"account", func(c *Client, x context.Context, r GetAccountRequest) error { _, _, e := c.GetAccount(x, r); return e }},
		{"resource", func(c *Client, x context.Context, r GetAccountRequest) error {
			_, e := c.GetAccountResource(x, r)
			return e
		}},
		{"solid", func(c *Client, x context.Context, r GetAccountRequest) error {
			_, _, e := c.GetSolidifiedAccount(x, r)
			return e
		}},
	} {
		for _, tc := range []struct {
			name           string
			address        Address
			visible, valid bool
		}{
			{"hex", hexAddress, false, true},
			{"uppercase", Address(strings.ToUpper(string(hexAddress))), false, true},
			{"base58", base58Address, true, true},
			{"empty", "", false, false},
			{"prefix", "42" + Address(strings.Repeat("a", 40)), false, false},
			{"nonhex", "41" + Address(strings.Repeat("z", 40)), false, false},
			{"short", Address(string(hexAddress)[:41]), false, false},
			{"hex-visible", hexAddress, true, false},
			{"base58-hidden", base58Address, false, false},
			{"base58-zero", Address("T0" + strings.Repeat("1", 32)), true, false},
			{"base58-O", Address("TO" + strings.Repeat("1", 32)), true, false},
			{"base58-I", Address("TI" + strings.Repeat("1", 32)), true, false},
			{"base58-l", Address("Tl" + strings.Repeat("1", 32)), true, false},
			{"base58-prefix", Address("A" + strings.Repeat("1", 33)), true, false},
			{"base58-short", Address("T" + strings.Repeat("1", 32)), true, false},
		} {
			t.Run(method.name+tc.name, func(t *testing.T) {
				p := &fakePoster{response: `{"address":"native"}`}
				ctx := context.Background()
				err := method.call(newClient(t, p), ctx, GetAccountRequest{Address: tc.address, Visible: tc.visible})
				if tc.valid {
					if err != nil {
						t.Error("valid address rejected")
					}
					if p.calls != 1 {
						t.Error("valid address not sent once")
					}
					if p.ctx != ctx {
						t.Error("context changed")
					}
					visible := "false"
					if tc.visible {
						visible = "true"
					}
					checkJSON(t, p.body, `{"address":"`+string(tc.address)+`","visible":`+visible+`}`)
				} else {
					checkError(t, err, apierror.InvalidArgument, false)
					if p.calls != 0 {
						t.Error("invalid address reached Poster")
					}
				}
			})
		}
	}
}
func TestQueryInputValidation(t *testing.T) {
	for _, solid := range []bool{false, true} {
		for _, height := range []int64{-1, 0, 9223372036854775807} {
			t.Run("height", func(t *testing.T) {
				p := &fakePoster{response: `{"blockID":"native"}`}
				c := newClient(t, p)
				var err error
				if solid {
					_, _, err = c.GetSolidifiedBlockByNum(context.Background(), height)
				} else {
					_, _, err = c.GetBlockByNum(context.Background(), height)
				}
				if height < 0 {
					checkError(t, err, apierror.InvalidArgument, false)
					if p.calls != 0 {
						t.Error("negative height sent")
					}
				} else {
					if err != nil || p.calls != 1 {
						t.Error("valid height rejected")
					}
				}
			})
		}
	}
	for _, method := range []func(*Client, TxID) error{
		func(c *Client, id TxID) error { _, _, e := c.GetTransactionByID(context.Background(), id); return e },
		func(c *Client, id TxID) error {
			_, _, e := c.GetTransactionInfoByID(context.Background(), id)
			return e
		},
		func(c *Client, id TxID) error {
			_, _, e := c.GetSolidifiedTransactionByID(context.Background(), id)
			return e
		},
		func(c *Client, id TxID) error {
			_, _, e := c.GetSolidifiedTransactionInfoByID(context.Background(), id)
			return e
		},
	} {
		for _, id := range []TxID{"", "0x" + txID, TxID(strings.Repeat("g", 64)), TxID(strings.Repeat("a", 63)), TxID(strings.Repeat("a", 65)), txID, TxID(strings.ToUpper(string(txID)))} {
			t.Run("txid", func(t *testing.T) {
				valid := id == txID || id == TxID(strings.ToUpper(string(txID)))
				p := &fakePoster{response: `{"txID":"native","id":"native"}`}
				err := method(newClient(t, p), id)
				if valid {
					if err != nil || p.calls != 1 {
						t.Error("valid txid rejected")
					}
					checkJSON(t, p.body, `{"value":"`+string(id)+`"}`)
				} else {
					checkError(t, err, apierror.InvalidArgument, false)
					if p.calls != 0 {
						t.Error("invalid txid sent")
					}
				}
			})
		}
	}
}
func TestProto3ZeroAndStandardDecode(t *testing.T) {
	for _, response := range []string{`{"address":"native"}`, `{"address":"native","balance":0}`, `{"address":"native","balance":null,"future":{"x":true}}`, `{"address":"native","balance":1,"balance":0}`} {
		t.Run(response, func(t *testing.T) {
			p := &fakePoster{response: response}
			got, present, err := newClient(t, p).GetAccount(context.Background(), GetAccountRequest{Address: hexAddress})
			if err != nil || !present || got.Address != "native" || got.Balance != 0 {
				t.Error("zero balance / standard decoding not preserved")
			}
			if got.AccountResource != nil || got.AssetV2 != nil {
				t.Error("missing raw fields synthesized")
			}
		})
	}
	p := &fakePoster{response: `{}`}
	got, err := newClient(t, p).GetAccountResource(context.Background(), GetAccountRequest{Address: hexAddress})
	if err != nil || got != (AccountResource{}) {
		t.Error("omitted resource values must remain zero")
	}
}
