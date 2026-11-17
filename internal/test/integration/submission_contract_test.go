package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/bitcoin"
	"github.com/Stone-afk/chain-api-go/bnb"
	"github.com/Stone-afk/chain-api-go/ethereum"
	"github.com/Stone-afk/chain-api-go/solana"
	"github.com/Stone-afk/chain-api-go/tron"
)

func contractSubmissionMaterial(chain string) (submissionMaterial, []byte) {
	payload := []byte("aa")
	m := submissionMaterial{Version: 1, Chain: chain, Network: "fixture", EndpointAlias: chain + "_broadcast", AuthorizationID: "synthetic-authorization", Authorized: true, ExpiresAt: contractExpires, NetworkIdentity: contractIdentity(chain), PayloadPath: chain + ".payload", TransactionID: contractID(chain, 0), SendOptions: json.RawMessage(`{}`)}
	switch chain {
	case "ethereum", "bnb":
		m.Version = 2
		payload = []byte("0xaa")
		m.ExpectedExecution.EVMStatus = "0x1"
	case "bitcoin":
		m.ExpectedExecution.BitcoinMinConfirmations = 1
		m.BlockHash = contractID(chain, 7)
	case "solana":
		m.Version = 2
		payload = []byte("YQ==")
		m.SendOptions = json.RawMessage(`{"encoding":"base64","maxRetries":0,"skipPreflight":false,"minContextSlot":0}`)
		m.ExpectedExecution.SolanaCommitment = "confirmed"
		m.ExpectedExecution.SolanaErr = json.RawMessage(`null`)
	case "tron":
		payload = []byte(`{ "raw_data": {"synthetic":true}, "signature": ["aa"] }`)
		m.ExpectedExecution.TronReceiptResult = json.RawMessage(`"SUCCESS"`)
	}
	m.PayloadSHA256 = contractSHA(payload)
	return m, payload
}

// Emit only the chain-specific execution dictionary; JSON null is a Solana value, not a cross-chain default.
func contractMaterialJSON(m submissionMaterial) []byte {
	var obj map[string]any
	_ = json.Unmarshal(contractJSON(m), &obj)
	var expected map[string]any
	switch m.Chain {
	case "ethereum", "bnb":
		expected = map[string]any{"evm_status": m.ExpectedExecution.EVMStatus}
	case "bitcoin":
		expected = map[string]any{"bitcoin_min_confirmations": m.ExpectedExecution.BitcoinMinConfirmations}
	case "solana":
		expected = map[string]any{"solana_commitment": m.ExpectedExecution.SolanaCommitment, "solana_err": m.ExpectedExecution.SolanaErr}
	case "tron":
		expected = map[string]any{"tron_receipt_result": m.ExpectedExecution.TronReceiptResult}
	}
	obj["expected_execution"] = expected
	return contractJSON(obj)
}
func contractSubmissionCases(t *testing.T, chain string, m submissionMaterial, p []byte, claim claimMaterial, call contractCaller) ([]harnessCase, error) {
	t.Helper()
	switch chain {
	case "ethereum":
		c, e := ethereum.New(ethereum.Config{ExpectedChainID: *big.NewInt(1)}, call)
		if e != nil {
			t.Fatal(e)
		}
		return ethereumSubmissionCases(c, m, p, claim)
	case "bnb":
		c, e := bnb.New(bnb.Config{ExpectedChainID: *big.NewInt(1)}, call)
		if e != nil {
			t.Fatal(e)
		}
		return bnbSubmissionCases(c, m, p, claim)
	case "bitcoin":
		c, e := bitcoin.New(bitcoin.Config{}, call)
		if e != nil {
			t.Fatal(e)
		}
		return bitcoinSubmissionCases(c, m, p, claim)
	case "solana":
		c, e := solana.New(solana.Config{}, call)
		if e != nil {
			t.Fatal(e)
		}
		return solanaSubmissionCases(c, m, p, claim)
	default:
		c, e := tron.New(tron.Config{}, call)
		if e != nil {
			t.Fatal(e)
		}
		return tronSubmissionCases(c, m, p, claim)
	}
}
func contractSubmitWire(chain string) string {
	switch chain {
	case "ethereum", "bnb":
		return "eth_sendRawTransaction"
	case "bitcoin":
		return "sendrawtransaction"
	case "solana":
		return "sendTransaction"
	default:
		return "/wallet/broadcasttransaction"
	}
}
func contractLookupWire(chain string) string {
	switch chain {
	case "ethereum", "bnb":
		return "eth_getTransactionReceipt"
	case "bitcoin":
		return "getrawtransaction"
	case "solana":
		return "getSignatureStatuses"
	default:
		return "/walletsolidity/gettransactioninfobyid"
	}
}
func TestSubmissionContractDisabled(t *testing.T) {
	contractDisabled(t, true, "0", "1", []string{"CHAIN_API_GO_LIVE_TEST"})
	for _, v := range []string{"", "0", "true"} {
		t.Run(v, func(t *testing.T) {
			contractDisabled(t, true, "1", v, []string{"CHAIN_API_GO_LIVE_TEST", "CHAIN_API_GO_LIVE_BROADCAST"})
		})
	}
}

// Removing any identity/claim/accepted/unknown transition must change calls or ledger status.
func TestSubmissionContractStateMachine(t *testing.T) {
	for _, chain := range contractChains() {
		for _, scenario := range []string{"accepted", "identity", "second_identity", "claim_failure", "unknown", "rejected", "wrong_id", "pending", "execution_mismatch"} {
			if scenario == "second_identity" && chain != "bitcoin" {
				continue
			}
			t.Run(chain+"/"+scenario, func(t *testing.T) {
				m, payload := contractSubmissionMaterial(chain)
				var events []string
				claims, sends, lookups := 0, 0, 0
				claim := func() error {
					claims++
					events = append(events, "claim")
					if scenario == "claim_failure" {
						return errors.New(contractSecret)
					}
					return nil
				}
				call := contractCaller(func(_ context.Context, method string, p, out any) error {
					events = append(events, method)
					raw := contractResponse(chain, method, contractJSON(p))
					if scenario == "identity" && len(events) == 1 {
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
						return json.Unmarshal(raw, out)
					}
					if scenario == "second_identity" && method == "getblockhash" {
						raw = contractJSON(strings.Repeat("f", 64))
					}
					if method == contractSubmitWire(chain) {
						sends++
						if len(events) < 2 || events[len(events)-2] != "claim" {
							t.Error("send occurred before durable claim")
						}
						wire := string(contractJSON(p))
						if chain == "tron" {
							raw, ok := p.(json.RawMessage)
							if !ok || string(raw) != string(payload) {
								t.Error("Tron payload reconstructed before transport")
							}
						}
						if chain == "solana" && (!strings.Contains(wire, `"maxRetries":0`) || !strings.Contains(wire, `"skipPreflight":false`)) {
							t.Error("Solana explicit send options lost")
						}
						if chain != "tron" && !strings.Contains(wire, string(payload)) {
							t.Error("submission payload changed")
						}
						if scenario == "unknown" {
							return apierror.Error{Kind: apierror.Transport, Message: contractSecret, Cause: errors.New(contractSecret)}
						}
						if scenario == "rejected" {
							return apierror.Error{Kind: apierror.RPC, HasRPCCode: true, RPCCode: -1, NodeCode: "REJECT", Message: contractSecret}
						}
						if scenario == "wrong_id" {
							raw = contractJSON(contractID(chain, 1))
							if chain == "tron" {
								raw = json.RawMessage(fmt.Sprintf(`{"result":true,"txid":%q}`, contractID(chain, 1)))
							}
						}
					}
					if method == contractLookupWire(chain) {
						lookups++
						wire := string(contractJSON(p))
						if !strings.Contains(wire, m.TransactionID) {
							t.Error("lookup did not use original ID")
						}
						if chain == "bitcoin" && !strings.Contains(wire, m.BlockHash) {
							t.Error("lookup lost explicit block hash")
						}
						if chain == "solana" && !strings.Contains(wire, `"searchTransactionHistory":true`) {
							t.Error("lookup omitted historical search")
						}
						if scenario == "pending" {
							switch chain {
							case "bitcoin":
								raw = json.RawMessage(`{"confirmations":0}`)
							case "solana":
								raw = json.RawMessage(`{"context":{"slot":9},"value":[null]}`)
							case "tron":
								raw = json.RawMessage(`{}`)
							default:
								raw = json.RawMessage(`null`)
							}
						}
						if scenario == "execution_mismatch" {
							switch chain {
							case "bitcoin":
								raw = json.RawMessage(`{"confirmations":"invalid"}`)
							case "solana":
								raw = json.RawMessage(`{"context":{"slot":9},"value":[{"confirmationStatus":"finalized","err":{"InstructionError":[0,"x"]}}]}`)
							case "tron":
								raw = json.RawMessage(fmt.Sprintf(`{"id":%q,"receipt":{"result":"FAILED"}}`, m.TransactionID))
							default:
								raw = json.RawMessage(fmt.Sprintf(`{"transactionHash":%q,"status":"0x0"}`, m.TransactionID))
							}
						}
					}
					return json.Unmarshal(raw, out)
				})
				cases, err := contractSubmissionCases(t, chain, m, payload, claim, call)
				if err != nil {
					t.Errorf("valid submission factory rejected: %v", err)
				}
				ids := []string{}
				for _, c := range cases {
					ids = append(ids, c.ID)
				}
				if !reflect.DeepEqual(ids, contractRequired(chain, true)) {
					t.Errorf("broadcast catalog/order=%v", ids)
				}
				budget := 3
				if chain == "bitcoin" {
					budget = 4
				}
				got := contractHarness(chain, cases, true, budget)
				want := "PASS"
				wantClaims, wantSends, wantLookups := 1, 1, 1
				switch scenario {
				case "identity", "second_identity":
					want = "BLOCKED"
					wantClaims, wantSends, wantLookups = 0, 0, 0
				case "claim_failure":
					want = "BLOCKED"
					wantSends, wantLookups = 0, 0
				case "unknown", "pending":
					want = "BLOCKED"
				case "rejected", "wrong_id":
					want = "FAIL"
					wantLookups = 0
				case "execution_mismatch":
					want = "FAIL"
				}
				if got.Status != want || claims != wantClaims || sends != wantSends || lookups != wantLookups {
					t.Errorf("state result=%s want=%s claim/send/lookup=%d/%d/%d want=%d/%d/%d", got.Status, want, claims, sends, lookups, wantClaims, wantSends, wantLookups)
				}
				if scenario == "unknown" && (got.Cases[chain+".submit"] != "BLOCKED" || got.Cases[chain+".lookup"] != "PASS") {
					t.Error("unknown lookup overwrote original submission fact")
				}
				if scenario == "accepted" && got.Requests != budget {
					t.Errorf("request budget=%d want=%d", got.Requests, budget)
				}
				assertRedacted(t, got, contractSecret)
			})
		}
	}
}

func TestSubmissionContractClaimJournal(t *testing.T) {
	c := materialClaim{Chain: "ethereum", AuthorizationID: "auth", PayloadSHA256: strings.Repeat("a", 64), MaterialSHA256: strings.Repeat("b", 64)}
	for _, scenario := range []string{"new", "same_auth", "same_payload", "other_chain", "append_failure"} {
		t.Run(scenario, func(t *testing.T) {
			previous := []materialClaim{}
			if scenario == "same_auth" {
				old := c
				old.PayloadSHA256 = strings.Repeat("c", 64)
				previous = append(previous, old)
			}
			if scenario == "same_payload" {
				old := c
				old.AuthorizationID = "other"
				previous = append(previous, old)
			}
			if scenario == "other_chain" {
				old := c
				old.Chain = "bnb"
				previous = append(previous, old)
			}
			writes := 0
			var record []byte
			err := claimSubmission(c, contractObserved, previous, func(b []byte) error {
				writes++
				record = append([]byte(nil), b...)
				if scenario == "append_failure" {
					return errors.New(contractSecret)
				}
				return nil
			})
			valid := scenario == "new" || scenario == "other_chain"
			if (err == nil) != valid {
				t.Errorf("claim outcome wrong: valid=%v error=%v", valid, err)
			}
			wantWrites := 1
			if scenario == "same_auth" || scenario == "same_payload" {
				wantWrites = 0
			}
			if writes != wantWrites {
				t.Errorf("append calls=%d want=%d", writes, wantWrites)
			}
			if valid {
				if !strings.HasSuffix(string(record), "\n") {
					t.Error("claim missing JSONL terminator")
				}
				var got broadcastRecord
				if json.Unmarshal(record, &got) != nil || got.Version != 1 || got.Type != "claim" || got.RecordedAt != contractObserved || got.Claim != c || !reflect.DeepEqual(got.Report, suiteReport{}) {
					t.Error("claim record not complete/isolated")
				}
				claims, e := readMaterialClaims(record)
				if e != nil || !reflect.DeepEqual(claims, []materialClaim{c}) {
					t.Error("durable claim not recognized")
				}
			}
		})
	}
	good := string(contractJSON(broadcastRecord{Version: 1, Type: "claim", RecordedAt: contractObserved, Claim: c})) + "\n"
	for _, bad := range []string{"{", good + "{", strings.Replace(good, `"version":1`, `"version":2`, 1), strings.Replace(good, `"type":"claim"`, `"type":"other"`, 1), strings.Replace(good, `"authorization_id":"auth"`, `"authorization_id":""`, 1), `{"version":1,"type":"report","report":{}}` + "\n"} {
		t.Run("invalid_journal", func(t *testing.T) {
			if _, err := readMaterialClaims([]byte(bad)); err == nil {
				t.Error("invalid/truncated journal accepted")
			}
		})
	}
	claims, err := readMaterialClaims(nil)
	if err != nil || len(claims) != 0 {
		t.Error("empty journal must allow first claim")
	}
}

func TestSubmissionContractEnabledJournalAndReports(t *testing.T) {
	for _, scenario := range []string{"pass", "existing_claim", "truncated", "claim_write_failure", "report_write_failure", "assertion_and_report_failure", "journal_read_failure", "unknown", "assertion_saved"} {
		t.Run(scenario, func(t *testing.T) {
			h := contractHostFor(t, "broadcast")
			want := "PASS"
			if scenario == "existing_claim" {
				m, _ := contractSubmissionMaterial("ethereum")
				h.journal = append(contractJSON(broadcastRecord{Version: 1, Type: "claim", RecordedAt: contractObserved, Claim: materialClaim{Chain: m.Chain, AuthorizationID: m.AuthorizationID, PayloadSHA256: m.PayloadSHA256, MaterialSHA256: contractSHA(h.files["ethereum.material"])}}), '\n')
				want = "BLOCKED"
			}
			if scenario == "truncated" {
				h.journal = []byte(`{"version":`)
				want = "BLOCKED"
			}
			if scenario == "journal_read_failure" {
				h.deps.ReadBroadcastJournal = func() ([]byte, error) { return nil, errors.New(contractSecret) }
				want = "BLOCKED"
			}
			initialPrefix := string(h.journal)
			claimPrefix := ""
			hostAppend := h.deps.AppendBroadcastRecord
			h.deps.AppendBroadcastRecord = func(b []byte) error {
				var record map[string]json.RawMessage
				_ = json.Unmarshal(b, &record)
				if string(record["type"]) == `"report"` {
					claimPrefix = string(h.journal)
				}
				return hostAppend(b)
			}
			appendRecord := h.deps.AppendBroadcastRecord
			if scenario == "claim_write_failure" || scenario == "report_write_failure" || scenario == "assertion_and_report_failure" {
				want = "BLOCKED"
				h.deps.AppendBroadcastRecord = func(b []byte) error {
					var r broadcastRecord
					if json.Unmarshal(b, &r) != nil {
						t.Error("invalid record passed to append")
					}
					if r.Type == "claim" && scenario == "claim_write_failure" || r.Type == "report" && scenario != "claim_write_failure" {
						return errors.New(contractSecret)
					}
					return appendRecord(b)
				}
			}
			if scenario == "assertion_and_report_failure" || scenario == "assertion_saved" {
				want = "FAIL"
				h.mutate = func(c, m string, b json.RawMessage) (json.RawMessage, error) {
					if c == "ethereum" && m == "eth_sendRawTransaction" {
						return contractJSON(contractID(c, 1)), nil
					}
					return b, nil
				}
			}
			if scenario == "unknown" {
				want = "BLOCKED"
				h.mutate = func(c, m string, b json.RawMessage) (json.RawMessage, error) {
					if m == contractSubmitWire(c) {
						return nil, errors.New(contractSecret)
					}
					return b, nil
				}
			}
			got := runLiveBroadcast(context.Background(), h.deps)
			contractCheckSuite(t, got, "broadcast", want)
			if scenario == "pass" || scenario == "unknown" || scenario == "assertion_saved" || scenario == "existing_claim" {
				lines := strings.Split(strings.TrimSuffix(string(h.journal), "\n"), "\n")
				last := []byte(lines[len(lines)-1])
				fields := contractObjectKeys(t, last, "version type recorded_at claim report")
				contractObjectKeys(t, fields["claim"], "chain authorization_id payload_sha256 material_sha256")
				contractReportKeys(t, fields["report"])
				var record broadcastRecord
				if json.Unmarshal(last, &record) != nil || record.Version != 1 || record.Type != "report" || record.RecordedAt != "2099-01-01T00:00:00Z" || record.Claim != (materialClaim{}) || !reflect.DeepEqual(record.Report, got) {
					t.Error("persisted final report differs from returned facts or fixed record metadata")
				}
				if claimPrefix == "" || !strings.HasPrefix(claimPrefix, initialPrefix) || string(h.journal) != claimPrefix+string(last)+"\n" {
					t.Error("final report rewrote or omitted durable claim prefix")
				}
			}
			for _, chain := range contractChains() {
				sends, lookups := 0, 0
				for _, m := range h.calls[chain] {
					if m == contractSubmitWire(chain) {
						sends++
					}
					if m == contractLookupWire(chain) {
						lookups++
					}
				}
				expected := 1
				if scenario == "truncated" || scenario == "journal_read_failure" || scenario == "claim_write_failure" || scenario == "existing_claim" && chain == "ethereum" {
					expected = 0
				}
				if sends != expected {
					t.Errorf("%s sends=%d want=%d", chain, sends, expected)
				}
				if scenario == "unknown" && lookups != 1 {
					t.Error("unknown did not query exactly once")
				}
			}
			if scenario == "pass" {
				if h.reads["explicit-config"] == 0 || len(h.creates) != 5 || !reflect.DeepEqual(h.creates, h.cleanups) {
					t.Error("broadcast failed explicit assembly/cleanup")
				}
				lines := strings.Split(strings.TrimSuffix(string(h.journal), "\n"), "\n")
				if len(lines) != 6 {
					t.Errorf("journal lines=%d want 5 claims+report", len(lines))
				}
				for _, chain := range contractChains() {
					if h.reads[chain+".payload"] != 1 || h.reads[chain+".material"] != 1 || h.reads[chain+".dynamic"] != 1 {
						t.Error("broadcast material not read once")
					}
				}
				claims, err := readMaterialClaims(h.journal)
				if err != nil || len(claims) != 5 {
					t.Error("complete journal did not retain five claims")
				}
				for _, chain := range contractChains() {
					send := -1
					for i, e := range h.events {
						if e == chain+":"+contractSubmitWire(chain) {
							send = i
							break
						}
					}
					if send < 1 || h.events[send-1] != "append" {
						t.Error("durable append must immediately precede send")
					}
				}
			}
			assertRedacted(t, string(h.journal), contractSecret, "https://fixture.invalid", "Authorization", "raw_data")
		})
	}
}

func TestSubmissionContractMaterialPrerequisites(t *testing.T) {
	for _, chain := range contractChains() {
		for _, bad := range []string{"unauthorized", "alias", "expired", "identity", "payload_digest", "material_digest", "missing_payload", "missing_preflight", "nonempty_inputs", "wrong_execution", "unknown_field", "duplicate_field"} {
			t.Run(chain+"/"+bad, func(t *testing.T) {
				h := contractHostFor(t, "broadcast")
				m, _ := contractSubmissionMaterial(chain)
				var d dynamicInputs
				_ = json.Unmarshal(h.files[chain+".dynamic"], &d)
				switch bad {
				case "unauthorized":
					m.Authorized = false
				case "alias":
					m.EndpointAlias = "other_broadcast"
				case "expired":
					m.ExpiresAt = contractObserved
				case "identity":
					m.NetworkIdentity = networkIdentity{}
				case "payload_digest":
					m.PayloadSHA256 = strings.Repeat("f", 64)
				case "material_digest":
					d.ManifestSHA256 = strings.Repeat("f", 64)
				case "missing_payload":
					delete(h.files, m.PayloadPath)
				case "missing_preflight":
					d.Preflight.Methods = map[string]rpcExpectation{}
				case "nonempty_inputs":
					d.Inputs = json.RawMessage(`{"unexpected":true}`)
				case "wrong_execution":
					m.ExpectedExecution = executionExpectation{}
					if chain != "solana" {
						m.ExpectedExecution.SolanaCommitment = "finalized"
					}
					if chain == "solana" {
						m.ExpectedExecution.EVMStatus = "0x1"
					}
				}
				raw := contractMaterialJSON(m)
				if bad == "unknown_field" {
					raw = []byte(strings.TrimSuffix(string(raw), "}") + `,"private_key":"SYNTHETIC_PRIVATE_MARKER"}`)
				}
				if bad == "duplicate_field" {
					raw = []byte(strings.TrimSuffix(string(raw), "}") + `,"authorized":true}`)
				}
				if bad != "material_digest" {
					d.ManifestSHA256 = contractSHA(raw)
				}
				h.files[chain+".material"] = raw
				h.files[chain+".dynamic"] = contractJSON(d)
				got := runLiveBroadcast(context.Background(), h.deps)
				contractCheckSuite(t, got, "broadcast", "BLOCKED")
				if len(h.calls[chain]) != 0 {
					t.Error("invalid broadcast material reached identity/send")
				}
			})
		}
	}
}

func TestSubmissionContractReentryNeverResends(t *testing.T) {
	h := contractHostFor(t, "broadcast")
	first := runLiveBroadcast(context.Background(), h.deps)
	contractCheckSuite(t, first, "broadcast", "PASS")
	before := map[string]int{}
	for _, c := range contractChains() {
		before[c] = len(h.calls[c])
	}
	second := runLiveBroadcast(context.Background(), h.deps)
	contractCheckSuite(t, second, "broadcast", "BLOCKED")
	for _, c := range contractChains() {
		for _, m := range h.calls[c][before[c]:] {
			if m == contractSubmitWire(c) {
				t.Errorf("consumed material resent for %s", c)
			}
		}
	}
	claims, err := readMaterialClaims(h.journal)
	if err != nil || len(claims) != 5 {
		t.Error("reentry changed permanent material claims")
	}
}

func TestNativeEVMSubmissionVersionGate(t *testing.T) {
	for _, chain := range []string{"ethereum", "bnb"} {
		for _, version := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/v%d", chain, version), func(t *testing.T) {
				h := contractHostFor(t, "broadcast")
				m, _ := contractSubmissionMaterial(chain)
				m.Version = version
				raw := contractMaterialJSON(m)
				var d dynamicInputs
				if err := json.Unmarshal(h.files[chain+".dynamic"], &d); err != nil {
					t.Fatal(err)
				}
				d.ManifestSHA256 = contractSHA(raw)
				h.files[chain+".material"] = raw
				h.files[chain+".dynamic"] = contractJSON(d)
				got := runLiveBroadcast(context.Background(), h.deps)
				for _, c := range got.Chains {
					if c.Chain == chain {
						if version == 1 {
							if c.Result.Status != "BLOCKED" || len(h.calls[chain]) != 0 {
								t.Fatal("NS-08: obsolete submission material reached Caller")
							}
						} else if c.Result.Status != "PASS" {
							t.Fatalf("NS-08: native v2 submission status=%s", c.Result.Status)
						}
					}
				}
			})
		}
	}
}

func TestNativeSolanaSubmissionVersionGate(t *testing.T) {
	for _, chain := range []string{"solana"} {
		for _, version := range []int{1, 2} {
			t.Run(fmt.Sprintf("%s/v%d", chain, version), func(t *testing.T) {
				h := contractHostFor(t, "broadcast")
				m, _ := contractSubmissionMaterial(chain)
				m.Version = version
				raw := contractMaterialJSON(m)
				var d dynamicInputs
				if err := json.Unmarshal(h.files[chain+".dynamic"], &d); err != nil {
					t.Fatal(err)
				}
				d.ManifestSHA256 = contractSHA(raw)
				h.files[chain+".material"] = raw
				h.files[chain+".dynamic"] = contractJSON(d)
				got := runLiveBroadcast(context.Background(), h.deps)
				for _, c := range got.Chains {
					if c.Chain == chain {
						if version == 1 {
							if c.Result.Status != "BLOCKED" || len(h.calls[chain]) != 0 {
								t.Fatal("NS-08: obsolete submission material reached Caller")
							}
						} else if c.Result.Status != "PASS" {
							t.Fatalf("NS-08: native v2 submission status=%s", c.Result.Status)
						}
					}
				}
			})
		}
	}
}
