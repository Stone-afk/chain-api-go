package integration

import (
	"context"
	"errors"
	"net/http"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/bitcoin"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/Stone-afk/chain-api-go/transport"
)

const (
	scenarioBitcoinConfirmations    = 6
	scenarioBitcoinConfTarget       = 6
	scenarioBitcoinMaxResponseBytes = 8 << 20
	scenarioBitcoinNotFoundCode     = -5
)

// runScenarioBitcoin runs Bitcoin through doer and returns a classified issue
// for every non-submit catalog ID of the chain.
//
// chainCfg.Params carries expected_chain (getblockchaininfo chain, e.g. "main"),
// expected_genesis (height-0 block hash) and an optional jsonrpc_version
// ("1.0", the default, or "2.0"). GetBlockchainInfo and the
// genesis GetBlockHash run first; on mismatch every other entry is BLOCKED
// without a request. Reads are anchored on a confirmed height and its
// coinbase transaction. GetMempoolEntry passes only on RPC code -5 and
// TestMempoolAccept passes only when the coinbase is not allowed. Submit
// methods are never called.
func runScenarioBitcoin(ctx context.Context, cfg scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]scenarioIssue {
	const chain = "bitcoin"
	out := map[string]scenarioIssue{}
	ids := []string{}
	for _, e := range scenarioCatalog() {
		if e.Chain == chain && e.Kind != "submit" {
			out[e.ID] = scenarioIssue{"NOT_RUN", "unimplemented"}
			ids = append(ids, e.ID)
		}
	}
	fill := func(issue scenarioIssue) {
		for _, id := range ids {
			if out[id].Status == "NOT_RUN" {
				out[id] = issue
			}
		}
	}
	known := map[string]bool{}
	for _, id := range cfg.KnownUnsupported {
		known[id] = true
	}
	record := func(method string, err error) bool {
		id := chain + "/" + method
		issue := classifyScenarioError(err)
		if issue == (scenarioIssue{"FAIL", "rpc"}) && known[id] {
			issue = scenarioIssue{"BLOCKED", "capability"}
		}
		out[id] = issue
		return err == nil
	}
	check := func(method string, ok bool) {
		if !ok {
			out[chain+"/"+method] = scenarioIssueMismatch
		}
	}
	setAll := func(issue scenarioIssue, methods ...string) {
		for _, m := range methods {
			out[chain+"/"+m] = issue
		}
	}

	expectedChain := chainCfg.Params["expected_chain"]
	expectedGenesis := bitcoin.BlockHash(chainCfg.Params["expected_genesis"])
	version := jsonrpc.Bitcoin
	switch chainCfg.Params["jsonrpc_version"] {
	case "", jsonrpc.Bitcoin.ToString():
	case jsonrpc.V2.ToString():
		version = jsonrpc.V2
	default:
		version = ""
	}
	if expectedChain == "" || !scenarioBitcoinHash(expectedGenesis.ToString()) || version == "" {
		fill(classifyScenarioError(apierror.Error{Kind: apierror.Configuration}))
		return out
	}
	headers := http.Header{}
	for k, v := range chainCfg.Headers {
		headers.Set(k, v)
	}
	rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: chainCfg.Endpoint, Headers: headers, MaxResponseBytes: scenarioBitcoinMaxResponseBytes, Version: version}, doer)
	if err != nil {
		fill(classifyScenarioError(err))
		return out
	}
	client, err := bitcoin.New(rpc)
	if err != nil {
		fill(classifyScenarioError(err))
		return out
	}
	mismatch := classifyScenarioError(apierror.Error{Kind: apierror.NetworkMismatch})

	info, err := client.GetBlockchainInfo(ctx)
	if !record("GetBlockchainInfo", err) {
		fill(scenarioIssueIdentity)
		return out
	}
	if info.Chain != expectedChain {
		out[chain+"/GetBlockchainInfo"] = mismatch
		fill(scenarioIssueIdentity)
		return out
	}
	genesis, err := client.GetBlockHash(ctx, 0)
	if !record("GetBlockHash", err) {
		fill(scenarioIssueIdentity)
		return out
	}
	if genesis != expectedGenesis {
		out[chain+"/GetBlockHash"] = mismatch
		fill(scenarioIssueIdentity)
		return out
	}
	check("GetBlockchainInfo", info.Blocks >= 0 && scenarioBitcoinHash(info.BestBlockHash))

	head, err := client.GetBlockCount(ctx)
	if record("GetBlockCount", err) && head < scenarioBitcoinConfirmations {
		out[chain+"/GetBlockCount"] = scenarioIssueMaterial
	}
	var coinbase bitcoin.Transaction
	if out[chain+"/GetBlockCount"].Status != "PASS" {
		setAll(scenarioIssueDependency, "GetBlock", "GetRawTransaction", "GetTxOut", "GetMempoolEntry", "TestMempoolAccept")
	} else {
		anchor := head - scenarioBitcoinConfirmations
		anchorHash, err := client.GetBlockHash(ctx, anchor)
		var block bitcoin.Block
		if record("GetBlockHash", err) {
			block, err = client.GetBlock(ctx, anchorHash)
			if record("GetBlock", err) {
				check("GetBlock", block.Hash == anchorHash.ToString() && block.Height == anchor && block.Confirmations >= scenarioBitcoinConfirmations)
			}
		}
		switch {
		case out[chain+"/GetBlock"].Status != "PASS":
			setAll(scenarioIssueDependency, "GetRawTransaction", "GetTxOut", "GetMempoolEntry", "TestMempoolAccept")
		case len(block.TxIDs) == 0 || !scenarioBitcoinHash(block.TxIDs[0]):
			setAll(scenarioIssueMaterial, "GetRawTransaction", "GetTxOut", "GetMempoolEntry", "TestMempoolAccept")
		default:
			txid := bitcoin.TxID(block.TxIDs[0])
			tx, err := client.GetRawTransaction(ctx, bitcoin.GetRawTransactionRequest{TxID: txid, BlockHash: anchorHash, HasBlockHash: true})
			if record("GetRawTransaction", err) {
				check("GetRawTransaction", tx.TxID == txid.ToString() && tx.Hex != "" && len(tx.Vin) > 0 && tx.Vin[0].Coinbase != nil)
				if out[chain+"/GetRawTransaction"].Status == "PASS" {
					coinbase = tx
				}
			}

			vout, spendable := uint32(0), false
			for _, o := range coinbase.Vout {
				if o.ScriptPubKey.Type != "nulldata" {
					vout, spendable = o.N, true
					break
				}
			}
			switch {
			case out[chain+"/GetRawTransaction"].Status != "PASS":
				out[chain+"/GetTxOut"] = scenarioIssueDependency
			case !spendable:
				out[chain+"/GetTxOut"] = scenarioIssueMaterial
			default:
				utxo, found, err := client.GetTxOut(ctx, bitcoin.GetTxOutRequest{TxID: txid, Vout: vout})
				if record("GetTxOut", err) {
					check("GetTxOut", found && utxo.Coinbase && utxo.Value >= 0 && scenarioBitcoinHash(utxo.BestBlock))
				}
			}

			_, err = client.GetMempoolEntry(ctx, txid)
			var apiErr apierror.Error
			switch {
			case err == nil:
				out[chain+"/GetMempoolEntry"] = scenarioIssueMismatch
			case errors.As(err, &apiErr) && apiErr.Kind == apierror.RPC && apiErr.HasRPCCode && apiErr.RPCCode == scenarioBitcoinNotFoundCode:
				out[chain+"/GetMempoolEntry"] = scenarioIssue{"PASS", ""}
			default:
				record("GetMempoolEntry", err)
			}

			if coinbase.Hex == "" {
				out[chain+"/TestMempoolAccept"] = scenarioIssueDependency
			} else {
				accept, err := client.TestMempoolAccept(ctx, bitcoin.RawTransaction(coinbase.Hex))
				if record("TestMempoolAccept", err) {
					check("TestMempoolAccept", !accept.Allowed && accept.TxID == txid.ToString())
				}
			}
		}
	}

	fee, hasEstimate, err := client.EstimateSmartFee(ctx, bitcoin.EstimateSmartFeeRequest{ConfTarget: scenarioBitcoinConfTarget})
	if record("EstimateSmartFee", err) {
		check("EstimateSmartFee", hasEstimate && fee.FeeRate > 0)
	}
	fill(scenarioIssueDependency)
	return out
}

func scenarioBitcoinHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// scenarioBitcoinRunner adapts runScenarioBitcoin to the status-only runner contract.
func scenarioBitcoinRunner() scenarioRunner {
	return func(ctx context.Context, cfg scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]string {
		got := map[string]string{}
		for id, issue := range runScenarioBitcoin(ctx, cfg, chainCfg, doer) {
			got[id] = issue.Status
		}
		return got
	}
}
