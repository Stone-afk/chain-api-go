package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/bitcoin"
)

type bitcoinReadInputs struct {
	TxOut            bitcoin.GetTxOutRequest         `json:"txout"`
	MempoolTxID      bitcoin.TxID                    `json:"mempool_txid"`
	Estimate         bitcoin.EstimateSmartFeeRequest `json:"estimate"`
	PreflightPayload bitcoin.RawTransaction          `json:"preflight_payload"`
}

type bitcoinSampleExpected struct {
	Transaction bitcoin.Transaction `json:"transaction"`
	BlockHash   string              `json:"block_hash"`
	BlockHeight int64               `json:"block_height"`
	TxIDs       []string            `json:"txids"`
}

func bitcoinReadCases(c *bitcoin.Client, m readManifest, d dynamicInputs) ([]harnessCase, error) {
	var in bitcoinReadInputs
	if decodeMaterial(d.Inputs, &in) != nil || len(m.Samples) < 3 {
		return nil, blocked()
	}
	ex := make([]bitcoinSampleExpected, len(m.Samples))
	for i, s := range m.Samples {
		if decodeMaterial(s.Expected, &ex[i]) != nil {
			return nil, blocked()
		}
	}
	chainPassed, identityPassed := false, false
	gate := func() bool { return identityPassed }
	prefix := "bitcoin."
	s, e := m.Samples[0], ex[0]
	tx := func(s readSample, e bitcoinSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, err := c.GetRawTransaction(ctx, bitcoin.GetRawTransactionRequest{TxID: bitcoin.TxID(s.TransactionID), BlockHash: bitcoin.BlockHash(s.BlockHash), HasBlockHash: true})
			var confirmations int64
			ok := json.Unmarshal(v.Confirmations, &confirmations) == nil && string(v.Confirmations) != "null" && confirmations >= 0
			v.Confirmations = e.Transaction.Confirmations
			return checked(err, ok && v.TxID == s.TransactionID && bitcoinTransactionEqual(v, e.Transaction))
		}
	}
	block := func(s readSample, e bitcoinSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, err := c.GetBlock(ctx, bitcoin.BlockHash(s.BlockHash))
			return checked(err, v.Hash == s.BlockHash && v.Hash == e.BlockHash && v.Height == e.BlockHeight && valueEqual(v.TxIDs, e.TxIDs))
		}
	}
	cases := []harnessCase{
		methodCase(prefix+"GetBlockchainInfo", func() bool { return true }, func(ctx context.Context) error {
			v, err := c.GetBlockchainInfo(ctx)
			chainPassed = err == nil && v.Chain == m.NetworkIdentity.BitcoinChain
			return identityResult(err, chainPassed)
		}),
		methodCase(prefix+"GetBlockHash", func() bool { return chainPassed }, func(ctx context.Context) error {
			v, err := c.GetBlockHash(ctx, 0)
			identityPassed = err == nil && string(v) == m.NetworkIdentity.GenesisHash
			return identityResult(err, identityPassed)
		}),
		methodCase(prefix+"GetBlockCount", gate, func(ctx context.Context) error {
			v, err := c.GetBlockCount(ctx)
			return checked(err, v >= 0 && uint64(v) >= s.HeightOrSlot)
		}),
		methodCase(prefix+"GetBlock", gate, block(s, e)), methodCase(prefix+"GetRawTransaction", gate, tx(s, e)),
		methodCase(prefix+"GetTxOut", gate, func(ctx context.Context) error {
			v, p, err := c.GetTxOut(ctx, in.TxOut)
			return expectation(d.Preflight.Methods["GetTxOut"], err, rawEqual(jsonBytes(p), d.Preflight.Methods["GetTxOut"].Expected) && (!p || v.Value >= 0 && v.BestBlock != ""))
		}),
		methodCase(prefix+"GetMempoolEntry", gate, func(ctx context.Context) error {
			v, err := c.GetMempoolEntry(ctx, in.MempoolTxID)
			return expectation(d.Preflight.Methods["GetMempoolEntry"], err, v.BaseFee >= 0 && v.VSize > 0)
		}),
		methodCase(prefix+"EstimateSmartFee", gate, func(ctx context.Context) error {
			v, p, err := c.EstimateSmartFee(ctx, in.Estimate)
			return expectation(d.Preflight.Methods["EstimateSmartFee"], err, rawEqual(jsonBytes(p), d.Preflight.Methods["EstimateSmartFee"].Expected) && (!p || v.FeeRate >= 0))
		}),
		methodCase(prefix+"TestMempoolAccept", gate, func(ctx context.Context) error {
			v, err := c.TestMempoolAccept(ctx, in.PreflightPayload)
			return expectation(d.Preflight.Methods["TestMempoolAccept"], err, rawEqual(jsonBytes(v.Allowed), d.Preflight.Methods["TestMempoolAccept"].Expected))
		})}
	for i := 0; i < 3; i++ {
		cases = append(cases, sampleCase(fmt.Sprintf("bitcoin.sample.%d", i+1), gate, tx(m.Samples[i], ex[i]), block(m.Samples[i], ex[i])))
	}
	return cases, nil
}
func bitcoinSubmissionCases(c *bitcoin.Client, m submissionMaterial, payload []byte, claim claimMaterial) ([]harnessCase, error) {
	identityPassed, claimSucceeded, sendAttempted, accepted, unknown := false, false, false, false, false
	return []harnessCase{
		{ID: "bitcoin.identity", Run: func(_ context.Context, r harnessRequest) (string, error) {
			e := r(func(ctx context.Context) error {
				v, e := c.GetBlockchainInfo(ctx)
				return identityResult(e, v.Chain == m.NetworkIdentity.BitcoinChain)
			})
			if e != nil {
				return resultStatus(e)
			}
			e = r(func(ctx context.Context) error {
				v, e := c.GetBlockHash(ctx, 0)
				return identityResult(e, string(v) == m.NetworkIdentity.GenesisHash)
			})
			identityPassed = e == nil
			return resultStatus(e)
		}},
		{ID: "bitcoin.submit", Run: func(_ context.Context, r harnessRequest) (string, error) {
			if !identityPassed || claimSucceeded {
				return resultStatus(blocked())
			}
			if e := claim(); e != nil {
				return resultStatus(blocked())
			}
			claimSucceeded = true
			return resultStatus(r(func(ctx context.Context) error {
				sendAttempted = true
				v, e := c.SendRawTransaction(ctx, bitcoin.RawTransaction(string(payload)))
				var a apierror.Error
				if errors.As(e, &a) && a.SubmissionUnknown {
					unknown = true
					return blocked()
				}
				accepted = e == nil && string(v) == m.TransactionID
				return checked(e, accepted)
			}))
		}},
		methodCase("bitcoin.lookup", func() bool { return sendAttempted && (accepted || unknown) }, func(ctx context.Context) error {
			v, e := c.GetRawTransaction(ctx, bitcoin.GetRawTransactionRequest{TxID: bitcoin.TxID(m.TransactionID), BlockHash: bitcoin.BlockHash(m.BlockHash), HasBlockHash: m.BlockHash != ""})
			if e != nil {
				return checked(e, false)
			}
			if v.Confirmations == nil || string(v.Confirmations) == "null" {
				return blocked()
			}
			var n int64
			if json.Unmarshal(v.Confirmations, &n) != nil {
				return assertion(false)
			}
			if n < m.ExpectedExecution.BitcoinMinConfirmations {
				return blocked()
			}
			return nil
		})}, nil
}
