package integration

import (
	"context"
	"encoding/base64"
	"net/http"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/jsonrpc"
	"github.com/Stone-afk/chain-api-go/solana"
	"github.com/Stone-afk/chain-api-go/transport"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

const (
	scenarioSolanaMaxResponseBytes = 8 << 20
	scenarioSolanaSignatureSize    = 64
	scenarioSolanaKeySize          = 32
	scenarioSolanaHeaderSize       = 3
)

// runScenarioSolana runs Solana through doer with JSON-RPC 2.0 and returns a
// classified issue for every non-submit catalog ID of the chain.
//
// chainCfg.Params carries expected_genesis (base58 genesis hash) and an
// optional account (base58 public key). GetGenesisHash runs first; on
// mismatch every other entry is BLOCKED without a request. Reads are anchored
// on the finalized slot, its block fetched with signature details only, and
// its first transaction fetched as base64 bytes. The message is sliced from
// those bytes by byte offsets only; an invalid signature section is BLOCKED
// material. Balance and account reads target account, or the fee payer (first
// message account key) when account is absent. Simulation uses the fetched
// bytes with sigVerify false. A null fee value is PASS. Submit methods are
// never called.
func runScenarioSolana(ctx context.Context, _ scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]scenarioIssue {
	const chain = "solana"
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
	record := func(method string, err error) bool {
		out[chain+"/"+method] = classifyScenarioError(err)
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
	configuration := classifyScenarioError(apierror.Error{Kind: apierror.Configuration})

	expectedGenesis, err := ns.HashFromBase58(chainCfg.Params["expected_genesis"])
	if err != nil || expectedGenesis.IsZero() {
		fill(configuration)
		return out
	}
	var account ns.PublicKey
	hasAccount := chainCfg.Params["account"] != ""
	if hasAccount {
		if account, err = ns.PublicKeyFromBase58(chainCfg.Params["account"]); err != nil {
			fill(configuration)
			return out
		}
	}
	headers := http.Header{}
	for k, v := range chainCfg.Headers {
		headers.Set(k, v)
	}
	rpc, err := jsonrpc.New(jsonrpc.Config{Endpoint: chainCfg.Endpoint, Headers: headers, MaxResponseBytes: scenarioSolanaMaxResponseBytes, Version: jsonrpc.V2}, doer)
	if err != nil {
		fill(classifyScenarioError(err))
		return out
	}
	client, err := solana.New(rpc)
	if err != nil {
		fill(classifyScenarioError(err))
		return out
	}

	genesis, err := client.GetGenesisHash(ctx)
	if !record("GetGenesisHash", err) {
		fill(scenarioIssueIdentity)
		return out
	}
	if genesis != expectedGenesis {
		out[chain+"/GetGenesisHash"] = classifyScenarioError(apierror.Error{Kind: apierror.NetworkMismatch})
		fill(scenarioIssueIdentity)
		return out
	}

	maxVersion := srpc.MaxSupportedTransactionVersion0
	noRewards := false
	slot, err := client.GetSlotWithOpts(ctx, &srpc.GetSlotOpts{Commitment: srpc.CommitmentFinalized})
	if record("GetSlotWithOpts", err) {
		check("GetSlotWithOpts", slot > 0)
	}
	if out[chain+"/GetSlotWithOpts"].Status != "PASS" {
		out[chain+"/GetFirstAvailableBlock"] = scenarioIssueDependency
	} else {
		first, err := client.GetFirstAvailableBlock(ctx)
		if record("GetFirstAvailableBlock", err) {
			check("GetFirstAvailableBlock", first <= slot)
		}
	}
	var raw []byte
	var signature ns.Signature
	if out[chain+"/GetSlotWithOpts"].Status != "PASS" {
		setAll(scenarioIssueDependency, "GetBlockWithOpts", "GetBlocks", "GetTransaction", "GetSignatureStatuses")
	} else {
		block, err := client.GetBlockWithOpts(ctx, slot, &srpc.GetBlockOpts{TransactionDetails: srpc.TransactionDetailsSignatures, Rewards: &noRewards, Commitment: srpc.CommitmentFinalized, MaxSupportedTransactionVersion: &maxVersion})
		if record("GetBlockWithOpts", err) {
			check("GetBlockWithOpts", block != nil && !block.Blockhash.IsZero() && block.ParentSlot < slot)
		}
		switch {
		case out[chain+"/GetBlockWithOpts"].Status != "PASS":
			setAll(scenarioIssueDependency, "GetTransaction", "GetSignatureStatuses")
		case len(block.Signatures) == 0 || block.Signatures[0].IsZero():
			setAll(scenarioIssueMaterial, "GetTransaction", "GetSignatureStatuses")
		default:
			signature = block.Signatures[0]
			tx, err := client.GetTransaction(ctx, signature, &srpc.GetTransactionOpts{Encoding: ns.EncodingBase64, Commitment: srpc.CommitmentFinalized, MaxSupportedTransactionVersion: &maxVersion})
			if record("GetTransaction", err) {
				ok := tx != nil && tx.Slot == slot && tx.Transaction != nil && tx.Meta != nil
				if ok {
					raw = tx.Transaction.GetBinary()
				}
				check("GetTransaction", ok && len(raw) > 0)
			}
			if out[chain+"/GetTransaction"].Status != "PASS" {
				raw = nil
			}

			statuses, err := client.GetSignatureStatuses(ctx, true, signature)
			if record("GetSignatureStatuses", err) {
				check("GetSignatureStatuses", statuses != nil && len(statuses.Value) == 1 && statuses.Value[0] != nil && statuses.Value[0].Slot == slot && statuses.Value[0].ConfirmationStatus == srpc.ConfirmationStatusFinalized)
			}
		}
		if out[chain+"/GetBlockWithOpts"].Status == "PASS" {
			blocks, err := client.GetBlocks(ctx, slot, &slot, srpc.CommitmentFinalized)
			if record("GetBlocks", err) {
				check("GetBlocks", len(blocks) == 1 && blocks[0] == slot)
			}
		} else {
			out[chain+"/GetBlocks"] = scenarioIssueDependency
		}
	}

	message, feePayer, sliced := scenarioSolanaSlice(raw, signature)
	if !hasAccount && sliced {
		account, hasAccount = feePayer, true
	}
	if !hasAccount {
		missing := scenarioIssueMaterial
		if raw == nil {
			missing = scenarioIssueDependency
		}
		setAll(missing, "GetBalanceWithOpts", "GetAccountInfoWithOpts")
	} else {
		balance, err := client.GetBalanceWithOpts(ctx, account, &srpc.GetBalanceOpts{Commitment: srpc.CommitmentFinalized})
		if record("GetBalanceWithOpts", err) {
			check("GetBalanceWithOpts", balance != nil)
		}
		info, err := client.GetAccountInfoWithOpts(ctx, account, &srpc.GetAccountInfoOpts{Encoding: ns.EncodingBase64, Commitment: srpc.CommitmentFinalized})
		if record("GetAccountInfoWithOpts", err) {
			check("GetAccountInfoWithOpts", info != nil && info.Value != nil)
		}
	}

	latest, err := client.GetLatestBlockhashWithOpts(ctx, &srpc.GetLatestBlockhashOpts{Commitment: srpc.CommitmentFinalized})
	if record("GetLatestBlockhashWithOpts", err) {
		check("GetLatestBlockhashWithOpts", latest != nil && latest.Value != nil && !latest.Value.Blockhash.IsZero() && latest.Value.LastValidBlockHeight > 0)
	}

	switch {
	case raw == nil:
		setAll(scenarioIssueDependency, "GetFeeForMessageWithOpts", "SimulateRawTransactionWithOpts")
	case !sliced:
		setAll(scenarioIssueMaterial, "GetFeeForMessageWithOpts", "SimulateRawTransactionWithOpts")
	default:
		fee, err := client.GetFeeForMessageWithOpts(ctx, base64.StdEncoding.EncodeToString(message), &srpc.GetFeeForMessageOpts{Commitment: srpc.CommitmentFinalized})
		if record("GetFeeForMessageWithOpts", err) {
			check("GetFeeForMessageWithOpts", fee != nil)
		}
		sim, err := client.SimulateRawTransactionWithOpts(ctx, raw, &srpc.SimulateTransactionOpts{SigVerify: false, Commitment: srpc.CommitmentFinalized})
		if record("SimulateRawTransactionWithOpts", err) {
			check("SimulateRawTransactionWithOpts", sim != nil && sim.Value != nil)
		}
	}
	fill(scenarioIssueDependency)
	return out
}

// scenarioSolanaSlice splits wire transaction bytes into the message and its
// fee payer by byte offsets only: a compact-u16 signature count, the
// signatures, then the message whose header must require that many
// signatures. The first signature must equal want.
func scenarioSolanaSlice(raw []byte, want ns.Signature) ([]byte, ns.PublicKey, bool) {
	count, n, ok := scenarioSolanaShortVec(raw)
	if !ok || count == 0 {
		return nil, ns.PublicKey{}, false
	}
	start := n + count*scenarioSolanaSignatureSize
	if start >= len(raw) || ns.SignatureFromBytes(raw[n:n+scenarioSolanaSignatureSize]) != want {
		return nil, ns.PublicKey{}, false
	}
	header := start
	if raw[start]&0x80 != 0 {
		header++
	}
	message := raw[header:]
	if len(message) <= scenarioSolanaHeaderSize || int(message[0]) != count {
		return nil, ns.PublicKey{}, false
	}
	keys, kn, ok := scenarioSolanaShortVec(message[scenarioSolanaHeaderSize:])
	keyStart := scenarioSolanaHeaderSize + kn
	if !ok || keys < count || keyStart+scenarioSolanaKeySize > len(message) {
		return nil, ns.PublicKey{}, false
	}
	return raw[start:], ns.PublicKeyFromBytes(message[keyStart : keyStart+scenarioSolanaKeySize]), true
}

// scenarioSolanaShortVec decodes a compact-u16 length prefix.
func scenarioSolanaShortVec(b []byte) (int, int, bool) {
	v := 0
	for i := 0; i < 3 && i < len(b); i++ {
		v |= int(b[i]&0x7f) << (7 * i)
		if b[i]&0x80 == 0 {
			return v, i + 1, true
		}
	}
	return 0, 0, false
}

// scenarioSolanaRunner adapts runScenarioSolana to the status-only runner contract.
func scenarioSolanaRunner() scenarioRunner {
	return func(ctx context.Context, cfg scenarioConfig, chainCfg scenarioChainConfig, doer transport.HTTPDoer) map[string]string {
		got := map[string]string{}
		for id, issue := range runScenarioSolana(ctx, cfg, chainCfg, doer) {
			got[id] = issue.Status
		}
		return got
	}
}
