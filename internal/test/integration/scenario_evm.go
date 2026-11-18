package integration

import (
	"context"
	"math/big"
	"net/http"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/bnb"
	"github.com/Stone-afk/chain-api-go/ethereum"
	"github.com/Stone-afk/chain-api-go/evm"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/Stone-afk/chain-api-go/transport"
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	scenarioEVMConfirmations    = 12
	scenarioEVMMaxResponseBytes = 8 << 20
)

var (
	scenarioIssueMismatch   = scenarioIssue{"FAIL", "mismatch"}
	scenarioIssueIdentity   = scenarioIssue{"BLOCKED", "identity"}
	scenarioIssueDependency = scenarioIssue{"BLOCKED", "dependency"}
	scenarioIssueMaterial   = scenarioIssue{"BLOCKED", "material"}
)

// runScenarioEVM runs one EVM chain ("ethereum" or "bnb") through doer and
// returns a classified issue for every non-submit catalog ID of that chain.
//
// chainCfg.Params carries expected_chain_id (decimal), contract (hex address)
// and call_data (hex). ChainID and ValidateNetwork are always attempted first;
// if either fails, every other entry is BLOCKED without a request. Reads are
// anchored on a confirmed block and cross-checked against the same block and
// its first transaction. An RPC error is FAIL unless the ID is listed in
// cfg.KnownUnsupported, in which case it is BLOCKED with category capability.
// A block without transactions makes transaction lookups BLOCKED with
// category material. Submit methods are never called.
func runScenarioEVM(ctx context.Context, chain string, cfg scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]scenarioIssue {
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

	expected, okID := new(big.Int).SetString(chainCfg.Params["expected_chain_id"], 10)
	contractHex := chainCfg.Params["contract"]
	callData, errData := hexutil.Decode(chainCfg.Params["call_data"])
	if !okID || expected.Sign() <= 0 || !common.IsHexAddress(contractHex) || errData != nil || len(callData) == 0 {
		fill(classifyScenarioError(apierror.Error{Kind: apierror.Configuration}))
		return out
	}
	contract := common.HexToAddress(contractHex)
	headers := http.Header{}
	for k, v := range chainCfg.Headers {
		headers.Set(k, v)
	}
	rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: chainCfg.Endpoint, Headers: headers, MaxResponseBytes: scenarioEVMMaxResponseBytes, Version: jsonrpc.V2}, doer)
	if err != nil {
		fill(classifyScenarioError(err))
		return out
	}
	var client *evm.Client
	var validate func(context.Context) error
	switch chain {
	case "ethereum":
		c, err := ethereum.New(ethereum.Config{RawURL: chainCfg.Endpoint, ExpectedChainID: *expected}, rpc)
		if err != nil {
			fill(classifyScenarioError(err))
			return out
		}
		client, validate = c.Client, c.ValidateNetwork
	case "bnb":
		c, err := bnb.New(bnb.Config{RawURL: chainCfg.Endpoint, ExpectedChainID: *expected}, rpc)
		if err != nil {
			fill(classifyScenarioError(err))
			return out
		}
		client, validate = c.Client, c.ValidateNetwork
	default:
		fill(classifyScenarioError(apierror.Error{Kind: apierror.Configuration}))
		return out
	}

	id, err := client.ChainID(ctx)
	identityOK := record("ChainID", err)
	if identityOK && id.Cmp(expected) != 0 {
		out[chain+"/ChainID"] = classifyScenarioError(apierror.Error{Kind: apierror.NetworkMismatch})
		identityOK = false
	}
	if !record("ValidateNetwork", validate(ctx)) {
		identityOK = false
	}
	if !identityOK {
		fill(scenarioIssueIdentity)
		return out
	}

	head, err := client.BlockNumber(ctx)
	if record("BlockNumber", err) && head < scenarioEVMConfirmations {
		out[chain+"/BlockNumber"] = scenarioIssueMaterial
	}
	var anchor *big.Int
	if out[chain+"/BlockNumber"].Status == "PASS" {
		anchor = new(big.Int).SetUint64(head - scenarioEVMConfirmations)
	} else {
		for _, m := range []string{"BlockByNumber", "TransactionByHash", "TransactionReceipt", "BalanceAt", "NonceAt", "CodeAt", "CallContract", "FilterLogs", "FeeHistory"} {
			out[chain+"/"+m] = scenarioIssueDependency
		}
	}

	if anchor != nil {
		block, err := client.BlockByNumber(ctx, anchor)
		if record("BlockByNumber", err) {
			check("BlockByNumber", block.Number().Cmp(anchor) == 0)
		}
		switch {
		case out[chain+"/BlockByNumber"].Status != "PASS":
			out[chain+"/TransactionByHash"], out[chain+"/TransactionReceipt"] = scenarioIssueDependency, scenarioIssueDependency
		case len(block.Transactions()) == 0:
			out[chain+"/TransactionByHash"], out[chain+"/TransactionReceipt"] = scenarioIssueMaterial, scenarioIssueMaterial
		default:
			first := block.Transactions()[0]
			tx, pending, err := client.TransactionByHash(ctx, first.Hash())
			if record("TransactionByHash", err) {
				check("TransactionByHash", !pending && tx.Hash() == first.Hash())
			}
			receipt, err := client.TransactionReceipt(ctx, first.Hash())
			if record("TransactionReceipt", err) {
				check("TransactionReceipt", receipt.TxHash == first.Hash() && receipt.BlockHash == block.Hash() && receipt.BlockNumber != nil && receipt.BlockNumber.Cmp(anchor) == 0)
			}
		}

		balance, err := client.BalanceAt(ctx, contract, anchor)
		if record("BalanceAt", err) {
			check("BalanceAt", balance != nil && balance.Sign() >= 0)
		}
		_, err = client.NonceAt(ctx, contract, anchor)
		record("NonceAt", err)
		code, err := client.CodeAt(ctx, contract, anchor)
		if record("CodeAt", err) {
			check("CodeAt", len(code) > 0)
		}
		result, err := client.CallContract(ctx, geth.CallMsg{To: &contract, Data: callData}, anchor)
		if record("CallContract", err) {
			check("CallContract", len(result) > 0)
		}
		logs, err := client.FilterLogs(ctx, geth.FilterQuery{FromBlock: anchor, ToBlock: anchor, Addresses: []common.Address{contract}})
		if record("FilterLogs", err) {
			ok := true
			for _, l := range logs {
				ok = ok && l.Address == contract && l.BlockNumber == anchor.Uint64()
			}
			check("FilterLogs", ok)
		}
		history, err := client.FeeHistory(ctx, 1, anchor, []float64{50})
		if record("FeeHistory", err) {
			check("FeeHistory", history.OldestBlock != nil && len(history.BaseFee) > 0)
		}
	}

	gas, err := client.EstimateGas(ctx, geth.CallMsg{To: &contract, Data: callData})
	if record("EstimateGas", err) {
		check("EstimateGas", gas > 0)
	}
	price, err := client.SuggestGasPrice(ctx)
	if record("SuggestGasPrice", err) {
		check("SuggestGasPrice", price != nil && price.Sign() > 0)
	}
	if _, raw := out[chain+"/jsonrpc.RawCall"]; raw {
		var rawID hexutil.Big
		if record("jsonrpc.RawCall", rpc.RawCall(ctx, "eth_chainId", []any{}, &rawID)) {
			check("jsonrpc.RawCall", rawID.ToInt().Cmp(expected) == 0)
		}
	}
	fill(scenarioIssueDependency)
	return out
}

// scenarioEVMRunner adapts runScenarioEVM to the status-only runner contract.
func scenarioEVMRunner(chain string) scenarioRunner {
	return func(ctx context.Context, cfg scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]string {
		got := map[string]string{}
		for id, issue := range runScenarioEVM(ctx, chain, cfg, chainCfg, doer) {
			got[id] = issue.Status
		}
		return got
	}
}
