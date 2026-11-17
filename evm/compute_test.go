package evm

import (
	"context"
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestNativeEVMCompute(t *testing.T) {
	ctx := context.Background()
	to := common.HexToAddress(a1)
	msg := geth.CallMsg{To: &to, Gas: 21000, Value: big.NewInt(0), GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(1), Data: []byte{0, 255}}
	arg := `{"from":"0x0000000000000000000000000000000000000000","to":"` + a1 + `","gas":"0x5208","value":"0x0","maxFeePerGas":"0xa","maxPriorityFeePerGas":"0x1","input":"0x00ff"}`
	for _, tc := range []struct {
		name, method, params, reply string
		run                         func(nativeContract) (any, error)
		want                        any
	}{
		{"call", "eth_call", "[" + arg + `,"pending"]`, `"0x00ff"`, func(c nativeContract) (any, error) { return c.CallContract(ctx, msg, big.NewInt(-1)) }, []byte{0, 255}},
		{"estimate", "eth_estimateGas", "[" + arg + "]", `"0x5208"`, func(c nativeContract) (any, error) { return c.EstimateGas(ctx, msg) }, uint64(21000)},
		{"price", "eth_gasPrice", "[]", `"0x10000000000000001"`, func(c nativeContract) (any, error) { return c.SuggestGasPrice(ctx) }, new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1))},
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
			equal(t, msg.Value, big.NewInt(0))
			equal(t, msg.Data, []byte{0, 255})
		})
	}
}
func TestNativeEVMFeeHistory(t *testing.T) {
	f := &rpcFake{reply: `{"oldestBlock":"0xf","baseFeePerGas":["0x10000000000000001","0x2","0x3"],"gasUsedRatio":[0.5,1],"reward":[["0x4","0x5"],["0x6","0x7"]]}`}
	c := nativeClient(t, f)
	percentiles := []float64{25, 75}
	v, err := c.FeeHistory(context.Background(), 2, nil, percentiles)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, v.OldestBlock, big.NewInt(15))
	equal(t, v.BaseFee[0].String(), "18446744073709551617")
	equal(t, v.GasUsedRatio, []float64{0.5, 1})
	equal(t, v.Reward[1][1], big.NewInt(7))
	equal(t, percentiles, []float64{25, 75})
	assertWire(t, f, "eth_feeHistory", `["0x2","latest",[25,75]]`, context.Background())
}
