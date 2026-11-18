package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	scenarioEVMContract = "0xdAC17F958D2ee523a2206206994597C13D831ec7"
	scenarioEVMHead     = 100
)

// scenarioEVMNode is a deterministic in-memory EVM JSON-RPC node keyed by method name.
type scenarioEVMNode struct {
	mu                                        sync.Mutex
	chainID                                   uint64
	uncles                                    int
	emptyBlock, tamperReceipt, feeHistoryFail bool
	transportErr                              bool
	log                                       []string
	header                                    *types.Header
	tx                                        *types.Transaction
	uncleHeaders                              []*types.Header
}

func (n *scenarioEVMNode) Do(req *http.Request) (*http.Response, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.transportErr {
		n.log = append(n.log, "transport")
		return nil, &url.Error{Op: "Post", URL: req.URL.String(), Err: errors.New("dial refused " + scenarioPathSecret)}
	}
	var in struct {
		ID     json.RawMessage   `json:"id"`
		Method string            `json:"method"`
		Params []json.RawMessage `json:"params"`
	}
	raw, _ := io.ReadAll(req.Body)
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	params, _ := json.Marshal(in.Params)
	n.log = append(n.log, in.Method+" "+string(params))
	result, rpcErr := n.dispatch(in.Method, in.Params)
	env := map[string]any{"jsonrpc": "2.0", "id": in.ID}
	if rpcErr != "" {
		env["error"] = map[string]any{"code": -32601, "message": rpcErr}
	} else {
		env["result"] = result
	}
	body, _ := json.Marshal(env)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
}

func (n *scenarioEVMNode) build(number uint64) {
	contract := common.HexToAddress(scenarioEVMContract)
	header := func(num uint64, extra byte) *types.Header {
		return &types.Header{ParentHash: common.HexToHash("0xaa"), UncleHash: types.EmptyUncleHash, Coinbase: contract, Root: common.HexToHash("0xbb"), TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash, Difficulty: big.NewInt(1), Number: new(big.Int).SetUint64(num), GasLimit: 30_000_000, GasUsed: 21_000, Time: 1_700_000_000, Extra: []byte{extra}}
	}
	n.header, n.tx, n.uncleHeaders = header(number, 0xff), nil, nil
	for i := 0; i < n.uncles; i++ {
		n.uncleHeaders = append(n.uncleHeaders, header(number-1, byte(i)))
	}
	if len(n.uncleHeaders) > 0 {
		n.header.UncleHash = types.CalcUncleHash(n.uncleHeaders)
	}
	if !n.emptyBlock {
		n.tx = types.NewTx(&types.LegacyTx{Nonce: number, To: &contract, Value: big.NewInt(1), Gas: 21_000, GasPrice: big.NewInt(1)})
		n.header.TxHash = common.HexToHash("0x01")
	}
}

func scenarioEVMObject(v any, extra map[string]any) map[string]any {
	b, _ := json.Marshal(v)
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	for k, x := range extra {
		m[k] = x
	}
	return m
}

func (n *scenarioEVMNode) txObject() map[string]any {
	return scenarioEVMObject(n.tx, map[string]any{"blockHash": n.header.Hash(), "blockNumber": hexutil.EncodeBig(n.header.Number), "from": common.HexToAddress("0x01"), "transactionIndex": "0x0"})
}

func (n *scenarioEVMNode) dispatch(method string, params []json.RawMessage) (any, string) {
	str := func(i int) string {
		var s string
		if i < len(params) {
			_ = json.Unmarshal(params[i], &s)
		}
		return s
	}
	switch method {
	case "eth_chainId":
		return hexutil.EncodeUint64(n.chainID), ""
	case "eth_blockNumber":
		return hexutil.EncodeUint64(scenarioEVMHead), ""
	case "eth_getBlockByNumber":
		number, err := hexutil.DecodeUint64(str(0))
		if err != nil {
			number = scenarioEVMHead
		}
		n.build(number)
		hashes := []common.Hash{}
		for _, u := range n.uncleHeaders {
			hashes = append(hashes, u.Hash())
		}
		txs := []any{}
		if n.tx != nil {
			txs = append(txs, n.txObject())
		}
		return scenarioEVMObject(n.header, map[string]any{"transactions": txs, "uncles": hashes}), ""
	case "eth_getUncleByBlockHashAndIndex":
		i, err := hexutil.DecodeUint64(str(1))
		if n.header == nil || err != nil || int(i) >= len(n.uncleHeaders) || common.HexToHash(str(0)) != n.header.Hash() {
			return nil, ""
		}
		return n.uncleHeaders[i], ""
	case "eth_getTransactionByHash":
		if n.tx == nil || common.HexToHash(str(0)) != n.tx.Hash() {
			return nil, ""
		}
		return n.txObject(), ""
	case "eth_getTransactionReceipt":
		if n.tx == nil || common.HexToHash(str(0)) != n.tx.Hash() {
			return nil, ""
		}
		r := &types.Receipt{Status: 1, CumulativeGasUsed: 21_000, Logs: []*types.Log{}, TxHash: n.tx.Hash(), GasUsed: 21_000, EffectiveGasPrice: big.NewInt(1), BlockHash: n.header.Hash(), BlockNumber: n.header.Number}
		if n.tamperReceipt {
			r.BlockHash = common.HexToHash("0xdead")
		}
		return r, ""
	case "eth_getBalance", "eth_getTransactionCount", "eth_gasPrice":
		return "0x1", ""
	case "eth_getCode":
		return "0x6080", ""
	case "eth_call":
		return "0x" + strings.Repeat("0", 63) + "1", ""
	case "eth_estimateGas":
		return "0x5208", ""
	case "eth_getLogs":
		return []any{}, ""
	case "eth_feeHistory":
		if n.feeHistoryFail {
			return nil, "the method eth_feeHistory does not exist"
		}
		return map[string]any{"oldestBlock": "0x5a", "baseFeePerGas": []string{"0x1", "0x1"}, "gasUsedRatio": []float64{0.5}, "reward": [][]string{{"0x1"}}}, ""
	}
	return nil, "method not found"
}

func (n *scenarioEVMNode) count(prefix string) int {
	c := 0
	for _, l := range n.log {
		if strings.HasPrefix(l, prefix) {
			c++
		}
	}
	return c
}

func scenarioEVMRun(chain string, expected uint64, known []string, max int, node *scenarioEVMNode) (map[string]scenarioIssue, int) {
	cfg := scenarioConfig{MaxRequests: max, MinInterval: time.Millisecond, TotalTimeout: time.Minute, RequiredChains: []string{chain}, KnownUnsupported: known}
	chainCfg := scenarioChainConfig{
		Endpoint:       "https://" + chain + ".invalid/v3/" + scenarioPathSecret,
		Headers:        map[string]string{"X-Api-Key": scenarioHeaderSecret},
		Params:         map[string]string{"expected_chain_id": fmt.Sprint(expected), "contract": scenarioEVMContract, "call_data": "0x18160ddd"},
		RequestTimeout: time.Minute,
	}
	cfg.Chains = map[string]scenarioChainConfig{chain: chainCfg}
	budget := newScenarioBudget(node, max, 0, func(context.Context, time.Duration) error { return nil })
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return runScenarioEVM(ctx, chain, cfg, chainCfg, budget), budget.Requests()
}

func TestScenarioEVM(t *testing.T) {
	identity := map[string]bool{"ChainID": true, "ValidateNetwork": true}
	pass := scenarioIssue{"PASS", ""}
	for _, tc := range []struct {
		name     string
		chain    string
		expected uint64
		node     *scenarioEVMNode
		known    []string
		check    func(t *testing.T, got map[string]scenarioIssue, n *scenarioEVMNode, requests int)
	}{
		// LS-07: one consistent replay per chain; submit is never touched.
		{name: "replay_ethereum", chain: "ethereum", expected: 1, node: &scenarioEVMNode{chainID: 1}},
		{name: "replay_bnb", chain: "bnb", expected: 56, node: &scenarioEVMNode{chainID: 56}},
		// LS-06: identity mismatch sends only the two identity requests.
		{name: "chain_id_mismatch", chain: "ethereum", expected: 1, node: &scenarioEVMNode{chainID: 56}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioEVMNode, requests int) {
			if requests != 2 || n.count("eth_chainId") != 2 || len(n.log) != 2 {
				t.Errorf("requests=%d log=%q; want exactly two eth_chainId", requests, n.log)
			}
			for _, id := range scenarioReadIDs("ethereum") {
				method := strings.TrimPrefix(id, "ethereum/")
				if identity[method] && got[id].Status == "PASS" || !identity[method] && got[id].Status != "BLOCKED" {
					t.Errorf("%s = %+v after identity mismatch", id, got[id])
				}
			}
		}},
		{name: "receipt_block_hash_tampered", chain: "bnb", expected: 56, node: &scenarioEVMNode{chainID: 56, tamperReceipt: true}, check: func(t *testing.T, got map[string]scenarioIssue, _ *scenarioEVMNode, _ int) {
			if got["bnb/TransactionReceipt"].Status != "FAIL" {
				t.Errorf("TransactionReceipt = %+v; want FAIL", got["bnb/TransactionReceipt"])
			}
		}},
		{name: "empty_block", chain: "ethereum", expected: 1, node: &scenarioEVMNode{chainID: 1, emptyBlock: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioEVMNode, _ int) {
			for _, id := range []string{"ethereum/TransactionByHash", "ethereum/TransactionReceipt"} {
				if got[id] != (scenarioIssue{"BLOCKED", "material"}) {
					t.Errorf("%s = %+v; want BLOCKED material", id, got[id])
				}
			}
			if n.count("eth_getTransactionByHash")+n.count("eth_getTransactionReceipt") != 0 || got["ethereum/BlockByNumber"] != pass {
				t.Errorf("empty block: BlockByNumber=%+v log=%q", got["ethereum/BlockByNumber"], n.log)
			}
		}},
		{name: "fee_history_unlisted_fail", chain: "ethereum", expected: 1, node: &scenarioEVMNode{chainID: 1, feeHistoryFail: true}, check: func(t *testing.T, got map[string]scenarioIssue, _ *scenarioEVMNode, _ int) {
			if got["ethereum/FeeHistory"] != (scenarioIssue{"FAIL", "rpc"}) || got["ethereum/SuggestGasPrice"] != pass {
				t.Errorf("FeeHistory=%+v SuggestGasPrice=%+v; want FAIL rpc, PASS", got["ethereum/FeeHistory"], got["ethereum/SuggestGasPrice"])
			}
		}},
		{name: "fee_history_known_unsupported_blocked", chain: "bnb", expected: 56, node: &scenarioEVMNode{chainID: 56, feeHistoryFail: true}, known: []string{"bnb/FeeHistory"}, check: func(t *testing.T, got map[string]scenarioIssue, _ *scenarioEVMNode, _ int) {
			if got["bnb/FeeHistory"] != (scenarioIssue{"BLOCKED", "capability"}) || got["bnb/SuggestGasPrice"] != pass {
				t.Errorf("FeeHistory=%+v SuggestGasPrice=%+v; want BLOCKED capability, PASS", got["bnb/FeeHistory"], got["bnb/SuggestGasPrice"])
			}
		}},
		// LS-11: a transport error carrying the URL token is classified without retaining it.
		{name: "transport_error_redacted", chain: "ethereum", expected: 1, node: &scenarioEVMNode{transportErr: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioEVMNode, _ int) {
			scenarioRedacted(t, got)
			if got["ethereum/ChainID"] != (scenarioIssue{"BLOCKED", "environment"}) || got["ethereum/BlockNumber"].Status != "BLOCKED" || n.count("transport") == 0 {
				t.Errorf("ChainID=%+v BlockNumber=%+v transport attempts=%d", got["ethereum/ChainID"], got["ethereum/BlockNumber"], n.count("transport"))
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			node := tc.node
			got, requests := scenarioEVMRun(tc.chain, tc.expected, tc.known, 1000, node)
			ids := scenarioReadIDs(tc.chain)
			if len(got) != len(ids) || node.count("eth_sendRawTransaction") != 0 {
				t.Errorf("result covers %d IDs (want %d); eth_sendRawTransaction calls=%d", len(got), len(ids), node.count("eth_sendRawTransaction"))
			}
			if _, ok := got[tc.chain+"/SendRawTransaction"]; ok {
				t.Error("submit entry must not be reported by the EVM runner")
			}
			if tc.check != nil {
				tc.check(t, got, node, requests)
				return
			}
			for _, id := range ids {
				if got[id] != pass {
					t.Errorf("%s = %+v; want PASS", id, got[id])
				}
			}
			if tc.chain == "ethereum" && node.count("eth_chainId") != 3 {
				t.Errorf("eth_chainId calls = %d; want ChainID, ValidateNetwork and one RawCall", node.count("eth_chainId"))
			}
		})
	}
	t.Run("two_uncles_budget_stops_third_block_request", testScenarioEVMUncleBudget)
}

// BlockByNumber with two uncles costs three requests; the budget must stop the third.
func testScenarioEVMUncleBudget(t *testing.T) {
	full := &scenarioEVMNode{chainID: 1, uncles: 2}
	got, _ := scenarioEVMRun("ethereum", 1, nil, 1000, full)
	second := -1
	for i, l := range full.log {
		if strings.HasPrefix(l, "eth_getUncleByBlockHashAndIndex") && strings.Contains(l, `"0x1"`) {
			second = i
			break
		}
	}
	if second < 2 || !strings.HasPrefix(full.log[second-1], "eth_getUncleByBlockHashAndIndex") || !strings.HasPrefix(full.log[second-2], "eth_getBlockByNumber") || got["ethereum/BlockByNumber"] != (scenarioIssue{"PASS", ""}) {
		t.Fatalf("unbudgeted run: second uncle at %d, BlockByNumber=%+v, log=%q", second, got["ethereum/BlockByNumber"], full.log)
	}
	limited := &scenarioEVMNode{chainID: 1, uncles: 2}
	got, requests := scenarioEVMRun("ethereum", 1, nil, second, limited)
	if requests != second || len(limited.log) != second {
		t.Errorf("requests=%d downstream=%d; want %d", requests, len(limited.log), second)
	}
	for _, l := range limited.log {
		if strings.HasPrefix(l, "eth_getUncleByBlockHashAndIndex") && strings.Contains(l, `"0x1"`) {
			t.Error("second uncle request was sent beyond the budget")
		}
	}
	if got["ethereum/BlockByNumber"] != (scenarioIssue{"BLOCKED", "budget"}) {
		t.Errorf("BlockByNumber = %+v; want BLOCKED budget", got["ethereum/BlockByNumber"])
	}
}
