package integration

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/bitcoin"
	"github.com/Stone-afk/chain-api-go/bnb"
	"github.com/Stone-afk/chain-api-go/ethereum"
	"github.com/Stone-afk/chain-api-go/evm"
	"github.com/Stone-afk/chain-api-go/httpjson"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/Stone-afk/chain-api-go/solana"
	"github.com/Stone-afk/chain-api-go/transport"
	"github.com/Stone-afk/chain-api-go/tron"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

// Fixed malformed probe payloads; nodes must reject every one of them.
const (
	scenarioProbeEVMPayload     = "0x00"
	scenarioProbeBitcoinPayload = "00"
	scenarioProbeSolanaPayload  = "AA=="
	scenarioProbeTronPayload    = `{"visible":false}`
)

const (
	scenarioProbeMaxResponseBytes  = 8 << 20
	scenarioProbeMethodNotFound    = -32601
	scenarioProbePlaceholderPrefix = "REPLACE_ME"
)

// scenarioProbeIdentity verifies the endpoint belongs to the configured network before any submission.
type scenarioProbeIdentity func(ctx context.Context, chain string, cfg scenarioChainConfig, doer transport.HTTPDoer) error

type scenarioProbeDeps struct {
	LookupEnv func(string) (string, bool)
	ReadFile  func(string) ([]byte, error)
	NewDoer   func(scenarioChainConfig) (transport.HTTPDoer, func(), error)
	Now       func() time.Time
	Wait      func(context.Context, time.Duration) error
	Identity  map[string]scenarioProbeIdentity
}

type scenarioProbeReport struct {
	Status, Category string
	Requests         int
	Chains           map[string]scenarioIssue
}

var (
	scenarioProbeIssueAccepted   = scenarioIssue{"FAIL", "accepted"}
	scenarioProbeIssueCapability = scenarioIssue{"BLOCKED", "capability"}
	scenarioProbeIssueStopped    = scenarioIssue{"NOT_RUN", "stopped"}
	scenarioProbeIssueSkipped    = scenarioIssue{"NOT_RUN", "not_required"}
)

// runScenarioSubmitProbe sends each fixed malformed payload once per required
// chain, in scenarioChains order, through the chain's existing submit method.
//
// Both switches must equal "1" and submit_probe_authorization must be set to a
// non-placeholder value; otherwise no request is made. A failed identity check
// skips that chain's submission. A node rejection is PASS; an acceptance is
// FAIL and stops every later chain; transport errors are BLOCKED and are never
// retried; a missing method is BLOCKED capability.
func runScenarioSubmitProbe(ctx context.Context, deps scenarioProbeDeps) scenarioProbeReport {
	scenario, _ := deps.LookupEnv(scenarioSwitch)
	probe, _ := deps.LookupEnv(scenarioProbeSwitch)
	if scenario != "1" || probe != "1" {
		return scenarioProbeReport{Status: "NOT_RUN", Category: "disabled"}
	}
	raw, err := deps.ReadFile(scenarioConfigPath)
	if errors.Is(err, fs.ErrNotExist) {
		return scenarioProbeReport{Status: "BLOCKED", Category: "config_missing"}
	}
	if err != nil {
		return scenarioProbeReport{Status: "BLOCKED", Category: "config_read"}
	}
	cfg, err := parseScenarioConfig(raw)
	if errors.Is(err, errScenarioYAML) {
		return scenarioProbeReport{Status: "BLOCKED", Category: "config_yaml"}
	}
	if err != nil {
		return scenarioProbeReport{Status: "BLOCKED", Category: "config_invalid"}
	}
	auth := strings.TrimSpace(cfg.SubmitProbeAuthorization)
	if auth == "" || strings.HasPrefix(auth, scenarioProbePlaceholderPrefix) {
		return scenarioProbeReport{Status: "BLOCKED", Category: "authorization"}
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.TotalTimeout)
	defer cancel()
	start := deps.Now()
	required := map[string]bool{}
	for _, c := range cfg.RequiredChains {
		required[c] = true
	}
	report := scenarioProbeReport{Chains: map[string]scenarioIssue{}}
	stopped := false
	for _, chain := range scenarioChains {
		switch {
		case stopped:
			report.Chains[chain] = scenarioProbeIssueStopped
			continue
		case !required[chain]:
			report.Chains[chain] = scenarioProbeIssueSkipped
			continue
		case deps.Now().Sub(start) >= cfg.TotalTimeout:
			report.Chains[chain] = classifyScenarioError(context.DeadlineExceeded)
			continue
		}
		chainCfg := cfg.Chains[chain]
		chainCfg.RequestTimeout = cfg.RequestTimeout
		if chainCfg.RequestTimeout == 0 {
			chainCfg.RequestTimeout = cfg.TotalTimeout
		}
		issue, requests := runScenarioProbeChain(ctx, chain, cfg.MaxRequests-report.Requests, cfg.MinInterval, chainCfg, deps)
		report.Requests += requests
		report.Chains[chain] = issue
		if issue.Status == "FAIL" {
			stopped = true
		}
	}

	report.Status = "PASS"
	for _, chain := range scenarioChains {
		switch report.Chains[chain].Status {
		case "FAIL":
			report.Status, report.Category = "FAIL", "chain_result"
		case "BLOCKED":
			if report.Status != "FAIL" {
				report.Status, report.Category = "BLOCKED", "chain_result"
			}
		}
	}
	return report
}

func runScenarioProbeChain(ctx context.Context, chain string, maxRequests int, interval time.Duration, chainCfg scenarioChainConfig, deps scenarioProbeDeps) (scenarioIssue, int) {
	identity, wired := deps.Identity[chain]
	if !wired {
		return scenarioIssueIdentity, 0
	}
	doer, release, err := deps.NewDoer(chainCfg)
	if err != nil {
		return scenarioIssue{"BLOCKED", "environment"}, 0
	}
	if release != nil {
		defer release()
	}
	budget := newScenarioBudget(doer, maxRequests, interval, deps.Wait)
	if err := identity(ctx, chain, chainCfg, budget); err != nil {
		if errors.Is(err, errScenarioBudget) {
			return classifyScenarioError(err), budget.Requests()
		}
		return scenarioIssueIdentity, budget.Requests()
	}
	err = scenarioProbeSubmit(ctx, chain, chainCfg, budget)
	return classifyScenarioProbeError(chain, err), budget.Requests()
}

// scenarioProbeSubmit calls the chain's existing submit method exactly once with its fixed payload.
func scenarioProbeSubmit(ctx context.Context, chain string, chainCfg scenarioChainConfig, doer transport.HTTPDoer) error {
	if chain == "tron" {
		client, err := scenarioProbeTron(chainCfg, doer)
		if err != nil {
			return err
		}
		_, _, err = client.BroadcastTransaction(ctx, tron.TransactionPayload{JSON: json.RawMessage(scenarioProbeTronPayload)})
		return err
	}
	version := jsonrpc.V2
	if chain == "bitcoin" {
		version = jsonrpc.Bitcoin
	}
	rpc, err := scenarioProbeRPC(chainCfg, version, doer)
	if err != nil {
		return err
	}
	switch chain {
	case "ethereum", "bnb":
		client, err := evm.New(chainCfg.Endpoint, rpc)
		if err != nil {
			return err
		}
		_, err = client.SendRawTransaction(ctx, scenarioProbeEVMPayload)
		return err
	case "bitcoin":
		client, err := bitcoin.New(rpc)
		if err != nil {
			return err
		}
		_, err = client.SendRawTransaction(ctx, bitcoin.RawTransaction(scenarioProbeBitcoinPayload))
		return err
	case "solana":
		client, err := solana.New(rpc)
		if err != nil {
			return err
		}
		var noRetries uint
		_, err = client.SendEncodedTransactionWithOpts(ctx, scenarioProbeSolanaPayload, srpc.TransactionOpts{Encoding: ns.EncodingBase64, MaxRetries: &noRetries})
		return err
	}
	return apierror.Error{Kind: apierror.Configuration}
}

// classifyScenarioProbeError maps a single submit outcome: a node rejection is
// the expected PASS and an acceptance is FAIL.
func classifyScenarioProbeError(chain string, err error) scenarioIssue {
	if err == nil {
		return scenarioProbeIssueAccepted
	}
	var apiErr apierror.Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Kind == apierror.RPC && apiErr.HasRPCCode && apiErr.RPCCode == scenarioProbeMethodNotFound:
			return scenarioProbeIssueCapability
		case apiErr.Kind == apierror.HTTP && chain == "tron" && apiErr.HTTPStatus == http.StatusNotFound:
			return scenarioProbeIssueCapability
		case apiErr.Kind == apierror.RPC:
			return scenarioIssue{"PASS", ""}
		}
	}
	return classifyScenarioError(err)
}

// scenarioProbeIdentities verifies each live endpoint with the same identity reads as the read scenario.
func scenarioProbeIdentities() map[string]scenarioProbeIdentity {
	return map[string]scenarioProbeIdentity{
		"ethereum": scenarioProbeEVMIdentity,
		"bnb":      scenarioProbeEVMIdentity,
		"bitcoin":  scenarioProbeBitcoinIdentity,
		"solana":   scenarioProbeSolanaIdentity,
		"tron":     scenarioProbeTronIdentity,
	}
}

func scenarioProbeRPC(chainCfg scenarioChainConfig, version jsonrpc.Version, doer transport.HTTPDoer) (*jsonrpc.Client, error) {
	headers := http.Header{}
	for k, v := range chainCfg.Headers {
		headers.Set(k, v)
	}
	return jsonrpc.New(jsonrpc.Config{Endpoint: chainCfg.Endpoint, Headers: headers, MaxResponseBytes: scenarioProbeMaxResponseBytes, Version: version}, doer)
}

func scenarioProbeTron(chainCfg scenarioChainConfig, doer transport.HTTPDoer) (*tron.Client, error) {
	headers := http.Header{}
	for k, v := range chainCfg.Headers {
		headers.Set(k, v)
	}
	poster, err := httpjson.New(httpjson.Config{Endpoint: chainCfg.Endpoint, Headers: headers, MaxResponseBytes: scenarioProbeMaxResponseBytes}, doer)
	if err != nil {
		return nil, err
	}
	return tron.New(poster)
}

func scenarioProbeEVMIdentity(ctx context.Context, chain string, chainCfg scenarioChainConfig, doer transport.HTTPDoer) error {
	expected, ok := new(big.Int).SetString(chainCfg.Params["expected_chain_id"], 10)
	if !ok || expected.Sign() <= 0 {
		return apierror.Error{Kind: apierror.Configuration}
	}
	rpc, err := scenarioProbeRPC(chainCfg, jsonrpc.V2, doer)
	if err != nil {
		return err
	}
	switch chain {
	case "ethereum":
		c, err := ethereum.New(ethereum.Config{RawURL: chainCfg.Endpoint, ExpectedChainID: *expected}, rpc)
		if err != nil {
			return err
		}
		return c.ValidateNetwork(ctx)
	case "bnb":
		c, err := bnb.New(bnb.Config{RawURL: chainCfg.Endpoint, ExpectedChainID: *expected}, rpc)
		if err != nil {
			return err
		}
		return c.ValidateNetwork(ctx)
	}
	return apierror.Error{Kind: apierror.Configuration}
}

func scenarioProbeBitcoinIdentity(ctx context.Context, _ string, chainCfg scenarioChainConfig, doer transport.HTTPDoer) error {
	expectedChain := chainCfg.Params["expected_chain"]
	expectedGenesis := bitcoin.BlockHash(chainCfg.Params["expected_genesis"])
	if expectedChain == "" || !scenarioBitcoinHash(expectedGenesis.ToString()) {
		return apierror.Error{Kind: apierror.Configuration}
	}
	rpc, err := scenarioProbeRPC(chainCfg, jsonrpc.Bitcoin, doer)
	if err != nil {
		return err
	}
	client, err := bitcoin.New(rpc)
	if err != nil {
		return err
	}
	info, err := client.GetBlockchainInfo(ctx)
	if err != nil {
		return err
	}
	if info.Chain != expectedChain {
		return apierror.Error{Kind: apierror.NetworkMismatch}
	}
	genesis, err := client.GetBlockHash(ctx, 0)
	if err != nil {
		return err
	}
	if genesis != expectedGenesis {
		return apierror.Error{Kind: apierror.NetworkMismatch}
	}
	return nil
}

func scenarioProbeSolanaIdentity(ctx context.Context, _ string, chainCfg scenarioChainConfig, doer transport.HTTPDoer) error {
	expected, err := ns.HashFromBase58(chainCfg.Params["expected_genesis"])
	if err != nil || expected.IsZero() {
		return apierror.Error{Kind: apierror.Configuration}
	}
	rpc, err := scenarioProbeRPC(chainCfg, jsonrpc.V2, doer)
	if err != nil {
		return err
	}
	client, err := solana.New(rpc)
	if err != nil {
		return err
	}
	genesis, err := client.GetGenesisHash(ctx)
	if err != nil {
		return err
	}
	if genesis != expected {
		return apierror.Error{Kind: apierror.NetworkMismatch}
	}
	return nil
}

func scenarioProbeTronIdentity(ctx context.Context, _ string, chainCfg scenarioChainConfig, doer transport.HTTPDoer) error {
	expected := chainCfg.Params["expected_genesis"]
	if !scenarioTronHash(expected) {
		return apierror.Error{Kind: apierror.Configuration}
	}
	client, err := scenarioProbeTron(chainCfg, doer)
	if err != nil {
		return err
	}
	genesis, found, err := client.GetBlockByNum(ctx, 0)
	if err != nil {
		return err
	}
	if !found || !strings.EqualFold(genesis.BlockID, expected) {
		return apierror.Error{Kind: apierror.NetworkMismatch}
	}
	return nil
}
