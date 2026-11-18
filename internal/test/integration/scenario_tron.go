package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/httpjson"
	"github.com/Stone-afk/chain-api-go/transport"
	"github.com/Stone-afk/chain-api-go/tron"
)

const (
	scenarioTronMaxResponseBytes = 8 << 20
	scenarioTronHashSize         = 64
)

// runScenarioTron runs Tron through doer with httpjson and returns a
// classified issue for every non-submit catalog ID of the chain.
//
// chainCfg.Params carries expected_genesis (block 0 blockID), account,
// contract and function_selector (visible base58 addresses) and an optional
// hex parameter. GetBlockByNum(0) runs first; on mismatch every other entry is
// BLOCKED without a request. Reads are anchored on the solidified head block
// and its first transaction; a block without transactions makes transaction
// lookups BLOCKED material. httpjson.RawPost re-reads block 0 once. An RPC
// error is FAIL unless the ID is listed in cfg.KnownUnsupported, in which case
// it is BLOCKED capability. Submit methods are never called.
func runScenarioTron(ctx context.Context, cfg scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]scenarioIssue {
	const chain = "tron"
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

	expectedGenesis := chainCfg.Params["expected_genesis"]
	account := tron.Address(chainCfg.Params["account"])
	contract := tron.Address(chainCfg.Params["contract"])
	selector := chainCfg.Params["function_selector"]
	if !scenarioTronHash(expectedGenesis) || account == "" || contract == "" || selector == "" {
		fill(classifyScenarioError(apierror.Error{Kind: apierror.Configuration}))
		return out
	}
	headers := http.Header{}
	for k, v := range chainCfg.Headers {
		headers.Set(k, v)
	}
	poster, err := httpjson.New(httpjson.Config{Endpoint: chainCfg.Endpoint, Headers: headers, MaxResponseBytes: scenarioTronMaxResponseBytes}, doer)
	if err != nil {
		fill(classifyScenarioError(err))
		return out
	}
	client, err := tron.New(poster)
	if err != nil {
		fill(classifyScenarioError(err))
		return out
	}

	genesis, found, err := client.GetBlockByNum(ctx, 0)
	if !record("GetBlockByNum", err) {
		fill(scenarioIssueIdentity)
		return out
	}
	if !found || !strings.EqualFold(genesis.BlockID, expectedGenesis) {
		out[chain+"/GetBlockByNum"] = classifyScenarioError(apierror.Error{Kind: apierror.NetworkMismatch})
		fill(scenarioIssueIdentity)
		return out
	}

	info, err := client.GetNodeInfo(ctx)
	if record("GetNodeInfo", err) {
		check("GetNodeInfo", info.Block != "" && info.SolidityBlock != "" && info.CurrentConnectCount >= 0)
	}

	head, found, err := client.GetNowBlock(ctx)
	headHeight, headOK := scenarioTronHeight(head)
	if record("GetNowBlock", err) {
		check("GetNowBlock", found && scenarioTronHash(head.BlockID) && headOK && headHeight > 0)
	}

	solid, found, err := client.GetSolidifiedNowBlock(ctx)
	solidHeight, solidOK := scenarioTronHeight(solid)
	if record("GetSolidifiedNowBlock", err) {
		check("GetSolidifiedNowBlock", found && scenarioTronHash(solid.BlockID) && solidOK && solidHeight > 0 && (!headOK || solidHeight <= headHeight))
	}
	txLookups := []string{"GetTransactionByID", "GetTransactionInfoByID", "GetSolidifiedTransactionByID", "GetSolidifiedTransactionInfoByID"}
	if out[chain+"/GetSolidifiedNowBlock"].Status != "PASS" {
		setAll(scenarioIssueDependency, append([]string{"GetSolidifiedBlockByNum"}, txLookups...)...)
	} else {
		byNum, found, err := client.GetSolidifiedBlockByNum(ctx, solidHeight)
		if record("GetSolidifiedBlockByNum", err) {
			check("GetSolidifiedBlockByNum", found && strings.EqualFold(byNum.BlockID, solid.BlockID))
		}
		if len(solid.Transactions) == 0 || !scenarioTronHash(solid.Transactions[0].TxID) {
			setAll(scenarioIssueMaterial, txLookups...)
		} else {
			anchor := solid.Transactions[0]
			id := tron.TxID(anchor.TxID)
			sameTx := func(tx tron.Transaction, found bool) bool {
				return found && strings.EqualFold(tx.TxID, anchor.TxID) && anchor.RawDataHex != "" && strings.EqualFold(tx.RawDataHex, anchor.RawDataHex)
			}
			sameInfo := func(i tron.TransactionInfo, found bool) bool {
				return found && strings.EqualFold(i.ID, anchor.TxID) && i.BlockNumber == solidHeight
			}

			tx, found, err := client.GetTransactionByID(ctx, id)
			if record("GetTransactionByID", err) {
				check("GetTransactionByID", sameTx(tx, found))
			}
			txInfo, found, err := client.GetTransactionInfoByID(ctx, id)
			if record("GetTransactionInfoByID", err) {
				check("GetTransactionInfoByID", sameInfo(txInfo, found))
			}
			solidTx, found, err := client.GetSolidifiedTransactionByID(ctx, id)
			if record("GetSolidifiedTransactionByID", err) {
				check("GetSolidifiedTransactionByID", sameTx(solidTx, found))
			}
			solidInfo, found, err := client.GetSolidifiedTransactionInfoByID(ctx, id)
			if record("GetSolidifiedTransactionInfoByID", err) {
				check("GetSolidifiedTransactionInfoByID", sameInfo(solidInfo, found))
			}
		}
	}

	accountReq := tron.GetAccountRequest{Address: account, Visible: true}
	acc, found, err := client.GetAccount(ctx, accountReq)
	if record("GetAccount", err) {
		check("GetAccount", found && acc.Address == account.ToString() && acc.Balance >= 0)
	}
	solidAcc, found, err := client.GetSolidifiedAccount(ctx, accountReq)
	if record("GetSolidifiedAccount", err) {
		check("GetSolidifiedAccount", found && solidAcc.Address == account.ToString() && solidAcc.Balance >= 0)
	}
	resource, err := client.GetAccountResource(ctx, accountReq)
	if record("GetAccountResource", err) {
		check("GetAccountResource", resource.TotalNetLimit > 0 && resource.TotalEnergyLimit > 0)
	}

	params, err := client.GetChainParameters(ctx)
	if record("GetChainParameters", err) {
		check("GetChainParameters", len(params.ChainParameter) > 0)
	}

	call := tron.ContractCallRequest{OwnerAddress: account, ContractAddress: contract, FunctionSelector: selector, Parameter: chainCfg.Params["parameter"], Visible: true}
	constant, err := client.TriggerConstantContract(ctx, call)
	if record("TriggerConstantContract", err) {
		check("TriggerConstantContract", constant.Result.Result && len(constant.ConstantResult) > 0)
	}
	energy, err := client.EstimateEnergy(ctx, call)
	if record("EstimateEnergy", err) {
		check("EstimateEnergy", energy.Result.Result && energy.EnergyRequired > 0)
	}

	var raw struct {
		BlockID string `json:"blockID"`
	}
	err = poster.RawPost(ctx, "/wallet/getblockbynum", map[string]any{"num": 0}, &raw)
	if record("httpjson.RawPost", err) {
		check("httpjson.RawPost", strings.EqualFold(raw.BlockID, expectedGenesis))
	}
	fill(scenarioIssueDependency)
	return out
}

func scenarioTronHash(s string) bool {
	if len(s) != scenarioTronHashSize {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

// scenarioTronHeight reads block_header.raw_data.number.
func scenarioTronHeight(b tron.Block) (int64, bool) {
	var header struct {
		RawData struct {
			Number json.RawMessage `json:"number"`
		} `json:"raw_data"`
	}
	var height int64
	if len(b.BlockHeader) == 0 || json.Unmarshal(b.BlockHeader, &header) != nil || len(header.RawData.Number) == 0 || json.Unmarshal(header.RawData.Number, &height) != nil {
		return 0, false
	}
	return height, true
}

func scenarioTronRunner() scenarioRunner {
	return func(ctx context.Context, cfg scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]string {
		got := map[string]string{}
		for id, issue := range runScenarioTron(ctx, cfg, chainCfg, doer) {
			got[id] = issue.Status
		}
		return got
	}
}
