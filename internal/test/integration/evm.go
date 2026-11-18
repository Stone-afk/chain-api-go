package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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
	geth "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
	"math/big"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type runnerDeps struct {
	LookupEnv             func(string) (string, bool)
	ReadFile              func(string) ([]byte, error)
	NewDoer               func(harnessConfig) (transport.HTTPDoer, func(), error)
	Now                   func() time.Time
	Wait                  func(context.Context, time.Duration) error
	WriteReadReport       func([]byte) error
	ReadBroadcastJournal  func() ([]byte, error)
	AppendBroadcastRecord func([]byte) error
}

type claimMaterial func() error

type networkIdentity struct {
	ChainID            string `json:"chain_id"`
	BitcoinChain       string `json:"bitcoin_chain"`
	GenesisHash        string `json:"genesis_hash"`
	TronGenesisBlockID string `json:"tron_genesis_block_id"`
}

type rpcExpectation struct {
	Outcome    string          `json:"outcome"`
	RPCCode    int             `json:"rpc_code"`
	HasRPCCode bool            `json:"has_rpc_code"`
	NodeCode   string          `json:"node_code"`
	Expected   json.RawMessage `json:"expected"`
}

type preflightRecord struct {
	Status          string                    `json:"status"`
	ObservedAt      string                    `json:"observed_at"`
	EndpointAlias   string                    `json:"endpoint_alias"`
	NetworkIdentity networkIdentity           `json:"network_identity"`
	Methods         map[string]rpcExpectation `json:"methods"`
}

type readSample struct {
	SampleID      string            `json:"sample_id"`
	Category      string            `json:"category"`
	SourceURL     string            `json:"source_url"`
	ObservedAt    string            `json:"observed_at"`
	TransactionID string            `json:"transaction_id"`
	BlockHash     string            `json:"block_hash"`
	HeightOrSlot  uint64            `json:"height_or_slot"`
	Addresses     []string          `json:"addresses"`
	Expected      json.RawMessage   `json:"expected"`
	FieldSources  map[string]string `json:"field_sources"`
	RPCPreflight  preflightRecord   `json:"rpc_preflight"`
	Requirements  []string          `json:"requirements"`
}

type readManifest struct {
	ManifestVersion int             `json:"manifest_version"`
	Chain           string          `json:"chain"`
	Network         string          `json:"network"`
	NetworkIdentity networkIdentity `json:"network_identity"`
	Samples         []readSample    `json:"samples"`
}

type dynamicInputs struct {
	ManifestSHA256   string          `json:"manifest_sha256"`
	MaxResponseBytes int64           `json:"max_response_bytes"`
	CodeSnapshot     string          `json:"code_snapshot"`
	Inputs           json.RawMessage `json:"inputs"`
	Preflight        preflightRecord `json:"preflight"`
	ObservedAt       string          `json:"observed_at"`
	ExpiresAt        string          `json:"expires_at"`
}

type evmReadInputs struct {
	Address        common.Address   `json:"address"`
	AtBlockNumber  uint64           `json:"at_block_number"`
	Call           geth.CallMsg     `json:"call"`
	Estimate       geth.CallMsg     `json:"estimate"`
	LogFilter      geth.FilterQuery `json:"log_filter"`
	FeeCount       uint64           `json:"fee_count"`
	FeePercentiles []float64        `json:"fee_percentiles"`
}
type evmSampleExpected struct {
	Transaction *types.Transaction `json:"transaction"`
	Receipt     types.Receipt      `json:"receipt"`
	BlockHash   common.Hash        `json:"block_hash"`
	BlockNumber uint64             `json:"block_number"`
	ParentHash  common.Hash        `json:"parent_hash"`
}

type executionExpectation struct {
	EVMStatus               string              `json:"evm_status"`
	BitcoinMinConfirmations int64               `json:"bitcoin_min_confirmations"`
	SolanaCommitment        srpc.CommitmentType `json:"solana_commitment"`
	SolanaErr               json.RawMessage     `json:"solana_err"`
	TronReceiptResult       json.RawMessage     `json:"tron_receipt_result"`
}

type submissionMaterial struct {
	Version           int                  `json:"version"`
	Chain             string               `json:"chain"`
	Network           string               `json:"network"`
	EndpointAlias     string               `json:"endpoint_alias"`
	AuthorizationID   string               `json:"authorization_id"`
	Authorized        bool                 `json:"authorized"`
	ExpiresAt         string               `json:"expires_at"`
	NetworkIdentity   networkIdentity      `json:"network_identity"`
	PayloadPath       string               `json:"payload_path"`
	PayloadSHA256     string               `json:"payload_sha256"`
	TransactionID     string               `json:"transaction_id"`
	BlockHash         string               `json:"block_hash"`
	SendOptions       json.RawMessage      `json:"send_options"`
	ExpectedExecution executionExpectation `json:"expected_execution"`
}

type materialClaim struct {
	Chain           string `json:"chain"`
	AuthorizationID string `json:"authorization_id"`
	PayloadSHA256   string `json:"payload_sha256"`
	MaterialSHA256  string `json:"material_sha256"`
}

type broadcastRecord struct {
	Version    int           `json:"version"`
	Type       string        `json:"type"`
	RecordedAt string        `json:"recorded_at"`
	Claim      materialClaim `json:"claim"`
	Report     suiteReport   `json:"report"`
}

type chainReport struct {
	Chain          string        `json:"chain"`
	Alias          string        `json:"alias"`
	CodeSnapshot   string        `json:"code_snapshot"`
	MaterialDigest string        `json:"material_digest"`
	Result         harnessReport `json:"result"`
}

type suiteReport struct {
	Version    int           `json:"version"`
	Mode       string        `json:"mode"`
	Status     string        `json:"status"`
	StartedAt  string        `json:"started_at"`
	FinishedAt string        `json:"finished_at"`
	Chains     []chainReport `json:"chains"`
}

func blocked() error { return harnessFailure{"BLOCKED", "environment"} }
func assertion(ok bool) error {
	if !ok {
		return harnessFailure{"FAIL", "assertion"}
	}
	return nil
}
func checked(err error, ok bool) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return assertion(err == nil && ok)
}
func identityResult(err error, ok bool) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err != nil || !ok {
		return blocked()
	}
	return nil
}
func resultStatus(err error) (string, error) {
	if err != nil {
		return classifyHarnessError(err).status, err
	}
	return "PASS", nil
}
func jsonBytes(v any) []byte { b, _ := json.Marshal(v); return b }
func rawEqual(a, b json.RawMessage) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	var x, y any
	da, db := json.NewDecoder(bytes.NewReader(a)), json.NewDecoder(bytes.NewReader(b))
	da.UseNumber()
	db.UseNumber()
	return da.Decode(&x) == nil && db.Decode(&y) == nil && reflect.DeepEqual(x, y)
}
func valueEqual(a, b any) bool { return rawEqual(jsonBytes(a), jsonBytes(b)) }
func decodeMaterial(raw []byte, out any) error {
	if _, err := strictObject(raw); err != nil {
		return blocked()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return blocked()
	}
	return nil
}
func digestBytes(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func token(s string) bool         { return regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`).MatchString(s) }
func digestOK(s string) bool      { return regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(s) }
func observed(s string, now time.Time) bool {
	t, e := time.Parse(time.RFC3339, s)
	return e == nil && !t.After(now)
}
func unexpired(s string, now time.Time) bool {
	t, e := time.Parse(time.RFC3339, s)
	return e == nil && now.Before(t)
}
func publicURL(s string) bool {
	u, e := url.Parse(s)
	return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}
func validIdentity(chain string, n networkIdentity) bool {
	switch chain {
	case "ethereum", "bnb":
		v, ok := new(big.Int).SetString(n.ChainID, 10)
		return ok && regexp.MustCompile(`^[0-9]+$`).MatchString(n.ChainID) && v.Sign() > 0 && n.BitcoinChain == "" && n.GenesisHash == "" && n.TronGenesisBlockID == ""
	case "bitcoin":
		return n.BitcoinChain != "" && n.GenesisHash != "" && n.ChainID == "" && n.TronGenesisBlockID == ""
	case "solana":
		return n.GenesisHash != "" && n.ChainID == "" && n.BitcoinChain == "" && n.TronGenesisBlockID == ""
	case "tron":
		return n.TronGenesisBlockID != "" && n.ChainID == "" && n.BitcoinChain == "" && n.GenesisHash == ""
	}
	return false
}

const evmMethods = "ChainID BlockNumber BlockByNumber BalanceAt NonceAt CodeAt TransactionByHash TransactionReceipt FilterLogs CallContract EstimateGas SuggestGasPrice FeeHistory"
const bitcoinMethods = "GetBlockchainInfo GetBlockHash GetBlockCount GetBlock GetRawTransaction GetTxOut GetMempoolEntry EstimateSmartFee TestMempoolAccept"
const solanaMethods = "GetGenesisHash GetSlotWithOpts GetBlockWithOpts GetBalanceWithOpts GetAccountInfoWithOpts GetLatestBlockhashWithOpts GetSignatureStatuses GetTransaction GetFeeForMessageWithOpts SimulateRawTransactionWithOpts"
const tronMethods = "GetBlockByNum GetNodeInfo GetNowBlock GetAccount GetAccountResource GetChainParameters GetTransactionByID GetTransactionInfoByID GetSolidifiedAccount GetSolidifiedNowBlock GetSolidifiedBlockByNum GetSolidifiedTransactionByID GetSolidifiedTransactionInfoByID TriggerConstantContract EstimateEnergy"

func requiredIDs(chain, methods string, read bool) []string {
	names := strings.Fields(methods)
	if read {
		names = append(names, "sample.1", "sample.2", "sample.3")
	}
	for i := range names {
		names[i] = chain + "." + names[i]
	}
	return names
}
func preflightOK(p preflightRecord, chain, alias string, n networkIdentity, methods, allowed string, now time.Time) bool {
	if p.Status != "PASS" || p.EndpointAlias != alias || p.NetworkIdentity != n || !observed(p.ObservedAt, now) {
		return false
	}
	for k, v := range p.Methods {
		if !strings.Contains(" "+methods+" ", " "+k+" ") {
			return false
		}
		if v.Outcome == "success" {
			if v.HasRPCCode || v.NodeCode != "" || !validExpected(chain, k, v.Expected) {
				return false
			}
		} else if v.Outcome == "rpc_error" {
			if !strings.Contains(" "+allowed+" ", " "+k+" ") || chain != "tron" && !v.HasRPCCode {
				return false
			}
		} else {
			return false
		}
	}
	for _, k := range strings.Fields(methods) {
		if _, ok := p.Methods[k]; !ok {
			return false
		}
	}
	return true
}
func expectation(e rpcExpectation, err error, ok bool) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if e.Outcome == "rpc_error" {
		var a apierror.Error
		return checked(nil, errors.As(err, &a) && a.Kind == apierror.RPC && ((e.HasRPCCode && a.HasRPCCode && a.RPCCode == e.RPCCode) || (!e.HasRPCCode && a.NodeCode == e.NodeCode)))
	}
	return checked(err, ok)
}
func methodCase(id string, gate func() bool, op func(context.Context) error) harnessCase {
	return harnessCase{ID: id, Run: func(_ context.Context, r harnessRequest) (string, error) {
		if !gate() {
			return resultStatus(blocked())
		}
		return resultStatus(r(op))
	}}
}
func sampleCase(id string, gate func() bool, ops ...func(context.Context) error) harnessCase {
	return harnessCase{ID: id, Run: func(_ context.Context, r harnessRequest) (string, error) {
		if !gate() {
			return resultStatus(blocked())
		}
		var final error
		for _, op := range ops {
			if e := r(op); e != nil {
				final = e
			}
		}
		return resultStatus(final)
	}}
}
func validateRead(m readManifest, d dynamicInputs, cfg harnessConfig, chain, methods, allowed, history, keys string, now time.Time) error {
	if m.ManifestVersion != materialVersion(chain) || m.Chain != chain || m.Network != cfg.Network || !token(m.Network) || !validIdentity(chain, m.NetworkIdentity) || len(m.Samples) < 3 || !preflightOK(d.Preflight, chain, cfg.Alias, m.NetworkIdentity, methods, allowed, now) {
		return blocked()
	}
	categories := []string{"native", "token", "failed"}
	if chain == "bnb" {
		categories[2] = "contract"
	}
	if chain == "bitcoin" {
		categories = []string{"multi_output", "segwit", "structure"}
	}
	seen := map[string]bool{}
	for i, s := range m.Samples {
		if !token(s.SampleID) || s.TransactionID == "" || seen[s.TransactionID] || s.BlockHash == "" || !publicURL(s.SourceURL) || !observed(s.ObservedAt, now) || i < 3 && s.Category != categories[i] || !preflightOK(s.RPCPreflight, chain, cfg.Alias, m.NetworkIdentity, history, "", now) {
			return blocked()
		}
		seen[s.TransactionID] = true
		obj, e := strictObject(s.Expected)
		if e != nil || len(obj) != len(strings.Fields(keys)) || len(s.FieldSources) != len(obj) {
			return blocked()
		}
		for _, key := range strings.Fields(keys) {
			if _, ok := obj[key]; !ok {
				return blocked()
			}
			source := s.FieldSources[key]
			if strings.HasPrefix(source, "browser:") {
				if !publicURL(strings.TrimPrefix(source, "browser:")) {
					return blocked()
				}
			} else if !strings.HasPrefix(source, "independent_rpc:") || !strings.Contains(" "+methods+" ", " "+strings.TrimPrefix(source, "independent_rpc:")+" ") {
				return blocked()
			}
		}
	}
	return nil
}
func readMaterialClaims(raw []byte) ([]materialClaim, error) {
	var claims []materialClaim
	if len(raw) == 0 {
		return claims, nil
	}
	if raw[len(raw)-1] != '\n' {
		return nil, blocked()
	}
	for _, line := range bytes.Split(raw[:len(raw)-1], []byte{'\n'}) {
		var r broadcastRecord
		if decodeMaterial(line, &r) != nil || r.Version != 1 {
			return nil, blocked()
		}
		if _, e := time.Parse(time.RFC3339, r.RecordedAt); e != nil {
			return nil, blocked()
		}
		switch r.Type {
		case "claim":
			if !validClaim(r.Claim) || !reflect.DeepEqual(r.Report, suiteReport{}) {
				return nil, blocked()
			}
			claims = append(claims, r.Claim)
		case "report":
			if r.Claim != (materialClaim{}) || !validStoredReport(r.Report) {
				return nil, blocked()
			}
		default:
			return nil, blocked()
		}
	}
	return claims, nil
}
func validClaim(c materialClaim) bool {
	return strings.Contains(" ethereum bnb bitcoin solana tron ", " "+c.Chain+" ") && token(c.AuthorizationID) && digestOK(c.PayloadSHA256) && digestOK(c.MaterialSHA256)
}
func claimSubmission(c materialClaim, at string, previous []materialClaim, appendRecord func([]byte) error) error {
	if !validClaim(c) {
		return blocked()
	}
	if _, e := time.Parse(time.RFC3339, at); e != nil {
		return blocked()
	}
	for _, p := range previous {
		if p.Chain == c.Chain && (p.AuthorizationID == c.AuthorizationID || p.PayloadSHA256 == c.PayloadSHA256) {
			return blocked()
		}
	}
	return appendRecord(append(jsonBytes(broadcastRecord{Version: 1, Type: "claim", RecordedAt: at, Claim: c}), '\n'))
}

func bitcoinTransactionEqual(a, b bitcoin.Transaction) bool {
	return valueEqual(a.TxID, b.TxID) && valueEqual(a.WTxID, b.WTxID) && valueEqual(a.Hex, b.Hex) && valueEqual(a.Version, b.Version) && valueEqual(a.LockTime, b.LockTime) && valueEqual(a.Size, b.Size) && valueEqual(a.VSize, b.VSize) && valueEqual(a.Weight, b.Weight) && bitcoinTxInsEqual(a.Vin, b.Vin) && bitcoinTxOutputsEqual(a.Vout, b.Vout) && rawEqual(a.BlockHash, b.BlockHash) && rawEqual(a.Confirmations, b.Confirmations) && rawEqual(a.BlockTime, b.BlockTime) && rawEqual(a.InActiveChain, b.InActiveChain)
}

func bitcoinTxInEqual(a, b bitcoin.TxIn) bool {
	return valueEqual(a.Sequence, b.Sequence) && rawEqual(a.TxID, b.TxID) && rawEqual(a.Vout, b.Vout) && rawEqual(a.ScriptSig, b.ScriptSig) && rawEqual(a.Coinbase, b.Coinbase) && rawEqual(a.Witness, b.Witness)
}

func bitcoinTxOutputEqual(a, b bitcoin.TxOutput) bool {
	return valueEqual(a.N, b.N) && valueEqual(a.Value, b.Value) && bitcoinScriptPubKeyEqual(a.ScriptPubKey, b.ScriptPubKey)
}

func bitcoinScriptPubKeyEqual(a, b bitcoin.ScriptPubKey) bool {
	return valueEqual(a.Hex, b.Hex) && valueEqual(a.Type, b.Type) && rawEqual(a.Address, b.Address)
}
func bitcoinTxInsEqual(a, b []bitcoin.TxIn) bool {
	if (a == nil) != (b == nil) || len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bitcoinTxInEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}
func bitcoinTxOutputsEqual(a, b []bitcoin.TxOutput) bool {
	if (a == nil) != (b == nil) || len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bitcoinTxOutputEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func solanaTransactionEqual(a, b srpc.GetTransactionResult) bool {
	return valueEqual(a, b)
}

func tronTransactionEqual(a, b tron.Transaction) bool {
	return valueEqual(a.TxID, b.TxID) && rawEqual(a.RawData, b.RawData) && valueEqual(a.RawDataHex, b.RawDataHex) && valueEqual(a.Signature, b.Signature) && rawEqual(a.Ret, b.Ret)
}

func tronTransactionInfoEqual(a, b tron.TransactionInfo) bool {
	return valueEqual(a.ID, b.ID) && valueEqual(a.Fee, b.Fee) && valueEqual(a.BlockNumber, b.BlockNumber) && valueEqual(a.BlockTimeStamp, b.BlockTimeStamp) && valueEqual(a.ContractResult, b.ContractResult) && valueEqual(a.ContractAddress, b.ContractAddress) && rawEqual(a.Receipt, b.Receipt) && rawEqual(a.Log, b.Log) && rawEqual(a.InternalTransactions, b.InternalTransactions) && rawEqual(a.Result, b.Result) && rawEqual(a.ResMessage, b.ResMessage)
}
func ethereumReadCases(c *ethereum.Client, m readManifest, d dynamicInputs) ([]harnessCase, error) {
	return evmReadCases(c.Client, m, d)
}
func bnbReadCases(c *bnb.Client, m readManifest, d dynamicInputs) ([]harnessCase, error) {
	return evmReadCases(c.Client, m, d)
}

// evmRequestContext admits each underlying Caller operation, including uncles.
// Dependency errors remain available to the case's frozen expectation; admission
// and deadline errors are still recorded by the harness.
type evmRequestContext struct {
	context.Context
	request harnessRequest
}

func (c evmRequestContext) CallRPC(call func(context.Context) error) error {
	var callErr error
	err := c.request(func(ctx context.Context) error { callErr = call(ctx); return nil })
	if err != nil {
		return err
	}
	return callErr
}
func evmMethodCase(id string, gate func() bool, op func(context.Context) error) harnessCase {
	return evmSampleCase(id, gate, op)
}
func evmSampleCase(id string, gate func() bool, ops ...func(context.Context) error) harnessCase {
	return harnessCase{ID: id, Run: func(ctx context.Context, r harnessRequest) (string, error) {
		if !gate() {
			return resultStatus(blocked())
		}
		var final error
		for _, op := range ops {
			if err := op(evmRequestContext{ctx, r}); err != nil {
				final = err
			}
		}
		return resultStatus(final)
	}}
}
func materialVersion(chain string) int {
	if chain == "ethereum" || chain == "bnb" || chain == "solana" {
		return 2
	}
	return 1
}
func evmChecked(err error, ok bool) error {
	var issue harnessIssue
	if errors.As(err, &issue) {
		return err
	}
	return checked(err, ok)
}
func evmExpectation(e rpcExpectation, err error, ok bool) error {
	var issue harnessIssue
	if errors.As(err, &issue) {
		return err
	}
	return expectation(e, err, ok)
}
func evmReadCases(c *evm.Client, m readManifest, d dynamicInputs) ([]harnessCase, error) {
	var in evmReadInputs
	if decodeMaterial(d.Inputs, &in) != nil || len(m.Samples) < 3 {
		return nil, blocked()
	}
	expected := make([]evmSampleExpected, len(m.Samples))
	for i, s := range m.Samples {
		if decodeMaterial(s.Expected, &expected[i]) != nil || expected[i].Transaction == nil {
			return nil, blocked()
		}
	}
	identityPassed := false
	gate := func() bool { return identityPassed }
	prefix := m.Chain + "."
	at := new(big.Int).SetUint64(in.AtBlockNumber)
	s, e := m.Samples[0], expected[0]
	tx := func(s readSample, e evmSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, pending, err := c.TransactionByHash(ctx, common.HexToHash(s.TransactionID))
			return evmChecked(err, err == nil && !pending && v.Hash() == common.HexToHash(s.TransactionID) && v.Hash() == e.Transaction.Hash())
		}
	}
	receipt := func(s readSample, e evmSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, err := c.TransactionReceipt(ctx, common.HexToHash(s.TransactionID))
			return evmChecked(err, err == nil && v.TxHash == common.HexToHash(s.TransactionID) && v.BlockHash == common.HexToHash(s.BlockHash) && reflect.DeepEqual(*v, e.Receipt))
		}
	}
	block := func(s readSample, e evmSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, err := c.BlockByNumber(ctx, new(big.Int).SetUint64(s.HeightOrSlot))
			return evmChecked(err, err == nil && v.Hash() == e.BlockHash && v.Hash() == common.HexToHash(s.BlockHash) && v.NumberU64() == e.BlockNumber && v.ParentHash() == e.ParentHash)
		}
	}
	cases := []harnessCase{
		evmMethodCase(prefix+"ChainID", func() bool { return true }, func(ctx context.Context) error {
			v, err := c.ChainID(ctx)
			want, _ := new(big.Int).SetString(m.NetworkIdentity.ChainID, 10)
			identityPassed = err == nil && want != nil && v.Cmp(want) == 0
			return identityResult(err, identityPassed)
		}),
		evmMethodCase(prefix+"BlockNumber", gate, func(ctx context.Context) error {
			v, err := c.BlockNumber(ctx)
			return evmChecked(err, v >= s.HeightOrSlot)
		}),
		evmMethodCase(prefix+"BlockByNumber", gate, block(s, e)),
		evmMethodCase(prefix+"BalanceAt", gate, func(ctx context.Context) error {
			v, err := c.BalanceAt(ctx, in.Address, at)
			var text string
			dec := json.Unmarshal(d.Preflight.Methods["BalanceAt"].Expected, &text)
			n, ok := new(big.Int).SetString(text, 10)
			return evmChecked(err, err == nil && dec == nil && ok && n.Sign() >= 0 && v.Cmp(n) == 0)
		}),
		evmMethodCase(prefix+"NonceAt", gate, func(ctx context.Context) error {
			v, err := c.NonceAt(ctx, in.Address, at)
			return evmChecked(err, rawEqual(jsonBytes(v), d.Preflight.Methods["NonceAt"].Expected))
		}),
		evmMethodCase(prefix+"CodeAt", gate, func(ctx context.Context) error {
			v, err := c.CodeAt(ctx, in.Address, at)
			return evmChecked(err, rawEqual(jsonBytes(v), d.Preflight.Methods["CodeAt"].Expected))
		}),
		evmMethodCase(prefix+"TransactionByHash", gate, tx(s, e)),
		evmMethodCase(prefix+"TransactionReceipt", gate, receipt(s, e)),
		evmMethodCase(prefix+"FilterLogs", gate, func(ctx context.Context) error {
			v, err := c.FilterLogs(ctx, in.LogFilter)
			var want []types.Log
			dec := json.Unmarshal(d.Preflight.Methods["FilterLogs"].Expected, &want)
			return evmChecked(err, dec == nil && reflect.DeepEqual(v, want))
		}),
		evmMethodCase(prefix+"CallContract", gate, func(ctx context.Context) error {
			v, err := c.CallContract(ctx, in.Call, at)
			return evmExpectation(d.Preflight.Methods["CallContract"], err, rawEqual(jsonBytes(v), d.Preflight.Methods["CallContract"].Expected))
		}),
		evmMethodCase(prefix+"EstimateGas", gate, func(ctx context.Context) error {
			v, err := c.EstimateGas(ctx, in.Estimate)
			return evmExpectation(d.Preflight.Methods["EstimateGas"], err, v > 0)
		}),
		evmMethodCase(prefix+"SuggestGasPrice", gate, func(ctx context.Context) error {
			v, err := c.SuggestGasPrice(ctx)
			return evmExpectation(d.Preflight.Methods["SuggestGasPrice"], err, err == nil && v.Sign() >= 0)
		}),
		evmMethodCase(prefix+"FeeHistory", gate, func(ctx context.Context) error {
			v, err := c.FeeHistory(ctx, in.FeeCount, at, in.FeePercentiles)
			var want geth.FeeHistory
			dec := json.Unmarshal(d.Preflight.Methods["FeeHistory"].Expected, &want)
			return evmExpectation(d.Preflight.Methods["FeeHistory"], err, err == nil && dec == nil && valueEqual(*v, want))
		}),
	}
	for i := 0; i < 3; i++ {
		cases = append(cases, evmSampleCase(fmt.Sprintf("%ssample.%d", prefix, i+1), gate, tx(m.Samples[i], expected[i]), receipt(m.Samples[i], expected[i]), block(m.Samples[i], expected[i])))
	}
	return cases, nil
}
func ethereumSubmissionCases(c *ethereum.Client, m submissionMaterial, p []byte, claim claimMaterial) ([]harnessCase, error) {
	return evmSubmissionCases(c.Client, m, p, claim)
}
func bnbSubmissionCases(c *bnb.Client, m submissionMaterial, p []byte, claim claimMaterial) ([]harnessCase, error) {
	return evmSubmissionCases(c.Client, m, p, claim)
}
func evmSubmissionCases(c *evm.Client, m submissionMaterial, payload []byte, claim claimMaterial) ([]harnessCase, error) {
	identityPassed, claimSucceeded, sendAttempted, accepted, unknown := false, false, false, false, false
	return []harnessCase{
		evmMethodCase(m.Chain+".identity", func() bool { return true }, func(ctx context.Context) error {
			v, err := c.ChainID(ctx)
			n, ok := new(big.Int).SetString(m.NetworkIdentity.ChainID, 10)
			identityPassed = err == nil && ok && v.Cmp(n) == 0
			return identityResult(err, identityPassed)
		}),
		{ID: m.Chain + ".submit", Run: func(ctx context.Context, r harnessRequest) (string, error) {
			if !identityPassed || claimSucceeded {
				return resultStatus(blocked())
			}
			if err := claim(); err != nil {
				return resultStatus(blocked())
			}
			claimSucceeded = true
			sendAttempted = true
			v, err := c.SendRawTransaction(evmRequestContext{ctx, r}, string(payload))
			var a apierror.Error
			if errors.As(err, &a) && a.SubmissionUnknown {
				unknown = true
				return resultStatus(blocked())
			}
			accepted = err == nil && v == common.HexToHash(m.TransactionID)
			return resultStatus(evmChecked(err, accepted))
		}},
		evmMethodCase(m.Chain+".lookup", func() bool { return sendAttempted && (accepted || unknown) }, func(ctx context.Context) error {
			v, err := c.TransactionReceipt(ctx, common.HexToHash(m.TransactionID))
			if errors.Is(err, geth.NotFound) {
				return blocked()
			}
			if err != nil {
				return evmChecked(err, false)
			}
			return assertion(v.TxHash == common.HexToHash(m.TransactionID) && fmt.Sprintf("0x%x", v.Status) == m.ExpectedExecution.EVMStatus)
		}),
	}, nil
}
func loadDynamic(deps runnerDeps, path, chain, mode string) (harnessConfig, dynamicInputs, error) {
	cfg, err := loadHarnessConfig(path, chain+"_"+mode, deps.ReadFile, deps.LookupEnv)
	if err != nil {
		return cfg, dynamicInputs{}, blocked()
	}
	raw, err := deps.ReadFile(cfg.DynamicInputsPath)
	var d dynamicInputs
	if err != nil || decodeMaterial(raw, &d) != nil {
		return cfg, d, blocked()
	}
	if !digestOK(d.ManifestSHA256) || d.MaxResponseBytes <= 0 || !regexp.MustCompile(`^([a-fA-F0-9]{40}|[a-fA-F0-9]{64})$`).MatchString(d.CodeSnapshot) || !observed(d.ObservedAt, deps.Now()) || !unexpired(d.ExpiresAt, deps.Now()) {
		return cfg, dynamicInputs{}, blocked()
	}
	return cfg, d, nil
}
func loadRead(deps runnerDeps, path, chain, methods, allowed, history, keys string) (harnessConfig, dynamicInputs, readManifest, []byte, error) {
	cfg, d, err := loadDynamic(deps, path, chain, "read")
	var m readManifest
	if err != nil {
		return cfg, d, m, nil, err
	}
	raw, err := deps.ReadFile("testdata/" + chain + "/samples.json")
	if err != nil || decodeMaterial(raw, &m) != nil {
		return cfg, d, m, raw, blocked()
	}
	if _, err = verifyManifest(raw, d.ManifestSHA256); err != nil {
		return cfg, d, m, raw, blocked()
	}
	return cfg, d, m, raw, validateRead(m, d, cfg, chain, methods, allowed, history, keys, deps.Now())
}
func validateSubmission(m submissionMaterial, d dynamicInputs, cfg harnessConfig, chain, identityMethods string, now time.Time) error {
	if m.Version != materialVersion(chain) || m.Chain != chain || m.Network != cfg.Network || !token(m.Network) || m.EndpointAlias != cfg.Alias || !token(m.AuthorizationID) || !m.Authorized || !unexpired(m.ExpiresAt, now) || !validIdentity(chain, m.NetworkIdentity) || m.PayloadPath == "" || !digestOK(m.PayloadSHA256) || m.TransactionID == "" || chain != "bitcoin" && m.BlockHash != "" || !preflightOK(d.Preflight, chain, cfg.Alias, m.NetworkIdentity, identityMethods, "", now) {
		return blocked()
	}
	inputs, err := strictObject(d.Inputs)
	if err != nil || len(inputs) != 0 {
		return blocked()
	}
	options, err := strictObject(m.SendOptions)
	if err != nil || chain != "solana" && len(options) != 0 {
		return blocked()
	}
	e := m.ExpectedExecution
	switch chain {
	case "ethereum", "bnb":
		if e.EVMStatus != "0x0" && e.EVMStatus != "0x1" {
			return blocked()
		}
		e.EVMStatus = ""
	case "bitcoin":
		if e.BitcoinMinConfirmations < 1 {
			return blocked()
		}
		e.BitcoinMinConfirmations = 0
	case "solana":
		if e.SolanaCommitment != srpc.CommitmentConfirmed && e.SolanaCommitment != srpc.CommitmentFinalized || e.SolanaErr == nil {
			return blocked()
		}
		var execution any
		decoder := json.NewDecoder(bytes.NewReader(e.SolanaErr))
		decoder.UseNumber()
		if decoder.Decode(&execution) != nil {
			return blocked()
		}
		switch execution.(type) {
		case nil, map[string]any, string:
		default:
			return blocked()
		}
		e.SolanaCommitment = ""
		e.SolanaErr = nil
		var opts srpc.TransactionOpts
		if decodeMaterial(m.SendOptions, &opts) != nil {
			return blocked()
		}
	case "tron":
		var result string
		if e.TronReceiptResult == nil || (json.Unmarshal(e.TronReceiptResult, &result) != nil || rawEqual(e.TronReceiptResult, json.RawMessage("null"))) && !rawEqual(e.TronReceiptResult, json.RawMessage(`{"absent":true}`)) {
			return blocked()
		}
		e.TronReceiptResult = nil
	}
	if !reflect.DeepEqual(e, executionExpectation{}) {
		return blocked()
	}
	return nil
}
func loadSubmission(deps runnerDeps, path, chain, identityMethods string) (harnessConfig, dynamicInputs, submissionMaterial, []byte, []byte, error) {
	cfg, d, err := loadDynamic(deps, path, chain, "broadcast")
	var m submissionMaterial
	if err != nil {
		return cfg, d, m, nil, nil, err
	}
	raw, err := deps.ReadFile(cfg.SubmissionMaterialPath)
	if err != nil || decodeMaterial(raw, &m) != nil {
		return cfg, d, m, raw, nil, blocked()
	}
	if _, err = verifyManifest(raw, d.ManifestSHA256); err != nil {
		return cfg, d, m, raw, nil, blocked()
	}
	if err = validateSubmission(m, d, cfg, chain, identityMethods, deps.Now()); err != nil {
		return cfg, d, m, raw, nil, err
	}
	payload, err := deps.ReadFile(m.PayloadPath)
	if err != nil || digestBytes(payload) != m.PayloadSHA256 {
		return cfg, d, m, raw, nil, blocked()
	}
	return cfg, d, m, raw, payload, nil
}
func harnessInputs(deps runnerDeps, cfg harnessConfig, d dynamicInputs, raw []byte) harnessDeps {
	return harnessDeps{Load: func() (harnessConfig, error) { return cfg, nil }, LoadManifest: func() ([]byte, string, error) { return raw, d.ManifestSHA256, nil }, Now: deps.Now, Wait: deps.Wait}
}
func protocolHeaders(cfg harnessConfig) http.Header {
	h := make(http.Header)
	for k, v := range cfg.Headers {
		h.Set(k, v)
	}
	return h
}
func initialChain(chain, mode string) chainReport {
	return chainReport{Chain: chain, Alias: chain + "_" + mode, Result: harnessReport{Status: "BLOCKED", Category: "environment"}}
}
func finishSuite(r suiteReport, deps runnerDeps) suiteReport {
	r.Status = "PASS"
	for _, c := range r.Chains {
		if c.Result.Status == "FAIL" {
			r.Status = "FAIL"
		} else if c.Result.Status != "PASS" && r.Status != "FAIL" {
			r.Status = "BLOCKED"
		}
	}
	r.FinishedAt = deps.Now().Format(time.RFC3339)
	var err error
	if r.Mode == "read" {
		err = deps.WriteReadReport(jsonBytes(r))
	} else {
		err = deps.AppendBroadcastRecord(append(jsonBytes(broadcastRecord{Version: 1, Type: "report", RecordedAt: r.FinishedAt, Report: r}), '\n'))
	}
	if err != nil && r.Status != "FAIL" {
		r.Status = "BLOCKED"
	}
	return r
}

func runLiveRead(ctx context.Context, deps runnerDeps) suiteReport {
	r := suiteReport{Mode: "read", Status: "NOT_RUN"}
	enabled, _ := deps.LookupEnv("CHAIN_API_GO_LIVE_TEST")
	if enabled != "1" {
		return r
	}
	r.Version = 1
	r.StartedAt = deps.Now().Format(time.RFC3339)
	path, _ := deps.LookupEnv("CHAIN_API_GO_LIVE_CONFIG")
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("ethereum", "read")
		cfg, d, m, raw, err := loadRead(deps, path, "ethereum", evmMethods, "CallContract EstimateGas SuggestGasPrice FeeHistory", "TransactionByHash TransactionReceipt BlockByNumber", "transaction receipt block_hash block_number parent_hash")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.V2}, doer)
		if err != nil {
			return out
		}
		n, _ := new(big.Int).SetString(m.NetworkIdentity.ChainID, 10)
		client, err := ethereum.New(ethereum.Config{RawURL: cfg.Endpoint, ExpectedChainID: *n}, rpc)
		if err != nil {
			return out
		}
		cases, err := ethereumReadCases(client, m, d)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("ethereum", evmMethods, true), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("bnb", "read")
		cfg, d, m, raw, err := loadRead(deps, path, "bnb", evmMethods, "CallContract EstimateGas SuggestGasPrice FeeHistory", "TransactionByHash TransactionReceipt BlockByNumber", "transaction receipt block_hash block_number parent_hash")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.V2}, doer)
		if err != nil {
			return out
		}
		n, _ := new(big.Int).SetString(m.NetworkIdentity.ChainID, 10)
		client, err := bnb.New(bnb.Config{RawURL: cfg.Endpoint, ExpectedChainID: *n}, rpc)
		if err != nil {
			return out
		}
		cases, err := bnbReadCases(client, m, d)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("bnb", evmMethods, true), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("bitcoin", "read")
		cfg, d, m, raw, err := loadRead(deps, path, "bitcoin", bitcoinMethods, "GetTxOut GetMempoolEntry EstimateSmartFee TestMempoolAccept", "GetRawTransaction GetBlock", "transaction block_hash block_height txids")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.Bitcoin}, doer)
		if err != nil {
			return out
		}
		client, err := bitcoin.New(rpc)
		if err != nil {
			return out
		}
		cases, err := bitcoinReadCases(client, m, d)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("bitcoin", bitcoinMethods, true), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("solana", "read")
		cfg, d, m, raw, err := loadRead(deps, path, "solana", solanaMethods, "GetFeeForMessageWithOpts SimulateRawTransactionWithOpts", "GetTransaction GetBlockWithOpts", "transaction blockhash previous_blockhash parent_slot")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.V2}, doer)
		if err != nil {
			return out
		}
		client, err := solana.New(rpc)
		if err != nil {
			return out
		}
		cases, err := solanaReadCases(client, m, d)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("solana", solanaMethods, true), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("tron", "read")
		cfg, d, m, raw, err := loadRead(deps, path, "tron", tronMethods, "GetAccountResource TriggerConstantContract EstimateEnergy", "GetTransactionByID GetTransactionInfoByID GetBlockByNum GetSolidifiedTransactionByID GetSolidifiedTransactionInfoByID", "transaction transaction_info block_id")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := httpjson.New(httpjson.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes}, doer)
		if err != nil {
			return out
		}
		client, err := tron.New(rpc)
		if err != nil {
			return out
		}
		cases, err := tronReadCases(client, m, d)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("tron", tronMethods, true), cases)
		return out
	}())
	return finishSuite(r, deps)
}

func runLiveBroadcast(ctx context.Context, deps runnerDeps) suiteReport {
	r := suiteReport{Mode: "broadcast", Status: "NOT_RUN"}
	enabled, _ := deps.LookupEnv("CHAIN_API_GO_LIVE_TEST")
	if enabled != "1" {
		return r
	}
	enabled, _ = deps.LookupEnv("CHAIN_API_GO_LIVE_BROADCAST")
	if enabled != "1" {
		return r
	}
	r.Version = 1
	r.StartedAt = deps.Now().Format(time.RFC3339)
	path, _ := deps.LookupEnv("CHAIN_API_GO_LIVE_CONFIG")
	journal, journalErr := deps.ReadBroadcastJournal()
	claims, parseErr := readMaterialClaims(journal)
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("ethereum", "broadcast")
		if journalErr != nil || parseErr != nil {
			return out
		}
		cfg, d, m, raw, payload, err := loadSubmission(deps, path, "ethereum", "ChainID")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.V2}, doer)
		if err != nil {
			return out
		}
		n, _ := new(big.Int).SetString(m.NetworkIdentity.ChainID, 10)
		client, err := ethereum.New(ethereum.Config{RawURL: cfg.Endpoint, ExpectedChainID: *n}, rpc)
		if err != nil {
			return out
		}
		claim := func() error {
			c := materialClaim{Chain: m.Chain, AuthorizationID: m.AuthorizationID, PayloadSHA256: m.PayloadSHA256, MaterialSHA256: d.ManifestSHA256}
			if e := claimSubmission(c, deps.Now().Format(time.RFC3339), claims, deps.AppendBroadcastRecord); e != nil {
				return e
			}
			claims = append(claims, c)
			return nil
		}
		cases, err := ethereumSubmissionCases(client, m, payload, claim)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("ethereum", "identity submit lookup", false), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("bnb", "broadcast")
		if journalErr != nil || parseErr != nil {
			return out
		}
		cfg, d, m, raw, payload, err := loadSubmission(deps, path, "bnb", "ChainID")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.V2}, doer)
		if err != nil {
			return out
		}
		n, _ := new(big.Int).SetString(m.NetworkIdentity.ChainID, 10)
		client, err := bnb.New(bnb.Config{RawURL: cfg.Endpoint, ExpectedChainID: *n}, rpc)
		if err != nil {
			return out
		}
		claim := func() error {
			c := materialClaim{Chain: m.Chain, AuthorizationID: m.AuthorizationID, PayloadSHA256: m.PayloadSHA256, MaterialSHA256: d.ManifestSHA256}
			if e := claimSubmission(c, deps.Now().Format(time.RFC3339), claims, deps.AppendBroadcastRecord); e != nil {
				return e
			}
			claims = append(claims, c)
			return nil
		}
		cases, err := bnbSubmissionCases(client, m, payload, claim)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("bnb", "identity submit lookup", false), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("bitcoin", "broadcast")
		if journalErr != nil || parseErr != nil {
			return out
		}
		cfg, d, m, raw, payload, err := loadSubmission(deps, path, "bitcoin", "GetBlockchainInfo GetBlockHash")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.Bitcoin}, doer)
		if err != nil {
			return out
		}
		client, err := bitcoin.New(rpc)
		if err != nil {
			return out
		}
		claim := func() error {
			c := materialClaim{Chain: m.Chain, AuthorizationID: m.AuthorizationID, PayloadSHA256: m.PayloadSHA256, MaterialSHA256: d.ManifestSHA256}
			if e := claimSubmission(c, deps.Now().Format(time.RFC3339), claims, deps.AppendBroadcastRecord); e != nil {
				return e
			}
			claims = append(claims, c)
			return nil
		}
		cases, err := bitcoinSubmissionCases(client, m, payload, claim)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("bitcoin", "identity submit lookup", false), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("solana", "broadcast")
		if journalErr != nil || parseErr != nil {
			return out
		}
		cfg, d, m, raw, payload, err := loadSubmission(deps, path, "solana", "GetGenesisHash")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes, Version: jsonrpc.V2}, doer)
		if err != nil {
			return out
		}
		client, err := solana.New(rpc)
		if err != nil {
			return out
		}
		claim := func() error {
			c := materialClaim{Chain: m.Chain, AuthorizationID: m.AuthorizationID, PayloadSHA256: m.PayloadSHA256, MaterialSHA256: d.ManifestSHA256}
			if e := claimSubmission(c, deps.Now().Format(time.RFC3339), claims, deps.AppendBroadcastRecord); e != nil {
				return e
			}
			claims = append(claims, c)
			return nil
		}
		cases, err := solanaSubmissionCases(client, m, payload, claim)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("solana", "identity submit lookup", false), cases)
		return out
	}())
	r.Chains = append(r.Chains, func() chainReport {
		out := initialChain("tron", "broadcast")
		if journalErr != nil || parseErr != nil {
			return out
		}
		cfg, d, m, raw, payload, err := loadSubmission(deps, path, "tron", "GetBlockByNum")
		if err != nil {
			return out
		}
		out.CodeSnapshot = d.CodeSnapshot
		out.MaterialDigest = d.ManifestSHA256
		doer, cleanup, err := deps.NewDoer(cfg)
		if err != nil {
			return out
		}
		defer cleanup()
		rpc, err := httpjson.New(httpjson.Config{Endpoint: cfg.Endpoint, Headers: protocolHeaders(cfg), MaxResponseBytes: d.MaxResponseBytes}, doer)
		if err != nil {
			return out
		}
		client, err := tron.New(rpc)
		if err != nil {
			return out
		}
		claim := func() error {
			c := materialClaim{Chain: m.Chain, AuthorizationID: m.AuthorizationID, PayloadSHA256: m.PayloadSHA256, MaterialSHA256: d.ManifestSHA256}
			if e := claimSubmission(c, deps.Now().Format(time.RFC3339), claims, deps.AppendBroadcastRecord); e != nil {
				return e
			}
			claims = append(claims, c)
			return nil
		}
		cases, err := tronSubmissionCases(client, m, payload, claim)
		if err != nil {
			return out
		}
		out.Result = runHarness(ctx, true, harnessInputs(deps, cfg, d, raw), requiredIDs("tron", "identity submit lookup", false), cases)
		return out
	}())
	return finishSuite(r, deps)
}

// Expected values belong to the frozen live material dictionary, not a node schema.
func validExpected(chain, method string, raw json.RawMessage) bool {
	if raw == nil {
		return false
	}
	var text string
	var number uint64
	var flag bool
	switch chain {
	case "ethereum", "bnb":
		switch method {
		case "BalanceAt":
			if json.Unmarshal(raw, &text) != nil {
				return false
			}
			n, ok := new(big.Int).SetString(text, 10)
			return ok && n.Sign() >= 0
		case "NonceAt":
			return !rawEqual(raw, json.RawMessage("null")) && json.Unmarshal(raw, &number) == nil
		case "CodeAt", "CallContract":
			var data []byte
			return !rawEqual(raw, json.RawMessage("null")) && json.Unmarshal(raw, &data) == nil
		case "FilterLogs":
			var logs []types.Log
			return len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '[' && json.Unmarshal(raw, &logs) == nil
		case "FeeHistory":
			var v geth.FeeHistory
			return decodeMaterial(raw, &v) == nil
		}
	case "bitcoin":
		if method == "GetTxOut" || method == "EstimateSmartFee" || method == "TestMempoolAccept" {
			return !rawEqual(raw, json.RawMessage("null")) && json.Unmarshal(raw, &flag) == nil
		}
	case "solana":
		if method == "GetAccountInfoWithOpts" {
			var account srpc.Account
			return decodeMaterial(raw, &account) == nil
		}
		if method == "GetFeeForMessageWithOpts" {
			var fee *uint64
			return json.Unmarshal(raw, &fee) == nil
		}
		if method == "SimulateRawTransactionWithOpts" {
			return json.Valid(raw)
		}
	case "tron":
		if method == "GetAccount" || method == "GetSolidifiedAccount" {
			return !rawEqual(raw, json.RawMessage("null")) && json.Unmarshal(raw, &flag) == nil
		}
		if method == "TriggerConstantContract" {
			var values []string
			return len(bytes.TrimSpace(raw)) > 0 && bytes.TrimSpace(raw)[0] == '[' && json.Unmarshal(raw, &values) == nil
		}
	}
	return rawEqual(raw, json.RawMessage("null"))
}

func validStoredReport(r suiteReport) bool {
	if r.Version != 1 || r.Mode != "broadcast" || len(r.Chains) != 5 || (r.Status != "PASS" && r.Status != "FAIL" && r.Status != "BLOCKED") {
		return false
	}
	for _, stamp := range []string{r.StartedAt, r.FinishedAt} {
		if _, e := time.Parse(time.RFC3339, stamp); e != nil {
			return false
		}
	}
	for i, chain := range []string{"ethereum", "bnb", "bitcoin", "solana", "tron"} {
		c := r.Chains[i]
		if c.Chain != chain || c.Alias != chain+"_broadcast" || c.Result.Requests < 0 {
			return false
		}
		if c.CodeSnapshot != "" && !regexp.MustCompile(`^([a-fA-F0-9]{40}|[a-fA-F0-9]{64})$`).MatchString(c.CodeSnapshot) || c.MaterialDigest != "" && !digestOK(c.MaterialDigest) || c.Result.Digest != "" && !digestOK(c.Result.Digest) {
			return false
		}
		if c.Result.Status != "PASS" && c.Result.Status != "FAIL" && c.Result.Status != "BLOCKED" {
			return false
		}
		for id, status := range c.Result.Cases {
			if id != chain+".identity" && id != chain+".submit" && id != chain+".lookup" {
				return false
			}
			if status != "PASS" && status != "FAIL" && status != "BLOCKED" && status != "NOT_RUN" {
				return false
			}
		}
		if c.Result.Status == "PASS" {
			for _, id := range requiredIDs(chain, "identity submit lookup", false) {
				if c.Result.Cases[id] != "PASS" {
					return false
				}
			}
		}
	}
	return true
}
