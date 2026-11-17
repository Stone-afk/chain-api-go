package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/tron"
)

type tronReadInputs struct {
	Account      tron.GetAccountRequest   `json:"account"`
	ContractCall tron.ContractCallRequest `json:"contract_call"`
}

type tronSampleExpected struct {
	Transaction     tron.Transaction     `json:"transaction"`
	TransactionInfo tron.TransactionInfo `json:"transaction_info"`
	BlockID         string               `json:"block_id"`
}

func tronReadCases(c *tron.Client, m readManifest, d dynamicInputs) ([]harnessCase, error) {
	var in tronReadInputs
	if decodeMaterial(d.Inputs, &in) != nil || len(m.Samples) < 3 {
		return nil, blocked()
	}
	ex := make([]tronSampleExpected, len(m.Samples))
	for i, s := range m.Samples {
		if s.HeightOrSlot > 1<<63-1 || decodeMaterial(s.Expected, &ex[i]) != nil {
			return nil, blocked()
		}
	}
	identityPassed := false
	gate := func() bool { return identityPassed }
	s, e := m.Samples[0], ex[0]
	tx := func(s readSample, e tronSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, p, err := c.GetTransactionByID(ctx, tron.TxID(s.TransactionID))
			return checked(err, p && v.TxID == s.TransactionID && tronTransactionEqual(v, e.Transaction))
		}
	}
	info := func(s readSample, e tronSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, p, err := c.GetTransactionInfoByID(ctx, tron.TxID(s.TransactionID))
			return checked(err, p && v.ID == s.TransactionID && tronTransactionInfoEqual(v, e.TransactionInfo))
		}
	}
	solidTx := func(s readSample, e tronSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, p, err := c.GetSolidifiedTransactionByID(ctx, tron.TxID(s.TransactionID))
			return checked(err, p && v.TxID == s.TransactionID && tronTransactionEqual(v, e.Transaction))
		}
	}
	solidInfo := func(s readSample, e tronSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, p, err := c.GetSolidifiedTransactionInfoByID(ctx, tron.TxID(s.TransactionID))
			return checked(err, p && v.ID == s.TransactionID && tronTransactionInfoEqual(v, e.TransactionInfo))
		}
	}
	block := func(s readSample, e tronSampleExpected) func(context.Context) error {
		return func(ctx context.Context) error {
			v, p, err := c.GetBlockByNum(ctx, int64(s.HeightOrSlot))
			return checked(err, p && v.BlockID == e.BlockID)
		}
	}
	cases := []harnessCase{
		methodCase("tron.GetBlockByNum", func() bool { return true }, func(ctx context.Context) error {
			v, p, err := c.GetBlockByNum(ctx, 0)
			identityPassed = err == nil && p && v.BlockID == m.NetworkIdentity.TronGenesisBlockID
			return identityResult(err, identityPassed)
		}),
		methodCase("tron.GetNodeInfo", gate, func(ctx context.Context) error {
			v, err := c.GetNodeInfo(ctx)
			return checked(err, v.Block != "" && v.SolidityBlock != "")
		}),
		methodCase("tron.GetNowBlock", gate, func(ctx context.Context) error {
			v, p, err := c.GetNowBlock(ctx)
			return checked(err, p && v.BlockID != "")
		}),
		methodCase("tron.GetAccount", gate, func(ctx context.Context) error {
			v, p, err := c.GetAccount(ctx, in.Account)
			return checked(err, rawEqual(jsonBytes(p), d.Preflight.Methods["GetAccount"].Expected) && (!p || v.Address != "" && v.Balance >= 0))
		}),
		methodCase("tron.GetAccountResource", gate, func(ctx context.Context) error {
			v, err := c.GetAccountResource(ctx, in.Account)
			return expectation(d.Preflight.Methods["GetAccountResource"], err, v.FreeNetUsed >= 0 && v.FreeNetLimit >= 0 && v.NetUsed >= 0 && v.NetLimit >= 0 && v.EnergyUsed >= 0 && v.EnergyLimit >= 0 && v.TotalNetLimit >= 0 && v.TotalNetWeight >= 0 && v.TotalEnergyLimit >= 0 && v.TotalEnergyWeight >= 0)
		}),
		methodCase("tron.GetChainParameters", gate, func(ctx context.Context) error {
			v, err := c.GetChainParameters(ctx)
			ok := len(v.ChainParameter) > 0
			for _, p := range v.ChainParameter {
				ok = ok && p.Key != ""
			}
			return checked(err, ok)
		}),
		methodCase("tron.GetTransactionByID", gate, tx(s, e)), methodCase("tron.GetTransactionInfoByID", gate, info(s, e)),
		methodCase("tron.GetSolidifiedAccount", gate, func(ctx context.Context) error {
			v, p, err := c.GetSolidifiedAccount(ctx, in.Account)
			return checked(err, rawEqual(jsonBytes(p), d.Preflight.Methods["GetSolidifiedAccount"].Expected) && (!p || v.Address != "" && v.Balance >= 0))
		}),
		methodCase("tron.GetSolidifiedNowBlock", gate, func(ctx context.Context) error {
			v, p, err := c.GetSolidifiedNowBlock(ctx)
			return checked(err, p && v.BlockID != "")
		}),
		methodCase("tron.GetSolidifiedBlockByNum", gate, func(ctx context.Context) error {
			v, p, err := c.GetSolidifiedBlockByNum(ctx, int64(s.HeightOrSlot))
			return checked(err, p && v.BlockID == e.BlockID)
		}),
		methodCase("tron.GetSolidifiedTransactionByID", gate, solidTx(s, e)), methodCase("tron.GetSolidifiedTransactionInfoByID", gate, solidInfo(s, e)),
		methodCase("tron.TriggerConstantContract", gate, func(ctx context.Context) error {
			v, err := c.TriggerConstantContract(ctx, in.ContractCall)
			return expectation(d.Preflight.Methods["TriggerConstantContract"], err, v.Result.Result && rawEqual(jsonBytes(v.ConstantResult), d.Preflight.Methods["TriggerConstantContract"].Expected))
		}),
		methodCase("tron.EstimateEnergy", gate, func(ctx context.Context) error {
			v, err := c.EstimateEnergy(ctx, in.ContractCall)
			return expectation(d.Preflight.Methods["EstimateEnergy"], err, v.Result.Result && v.EnergyRequired >= 0)
		})}
	for i := 0; i < 3; i++ {
		cases = append(cases, sampleCase(fmt.Sprintf("tron.sample.%d", i+1), gate, tx(m.Samples[i], ex[i]), info(m.Samples[i], ex[i]), block(m.Samples[i], ex[i]), solidTx(m.Samples[i], ex[i]), solidInfo(m.Samples[i], ex[i])))
	}
	return cases, nil
}
func tronSubmissionCases(c *tron.Client, m submissionMaterial, payload []byte, claim claimMaterial) ([]harnessCase, error) {
	identityPassed, claimSucceeded, sendAttempted, accepted, unknown := false, false, false, false, false
	return []harnessCase{
		methodCase("tron.identity", func() bool { return true }, func(ctx context.Context) error {
			v, p, e := c.GetBlockByNum(ctx, 0)
			identityPassed = e == nil && p && v.BlockID == m.NetworkIdentity.TronGenesisBlockID
			return identityResult(e, identityPassed)
		}),
		{ID: "tron.submit", Run: func(_ context.Context, r harnessRequest) (string, error) {
			if !identityPassed || claimSucceeded {
				return resultStatus(blocked())
			}
			if e := claim(); e != nil {
				return resultStatus(blocked())
			}
			claimSucceeded = true
			return resultStatus(r(func(ctx context.Context) error {
				sendAttempted = true
				v, p, e := c.BroadcastTransaction(ctx, tron.TransactionPayload{JSON: json.RawMessage(payload)})
				var a apierror.Error
				if errors.As(e, &a) && a.SubmissionUnknown {
					unknown = true
					return blocked()
				}
				accepted = e == nil && p && (v.TxID == "" || v.TxID == m.TransactionID)
				return checked(e, accepted)
			}))
		}},
		methodCase("tron.lookup", func() bool { return sendAttempted && (accepted || unknown) }, func(ctx context.Context) error {
			v, p, e := c.GetSolidifiedTransactionInfoByID(ctx, tron.TxID(m.TransactionID))
			if e != nil {
				return checked(e, false)
			}
			if !p {
				return blocked()
			}
			var receipt map[string]json.RawMessage
			if v.Receipt != nil && json.Unmarshal(v.Receipt, &receipt) != nil {
				return assertion(false)
			}
			result, present := receipt["result"]
			ok := rawEqual(result, m.ExpectedExecution.TronReceiptResult)
			if rawEqual(m.ExpectedExecution.TronReceiptResult, json.RawMessage(`{"absent":true}`)) {
				ok = !present
			}
			return assertion(v.ID == m.TransactionID && ok)
		})}, nil
}
