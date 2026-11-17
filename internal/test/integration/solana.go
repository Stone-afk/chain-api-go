package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/solana"
	ns "github.com/solana-foundation/solana-go/v2"
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

type solanaReadInputs struct {
	Account                  ns.PublicKey                 `json:"account"`
	Read                     srpc.GetSlotOpts             `json:"read"`
	Blockhash                srpc.GetLatestBlockhashOpts  `json:"blockhash"`
	AccountOptions           srpc.GetAccountInfoOpts      `json:"account_options"`
	TransactionOptions       srpc.GetTransactionOpts      `json:"transaction_options"`
	BlockOptions             srpc.GetBlockOpts            `json:"block_options"`
	SearchTransactionHistory bool                         `json:"search_transaction_history"`
	FeeMessage               string                       `json:"fee_message"`
	SimulateTransaction      []byte                       `json:"simulate_transaction"`
	SimulateOptions          srpc.SimulateTransactionOpts `json:"simulate_options"`
}

type solanaSampleExpected struct {
	Transaction       srpc.GetTransactionResult `json:"transaction"`
	Blockhash         string                    `json:"blockhash"`
	PreviousBlockhash string                    `json:"previous_blockhash"`
	ParentSlot        uint64                    `json:"parent_slot"`
}

func solanaReadCases(c *solana.Client, m readManifest, d dynamicInputs) ([]harnessCase, error) {
	var in solanaReadInputs
	if decodeMaterial(d.Inputs, &in) != nil || len(m.Samples) < 3 {
		return nil, blocked()
	}
	confirmed := func(v srpc.CommitmentType) bool {
		return v == srpc.CommitmentConfirmed || v == srpc.CommitmentFinalized
	}
	if in.TransactionOptions.Encoding != ns.EncodingJSON || in.TransactionOptions.MaxSupportedTransactionVersion == nil || *in.TransactionOptions.MaxSupportedTransactionVersion != 0 || !confirmed(in.TransactionOptions.Commitment) || in.BlockOptions.Encoding != ns.EncodingJSON || in.BlockOptions.MaxSupportedTransactionVersion == nil || *in.BlockOptions.MaxSupportedTransactionVersion != 0 || !confirmed(in.BlockOptions.Commitment) || !in.SearchTransactionHistory {
		return nil, blocked()
	}
	ex := make([]solanaSampleExpected, len(m.Samples))
	for i, s := range m.Samples {
		if decodeMaterial(s.Expected, &ex[i]) != nil {
			return nil, blocked()
		}
	}
	identityPassed := false
	gate := func() bool { return identityPassed }
	s, e := m.Samples[0], ex[0]
	tx := func(s readSample, e solanaSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			signature, err := ns.SignatureFromBase58(s.TransactionID)
			if err != nil {
				return blocked()
			}
			v, err := c.GetTransaction(ctx, signature, &in.TransactionOptions)
			return checked(err, v != nil && v.Slot == s.HeightOrSlot && solanaTransactionEqual(*v, e.Transaction))
		}
	}
	block := func(s readSample, e solanaSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, err := c.GetBlockWithOpts(ctx, s.HeightOrSlot, &in.BlockOptions)
			return checked(err, v != nil && v.Blockhash.String() == e.Blockhash && v.PreviousBlockhash.String() == e.PreviousBlockhash && v.ParentSlot == e.ParentSlot)
		}
	}
	cases := []harnessCase{
		methodCase("solana.GetGenesisHash", func() bool { return true }, func(ctx context.Context) error {
			v, err := c.GetGenesisHash(ctx)
			identityPassed = err == nil && v.String() == m.NetworkIdentity.GenesisHash
			return identityResult(err, identityPassed)
		}),
		methodCase("solana.GetSlotWithOpts", gate, func(ctx context.Context) error {
			v, err := c.GetSlotWithOpts(ctx, &in.Read)
			return checked(err, v >= s.HeightOrSlot)
		}),
		methodCase("solana.GetBlockWithOpts", gate, block(s, e)),
		methodCase("solana.GetBalanceWithOpts", gate, func(ctx context.Context) error {
			_, err := c.GetBalanceWithOpts(ctx, in.Account, &srpc.GetBalanceOpts{Commitment: in.Read.Commitment, MinContextSlot: in.Read.MinContextSlot})
			return checked(err, true)
		}),
		methodCase("solana.GetAccountInfoWithOpts", gate, func(ctx context.Context) error {
			v, err := c.GetAccountInfoWithOpts(ctx, in.Account, &in.AccountOptions)
			var expected srpc.Account
			ok := json.Unmarshal(d.Preflight.Methods["GetAccountInfoWithOpts"].Expected, &expected) == nil
			return checked(err, ok && v != nil && v.Value != nil && valueEqual(*v.Value, expected))
		}),
		methodCase("solana.GetLatestBlockhashWithOpts", gate, func(ctx context.Context) error {
			v, err := c.GetLatestBlockhashWithOpts(ctx, &in.Blockhash)
			return checked(err, v != nil && v.Value != nil && v.Value.Blockhash != (ns.Hash{}))
		}),
		methodCase("solana.GetSignatureStatuses", gate, func(ctx context.Context) error {
			signature, err := ns.SignatureFromBase58(s.TransactionID)
			if err != nil {
				return blocked()
			}
			v, err := c.GetSignatureStatuses(ctx, in.SearchTransactionHistory, signature)
			return checked(err, v != nil && len(v.Value) == 1 && v.Value[0] != nil && v.Value[0].Slot == s.HeightOrSlot)
		}),
		methodCase("solana.GetTransaction", gate, tx(s, e)),
		methodCase("solana.GetFeeForMessageWithOpts", gate, func(ctx context.Context) error {
			v, err := c.GetFeeForMessageWithOpts(ctx, in.FeeMessage, &srpc.GetFeeForMessageOpts{Commitment: in.Read.Commitment, MinContextSlot: in.Read.MinContextSlot})
			return expectation(d.Preflight.Methods["GetFeeForMessageWithOpts"], err, v != nil && rawEqual(jsonBytes(v.Value), d.Preflight.Methods["GetFeeForMessageWithOpts"].Expected))
		}),
		methodCase("solana.SimulateRawTransactionWithOpts", gate, func(ctx context.Context) error {
			v, err := c.SimulateRawTransactionWithOpts(ctx, in.SimulateTransaction, &in.SimulateOptions)
			return expectation(d.Preflight.Methods["SimulateRawTransactionWithOpts"], err, v != nil && v.Value != nil && rawEqual(jsonBytes(v.Value.Err), d.Preflight.Methods["SimulateRawTransactionWithOpts"].Expected))
		})}
	for i := 0; i < 3; i++ {
		cases = append(cases, sampleCase(fmt.Sprintf("solana.sample.%d", i+1), gate, tx(m.Samples[i], ex[i]), block(m.Samples[i], ex[i])))
	}
	return cases, nil
}
func solanaSubmissionCases(c *solana.Client, m submissionMaterial, payload []byte, claim claimMaterial) ([]harnessCase, error) {
	var opts srpc.TransactionOpts
	if decodeMaterial(m.SendOptions, &opts) != nil {
		return nil, blocked()
	}
	identityPassed, claimSucceeded, sendAttempted, accepted, unknown := false, false, false, false, false
	return []harnessCase{
		methodCase("solana.identity", func() bool { return true }, func(ctx context.Context) error {
			v, e := c.GetGenesisHash(ctx)
			identityPassed = e == nil && v.String() == m.NetworkIdentity.GenesisHash
			return identityResult(e, identityPassed)
		}),
		{ID: "solana.submit", Run: func(_ context.Context, r harnessRequest) (string, error) {
			if !identityPassed || claimSucceeded {
				return resultStatus(blocked())
			}
			if e := claim(); e != nil {
				return resultStatus(blocked())
			}
			claimSucceeded = true
			return resultStatus(r(func(ctx context.Context) error {
				sendAttempted = true
				v, e := c.SendEncodedTransactionWithOpts(ctx, string(payload), opts)
				var a apierror.Error
				if errors.As(e, &a) && a.SubmissionUnknown {
					unknown = true
					return blocked()
				}
				accepted = e == nil && v.String() == m.TransactionID
				return checked(e, accepted)
			}))
		}},
		methodCase("solana.lookup", func() bool { return sendAttempted && (accepted || unknown) }, func(ctx context.Context) error {
			signature, e := ns.SignatureFromBase58(m.TransactionID)
			if e != nil {
				return blocked()
			}
			v, e := c.GetSignatureStatuses(ctx, true, signature)
			if e != nil {
				return checked(e, false)
			}
			if v == nil || len(v.Value) != 1 {
				return assertion(false)
			}
			if v.Value[0] == nil {
				return blocked()
			}
			s := v.Value[0]
			if !rawEqual(jsonBytes(s.Err), m.ExpectedExecution.SolanaErr) {
				return assertion(false)
			}
			if s.ConfirmationStatus != srpc.ConfirmationStatusFinalized && (s.ConfirmationStatus != srpc.ConfirmationStatusConfirmed || m.ExpectedExecution.SolanaCommitment != srpc.CommitmentConfirmed) {
				return blocked()
			}
			return nil
		})}, nil
}
