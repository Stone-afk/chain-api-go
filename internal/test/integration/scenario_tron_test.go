package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	scenarioTRXGenesis  = "00000000000000001ebf88508a03865c71d452e25f4d51194196a1d22b6653dc"
	scenarioTRXHead     = 70_000_020
	scenarioTRXSolid    = 70_000_000
	scenarioTRXTxID     = "7777777777777777777777777777777777777777777777777777777777777777"
	scenarioTRXAccount  = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
	scenarioTRXContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	scenarioTRXHeader   = "TRON-PRO-API-KEY"
)

func scenarioTRXBlockID(height int64) string {
	if height == 0 {
		return scenarioTRXGenesis
	}
	return fmt.Sprintf("%016x%048x", height, 0x7e0)
}

// scenarioTRXNode is a deterministic in-memory java-tron HTTP API node.
type scenarioTRXNode struct {
	mu                         sync.Mutex
	genesis                    string
	emptyBlock, tamperSolidRaw bool
	reorderRaw                 bool
	rejectEnergy, proto3Energy bool
	missingHeader, badPath     int
	log                        []string
}

func (n *scenarioTRXNode) Do(req *http.Request) (*http.Response, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if req.Header.Get(scenarioTRXHeader) != scenarioHeaderSecret {
		n.missingHeader++
	}
	path, ok := strings.CutPrefix(req.URL.Path, "/"+scenarioPathSecret)
	if !ok || req.Method != http.MethodPost {
		n.badPath++
	}
	raw, _ := io.ReadAll(req.Body)
	var in map[string]any
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	compact, _ := json.Marshal(in)
	n.log = append(n.log, path+" "+string(compact))
	body, _ := json.Marshal(n.dispatch(path, in))
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func (n *scenarioTRXNode) tx(tamper bool) map[string]any {
	rawData := map[string]any{"ref_block_bytes": "1170", "ref_block_hash": "0b1c2d3e4f5a6b7c", "expiration": 1_700_000_060_000, "timestamp": 1_700_000_000_000,
		"contract": []any{map[string]any{"type": "TransferContract", "parameter": map[string]any{"value": map[string]any{"amount": 1, "owner_address": scenarioTRXAccount, "to_address": scenarioTRXContract}}}}}
	rawHex := "0a021170"
	if tamper {
		rawData["expiration"] = 1_700_000_090_000
		rawHex = "0a021171"
	}
	return map[string]any{"txID": scenarioTRXTxID, "raw_data": rawData, "raw_data_hex": rawHex, "signature": []string{strings.Repeat("ab", 65)}, "ret": []any{map[string]any{"contractRet": "SUCCESS"}}}
}

// scenarioTRXReversed re-encodes an object with its top-level keys in reverse
// order, as java-tron does between block and by-ID transaction encodings.
func scenarioTRXReversed(v map[string]any) json.RawMessage {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		key, _ := json.Marshal(k)
		val, _ := json.Marshal(v[k])
		b.Write(key)
		b.WriteByte(':')
		b.Write(val)
	}
	b.WriteByte('}')
	return b.Bytes()
}

func (n *scenarioTRXNode) block(height int64) map[string]any {
	b := map[string]any{"blockID": scenarioTRXBlockID(height), "block_header": map[string]any{"raw_data": map[string]any{"number": height, "timestamp": 1_700_000_000_000}}}
	if height == 0 {
		b["blockID"] = n.genesis
	} else if !n.emptyBlock {
		b["transactions"] = []any{n.tx(false)}
	}
	return b
}

func (n *scenarioTRXNode) dispatch(path string, in map[string]any) any {
	num, _ := in["num"].(float64)
	id, _ := in["value"].(string)
	info := map[string]any{"id": scenarioTRXTxID, "fee": 0, "blockNumber": scenarioTRXSolid, "blockTimeStamp": 1_700_000_000_000, "receipt": map[string]any{"net_usage": 268}}
	switch path {
	case "/wallet/getblockbynum", "/walletsolidity/getblockbynum":
		return n.block(int64(num))
	case "/wallet/getnowblock":
		return n.block(scenarioTRXHead)
	case "/walletsolidity/getnowblock":
		return n.block(scenarioTRXSolid)
	case "/wallet/getnodeinfo":
		return map[string]any{"block": fmt.Sprintf("Num:%d,ID:%s", scenarioTRXHead, scenarioTRXBlockID(scenarioTRXHead)), "solidityBlock": fmt.Sprintf("Num:%d,ID:%s", scenarioTRXSolid, scenarioTRXBlockID(scenarioTRXSolid)), "currentConnectCount": 8}
	case "/wallet/getaccount", "/walletsolidity/getaccount":
		return map[string]any{"address": scenarioTRXAccount, "balance": 1_000_000}
	case "/wallet/getaccountresource":
		return map[string]any{"freeNetLimit": 600, "TotalNetLimit": 43_200_000_000, "TotalNetWeight": 1, "TotalEnergyLimit": 90_000_000_000, "TotalEnergyWeight": 1}
	case "/wallet/getchainparameters":
		return map[string]any{"chainParameter": []any{map[string]any{"key": "getEnergyFee", "value": 420}}}
	case "/wallet/gettransactionbyid", "/walletsolidity/gettransactionbyid":
		if id != scenarioTRXTxID {
			return map[string]any{}
		}
		tx := n.tx(n.tamperSolidRaw && strings.HasPrefix(path, "/walletsolidity/"))
		if n.reorderRaw {
			tx["raw_data"] = scenarioTRXReversed(tx["raw_data"].(map[string]any))
		}
		return tx
	case "/wallet/gettransactioninfobyid", "/walletsolidity/gettransactioninfobyid":
		if id != scenarioTRXTxID {
			return map[string]any{}
		}
		return info
	case "/wallet/triggerconstantcontract":
		return map[string]any{"result": map[string]any{"result": true}, "constant_result": []string{strings.Repeat("0", 63) + "1"}, "energy_used": 900}
	case "/wallet/estimateenergy":
		if n.proto3Energy {
			return map[string]any{"result": map[string]any{"code": "CONTRACT_VALIDATE_ERROR", "message": "this node does not support estimate energy"}}
		}
		if n.rejectEnergy {
			return map[string]any{"result": map[string]any{"result": false, "code": "OTHER_ERROR", "message": "estimate energy disabled " + scenarioPathSecret + " " + scenarioHeaderSecret}}
		}
		return map[string]any{"result": map[string]any{"result": true}, "energy_required": 1500}
	}
	return map[string]any{"Error": "unknown path " + scenarioHeaderSecret}
}

func (n *scenarioTRXNode) count(prefix string) int {
	c := 0
	for _, l := range n.log {
		if strings.HasPrefix(l, prefix) {
			c++
		}
	}
	return c
}

func scenarioTRXRun(max int, known []string, node *scenarioTRXNode) (map[string]scenarioIssue, int) {
	chainCfg := scenarioChainConfig{
		Endpoint:       "https://tron.invalid/" + scenarioPathSecret,
		Headers:        map[string]string{scenarioTRXHeader: scenarioHeaderSecret},
		Params:         map[string]string{"expected_genesis": scenarioTRXGenesis, "account": scenarioTRXAccount, "contract": scenarioTRXContract, "function_selector": "totalSupply()"},
		RequestTimeout: time.Minute,
	}
	cfg := scenarioConfig{MaxRequests: max, MinInterval: time.Millisecond, TotalTimeout: time.Minute, RequiredChains: []string{"tron"}, KnownUnsupported: known, Chains: map[string]scenarioChainConfig{"tron": chainCfg}}
	budget := newScenarioBudget(node, max, 0, func(context.Context, time.Duration) error { return nil })
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return runScenarioTron(ctx, cfg, chainCfg, budget), budget.Requests()
}

func TestScenarioTron(t *testing.T) {
	pass := scenarioIssue{"PASS", ""}
	txLookups := []string{"tron/GetTransactionByID", "tron/GetTransactionInfoByID", "tron/GetSolidifiedTransactionByID", "tron/GetSolidifiedTransactionInfoByID"}
	allPass := func(t *testing.T, got map[string]scenarioIssue) {
		for _, id := range scenarioReadIDs("tron") {
			if got[id] != pass {
				t.Errorf("%s = %+v; want PASS", id, got[id])
			}
		}
	}
	for _, tc := range []struct {
		name  string
		max   int
		known []string
		node  *scenarioTRXNode
		check func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, requests int)
	}{
		// LS-10: consistent replay of fifteen reads plus one RawPost re-reading block 0.
		{name: "replay_all_pass", node: &scenarioTRXNode{genesis: scenarioTRXGenesis}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, requests int) {
			allPass(t, got)
			if n.count(`/wallet/getblockbynum {"num":0}`) != 2 {
				t.Errorf("block 0 reads=%d; want identity plus one RawPost", n.count(`/wallet/getblockbynum {"num":0}`))
			}
			if n.count("/wallet/estimateenergy") != 1 || n.count("/wallet/triggerconstantcontract") != 1 {
				t.Errorf("log=%q; want one estimateenergy and one triggerconstantcontract", n.log)
			}
			if requests == 0 || requests != len(n.log) {
				t.Errorf("requests=%d downstream=%d", requests, len(n.log))
			}
		}},
		// LS-06: genesis mismatch stops before any other request.
		{name: "genesis_mismatch_zero_requests", node: &scenarioTRXNode{genesis: scenarioTRXBlockID(1)}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, requests int) {
			if requests != 1 || len(n.log) != 1 || n.log[0] != `/wallet/getblockbynum {"num":0}` {
				t.Errorf("requests=%d log=%q; want only the genesis block read", requests, n.log)
			}
			if got["tron/GetBlockByNum"].Status != "FAIL" {
				t.Errorf("GetBlockByNum = %+v; want FAIL", got["tron/GetBlockByNum"])
			}
			for _, id := range scenarioReadIDs("tron") {
				if id != "tron/GetBlockByNum" && got[id].Status != "BLOCKED" {
					t.Errorf("%s = %+v after genesis mismatch; want BLOCKED", id, got[id])
				}
			}
		}},
		{name: "solidified_raw_data_tampered", node: &scenarioTRXNode{genesis: scenarioTRXGenesis, tamperSolidRaw: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, _ int) {
			if got["tron/GetSolidifiedTransactionByID"].Status != "FAIL" || n.count("/walletsolidity/gettransactionbyid") != 1 {
				t.Errorf("GetSolidifiedTransactionByID = %+v after %d requests; want FAIL after one", got["tron/GetSolidifiedTransactionByID"], n.count("/walletsolidity/gettransactionbyid"))
			}
		}},
		{name: "by_id_raw_data_key_order", node: &scenarioTRXNode{genesis: scenarioTRXGenesis, reorderRaw: true}, check: func(t *testing.T, got map[string]scenarioIssue, _ *scenarioTRXNode, _ int) {
			for _, id := range []string{"tron/GetTransactionByID", "tron/GetSolidifiedTransactionByID"} {
				if got[id].Status != "PASS" {
					t.Errorf("%s = %+v with reordered raw_data keys; want PASS", id, got[id])
				}
			}
		}},
		{name: "estimate_energy_proto3_rejection_known_unsupported", known: []string{"tron/EstimateEnergy"}, node: &scenarioTRXNode{genesis: scenarioTRXGenesis, proto3Energy: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, _ int) {
			if got["tron/EstimateEnergy"] != (scenarioIssue{"BLOCKED", "capability"}) || n.count("/wallet/estimateenergy") != 1 {
				t.Errorf("EstimateEnergy = %+v after %d requests; want BLOCKED capability after one", got["tron/EstimateEnergy"], n.count("/wallet/estimateenergy"))
			}
		}},
		{name: "empty_block_material", node: &scenarioTRXNode{genesis: scenarioTRXGenesis, emptyBlock: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, _ int) {
			for _, id := range txLookups {
				if got[id] != scenarioIssueMaterial {
					t.Errorf("%s = %+v; want BLOCKED material", id, got[id])
				}
			}
			if c := n.count("/wallet/gettransaction") + n.count("/walletsolidity/gettransaction"); c != 0 {
				t.Errorf("transaction lookups=%d on empty block; want 0", c)
			}
		}},
		{name: "estimate_energy_rejected_unlisted", node: &scenarioTRXNode{genesis: scenarioTRXGenesis, rejectEnergy: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, _ int) {
			if got["tron/EstimateEnergy"].Status != "FAIL" || n.count("/wallet/estimateenergy") != 1 {
				t.Errorf("EstimateEnergy = %+v after %d requests; want FAIL after one", got["tron/EstimateEnergy"], n.count("/wallet/estimateenergy"))
			}
		}},
		{name: "estimate_energy_rejected_known_unsupported", known: []string{"tron/EstimateEnergy"}, node: &scenarioTRXNode{genesis: scenarioTRXGenesis, rejectEnergy: true}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, _ int) {
			if got["tron/EstimateEnergy"] != (scenarioIssue{"BLOCKED", "capability"}) || n.count("/wallet/estimateenergy") != 1 {
				t.Errorf("EstimateEnergy = %+v after %d requests; want BLOCKED capability after one", got["tron/EstimateEnergy"], n.count("/wallet/estimateenergy"))
			}
		}},
		// LS-11: the YAML header reaches every request while output keeps no header value.
		{name: "header_sent_value_redacted", node: &scenarioTRXNode{genesis: scenarioTRXGenesis}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, requests int) {
			if requests == 0 || len(n.log) != requests || n.missingHeader != 0 {
				t.Errorf("requests=%d downstream=%d missing header=%d; want header on every request", requests, len(n.log), n.missingHeader)
			}
			b, _ := json.Marshal(got)
			if strings.Contains(string(b)+fmt.Sprintf("%#v", got), scenarioHeaderSecret) {
				t.Error("output retained header value")
			}
		}},
		// LS-04: the budget stops the next request.
		{name: "budget_exhausted", max: 3, node: &scenarioTRXNode{genesis: scenarioTRXGenesis}, check: func(t *testing.T, got map[string]scenarioIssue, n *scenarioTRXNode, requests int) {
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
			got, requests := scenarioTRXRun(max, tc.known, tc.node)
			scenarioRedacted(t, got)
			ids := scenarioReadIDs("tron")
			if len(got) != len(ids) || tc.node.count("/wallet/broadcast") != 0 {
				t.Errorf("result covers %d IDs (want %d); broadcast calls=%d", len(got), len(ids), tc.node.count("/wallet/broadcast"))
			}
			if _, ok := got["tron/BroadcastTransaction"]; ok {
				t.Error("submit entry must not be reported by the Tron runner")
			}
			if tc.node.badPath != 0 || tc.node.missingHeader != 0 {
				t.Errorf("requests off endpoint path or not POST=%d, requests without header=%d", tc.node.badPath, tc.node.missingHeader)
			}
			tc.check(t, got, tc.node, requests)
		})
	}
}
