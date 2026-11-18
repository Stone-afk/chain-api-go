package integration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/bitcoin"
	"github.com/Stone-afk/chain-api-go/bnb"
	"github.com/Stone-afk/chain-api-go/ethereum"
	"github.com/Stone-afk/chain-api-go/solana"
	"github.com/Stone-afk/chain-api-go/transport"
	"github.com/Stone-afk/chain-api-go/tron"
	"github.com/ethereum/go-ethereum/core/types"
	ns "github.com/solana-foundation/solana-go/v2"
)

// Fixed independent acceptance catalog: never derived from the runner's output.
func contractCatalog(chain string) []string {
	var methods string
	switch chain {
	case "ethereum", "bnb":
		methods = "ChainID BlockNumber BlockByNumber BalanceAt NonceAt CodeAt TransactionByHash TransactionReceipt FilterLogs CallContract EstimateGas SuggestGasPrice FeeHistory"
	case "bitcoin":
		methods = "GetBlockchainInfo GetBlockHash GetBlockCount GetBlock GetRawTransaction GetTxOut GetMempoolEntry EstimateSmartFee TestMempoolAccept"
	case "solana":
		methods = "GetGenesisHash GetSlotWithOpts GetBlockWithOpts GetBalanceWithOpts GetAccountInfoWithOpts GetLatestBlockhashWithOpts GetSignatureStatuses GetTransaction GetFeeForMessageWithOpts SimulateRawTransactionWithOpts"
	case "tron":
		methods = "GetBlockByNum GetNodeInfo GetNowBlock GetAccount GetAccountResource GetChainParameters GetTransactionByID GetTransactionInfoByID GetSolidifiedAccount GetSolidifiedNowBlock GetSolidifiedBlockByNum GetSolidifiedTransactionByID GetSolidifiedTransactionInfoByID TriggerConstantContract EstimateEnergy"
	}
	return strings.Fields(methods)
}
func contractChains() []string { return []string{"ethereum", "bnb", "bitcoin", "solana", "tron"} }
func contractJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
func contractSHA(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func contractID(chain string, n int) string {
	if chain == "solana" {
		if n == 7 {
			return (ns.Hash{9}).String()
		}
		return (ns.Signature{byte(n + 2)}).String()
	}
	s := strings.Repeat(fmt.Sprint(n+1), 64)
	if chain == "ethereum" || chain == "bnb" {
		if n == 7 {
			var header types.Header
			if err := json.Unmarshal(contractNativeHeader(), &header); err != nil {
				panic(err)
			}
			return header.Hash().Hex()
		}
		var tx types.Transaction
		if err := json.Unmarshal(contractNativeTransaction(n), &tx); err != nil {
			panic(err)
		}
		return tx.Hash().Hex()
	}
	return s
}
func contractIdentity(chain string) networkIdentity {
	switch chain {
	case "ethereum", "bnb":
		return networkIdentity{ChainID: "1"}
	case "bitcoin":
		return networkIdentity{BitcoinChain: "regtest", GenesisHash: strings.Repeat("a", 64)}
	case "solana":
		return networkIdentity{GenesisHash: strings.Repeat("1", 32)}
	default:
		return networkIdentity{TronGenesisBlockID: strings.Repeat("b", 64)}
	}
}

const contractObserved = "2098-01-01T00:00:00Z"
const contractExpires = "2100-01-01T00:00:00Z"
const contractSecret = "SYNTHETIC_PRIVATE_MARKER"

// All node responses and expected fields below are literal synthetic data, independent of SDK results.
func contractSample(chain string, n int) readSample {
	id := contractID(chain, n)
	block := contractID(chain, 7)
	categories := []string{"native", "token", "failed"}
	if chain == "bnb" {
		categories[2] = "contract"
	}
	if chain == "bitcoin" {
		categories = []string{"multi_output", "segwit", "structure"}
	}
	s := readSample{SampleID: fmt.Sprintf("sample-%d", n), Category: categories[n], SourceURL: "https://explorer.invalid/tx", ObservedAt: contractObserved, TransactionID: id, BlockHash: block, HeightOrSlot: 8, Addresses: []string{"public"}, Requirements: []string{"archive"}}
	var expected string
	var calls []string
	switch chain {
	case "ethereum", "bnb":
		expected = fmt.Sprintf(`{"transaction":%s,"receipt":%s,"block_hash":%q,"block_number":8,"parent_hash":%q}`, contractNativeTransaction(n), contractNativeReceipt(id, block), block, "0x"+strings.Repeat("0", 63)+"1")
		calls = []string{"TransactionByHash", "TransactionReceipt", "BlockByNumber"}
	case "bitcoin":
		expected = fmt.Sprintf(`{"transaction":{"txid":%q,"hash":"wtx","confirmations":2,"vin":[{"sequence":1,"txinwitness":["aa"]}],"vout":[{"n":0,"value":1,"scriptPubKey":{"hex":"51","type":"pubkey"}}]},"block_hash":%q,"block_height":8,"txids":[%q,%q,%q]}`, id, block, contractID(chain, 0), contractID(chain, 1), contractID(chain, 2))
		calls = []string{"GetRawTransaction", "GetBlock"}
	case "solana":
		expected = fmt.Sprintf(`{"transaction":{"slot":8,"transaction":{"signatures":[%q],"message":{"accountKeys":["11111111111111111111111111111111"],"recentBlockhash":"11111111111111111111111111111111","header":{},"instructions":[]}},"meta":{"err":null,"fee":9007199254740993,"preBalances":[7],"postBalances":[6]},"version":0},"blockhash":%q,"previous_blockhash":"11111111111111111111111111111111","parent_slot":7}`, id, block)
		calls = []string{"GetTransaction", "GetBlockWithOpts"}
	case "tron":
		expected = fmt.Sprintf(`{"transaction":{"txID":%q,"raw_data":{"contract":[{"type":"TransferContract"}]},"signature":["aa"]},"transaction_info":{"id":%q,"blockNumber":8,"fee":1,"receipt":{"result":"SUCCESS","energy_usage":1},"log":[]},"block_id":%q}`, id, id, block)
		calls = []string{"GetTransactionByID", "GetTransactionInfoByID", "GetBlockByNum", "GetSolidifiedTransactionByID", "GetSolidifiedTransactionInfoByID"}
	}
	s.Expected = json.RawMessage(expected)
	s.FieldSources = map[string]string{}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(s.Expected, &fields)
	for k := range fields {
		s.FieldSources[k] = "browser:https://explorer.invalid/tx"
	}
	s.RPCPreflight = preflightRecord{Status: "PASS", ObservedAt: contractObserved, EndpointAlias: chain + "_read", NetworkIdentity: contractIdentity(chain), Methods: map[string]rpcExpectation{}}
	for _, m := range calls {
		s.RPCPreflight.Methods[m] = rpcExpectation{Outcome: "success", Expected: json.RawMessage("null")}
	}
	return s
}
func contractReadMaterial(chain string) (readManifest, dynamicInputs) {
	m := readManifest{ManifestVersion: 1, Chain: chain, Network: "fixture", NetworkIdentity: contractIdentity(chain), Samples: []readSample{contractSample(chain, 0), contractSample(chain, 1), contractSample(chain, 2)}}
	d := dynamicInputs{MaxResponseBytes: 65536, CodeSnapshot: strings.Repeat("c", 40), ObservedAt: contractObserved, ExpiresAt: contractExpires, Preflight: preflightRecord{Status: "PASS", ObservedAt: contractObserved, EndpointAlias: chain + "_read", NetworkIdentity: m.NetworkIdentity, Methods: map[string]rpcExpectation{}}}
	expected := map[string]string{}
	switch chain {
	case "ethereum", "bnb":
		d.Inputs = json.RawMessage(`{"address":"0x1111111111111111111111111111111111111111","at_block_number":8,"call":{},"estimate":{},"log_filter":{"FromBlock":8,"ToBlock":8,"Addresses":[],"Topics":[]},"fee_count":1,"fee_percentiles":[]}`)
		m.ManifestVersion = 2
		expected = map[string]string{"BalanceAt": `"1"`, "NonceAt": "1", "CodeAt": `"qg=="`, "FilterLogs": "[]", "CallContract": `"qg=="`, "FeeHistory": `{"OldestBlock":8,"BaseFee":[1],"GasUsedRatio":[0.5],"Reward":[]}`}
	case "bitcoin":
		d.Inputs = json.RawMessage(fmt.Sprintf(`{"txout":{"TxID":%q},"mempool_txid":%q,"estimate":{"ConfTarget":2},"preflight_payload":"aa"}`, m.Samples[0].TransactionID, m.Samples[0].TransactionID))
		expected = map[string]string{"GetTxOut": "true", "EstimateSmartFee": "true", "TestMempoolAccept": "false"}
	case "solana":
		m.ManifestVersion = 2
		d.Inputs = json.RawMessage(`{"account":"11111111111111111111111111111111","read":{"Commitment":"confirmed"},"blockhash":{"Commitment":"confirmed"},"account_options":{"Encoding":"base64"},"transaction_options":{"Encoding":"json","Commitment":"confirmed","MaxSupportedTransactionVersion":0},"block_options":{"Encoding":"json","Commitment":"confirmed","MaxSupportedTransactionVersion":0},"search_transaction_history":true,"fee_message":"YQ==","simulate_transaction":"YQ==","simulate_options":{}}`)
		expected = map[string]string{"GetAccountInfoWithOpts": `{"lamports":1,"owner":"11111111111111111111111111111111","data":["","base64"],"executable":false,"rentEpoch":0}`, "GetFeeForMessageWithOpts": "1", "SimulateRawTransactionWithOpts": "null"}
	case "tron":
		addr := "41" + strings.Repeat("1", 40)
		d.Inputs = json.RawMessage(fmt.Sprintf(`{"account":{"Address":%q,"Visible":false},"contract_call":{"OwnerAddress":%q,"ContractAddress":%q,"FunctionSelector":"balanceOf(address)","Parameter":"aa"}}`, addr, addr, addr))
		expected = map[string]string{"GetAccount": "true", "GetSolidifiedAccount": "true", "TriggerConstantContract": `["aa"]`}
	}
	for _, method := range contractCatalog(chain) {
		v := "null"
		if s, ok := expected[method]; ok {
			v = s
		}
		d.Preflight.Methods[method] = rpcExpectation{Outcome: "success", Expected: json.RawMessage(v)}
	}
	d.ManifestSHA256 = contractSHA(contractJSON(m))
	return m, d
}

type contractCaller func(context.Context, string, any, any) error

func (f contractCaller) Call(c context.Context, m string, p, o any) error { return f(c, m, p, o) }
func (f contractCaller) Post(c context.Context, m string, p, o any) error { return f(c, m, p, o) }

type contractDoer func(*http.Request) (*http.Response, error)

func (f contractDoer) Do(r *http.Request) (*http.Response, error) { return f(r) }

// The fake is only the external transport. Actual concrete SDK clients and runner cases execute.
func contractResponse(chain, method string, params json.RawMessage) json.RawMessage {
	n := 0
	for i := 0; i < 3; i++ {
		if strings.Contains(string(params), contractID(chain, i)) {
			n = i
			break
		}
	}
	s := contractSample(chain, n)
	var e map[string]json.RawMessage
	_ = json.Unmarshal(s.Expected, &e)
	switch method {
	case "eth_chainId":
		return json.RawMessage(`"0x1"`)
	case "eth_blockNumber":
		return json.RawMessage(`"0x9"`)
	case "eth_getBalance", "eth_getTransactionCount", "eth_estimateGas", "eth_gasPrice":
		return json.RawMessage(`"0x1"`)
	case "eth_getCode", "eth_call":
		return json.RawMessage(`"0xaa"`)
	case "eth_getLogs":
		return json.RawMessage(`[]`)
	case "eth_feeHistory":
		return json.RawMessage(`{"oldestBlock":"0x8","baseFeePerGas":["0x1"],"gasUsedRatio":[0.5],"reward":null}`)
	case "eth_getTransactionByHash", "getTransaction", "/wallet/gettransactionbyid", "/walletsolidity/gettransactionbyid":
		if chain == "ethereum" || chain == "bnb" {
			return json.RawMessage(strings.TrimSuffix(string(e["transaction"]), "}") + fmt.Sprintf(`,"blockHash":%q,"blockNumber":"0x8"}`, contractID(chain, 7)))
		}
		return e["transaction"]
	case "eth_getTransactionReceipt":
		return e["receipt"]
	case "eth_getBlockByNumber":
		return json.RawMessage(strings.TrimSuffix(string(contractNativeHeader()), "}") + fmt.Sprintf(`,"hash":%s,"transactions":[],"uncles":[]}`, e["block_hash"]))
	case "eth_getUncleByBlockHashAndIndex":
		return contractNativeHeader()
	case "getblockchaininfo":
		return json.RawMessage(`{"chain":"regtest"}`)
	case "getblockhash":
		return contractJSON(strings.Repeat("a", 64))
	case "getblockcount", "getSlot":
		return json.RawMessage(`9`)
	case "getrawtransaction":
		return json.RawMessage(strings.ReplaceAll(string(e["transaction"]), `"value":1`, `"value":0.00000001`))
	case "getblock":
		return json.RawMessage(fmt.Sprintf(`{"hash":%s,"height":8,"tx":[%q,%q,%q]}`, e["block_hash"], contractID(chain, 0), contractID(chain, 1), contractID(chain, 2)))
	case "gettxout":
		return json.RawMessage(`{"bestblock":"block","value":0.00000001}`)
	case "getmempoolentry":
		return json.RawMessage(`{"vsize":1,"fees":{"base":0.00000001}}`)
	case "estimatesmartfee":
		return json.RawMessage(`{"feerate":0.00000001,"blocks":2}`)
	case "testmempoolaccept":
		return json.RawMessage(`[ {"allowed":false,"reject-reason":"synthetic"} ]`)
	case "getGenesisHash":
		return contractJSON(strings.Repeat("1", 32))
	case "getBlock":
		return json.RawMessage(fmt.Sprintf(`{"blockhash":%s,"previousBlockhash":"11111111111111111111111111111111","parentSlot":7}`, e["blockhash"]))
	case "getBalance", "getFeeForMessage":
		return json.RawMessage(`{"context":{"slot":9},"value":1}`)
	case "getAccountInfo":
		return json.RawMessage(`{"context":{"slot":9},"value":{"lamports":1,"owner":"11111111111111111111111111111111","data":["","base64"],"executable":false,"rentEpoch":0}}`)
	case "getLatestBlockhash":
		return json.RawMessage(fmt.Sprintf(`{"context":{"slot":9},"value":{"blockhash":%q,"lastValidBlockHeight":2}}`, contractID("solana", 7)))
	case "getSignatureStatuses":
		return json.RawMessage(`{"context":{"slot":9},"value":[{"slot":8,"err":null,"confirmationStatus":"finalized"}]}`)
	case "simulateTransaction":
		return json.RawMessage(`{"context":{"slot":9},"value":{"err":null}}`)
	case "/wallet/getblockbynum":
		if strings.Contains(string(params), `"num":0`) {
			return contractJSON(map[string]string{"blockID": strings.Repeat("b", 64)})
		}
		fallthrough
	case "/walletsolidity/getblockbynum", "/wallet/getnowblock", "/walletsolidity/getnowblock":
		return json.RawMessage(fmt.Sprintf(`{"blockID":%s}`, e["block_id"]))
	case "/wallet/getnodeinfo":
		return json.RawMessage(`{"block":"head","solidityBlock":"solid"}`)
	case "/wallet/getaccount", "/walletsolidity/getaccount":
		return json.RawMessage(`{"address":"account","balance":0}`)
	case "/wallet/getaccountresource":
		return json.RawMessage(`{}`)
	case "/wallet/getchainparameters":
		return json.RawMessage(`{"chainParameter":[{"key":"fee","value":1}]}`)
	case "/wallet/gettransactioninfobyid", "/walletsolidity/gettransactioninfobyid":
		return e["transaction_info"]
	case "/wallet/triggerconstantcontract":
		return json.RawMessage(`{"result":{"result":true},"constant_result":["aa"]}`)
	case "/wallet/estimateenergy":
		return json.RawMessage(`{"result":{"result":true},"energy_required":1}`)
	case "eth_sendRawTransaction", "sendrawtransaction", "sendTransaction":
		return contractJSON(contractID(chain, 0))
	case "/wallet/broadcasttransaction":
		return json.RawMessage(fmt.Sprintf(`{"result":true,"txid":%q}`, contractID(chain, 0)))
	}
	return json.RawMessage(`null`)
}

// Literal wire catalog catches swapped/duplicate closures even when total counts match.
func contractWireCounts(chain string) map[string]int {
	var methods string
	switch chain {
	case "ethereum", "bnb":
		methods = "eth_chainId eth_blockNumber eth_getBlockByNumber eth_getBalance eth_getTransactionCount eth_getCode eth_getTransactionByHash eth_getTransactionReceipt eth_getLogs eth_call eth_estimateGas eth_gasPrice eth_feeHistory"
	case "bitcoin":
		methods = "getblockchaininfo getblockhash getblockcount getblock getrawtransaction gettxout getmempoolentry estimatesmartfee testmempoolaccept"
	case "solana":
		methods = "getGenesisHash getSlot getBlock getBalance getAccountInfo getLatestBlockhash getSignatureStatuses getTransaction getFeeForMessage simulateTransaction"
	case "tron":
		methods = "/wallet/getblockbynum /wallet/getnodeinfo /wallet/getnowblock /wallet/getaccount /wallet/getaccountresource /wallet/getchainparameters /wallet/gettransactionbyid /wallet/gettransactioninfobyid /walletsolidity/getaccount /walletsolidity/getnowblock /walletsolidity/getblockbynum /walletsolidity/gettransactionbyid /walletsolidity/gettransactioninfobyid /wallet/triggerconstantcontract /wallet/estimateenergy"
	}
	counts := map[string]int{}
	for _, m := range strings.Fields(methods) {
		counts[m] = 1
	}
	extra := map[string][]string{"ethereum": {"eth_getBlockByNumber", "eth_getTransactionByHash", "eth_getTransactionReceipt"}, "bnb": {"eth_getBlockByNumber", "eth_getTransactionByHash", "eth_getTransactionReceipt"}, "bitcoin": {"getblock", "getrawtransaction"}, "solana": {"getBlock", "getTransaction"}, "tron": {"/wallet/getblockbynum", "/wallet/gettransactionbyid", "/wallet/gettransactioninfobyid", "/walletsolidity/gettransactionbyid", "/walletsolidity/gettransactioninfobyid"}}
	for _, m := range extra[chain] {
		counts[m] += 3
	}
	return counts
}
func contractReadParams(t *testing.T, chain, method string, p any) {
	t.Helper()
	raw := string(contractJSON(p))
	switch method {
	case "eth_getBalance", "eth_getTransactionCount", "eth_getCode", "eth_call":
		if !strings.HasSuffix(raw, `,"0x8"]`) {
			t.Error("historical/dynamic call lost explicit fixed block")
		}
	case "eth_getLogs":
		if !strings.Contains(raw, `"fromBlock":"0x8"`) || !strings.Contains(raw, `"toBlock":"0x8"`) {
			t.Error("logs used non-frozen range")
		}
	case "getrawtransaction":
		if !strings.HasSuffix(raw, `,`+string(contractJSON(contractID(chain, 7)))+`]`) {
			t.Error("historical Bitcoin lookup omitted explicit blockhash")
		}
	case "getBlock", "getTransaction":
		if !strings.Contains(raw, `"encoding":"json"`) || !strings.Contains(raw, `"maxSupportedTransactionVersion":0`) || !strings.Contains(raw, `"commitment":"confirmed"`) {
			t.Error("Solana historical options not frozen")
		}
	case "getSignatureStatuses":
		if !strings.Contains(raw, `"searchTransactionHistory":true`) {
			t.Error("Solana status search omitted history")
		}
	}
}
func contractReadCases(t *testing.T, chain string, m readManifest, d dynamicInputs, call contractCaller) ([]harnessCase, error) {
	t.Helper()
	switch chain {
	case "ethereum":
		c, e := ethereum.New(ethereum.Config{RawURL: "http://fixture.invalid", ExpectedChainID: *big.NewInt(1)}, call)
		if e != nil {
			t.Fatal(e)
		}
		return ethereumReadCases(c, m, d)
	case "bnb":
		c, e := bnb.New(bnb.Config{RawURL: "http://fixture.invalid", ExpectedChainID: *big.NewInt(1)}, call)
		if e != nil {
			t.Fatal(e)
		}
		return bnbReadCases(c, m, d)
	case "bitcoin":
		c, e := bitcoin.New(call)
		if e != nil {
			t.Fatal(e)
		}
		return bitcoinReadCases(c, m, d)
	case "solana":
		c, e := solana.New(call)
		if e != nil {
			t.Fatal(e)
		}
		return solanaReadCases(c, m, d)
	default:
		c, e := tron.New(call)
		if e != nil {
			t.Fatal(e)
		}
		return tronReadCases(c, m, d)
	}
}
func contractRequired(chain string, broadcast bool) []string {
	names := contractCatalog(chain)
	if broadcast {
		names = []string{"identity", "submit", "lookup"}
	} else {
		names = append(names, "sample.1", "sample.2", "sample.3")
	}
	ids := make([]string, len(names))
	for i, n := range names {
		ids[i] = chain + "." + n
	}
	return ids
}
func contractHarness(chain string, cases []harnessCase, broadcast bool, budget int) harnessReport {
	now := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	deps := harnessDeps{Load: func() (harnessConfig, error) {
		return harnessConfig{Alias: chain, Endpoint: "https://fixture.invalid", Network: "fixture", MaxRequests: budget, MaxConcurrent: 1, MinInterval: time.Nanosecond, TotalTimeout: time.Hour}, nil
	}, LoadManifest: func() ([]byte, string, error) { return []byte(fixtureManifest), fixtureDigest, nil }, Now: func() time.Time { return now }, Wait: func(_ context.Context, d time.Duration) error { now = now.Add(d); return nil }}
	return runHarness(context.Background(), true, deps, contractRequired(chain, broadcast), cases)
}

func contractDisabled(t *testing.T, broadcast bool, first, second string, wantKeys []string) {
	t.Helper()
	var keys []string
	forbidden := func() { t.Fatal("disabled runner performed non-switch work") }
	deps := runnerDeps{LookupEnv: func(k string) (string, bool) {
		keys = append(keys, k)
		switch k {
		case "CHAIN_API_GO_LIVE_TEST":
			return first, true
		case "CHAIN_API_GO_LIVE_BROADCAST":
			return second, true
		default:
			forbidden()
			return "", false
		}
	}, ReadFile: func(string) ([]byte, error) { forbidden(); return nil, nil }, Now: func() time.Time { forbidden(); return time.Time{} }, Wait: func(context.Context, time.Duration) error { forbidden(); return nil }, NewDoer: func(harnessConfig) (transport.HTTPDoer, func(), error) { forbidden(); return nil, nil, nil }, WriteReadReport: func([]byte) error { forbidden(); return nil }, ReadBroadcastJournal: func() ([]byte, error) { forbidden(); return nil, nil }, AppendBroadcastRecord: func([]byte) error { forbidden(); return nil }}
	mode := "read"
	var got suiteReport
	if broadcast {
		mode = "broadcast"
		got = runLiveBroadcast(context.Background(), deps)
	} else {
		got = runLiveRead(context.Background(), deps)
	}
	if got.Status != "NOT_RUN" || got.Mode != mode || !reflect.DeepEqual(keys, wantKeys) || got.StartedAt != "" || got.FinishedAt != "" || len(got.Chains) != 0 {
		t.Errorf("switch boundary/status wrong: keys=%v report=%+v", keys, got)
	}
}
func TestReadContractDisabled(t *testing.T) {
	for _, v := range []string{"", "0", "true"} {
		t.Run(v, func(t *testing.T) { contractDisabled(t, false, v, "1", []string{"CHAIN_API_GO_LIVE_TEST"}) })
	}
}

// Detect omitted/duplicated methods, unbudgeted calls, identity bypass, and historical/dynamic mixing.
func TestReadContractCatalogAndIdentity(t *testing.T) {
	for _, chain := range contractChains() {
		for _, badIdentity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/identity_bad=%v", chain, badIdentity), func(t *testing.T) {
				m, d := contractReadMaterial(chain)
				counts := map[string]int{}
				calls := 0
				call := contractCaller(func(_ context.Context, method string, p, out any) error {
					calls++
					counts[method]++
					contractReadParams(t, chain, method, p)
					raw := contractResponse(chain, method, contractJSON(p))
					if badIdentity {
						switch chain {
						case "ethereum", "bnb":
							raw = json.RawMessage(`"0x2"`)
						case "bitcoin":
							raw = json.RawMessage(`{"chain":"main"}`)
						case "solana":
							raw = contractJSON(strings.Repeat("2", 32))
						case "tron":
							raw = json.RawMessage(`{"blockID":"different"}`)
						}
					}
					return json.Unmarshal(raw, out)
				})
				cases, err := contractReadCases(t, chain, m, d, call)
				if err != nil {
					t.Errorf("valid material rejected: %v", err)
				}
				ids := []string{}
				for _, c := range cases {
					ids = append(ids, c.ID)
				}
				if !reflect.DeepEqual(ids, contractRequired(chain, false)) {
					t.Errorf("catalog/order=%v", ids)
				}
				budgets := map[string]int{"ethereum": 22, "bnb": 22, "bitcoin": 15, "solana": 16, "tron": 30}
				got := contractHarness(chain, cases, false, budgets[chain])
				want := "PASS"
				if badIdentity {
					want = "BLOCKED"
				}
				if got.Status != want {
					t.Errorf("status=%s want=%s", got.Status, want)
				}
				if !badIdentity {
					if !reflect.DeepEqual(counts, contractWireCounts(chain)) {
						t.Errorf("wire catalog counts=%v", counts)
					}
					if calls != budgets[chain] || got.Requests != calls {
						t.Errorf("request budget=%d/%d want=%d", calls, got.Requests, budgets[chain])
					}
					for _, id := range contractRequired(chain, false) {
						if got.Cases[id] != "PASS" {
							t.Errorf("required %s=%s", id, got.Cases[id])
						}
					}
				} else {
					if calls > 1 {
						t.Errorf("identity failure allowed %d calls", calls)
					}
					for _, id := range contractRequired(chain, false) {
						if got.Cases[id] == "PASS" {
							t.Errorf("identity failure leaked PASS: %s", id)
						}
					}
				}
			})
		}
	}
}

// Synthetic in-memory host: there is deliberately no os/env/HTTP client fallback.
type contractHost struct {
	files             map[string][]byte
	reads             map[string]int
	creates, cleanups []string
	calls             map[string][]string
	journal, output   []byte
	events            []string
	deps              runnerDeps
	mutate            func(string, string, json.RawMessage) (json.RawMessage, error)
}

func contractHostFor(t *testing.T, mode string) *contractHost {
	t.Helper()
	h := &contractHost{files: map[string][]byte{}, reads: map[string]int{}, calls: map[string][]string{}}
	targets := map[string]any{}
	for _, chain := range contractChains() {
		m, d := contractReadMaterial(chain)
		material := contractJSON(m)
		budget := map[string]int{"ethereum": 22, "bnb": 22, "bitcoin": 15, "solana": 16, "tron": 30}[chain]
		if mode == "broadcast" {
			mat, payload := contractSubmissionMaterial(chain)
			material = contractMaterialJSON(mat)
			h.files[mat.PayloadPath] = payload
			d.Inputs = json.RawMessage(`{}`)
			d.Preflight.EndpointAlias = chain + "_broadcast"
			d.Preflight.Methods = map[string]rpcExpectation{}
			names := []string{contractCatalog(chain)[0]}
			if chain == "bitcoin" {
				names = append(names, "GetBlockHash")
			}
			for _, n := range names {
				d.Preflight.Methods[n] = rpcExpectation{Outcome: "success", Expected: json.RawMessage(`null`)}
			}
			budget = 3
			if chain == "bitcoin" {
				budget = 4
			}
		}
		d.ManifestSHA256 = contractSHA(material)
		h.files[chain+".dynamic"] = contractJSON(d)
		h.files[chain+".material"] = material
		h.files["testdata/"+chain+"/samples.json"] = material
		targets[chain+"_"+mode] = map[string]any{"endpoint_env": "ENDPOINT", "header_env": map[string]string{"Authorization": "AUTH"}, "network": "fixture", "max_requests": budget, "max_concurrent": 1, "min_interval": "1ns", "total_timeout": "1h", "dynamic_inputs_path": chain + ".dynamic", "submission_material_path": chain + ".material"}
	}
	h.files["explicit-config"] = contractJSON(map[string]any{"targets": targets})
	now := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	h.deps = runnerDeps{LookupEnv: func(k string) (string, bool) {
		switch k {
		case "CHAIN_API_GO_LIVE_TEST", "CHAIN_API_GO_LIVE_BROADCAST":
			return "1", true
		case "CHAIN_API_GO_LIVE_CONFIG":
			return "explicit-config", true
		case "ENDPOINT":
			return "https://fixture.invalid/" + contractSecret, true
		case "AUTH":
			return contractSecret, true
		}
		t.Errorf("unapproved env lookup %s", k)
		return "", false
	}, ReadFile: func(p string) ([]byte, error) {
		h.reads[p]++
		v, ok := h.files[p]
		if !ok {
			return nil, errors.New(contractSecret)
		}
		return append([]byte(nil), v...), nil
	}, Now: func() time.Time { return now }, Wait: func(_ context.Context, d time.Duration) error { now = now.Add(d); return nil }, WriteReadReport: func(b []byte) error { h.output = append([]byte(nil), b...); return nil }, ReadBroadcastJournal: func() ([]byte, error) {
		h.events = append(h.events, "journal-read")
		return append([]byte(nil), h.journal...), nil
	}, AppendBroadcastRecord: func(b []byte) error {
		h.events = append(h.events, "append")
		h.journal = append(h.journal, b...)
		return nil
	}}
	h.deps.NewDoer = func(cfg harnessConfig) (transport.HTTPDoer, func(), error) {
		h.creates = append(h.creates, cfg.Alias)
		chain := strings.TrimSuffix(cfg.Alias, "_"+mode)
		return contractDoer(func(r *http.Request) (*http.Response, error) {
			body, e := io.ReadAll(r.Body)
			if e != nil {
				return nil, e
			}
			_ = r.Body.Close()
			method := strings.TrimPrefix(r.URL.Path, "/"+contractSecret)
			params := json.RawMessage(body)
			var req struct {
				Method  string          `json:"method"`
				Params  json.RawMessage `json:"params"`
				ID      json.RawMessage `json:"id"`
				Version string          `json:"jsonrpc"`
			}
			if chain != "tron" {
				if e = json.Unmarshal(body, &req); e != nil {
					t.Fatal(e)
				}
				method = req.Method
				params = req.Params
				want := "2.0"
				if chain == "bitcoin" {
					want = "1.0"
				}
				if req.Version != want {
					t.Errorf("protocol dialect %s=%q", chain, req.Version)
				}
			}
			h.calls[chain] = append(h.calls[chain], method)
			h.events = append(h.events, chain+":"+method)
			if r.Header.Get("Authorization") != contractSecret {
				t.Error("injected headers lost")
			}
			raw := contractResponse(chain, method, params)
			if h.mutate != nil {
				raw, e = h.mutate(chain, method, raw)
				if e != nil {
					return nil, e
				}
			}
			if chain == "bitcoin" {
				raw = contractJSON(map[string]any{"id": req.ID, "result": raw, "error": nil})
			} else if chain != "tron" {
				raw = contractJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": raw})
			}

			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
		}), func() { h.cleanups = append(h.cleanups, cfg.Alias) }, nil
	}
	return h
}
func contractCheckSuite(t *testing.T, got suiteReport, mode, status string) {
	t.Helper()
	if got.Version != 1 || got.Mode != mode || got.Status != status {
		t.Errorf("suite metadata=%+v want %s/%s", got, mode, status)
	}
	if len(got.Chains) != 5 {
		t.Errorf("chains=%d want 5, including blocked assemblies", len(got.Chains))
	}
	for i, c := range got.Chains {
		if i >= 5 {
			break
		}
		if c.Chain != contractChains()[i] || c.Alias != contractChains()[i]+"_"+mode {
			t.Error("chain/alias order changed")
		}
	}
	for _, date := range []string{got.StartedAt, got.FinishedAt} {
		if _, e := time.Parse(time.RFC3339, date); e != nil {
			t.Error("enabled report missing RFC3339 time")
		}
	}
	assertRedacted(t, got, contractSecret, "https://fixture.invalid", "Authorization", "raw_data", "payload_path")
}
func TestReadContractEnabledReports(t *testing.T) {
	for _, scenario := range []string{"pass", "missing_config", "missing_file", "assembly", "write_failure", "assertion_and_write_failure", "budget"} {
		t.Run(scenario, func(t *testing.T) {
			h := contractHostFor(t, "read")
			want := "PASS"
			switch scenario {
			case "missing_config":
				delete(h.files, "explicit-config")
				want = "BLOCKED"
			case "missing_file":
				delete(h.files, "bnb.dynamic")
				want = "BLOCKED"
			case "assembly":
				h.deps.NewDoer = func(harnessConfig) (transport.HTTPDoer, func(), error) { return nil, nil, errors.New(contractSecret) }
				want = "BLOCKED"
			case "write_failure":
				h.deps.WriteReadReport = func([]byte) error { return errors.New(contractSecret) }
				want = "BLOCKED"
			case "assertion_and_write_failure":
				h.mutate = func(c, m string, b json.RawMessage) (json.RawMessage, error) {
					if c == "ethereum" && m == "eth_getBalance" {
						return json.RawMessage(`"0x2"`), nil
					}
					return b, nil
				}
				h.deps.WriteReadReport = func([]byte) error { return errors.New(contractSecret) }
				want = "FAIL"
			case "budget":
				h.files["explicit-config"] = []byte(strings.ReplaceAll(string(h.files["explicit-config"]), `"max_requests":22`, `"max_requests":1`))
				want = "BLOCKED"
			}
			got := runLiveRead(context.Background(), h.deps)
			contractCheckSuite(t, got, "read", want)
			if scenario == "pass" {
				contractReportKeys(t, h.output)
				var stored suiteReport
				if json.Unmarshal(h.output, &stored) != nil || !reflect.DeepEqual(stored, got) {
					t.Error("saved read report not equal to returned report")
				}
				if len(h.creates) != 5 || !reflect.DeepEqual(h.creates, h.cleanups) {
					t.Error("client assembly/cleanup not exactly once per chain")
				}
				for _, chain := range contractChains() {
					if h.reads[chain+".dynamic"] != 1 || h.reads["testdata/"+chain+"/samples.json"] != 1 {
						t.Errorf("material read count %s", chain)
					}
				}
				for _, c := range got.Chains {
					if c.MaterialDigest != contractSHA(h.files["testdata/"+c.Chain+"/samples.json"]) || c.CodeSnapshot != strings.Repeat("c", 40) {
						t.Error("report snapshot/digest not bound")
					}
					for _, id := range contractRequired(c.Chain, false) {
						if c.Result.Cases[id] != "PASS" {
							t.Errorf("missing required %s", id)
						}
					}
				}
			}
		})
	}
}

func TestNativeSolanaReadCatalog(t *testing.T) {
	for _, chain := range []string{"solana"} {
		t.Run(chain, func(t *testing.T) {
			m, d := contractReadMaterial(chain)
			call := contractCaller(func(_ context.Context, method string, p, out any) error {
				return json.Unmarshal(contractResponse(chain, method, contractJSON(p)), out)
			})
			cases, err := contractReadCases(t, chain, m, d, call)
			if err != nil {
				t.Fatalf("NS-08: runner rejected frozen native Solana inputs from in-memory fixtures: %v", err)
			}
			ids := []string{}
			for _, c := range cases {
				ids = append(ids, c.ID)
			}
			if !reflect.DeepEqual(ids, contractRequired(chain, false)) {
				t.Fatalf("NS-08: native catalog=%v want=%v", ids, contractRequired(chain, false))
			}
		})
	}
}
func TestNativeSolanaReadVersionGate(t *testing.T) {
	for _, chain := range []string{"solana"} {
		for _, version := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/v%d", chain, version), func(t *testing.T) {
				h := contractHostFor(t, "read")
				m, d := contractReadMaterial(chain)
				m.ManifestVersion = version
				raw := contractJSON(m)
				d.ManifestSHA256 = contractSHA(raw)
				h.files["testdata/"+chain+"/samples.json"] = raw
				h.files[chain+".dynamic"] = contractJSON(d)
				got := runLiveRead(context.Background(), h.deps)
				for _, c := range got.Chains {
					if c.Chain == chain {
						if version == 1 {
							if c.Result.Status != "BLOCKED" || len(h.calls[chain]) != 0 {
								t.Fatal("NS-08: obsolete Solana material reached Caller")
							}
						} else if c.Result.Status != "PASS" {
							t.Fatalf("NS-08: native v2 material status=%s", c.Result.Status)
						}
					}
				}
			})
		}
	}
}

// Each mutation would otherwise convert a missing independent prerequisite into apparent PASS.
func TestReadContractMaterialPrerequisites(t *testing.T) {
	for _, chain := range contractChains() {
		for _, bad := range []string{"digest", "identity", "expired", "preflight_missing", "preflight_failed", "preflight_alias", "duplicate_sample", "category", "source", "field_sources", "expected_missing", "unknown_key", "trailing", "snapshot", "mixed_identity", "future", "invalid_outcome", "success_with_code", "historic_rpc_error", "sample_preflight", "solana_options", "tron_height"} {
			t.Run(chain+"/"+bad, func(t *testing.T) {
				if bad == "solana_options" && chain != "solana" || bad == "tron_height" && chain != "tron" {
					return
				}
				h := contractHostFor(t, "read")
				m, d := contractReadMaterial(chain)
				switch bad {
				case "digest":
					d.ManifestSHA256 = strings.Repeat("0", 64)
				case "identity":
					m.Network = "other"
				case "expired":
					d.ExpiresAt = contractObserved
				case "preflight_missing":
					delete(d.Preflight.Methods, contractCatalog(chain)[len(contractCatalog(chain))-1])
				case "preflight_failed":
					d.Preflight.Status = "FAIL"
				case "preflight_alias":
					d.Preflight.EndpointAlias = "other_read"
				case "duplicate_sample":
					m.Samples[1].TransactionID = m.Samples[0].TransactionID
				case "category":
					m.Samples[1].Category = "native"
				case "source":
					m.Samples[0].SourceURL = "https://secret@explorer.invalid"
				case "field_sources":
					m.Samples[0].FieldSources = map[string]string{}
				case "expected_missing":
					m.Samples[0].Expected = json.RawMessage(`{}`)
				case "snapshot":
					d.CodeSnapshot = contractSecret
				case "future":
					d.ObservedAt = contractExpires
				case "invalid_outcome":
					v := d.Preflight.Methods[contractCatalog(chain)[0]]
					v.Outcome = "accepted"
					d.Preflight.Methods[contractCatalog(chain)[0]] = v
				case "success_with_code":
					v := d.Preflight.Methods[contractCatalog(chain)[0]]
					v.HasRPCCode = true
					d.Preflight.Methods[contractCatalog(chain)[0]] = v
				case "historic_rpc_error":
					v := d.Preflight.Methods[contractCatalog(chain)[0]]
					v.Outcome = "rpc_error"
					v.HasRPCCode = true
					d.Preflight.Methods[contractCatalog(chain)[0]] = v
				case "sample_preflight":
					m.Samples[0].RPCPreflight.Methods = map[string]rpcExpectation{}
				case "solana_options":
					d.Inputs = []byte(strings.ReplaceAll(string(d.Inputs), `"MaxSupportedTransactionVersion":0`, `"MaxSupportedTransactionVersion":null`))
				case "tron_height":
					m.Samples[0].HeightOrSlot = 1 << 63
				case "mixed_identity":
					m.NetworkIdentity = networkIdentity{ChainID: "1", BitcoinChain: "regtest", GenesisHash: "genesis", TronGenesisBlockID: "block"}
				}
				raw := contractJSON(m)
				if bad == "unknown_key" {
					raw = []byte(strings.TrimSuffix(string(raw), "}") + `,"unapproved":1}`)
				}
				if bad == "trailing" {
					raw = append(raw, []byte(`{}`)...)
				}
				if bad != "digest" {
					d.ManifestSHA256 = contractSHA(raw)
				}
				h.files["testdata/"+chain+"/samples.json"] = raw
				h.files[chain+".dynamic"] = contractJSON(d)
				got := runLiveRead(context.Background(), h.deps)
				contractCheckSuite(t, got, "read", "BLOCKED")
				if len(h.calls[chain]) != 0 {
					t.Error("invalid local material reached SDK")
				}
				for _, c := range got.Chains {
					if c.Chain == chain && c.Result.Status != "BLOCKED" {
						t.Error("invalid chain silently passed")
					}
				}
			})
		}
	}
}

func TestReadContractExpectedRPCAndHistoricalComparison(t *testing.T) {
	for _, chain := range contractChains() {
		for _, scenario := range []string{"expected_rpc", "wrong_code", "transport_not_rpc", "history_change", "raw_null_vs_absent", "large_integer", "missing_case", "confirmation_growth"} {
			t.Run(chain+"/"+scenario, func(t *testing.T) {
				if scenario == "confirmation_growth" && chain != "bitcoin" || scenario == "large_integer" && chain != "solana" || scenario == "raw_null_vs_absent" && (chain == "bitcoin" || chain == "tron" || chain == "ethereum" || chain == "bnb") {
					return
				}
				m, d := contractReadMaterial(chain)
				method, wire := "CallContract", "eth_call"
				switch chain {
				case "bitcoin":
					method, wire = "TestMempoolAccept", "testmempoolaccept"
				case "solana":
					method, wire = "SimulateRawTransactionWithOpts", "simulateTransaction"
				case "tron":
					method, wire = "TriggerConstantContract", "/wallet/triggerconstantcontract"
				}
				want := "FAIL"
				if scenario == "confirmation_growth" {
					want = "PASS"
				}
				if scenario == "expected_rpc" || scenario == "wrong_code" || scenario == "transport_not_rpc" {
					d.Preflight.Methods[method] = rpcExpectation{Outcome: "rpc_error", RPCCode: -32001, HasRPCCode: chain != "tron", NodeCode: ""}
					if chain == "tron" {
						v := d.Preflight.Methods[method]
						v.NodeCode = "REJECT"
						d.Preflight.Methods[method] = v
					}
					if scenario == "expected_rpc" {
						want = "PASS"
					}
				}
				call := contractCaller(func(_ context.Context, method string, p, out any) error {
					raw := contractResponse(chain, method, contractJSON(p))
					if method == wire && (scenario == "expected_rpc" || scenario == "wrong_code" || scenario == "transport_not_rpc") {
						e := apierror.Error{Kind: apierror.RPC, RPCCode: -32001, HasRPCCode: chain != "tron", NodeCode: "REJECT", Message: contractSecret, Data: json.RawMessage(`{"secret":"SYNTHETIC_PRIVATE_MARKER"}`), ResponseBody: []byte(contractSecret), Cause: errors.New(contractSecret)}
						if scenario == "wrong_code" {
							e.RPCCode--
							e.NodeCode = "OTHER"
						}
						if scenario == "transport_not_rpc" {
							e.Kind = apierror.Transport
						}
						return e
					}
					if scenario == "confirmation_growth" && method == "getrawtransaction" {
						raw = []byte(strings.Replace(string(raw), `"confirmations":2`, `"confirmations":99`, 1))
					}
					if scenario == "history_change" || scenario == "raw_null_vs_absent" || scenario == "large_integer" {
						switch chain {
						case "ethereum", "bnb":
							if method == "eth_getTransactionByHash" {
								if scenario == "raw_null_vs_absent" {
									raw = []byte(strings.Replace(string(raw), `,"to":null`, "", 1))
								} else {
									raw = []byte(strings.Replace(string(raw), `"value":"0x1"`, `"value":"0x2"`, 1))
								}
							}
						case "bitcoin":
							if method == "getrawtransaction" {
								raw = []byte(strings.Replace(string(raw), `"sequence":1`, `"sequence":2`, 1))
							}
						case "solana":
							if method == "getTransaction" {
								if scenario == "raw_null_vs_absent" {
									raw = []byte(strings.Replace(string(raw), `"err":null,`, `"err":{"InstructionError":[0,"synthetic"]},`, 1))
								} else {
									raw = []byte(strings.Replace(string(raw), `9007199254740993`, `9007199254740992`, 1))
								}
							}
						case "tron":
							if method == "/wallet/gettransactioninfobyid" {
								raw = []byte(strings.Replace(string(raw), `"fee":1`, `"fee":2`, 1))
							}
						}
					}
					return json.Unmarshal(raw, out)
				})
				cases, err := contractReadCases(t, chain, m, d, call)
				if err != nil {
					t.Errorf("factory rejected valid material: %v", err)
				}
				if scenario == "missing_case" {
					want = "BLOCKED"
					if len(cases) > 0 {
						cases = cases[:len(cases)-1]
					}
				}
				got := contractHarness(chain, cases, false, 100)
				if got.Status != want {
					t.Errorf("%s status=%s want=%s", scenario, got.Status, want)
				}
				assertRedacted(t, got, contractSecret)
			})
		}
	}
}

// Dynamic null/presence and native compute rejection are data, not omitted coverage.
func TestReadContractDynamicNullAndExpected(t *testing.T) {
	for _, tc := range []struct{ chain, method, wire, expected, response, want string }{
		{"ethereum", "BalanceAt", "eth_getBalance", `"2"`, `"0x1"`, "FAIL"},
		{"bnb", "NonceAt", "eth_getTransactionCount", `2`, `"0x1"`, "FAIL"},
		{"ethereum", "CodeAt", "eth_getCode", `"0xbb"`, `"0xaa"`, "FAIL"},
		{"bnb", "FilterLogs", "eth_getLogs", `[]`, `[{"address":"unexpected"}]`, "FAIL"},
		{"ethereum", "EstimateGas", "eth_estimateGas", `null`, `"0x0"`, "FAIL"},
		{"bnb", "FeeHistory", "eth_feeHistory", `{"OldestBlock":8,"BaseFee":[0],"GasUsedRatio":[0.5],"Reward":[[0]]}`, `{"oldestBlock":"0x8","baseFeePerGas":["0x0"],"gasUsedRatio":[0.5],"reward":[["0x0"]]}`, "PASS"},
		{"bnb", "FeeHistory", "eth_feeHistory", `{"OldestBlock":8,"BaseFee":[1],"GasUsedRatio":[0.5],"Reward":[[0]]}`, `{"oldestBlock":"0x8","baseFeePerGas":["0x0"],"gasUsedRatio":[0.5],"reward":[["0x0"]]}`, "FAIL"},
		{"bitcoin", "GetTxOut", "gettxout", `false`, `null`, "PASS"},
		{"bitcoin", "EstimateSmartFee", "estimatesmartfee", `false`, `{"errors":["insufficient data"]}`, "PASS"},
		{"bitcoin", "TestMempoolAccept", "testmempoolaccept", `true`, `[{"allowed":false}]`, "FAIL"},
		{"solana", "GetAccountInfoWithOpts", "getAccountInfo", `null`, `{"context":{"slot":9},"value":null}`, "FAIL"},
		{"solana", "GetFeeForMessageWithOpts", "getFeeForMessage", `null`, `{"context":{"slot":9},"value":null}`, "PASS"},
		{"solana", "SimulateRawTransactionWithOpts", "simulateTransaction", `{"InstructionError":[0,"x"]}`, `{"context":{"slot":9},"value":{"err":{"InstructionError":[0,"x"]}}}`, "PASS"},
		{"tron", "GetAccount", "/wallet/getaccount", `false`, `{}`, "PASS"},
		{"tron", "GetSolidifiedAccount", "/walletsolidity/getaccount", `false`, `{}`, "PASS"},
		{"tron", "TriggerConstantContract", "/wallet/triggerconstantcontract", `["aa"]`, `{"result":{"result":true},"constant_result":["bb"]}`, "FAIL"},
	} {
		t.Run(tc.chain+"/"+tc.method, func(t *testing.T) {
			m, d := contractReadMaterial(tc.chain)
			e := d.Preflight.Methods[tc.method]
			e.Expected = json.RawMessage(tc.expected)
			d.Preflight.Methods[tc.method] = e
			call := contractCaller(func(_ context.Context, method string, p, out any) error {
				raw := contractResponse(tc.chain, method, contractJSON(p))
				if method == tc.wire {
					raw = json.RawMessage(tc.response)
				}
				return json.Unmarshal(raw, out)
			})
			cases, err := contractReadCases(t, tc.chain, m, d, call)
			if err != nil {
				t.Errorf("valid dynamic fixture rejected: %v", err)
			}
			got := contractHarness(tc.chain, cases, false, 100)
			if got.Status != tc.want {
				t.Errorf("dynamic result=%s want=%s", got.Status, tc.want)
			}
		})
	}
}

// Each independent return boundary must contribute to the required case verdict.
func TestReadContractHistoricalReturnBoundaries(t *testing.T) {
	for _, tc := range []struct{ chain, method, wire, old, new string }{
		{"evm", "TransactionReceipt", "eth_getTransactionReceipt", `"status":"0x1"`, `"status":"0x0"`},
		{"evm", "TransactionReceipt", "eth_getTransactionReceipt", `"logs":[]`, `"logs":[{"address":"changed"}]`},
		{"evm", "TransactionReceipt", "eth_getTransactionReceipt", `"blockNumber":"0x8"`, `"blockNumber":"0x9"`},
		{"evm", "TransactionReceipt", "eth_getTransactionReceipt", `"gasUsed":"0x5208"`, `"gasUsed":"0x9"`},
		{"evm", "BlockByNumber", "eth_getBlockByNumber", `"timestamp":"0x9"`, `"timestamp":"0xa"`},
		{"evm", "BlockByNumber", "eth_getBlockByNumber", `"number":"0x8"`, `"number":"0x9"`},
		{"evm", "BlockByNumber", "eth_getBlockByNumber", `"gasLimit":"0x100000"`, `"gasLimit":"0x200000"`},
		{"bitcoin", "GetBlock", "getblock", `"height":8`, `"height":9`},
		{"bitcoin", "GetBlock", "getblock", `"hash":"888`, `"hash":"999`},
		{"bitcoin", "GetBlock", "getblock", `"tx":[`, `"tx":["extra",`},
		{"solana", "GetBlockWithOpts", "getBlock", `"parentSlot":7`, `"parentSlot":6`},
		{"solana", "GetBlockWithOpts", "getBlock", `"previousBlockhash":"11111111111111111111111111111111"`, `"previousBlockhash":"` + (ns.Hash{8}).String() + `"`},
		{"solana", "GetBlockWithOpts", "getBlock", `"blockhash":"` + contractID("solana", 7) + `"`, `"blockhash":"` + (ns.Hash{8}).String() + `"`},
		{"tron", "GetBlockByNum", "/wallet/getblockbynum", `"blockID":"888`, `"blockID":"999`},
		{"tron", "GetSolidifiedTransactionByID", "/walletsolidity/gettransactionbyid", `"type":"TransferContract"`, `"type":"ChangedContract"`},
		{"tron", "GetSolidifiedTransactionByID", "/walletsolidity/gettransactionbyid", `"signature":["aa"]`, `"signature":["bb"]`},
		{"tron", "GetSolidifiedTransactionInfoByID", "/walletsolidity/gettransactioninfobyid", `"fee":1`, `"fee":2`},
		{"tron", "GetSolidifiedTransactionInfoByID", "/walletsolidity/gettransactioninfobyid", `"result":"SUCCESS"`, `"result":"FAILED"`},
		{"tron", "GetSolidifiedTransactionInfoByID", "/walletsolidity/gettransactioninfobyid", `"log":[]`, `"log":[{"data":"changed"}]`},
	} {
		chains := []string{tc.chain}
		if tc.chain == "evm" {
			chains = []string{"ethereum", "bnb"}
		}
		for _, chain := range chains {
			t.Run(chain+"/"+tc.method+"/"+tc.old, func(t *testing.T) {
				m, d := contractReadMaterial(chain)
				changed := 0
				call := contractCaller(func(_ context.Context, method string, p, out any) error {
					raw := contractResponse(chain, method, contractJSON(p))
					if method == tc.wire && !(chain == "tron" && method == "/wallet/getblockbynum" && strings.Contains(string(contractJSON(p)), `"num":0`)) {
						altered := strings.Replace(string(raw), tc.old, tc.new, 1)
						if altered == string(raw) {
							t.Error("historical mutation did not match literal response")
						} else {
							changed++
						}
						raw = json.RawMessage(altered)
					}
					return json.Unmarshal(raw, out)
				})
				cases, err := contractReadCases(t, chain, m, d, call)
				if err != nil {
					t.Errorf("valid historical fixture rejected: %v", err)
				}
				got := contractHarness(chain, cases, false, 100)
				if changed == 0 {
					t.Error("historical comparison boundary never executed")
				}
				wantMethod := "FAIL"
				if chain == "tron" && tc.method == "GetBlockByNum" {
					wantMethod = "PASS"
				} // Genesis identity is unchanged; the historical sample block must fail.
				if got.Status != "FAIL" || got.Category != "assertion" || got.Cases[chain+"."+tc.method] != wantMethod || got.Cases[chain+".sample.1"] != "FAIL" {
					t.Errorf("changed historical return was not FAIL/assertion: %+v", got)
				}
			})
		}
	}
}

// Literal output keys are independent of production struct tags and round trips.
func contractObjectKeys(t *testing.T, raw []byte, want string) map[string]json.RawMessage {
	t.Helper()
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil || obj == nil {
		t.Error("output is not a JSON object")
		return nil
	}
	keys := strings.Fields(want)
	if len(obj) != len(keys) {
		t.Errorf("output key count=%d want=%d", len(obj), len(keys))
	}
	for _, key := range keys {
		if _, ok := obj[key]; !ok {
			t.Errorf("missing literal output key %s", key)
		}
	}
	return obj
}
func contractReportKeys(t *testing.T, raw []byte) {
	t.Helper()
	suite := contractObjectKeys(t, raw, "version mode status started_at finished_at chains")
	var chains []json.RawMessage
	if json.Unmarshal(suite["chains"], &chains) != nil {
		t.Error("output chains is not an array")
	}
	for _, rawChain := range chains {
		chain := contractObjectKeys(t, rawChain, "chain alias code_snapshot material_digest result")
		contractObjectKeys(t, chain["result"], "Status Category Digest Requests Cases")
	}
}

func contractNativeHeader() json.RawMessage {
	return json.RawMessage(`{"parentHash":"0x` + strings.Repeat("0", 63) + `1","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","miner":"0x1111111111111111111111111111111111111111","stateRoot":"0x` + strings.Repeat("0", 63) + `1","transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","receiptsRoot":"0x` + strings.Repeat("0", 63) + `1","logsBloom":"0x` + strings.Repeat("00", 256) + `","difficulty":"0x1","number":"0x8","gasLimit":"0x100000","gasUsed":"0x0","timestamp":"0x9","extraData":"0x","mixHash":"0x` + strings.Repeat("0", 63) + `1","nonce":"0x0000000000000000"}`)
}

// Parse fixed synthetic wire transactions; never construct/sign/serialize a transaction.
func contractNativeTransaction(n int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"type":"0x0","nonce":"0x%x","gasPrice":"0x3","gas":"0x5208","to":null,"value":"0x1","input":"0x00ff","v":"0x1b","r":"0x1","s":"0x2"}`, n))
}
func contractNativeReceipt(id, block string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"type":"0x0","status":"0x1","cumulativeGasUsed":"0x5208","logsBloom":"0x%s","logs":[],"transactionHash":%q,"contractAddress":null,"gasUsed":"0x5208","effectiveGasPrice":"0x3","blockHash":%q,"blockNumber":"0x8","transactionIndex":"0x0"}`, strings.Repeat("00", 256), id, block))
}
func TestNativeEVMReadCatalog(t *testing.T) {
	for _, chain := range []string{"ethereum", "bnb"} {
		t.Run(chain, func(t *testing.T) {
			m, d := contractReadMaterial(chain)
			call := contractCaller(func(_ context.Context, method string, p, out any) error {
				return json.Unmarshal(contractResponse(chain, method, contractJSON(p)), out)
			})
			cases, err := contractReadCases(t, chain, m, d, call)
			if err != nil {
				t.Fatalf("native material rejected: %v", err)
			}
			ids := []string{}
			for _, c := range cases {
				ids = append(ids, c.ID)
			}
			if !reflect.DeepEqual(ids, contractRequired(chain, false)) {
				t.Fatalf("NS-08: native catalog=%v want=%v", ids, contractRequired(chain, false))
			}
		})
	}
}
func TestNativeEVMReadVersionGate(t *testing.T) {
	for _, chain := range []string{"ethereum", "bnb"} {
		for _, version := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/v%d", chain, version), func(t *testing.T) {
				h := contractHostFor(t, "read")
				m, d := contractReadMaterial(chain)
				m.ManifestVersion = version
				raw := contractJSON(m)
				d.ManifestSHA256 = contractSHA(raw)
				h.files["testdata/"+chain+"/samples.json"] = raw
				h.files[chain+".dynamic"] = contractJSON(d)
				got := runLiveRead(context.Background(), h.deps)
				for _, c := range got.Chains {
					if c.Chain == chain {
						if version == 1 {
							if c.Result.Status != "BLOCKED" || len(h.calls[chain]) != 0 {
								t.Fatal("NS-08: obsolete EVM material reached Caller")
							}
						} else if c.Result.Status != "PASS" {
							t.Fatalf("NS-08: native v2 material status=%s", c.Result.Status)
						}
					}
				}
			})
		}
	}
}
func TestNativeEVMUncleBudget(t *testing.T) {
	for _, budget := range []int{2, 3} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			m, d := contractReadMaterial("ethereum")
			header := strings.Replace(string(contractNativeHeader()), "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347", "0x"+strings.Repeat("1", 64), 1)
			var decoded types.Header
			if err := json.Unmarshal([]byte(header), &decoded); err != nil {
				t.Fatal(err)
			}
			blockHash := decoded.Hash().Hex()
			for i := range m.Samples {
				m.Samples[i].BlockHash = blockHash
				var expected map[string]json.RawMessage
				_ = json.Unmarshal(m.Samples[i].Expected, &expected)
				expected["block_hash"] = contractJSON(blockHash)
				m.Samples[i].Expected = contractJSON(expected)
			}
			calls := []string{}
			call := contractCaller(func(_ context.Context, method string, p, out any) error {
				calls = append(calls, method)
				raw := contractResponse("ethereum", method, contractJSON(p))
				if method == "eth_getBlockByNumber" {
					raw = []byte(strings.TrimSuffix(header, "}") + fmt.Sprintf(`,"hash":%q,"transactions":[],"uncles":[%q,%q]}`, blockHash, blockHash, blockHash))
				}
				return json.Unmarshal(raw, out)
			})
			cases, err := contractReadCases(t, "ethereum", m, d, call)
			if err != nil {
				t.Fatalf("native material rejected: %v", err)
			}
			ctx := context.Background()
			_, err = cases[0].Run(ctx, func(op func(context.Context) error) error { return op(ctx) })
			if err != nil {
				t.Fatal(err)
			}
			calls = nil
			var target harnessCase
			for _, c := range cases {
				if c.ID == "ethereum.BlockByNumber" {
					target = c
				}
			}
			if target.Run == nil {
				t.Fatal("NS-08: native BlockByNumber runner missing")
			}
			now := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
			deps := harnessDeps{Load: func() (harnessConfig, error) {
				return harnessConfig{Alias: "ethereum", Endpoint: "https://fixture.invalid", Network: "fixture", MaxRequests: budget, MaxConcurrent: 1, MinInterval: time.Nanosecond, TotalTimeout: time.Hour}, nil
			}, LoadManifest: func() ([]byte, string, error) { return []byte(fixtureManifest), fixtureDigest, nil }, Now: func() time.Time { return now }, Wait: func(_ context.Context, d time.Duration) error { now = now.Add(d); return nil }}
			got := runHarness(ctx, true, deps, []string{target.ID}, []harnessCase{target})
			want := "PASS"
			if budget == 2 {
				want = "BLOCKED"
			}
			if got.Status != want || got.Requests != budget || len(calls) != budget {
				t.Fatalf("NS-08: status=%s requests=%d calls=%v want %s/%d", got.Status, got.Requests, calls, want, budget)
			}
			expected := []string{"eth_getBlockByNumber", "eth_getUncleByBlockHashAndIndex", "eth_getUncleByBlockHashAndIndex"}[:budget]
			if !reflect.DeepEqual(calls, expected) {
				t.Fatalf("uncle sequence=%v", calls)
			}
		})
	}
}
