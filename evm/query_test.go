package evm

import (
	"context"
	"errors"
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"testing"
)

func TestNativeEVMQueryWire(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "synthetic-context")
	addr := common.HexToAddress(a1)
	for _, tc := range []struct {
		name, method, params, reply string
		run                         func(nativeContract) (any, error)
		want                        any
	}{
		{"chain", "eth_chainId", "[]", `"0x10000000000000001"`, func(c nativeContract) (any, error) { return c.ChainID(ctx) }, new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1))},
		{"height", "eth_blockNumber", "[]", `"0x10"`, func(c nativeContract) (any, error) { return c.BlockNumber(ctx) }, uint64(16)},
		{"balance", "eth_getBalance", `["` + a1 + `","latest"]`, `"0x10000000000000001"`, func(c nativeContract) (any, error) { return c.BalanceAt(ctx, addr, nil) }, new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1))},
		{"pending-nonce", "eth_getTransactionCount", `["` + a1 + `","pending"]`, `"0x0"`, func(c nativeContract) (any, error) { return c.NonceAt(ctx, addr, big.NewInt(-1)) }, uint64(0)},
		{"code", "eth_getCode", `["` + a1 + `","0x0"]`, `"0x00ff"`, func(c nativeContract) (any, error) { return c.CodeAt(ctx, addr, big.NewInt(0)) }, []byte{0, 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &rpcFake{reply: tc.reply}
			c := nativeClient(t, f)
			got, err := tc.run(c)
			if err != nil {
				t.Fatal(err)
			}
			equal(t, got, tc.want)
			assertWire(t, f, tc.method, tc.params, ctx)
		})
	}
}
func TestNativeEVMNullIsNotFound(t *testing.T) {
	for _, name := range []string{"block", "transaction", "receipt"} {
		t.Run(name, func(t *testing.T) {
			f := &rpcFake{reply: "null"}
			c := nativeClient(t, f)
			ctx := context.Background()
			var err error
			switch name {
			case "block":
				v, e := c.BlockByNumber(ctx, nil)
				equal(t, v == nil, true)
				err = e
			case "transaction":
				v, pending, e := c.TransactionByHash(ctx, common.HexToHash(h1))
				equal(t, v == nil, true)
				equal(t, pending, false)
				err = e
			case "receipt":
				v, e := c.TransactionReceipt(ctx, common.HexToHash(h1))
				equal(t, v == nil, true)
				err = e
			}
			if !errors.Is(err, geth.NotFound) {
				t.Fatalf("NS-03: null returned %v, want ethereum.NotFound", err)
			}
			equal(t, len(f.calls), 1)
		})
	}
}

// Literal serialized synthetic transaction; no signing, transaction construction or encoding.
const nativeTransaction = `{"type":"0x0","nonce":"0x2","gasPrice":"0x3","gas":"0x5208","to":null,"value":"0x10000000000000001","input":"0x00ff","v":"0x1b","r":"0x1","s":"0x2","hash":"` + h1 + `","from":"` + a1 + `"}`

func TestNativeEVMTransactionPending(t *testing.T) {
	for _, pending := range []bool{true, false} {
		t.Run(map[bool]string{true: "pending", false: "mined"}[pending], func(t *testing.T) {
			raw := nativeTransaction
			if !pending {
				raw = strings.TrimSuffix(raw, "}") + `,"blockNumber":"0x8","blockHash":"` + h1 + `"}`
			}
			f := &rpcFake{reply: raw}
			c := nativeClient(t, f)
			tx, p, err := c.TransactionByHash(context.Background(), common.HexToHash(h1))
			if err != nil {
				t.Fatal(err)
			}
			equal(t, p, pending)
			equal(t, tx.Nonce(), uint64(2))
			equal(t, tx.To() == nil, true)
			equal(t, tx.Value().String(), "18446744073709551617")
			equal(t, tx.Data(), []byte{0, 255})
			assertWire(t, f, "eth_getTransactionByHash", `["`+h1+`"]`, context.Background())
		})
	}
}
func TestNativeEVMLogsAndReceipt(t *testing.T) {
	log := `{"address":"` + a1 + `","topics":["` + h1 + `"],"data":"0x00ff","blockNumber":"0x8","transactionHash":"` + h1 + `","transactionIndex":"0x0","blockHash":"` + h1 + `","logIndex":"0x0","removed":true}`
	t.Run("logs", func(t *testing.T) {
		f := &rpcFake{reply: "[" + log + "]"}
		c := nativeClient(t, f)
		logs, err := c.FilterLogs(context.Background(), geth.FilterQuery{FromBlock: big.NewInt(0), ToBlock: big.NewInt(-1), Addresses: []common.Address{common.HexToAddress(a1)}, Topics: [][]common.Hash{{common.HexToHash(h1)}, nil}})
		if err != nil {
			t.Fatal(err)
		}
		equal(t, len(logs), 1)
		equal(t, logs[0].Removed, true)
		equal(t, logs[0].Data, []byte{0, 255})
		assertWire(t, f, "eth_getLogs", `[{"fromBlock":"0x0","toBlock":"pending","address":["`+a1+`"],"topics":[["`+h1+`"],null]}]`, context.Background())
	})
	t.Run("receipt", func(t *testing.T) {
		f := &rpcFake{reply: `{"type":"0x0","status":"0x0","cumulativeGasUsed":"0x5208","logsBloom":"0x` + strings.Repeat("00", 256) + `","logs":[` + log + `],"transactionHash":"` + h1 + `","contractAddress":null,"gasUsed":"0x5208","effectiveGasPrice":"0x10000000000000001","blockHash":"` + h1 + `","blockNumber":"0x8","transactionIndex":"0x0"}`}
		c := nativeClient(t, f)
		r, err := c.TransactionReceipt(context.Background(), common.HexToHash(h1))
		if err != nil {
			t.Fatal(err)
		}
		equal(t, r.Status, uint64(0))
		equal(t, r.BlockNumber, big.NewInt(8))
		equal(t, r.EffectiveGasPrice.String(), "18446744073709551617")
		equal(t, len(r.Logs), 1)
		assertWire(t, f, "eth_getTransactionReceipt", `["`+h1+`"]`, context.Background())
	})
}
