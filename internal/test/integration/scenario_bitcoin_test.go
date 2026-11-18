package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	scenarioBTCGenesis  = "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f"
	scenarioBTCHead     = 200
	scenarioBTCCoinbase = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	scenarioBTCRawCB    = "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0403c8000effffffff0100f2052a01000000160014000000000000000000000000000000000000000000000000"
)

// scenarioBTCNode is a deterministic in-memory Bitcoin Core node. It expects
// version ("1.0" when empty) and answers in Bitcoin Core's matching dialect:
// 1.0 carries both result and error, 2.0 carries exactly one of them.
type scenarioBTCNode struct {
	mu                               sync.Mutex
	chain, genesis, version          string
	tamperHeight, mempoolOK, allowed bool
	mempoolCode                      int
	badVersion, missingAuth          int
	log                              []string
}

func scenarioBTCHash(height int64) string {
	if height == 0 {
		return scenarioBTCGenesis
	}
	return fmt.Sprintf("%064x", 0xb10c0000+height)
}

func (n *scenarioBTCNode) Do(req *http.Request) (*http.Response, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if req.Header.Get("Authorization") != "Basic "+scenarioHeaderSecret {
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
	want := n.version
	if want == "" {
		want = "1.0"
	}
	if in.Version != want {
		n.badVersion++
	}
	params, _ := json.Marshal(in.Params)
	n.log = append(n.log, in.Method+" "+string(params))
	result, code := n.dispatch(in.Method, in.Params)
	env := map[string]any{"id": in.ID, "result": result, "error": nil}
	if code != 0 {
		env["result"], env["error"] = nil, map[string]any{"code": code, "message": "rejected " + scenarioPathSecret}
	}
	if want == "2.0" {
		env["jsonrpc"] = "2.0"
		if code != 0 {
			delete(env, "result")
		} else {
			delete(env, "error")
		}
	}
	body, _ := json.Marshal(env)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
}

func (n *scenarioBTCNode) dispatch(method string, params []json.RawMessage) (any, int) {
	var hash string
	var height int64
	if len(params) > 0 {
		_ = json.Unmarshal(params[0], &hash)
		_ = json.Unmarshal(params[0], &height)
	}
	switch method {
	case "getblockchaininfo":
		return map[string]any{"chain": n.chain, "blocks": scenarioBTCHead, "headers": scenarioBTCHead, "bestblockhash": scenarioBTCHash(scenarioBTCHead), "pruned": false}, 0
	case "getblockcount":
		return scenarioBTCHead, 0
	case "getblockhash":
		if height == 0 {
			return n.genesis, 0
		}
		return scenarioBTCHash(height), 0
	case "getblock":
		var h int64
		if _, err := fmt.Sscanf(hash, "%x", &h); err != nil || hash == scenarioBTCGenesis {
			return nil, -5
		}
		h -= 0xb10c0000
		if n.tamperHeight {
			h++
		}
		return map[string]any{"hash": hash, "height": h, "confirmations": scenarioBTCHead - h + 1, "time": 1_700_000_000, "tx": []string{scenarioBTCCoinbase}, "previousblockhash": scenarioBTCHash(h - 1)}, 0
	case "getrawtransaction", "gettxout":
		if hash != scenarioBTCCoinbase {
			return nil, -5
		}
		spk := map[string]any{"hex": "0014" + strings.Repeat("00", 20), "type": "witness_v0_keyhash"}
		if method == "gettxout" {
			return map[string]any{"bestblock": scenarioBTCHash(scenarioBTCHead), "confirmations": 7, "value": 50.0, "scriptPubKey": spk, "coinbase": true}, 0
		}
		return map[string]any{"txid": hash, "hash": hash, "hex": scenarioBTCRawCB, "version": 1, "locktime": 0, "size": 100, "vsize": 100, "weight": 400,
			"vin": []any{map[string]any{"coinbase": "03c8000e", "sequence": 4294967295}}, "vout": []any{map[string]any{"value": 50.0, "n": 0, "scriptPubKey": spk}}}, 0
	case "getmempoolentry":
		if n.mempoolOK {
			return map[string]any{"vsize": 100, "weight": 400, "time": 1_700_000_000, "height": scenarioBTCHead, "fees": map[string]any{"base": 0.00001}, "depends": []string{}, "spentby": []string{}}, 0
		}
		if n.mempoolCode != 0 {
			return nil, n.mempoolCode
		}
		return nil, -5
	case "testmempoolaccept":
		item := map[string]any{"txid": scenarioBTCCoinbase, "wtxid": scenarioBTCCoinbase, "allowed": n.allowed}
		if !n.allowed {
			item["reject-reason"] = "coinbase"
		}
		return []any{item}, 0
	case "estimatesmartfee":
		return map[string]any{"feerate": 0.00012, "blocks": 6}, 0
	}
	return nil, -32601
}

func (n *scenarioBTCNode) count(prefix string) int {
	c := 0
	for _, l := range n.log {
		if strings.HasPrefix(l, prefix) {
			c++
		}
	}
	return c
}

func scenarioBTCRun(max int, version string, node *scenarioBTCNode) (map[string]scenarioIssue, int) {
	chainCfg := scenarioChainConfig{
		Endpoint:       "https://bitcoin.invalid/" + scenarioPathSecret,
		Headers:        map[string]string{"Authorization": "Basic " + scenarioHeaderSecret},
		Params:         map[string]string{"expected_chain": "main", "expected_genesis": scenarioBTCGenesis},
		RequestTimeout: time.Minute,
	}
	if version != "" {
		chainCfg.Params["jsonrpc_version"] = version
	}
	cfg := scenarioConfig{MaxRequests: max, MinInterval: time.Millisecond, TotalTimeout: time.Minute, RequiredChains: []string{"bitcoin"}, Chains: map[string]scenarioChainConfig{"bitcoin": chainCfg}}
	budget := newScenarioBudget(node, max, 0, func(context.Context, time.Duration) error { return nil })
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return runScenarioBitcoin(ctx, cfg, chainCfg, budget), budget.Requests()
}

func TestScenarioBitcoin(t *testing.T) {
	pass := scenarioIssue{"PASS", ""}
	identityOnly := func(mismatched string) func(*testing.T, map[string]scenarioIssue, *scenarioBTCNode, int) {
		return func(t *testing.T, got map[string]scenarioIssue, n *scenarioBTCNode, requests int) {
			for _, l := range n.log {
				if !strings.HasPrefix(l, "getblockchaininfo") && l != "getblockhash [0]" {
					t.Errorf("non-identity request after mismatch: %q", l)
				}
			}
			if requests == 0 || requests > 2 || len(n.log) != requests {
				t.Errorf("requests=%d log=%q; want only identity requests", requests, n.log)
			}
			if got[mismatched].Status != "FAIL" {
				t.Errorf("%s = %+v; want FAIL", mismatched, got[mismatched])
			}
			for _, id := range scenarioReadIDs("bitcoin") {
				if id != "bitcoin/GetBlockchainInfo" && id != "bitcoin/GetBlockHash" && got[id].Status != "BLOCKED" {
					t.Errorf("%s = %+v after identity mismatch; want BLOCKED", id, got[id])
				}
			}
		}
	}
	wantFail := func(id string) func(*testing.T, map[string]scenarioIssue, *scenarioBTCNode, int) {
		return func(t *testing.T, got map[string]scenarioIssue, _ *scenarioBTCNode, _ int) {
			if got[id].Status != "FAIL" {
				t.Errorf("%s = %+v; want FAIL", id, got[id])
			}
		}
	}
	for _, tc := range []struct {
		name    string
		max     int
		version string
		node    *scenarioBTCNode
		check   func(t *testing.T, got map[string]scenarioIssue, n *scenarioBTCNode, requests int)
	}{
		// LS-08: consistent replay; mempool -5 and a rejected coinbase precheck are PASS.
		{name: "replay", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis}},
		{name: "replay_jsonrpc_1_explicit", version: "1.0", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis}},
		{name: "replay_jsonrpc_2", version: "2.0", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis, version: "2.0"}},
		{name: "jsonrpc_version_invalid", version: "1.1", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioBTCNode, requests int) {
			if requests != 0 || len(n.log) != 0 {
				t.Errorf("requests=%d log=%q; want none for an invalid jsonrpc_version", requests, n.log)
			}
			for _, id := range scenarioReadIDs("bitcoin") {
				if got[id] != (scenarioIssue{"FAIL", "configuration"}) {
					t.Errorf("%s = %+v; want FAIL configuration", id, got[id])
				}
			}
		}},
		// LS-06: identity mismatch sends no request beyond the identity pair.
		{name: "chain_mismatch", node: &scenarioBTCNode{chain: "test", genesis: scenarioBTCGenesis}, check: identityOnly("bitcoin/GetBlockchainInfo")},
		{name: "genesis_mismatch", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCHash(1)}, check: identityOnly("bitcoin/GetBlockHash")},
		{name: "block_height_tampered", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis, tamperHeight: true}, check: wantFail("bitcoin/GetBlock")},
		{name: "mempool_entry_found", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis, mempoolOK: true}, check: wantFail("bitcoin/GetMempoolEntry")},
		{name: "mempool_entry_wrong_code", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis, mempoolCode: -8}, check: wantFail("bitcoin/GetMempoolEntry")},
		{name: "coinbase_precheck_allowed", node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis, allowed: true}, check: wantFail("bitcoin/TestMempoolAccept")},
		// LS-04/LS-11: the budget stops the next request and output stays redacted.
		{name: "budget_exhausted", max: 3, node: &scenarioBTCNode{chain: "main", genesis: scenarioBTCGenesis}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioBTCNode, requests int) {
			if requests != 3 || len(n.log) != 3 {
				t.Errorf("requests=%d downstream=%d; want 3", requests, len(n.log))
			}
			budgeted := false
			for id, issue := range got {
				budgeted = budgeted || issue == (scenarioIssue{"BLOCKED", "budget"})
				if issue != pass && issue.Status != "BLOCKED" {
					t.Errorf("%s = %+v; want PASS or BLOCKED under budget", id, issue)
				}
			}
			if !budgeted {
				t.Errorf("no entry is BLOCKED budget: %+v", got)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			max := tc.max
			if max == 0 {
				max = 1000
			}
			got, requests := scenarioBTCRun(max, tc.version, tc.node)
			scenarioRedacted(t, got)
			ids := scenarioReadIDs("bitcoin")
			if len(got) != len(ids) || tc.node.count("sendrawtransaction") != 0 {
				t.Errorf("result covers %d IDs (want %d); sendrawtransaction calls=%d", len(got), len(ids), tc.node.count("sendrawtransaction"))
			}
			if _, ok := got["bitcoin/SendRawTransaction"]; ok {
				t.Error("submit entry must not be reported by the Bitcoin runner")
			}
			if tc.node.badVersion != 0 || tc.node.missingAuth != 0 {
				t.Errorf("wrong-version requests=%d, requests without Authorization=%d", tc.node.badVersion, tc.node.missingAuth)
			}
			if tc.check != nil {
				tc.check(t, got, tc.node, requests)
				return
			}
			for _, id := range ids {
				if got[id] != pass {
					t.Errorf("%s = %+v; want PASS", id, got[id])
				}
			}
			if tc.node.count("getmempoolentry") != 1 || tc.node.count(`testmempoolaccept [["`+scenarioBTCRawCB+`"]]`) != 1 {
				t.Errorf("log=%q; want one getmempoolentry and one coinbase testmempoolaccept", tc.node.log)
			}
		})
	}
}
