package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	ns "github.com/solana-foundation/solana-go/v2"
)

const (
	scenarioSOLGenesis = "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d"
	scenarioSOLSlot    = 300_000_000
)

var (
	scenarioSOLSig   = bytes.Repeat([]byte{0x5a}, 64)
	scenarioSOLPayer = bytes.Repeat([]byte{0x11}, 32)
)

// scenarioSOLMessage is a synthetic legacy message: header, two keys, blockhash, one empty instruction.
func scenarioSOLMessage() []byte {
	m := []byte{1, 0, 1, 2}
	m = append(m, scenarioSOLPayer...)
	m = append(m, make([]byte, 32)...)
	m = append(m, bytes.Repeat([]byte{0x22}, 32)...)
	return append(m, 1, 1, 0, 0)
}

func scenarioSOLRaw(sigCount byte) []byte {
	raw := append([]byte{sigCount}, scenarioSOLSig...)
	return append(raw, scenarioSOLMessage()...)
}

// scenarioSOLNode is a deterministic in-memory Solana node speaking JSON-RPC 2.0.
type scenarioSOLNode struct {
	mu                      sync.Mutex
	genesis                 string
	sigCount                byte
	tamperSlot, nullFee     bool
	balanceErr              bool
	blocksResult            []uint64
	badVersion, missingAuth int
	log                     []string
	params                  map[string][]json.RawMessage
}

func (n *scenarioSOLNode) Do(req *http.Request) (*http.Response, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if req.Header.Get("Authorization") != "Bearer "+scenarioHeaderSecret {
		n.missingAuth++
	}
	var in struct {
		Version string            `json:"jsonrpc"`
		ID      json.RawMessage   `json:"id"`
		Method  string            `json:"method"`
		Params  []json.RawMessage `json:"params"`
	}
	raw, _ := io.ReadAll(req.Body)
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if in.Version != "2.0" {
		n.badVersion++
	}
	n.log = append(n.log, in.Method)
	if n.params == nil {
		n.params = map[string][]json.RawMessage{}
	}
	n.params[in.Method] = in.Params
	result, code := n.dispatch(in.Method)
	env := map[string]any{"jsonrpc": "2.0", "id": in.ID, "result": result}
	if code != 0 {
		env = map[string]any{"jsonrpc": "2.0", "id": in.ID, "error": map[string]any{"code": code, "message": "rejected " + scenarioPathSecret + " " + scenarioHeaderSecret}}
	}
	body, _ := json.Marshal(env)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func (n *scenarioSOLNode) dispatch(method string) (any, int) {
	ctx := map[string]any{"slot": scenarioSOLSlot}
	sig := ns.SignatureFromBytes(scenarioSOLSig).String()
	hash := ns.HashFromBytes(bytes.Repeat([]byte{0x33}, 32)).String()
	switch method {
	case "getGenesisHash":
		return n.genesis, 0
	case "getSlot":
		return scenarioSOLSlot, 0
	case "getBlock":
		return map[string]any{"blockhash": hash, "previousBlockhash": hash, "parentSlot": scenarioSOLSlot - 1, "signatures": []string{sig}, "blockHeight": 280_000_000, "blockTime": 1_700_000_000}, 0
	case "getTransaction":
		slot := scenarioSOLSlot
		if n.tamperSlot {
			slot++
		}
		meta := map[string]any{"err": nil, "fee": 5000, "preBalances": []int{10_000, 1}, "postBalances": []int{5000, 1}, "status": map[string]any{"Ok": nil}}
		return map[string]any{"slot": slot, "blockTime": 1_700_000_000, "transaction": []string{base64.StdEncoding.EncodeToString(scenarioSOLRaw(n.sigCount)), "base64"}, "meta": meta, "version": "legacy"}, 0
	case "getSignatureStatuses":
		return map[string]any{"context": ctx, "value": []any{map[string]any{"slot": scenarioSOLSlot, "confirmations": nil, "err": nil, "status": map[string]any{"Ok": nil}, "confirmationStatus": "finalized"}}}, 0
	case "getBalance":
		if n.balanceErr {
			return nil, -32602
		}
		return map[string]any{"context": ctx, "value": 5000}, 0
	case "getAccountInfo":
		return map[string]any{"context": ctx, "value": map[string]any{"lamports": 5000, "owner": "11111111111111111111111111111111", "data": []string{"", "base64"}, "executable": false, "rentEpoch": 0, "space": 0}}, 0
	case "getLatestBlockhash":
		return map[string]any{"context": ctx, "value": map[string]any{"blockhash": hash, "lastValidBlockHeight": 280_000_150}}, 0
	case "getFeeForMessage":
		if n.nullFee {
			return map[string]any{"context": ctx, "value": nil}, 0
		}
		return map[string]any{"context": ctx, "value": 5000}, 0
	case "simulateTransaction":
		return map[string]any{"context": ctx, "value": map[string]any{"err": nil, "logs": []string{}, "accounts": nil, "unitsConsumed": 150}}, 0
	case "getFirstAvailableBlock":
		return uint64(0), 0
	case "getBlocks":
		if n.blocksResult != nil {
			return n.blocksResult, 0
		}
		return []uint64{scenarioSOLSlot}, 0
	}
	return nil, -32601
}

func (n *scenarioSOLNode) count(method string) int {
	c := 0
	for _, l := range n.log {
		if l == method {
			c++
		}
	}
	return c
}

func (n *scenarioSOLNode) opts(method string) map[string]any {
	p := n.params[method]
	if len(p) < 2 {
		return nil
	}
	var m map[string]any
	_ = json.Unmarshal(p[len(p)-1], &m)
	return m
}

func scenarioSOLRun(node *scenarioSOLNode) (map[string]scenarioIssue, int) {
	chainCfg := scenarioChainConfig{
		Endpoint:       "https://solana.invalid/v3/" + scenarioPathSecret,
		Headers:        map[string]string{"Authorization": "Bearer " + scenarioHeaderSecret},
		Params:         map[string]string{"expected_genesis": scenarioSOLGenesis},
		RequestTimeout: time.Minute,
	}
	cfg := scenarioConfig{MaxRequests: 1000, MinInterval: time.Millisecond, TotalTimeout: time.Minute, RequiredChains: []string{"solana"}, Chains: map[string]scenarioChainConfig{"solana": chainCfg}}
	budget := newScenarioBudget(node, 1000, 0, func(context.Context, time.Duration) error { return nil })
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return runScenarioSolana(ctx, cfg, chainCfg, budget), budget.Requests()
}

func TestScenarioSolana(t *testing.T) {
	pass := scenarioIssue{"PASS", ""}
	allPass := func(t *testing.T, got map[string]scenarioIssue) {
		for _, id := range scenarioReadIDs("solana") {
			if got[id] != pass {
				t.Errorf("%s = %+v; want PASS", id, got[id])
			}
		}
	}
	for _, tc := range []struct {
		name  string
		node  *scenarioSOLNode
		check func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, requests int)
	}{
		// LS-09: consistent replay of all twelve reads, block signatures only, base64 transaction bytes.
		{name: "replay_all_pass", node: &scenarioSOLNode{genesis: scenarioSOLGenesis, sigCount: 1}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, requests int) {
			allPass(t, got)
			if n.opts("getBlock")["transactionDetails"] != "signatures" || n.opts("getTransaction")["encoding"] != "base64" {
				t.Errorf("getBlock opts=%v getTransaction opts=%v; want signatures and base64", n.opts("getBlock"), n.opts("getTransaction"))
			}
			var msg string
			if p := n.params["getFeeForMessage"]; len(p) > 0 {
				_ = json.Unmarshal(p[0], &msg)
			}
			if msg != base64.StdEncoding.EncodeToString(scenarioSOLMessage()) {
				t.Errorf("getFeeForMessage message=%q; want byte-offset slice of fetched transaction", msg)
			}
			p := n.params["getBlocks"]
			if n.count("getBlocks") != 1 || len(p) != 3 {
				t.Errorf("getBlocks calls=%d params=%v; want one call with [slot,slot,commitment]", n.count("getBlocks"), p)
			} else {
				var start, end uint64
				var opts map[string]any
				_ = json.Unmarshal(p[0], &start)
				_ = json.Unmarshal(p[1], &end)
				_ = json.Unmarshal(p[2], &opts)
				if start != scenarioSOLSlot || end != scenarioSOLSlot || opts["commitment"] != "finalized" {
					t.Errorf("getBlocks params=%v; want [%d,%d,{commitment:finalized}]", p, scenarioSOLSlot, scenarioSOLSlot)
				}
			}
			if n.count("getFirstAvailableBlock") != 1 {
				t.Errorf("getFirstAvailableBlock calls=%d; want 1", n.count("getFirstAvailableBlock"))
			}
			if requests == 0 || requests != len(n.log) {
				t.Errorf("requests=%d downstream=%d", requests, len(n.log))
			}
		}},
		{name: "get_blocks_mismatch_fail", node: &scenarioSOLNode{genesis: scenarioSOLGenesis, sigCount: 1, blocksResult: []uint64{scenarioSOLSlot - 1}}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, _ int) {
			if got["solana/GetBlocks"].Status != "FAIL" || n.count("getBlocks") != 1 {
				t.Errorf("GetBlocks = %+v, getBlocks calls=%d; want FAIL after one request", got["solana/GetBlocks"], n.count("getBlocks"))
			}
		}},
		// LS-06: genesis mismatch stops before any other request.
		{name: "genesis_mismatch_zero_requests", node: &scenarioSOLNode{genesis: "4uhcVJyU9pJkvQyS88uRDiswHXSCkY3zQawwpjk2NsNY", sigCount: 1}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, requests int) {
			if requests != 1 || len(n.log) != 1 || n.log[0] != "getGenesisHash" {
				t.Errorf("requests=%d log=%q; want only getGenesisHash", requests, n.log)
			}
			if got["solana/GetGenesisHash"].Status != "FAIL" {
				t.Errorf("GetGenesisHash = %+v; want FAIL", got["solana/GetGenesisHash"])
			}
			for _, id := range scenarioReadIDs("solana") {
				if id != "solana/GetGenesisHash" && got[id].Status != "BLOCKED" {
					t.Errorf("%s = %+v after genesis mismatch; want BLOCKED", id, got[id])
				}
			}
		}},
		{name: "transaction_slot_tampered", node: &scenarioSOLNode{genesis: scenarioSOLGenesis, sigCount: 1, tamperSlot: true}, check: func(t *testing.T, got map[string]scenarioIssue, _ *scenarioSOLNode, _ int) {
			if got["solana/GetTransaction"].Status != "FAIL" {
				t.Errorf("GetTransaction = %+v; want FAIL", got["solana/GetTransaction"])
			}
		}},
		{name: "signature_section_invalid", node: &scenarioSOLNode{genesis: scenarioSOLGenesis, sigCount: 5}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, _ int) {
			if got["solana/GetFeeForMessageWithOpts"] != scenarioIssueMaterial || n.count("getFeeForMessage") != 0 {
				t.Errorf("GetFeeForMessageWithOpts = %+v, getFeeForMessage calls=%d; want BLOCKED material without request", got["solana/GetFeeForMessageWithOpts"], n.count("getFeeForMessage"))
			}
		}},
		{name: "null_fee_pass", node: &scenarioSOLNode{genesis: scenarioSOLGenesis, sigCount: 1, nullFee: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, _ int) {
			allPass(t, got)
			if n.count("getFeeForMessage") != 1 {
				t.Errorf("getFeeForMessage calls=%d; want 1", n.count("getFeeForMessage"))
			}
		}},
		{name: "simulate_without_sigverify", node: &scenarioSOLNode{genesis: scenarioSOLGenesis, sigCount: 1}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, _ int) {
			var tx string
			if p := n.params["simulateTransaction"]; len(p) > 0 {
				_ = json.Unmarshal(p[0], &tx)
			}
			opts := n.opts("simulateTransaction")
			if n.count("simulateTransaction") != 1 || tx != base64.StdEncoding.EncodeToString(scenarioSOLRaw(1)) || opts == nil || opts["sigVerify"] == true {
				t.Errorf("simulate calls=%d tx=%q opts=%v; want one fetched-bytes simulation with sigVerify false", n.count("simulateTransaction"), tx, opts)
			}
			if got["solana/SimulateRawTransactionWithOpts"] != pass {
				t.Errorf("SimulateRawTransactionWithOpts = %+v; want PASS", got["solana/SimulateRawTransactionWithOpts"])
			}
		}},
		// LS-11: node error text carrying synthetic secrets is classified, not retained.
		{name: "redacted_rpc_error", node: &scenarioSOLNode{genesis: scenarioSOLGenesis, sigCount: 1, balanceErr: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioSOLNode, _ int) {
			if got["solana/GetBalanceWithOpts"] != (scenarioIssue{"FAIL", "rpc"}) || n.count("getBalance") != 1 {
				t.Errorf("GetBalanceWithOpts = %+v, getBalance calls=%d; want FAIL rpc after one request", got["solana/GetBalanceWithOpts"], n.count("getBalance"))
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, requests := scenarioSOLRun(tc.node)
			scenarioRedacted(t, got)
			ids := scenarioReadIDs("solana")
			if len(got) != len(ids) || tc.node.count("sendTransaction") != 0 {
				t.Errorf("result covers %d IDs (want %d); sendTransaction calls=%d", len(got), len(ids), tc.node.count("sendTransaction"))
			}
			if _, ok := got["solana/SendEncodedTransactionWithOpts"]; ok {
				t.Error("submit entry must not be reported by the Solana runner")
			}
			if tc.node.badVersion != 0 || tc.node.missingAuth != 0 {
				t.Errorf("non-2.0 requests=%d, requests without Authorization=%d", tc.node.badVersion, tc.node.missingAuth)
			}
			tc.check(t, got, tc.node, requests)
		})
	}
}
