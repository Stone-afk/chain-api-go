package integration

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Stone-afk/chain-api-go/transport"
)

func liveScenarioDeps() scenarioDeps {
	return scenarioDeps{
		LookupEnv: os.LookupEnv, ReadFile: os.ReadFile, Now: time.Now, Wait: timerWait,
		NewDoer: func(c scenarioChainConfig) (transport.HTTPDoer, func(), error) {
			tr := &http.Transport{}
			client := &http.Client{Transport: tr, Timeout: c.RequestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			return client, tr.CloseIdleConnections, nil
		},
		Runners: map[string]scenarioRunner{"ethereum": scenarioEVMRunner("ethereum"), "bnb": scenarioEVMRunner("bnb"), "bitcoin": scenarioBitcoinRunner(), "solana": scenarioSolanaRunner(), "tron": scenarioTronRunner()},
	}
}

func TestLiveScenario(t *testing.T) {
	runLiveScenario(t, liveScenarioDeps())
}

func TestLiveScenarioSubmitProbe(t *testing.T) {
	live := liveScenarioDeps()
	r := runScenarioSubmitProbe(context.Background(), scenarioProbeDeps{
		LookupEnv: live.LookupEnv, ReadFile: live.ReadFile, NewDoer: live.NewDoer, Now: live.Now, Wait: live.Wait,
		Identity: scenarioProbeIdentities(),
	})
	if r.Status == "NOT_RUN" {
		t.Skip("live submit probe disabled")
	}
	for _, chain := range scenarioChains {
		if issue, ok := r.Chains[chain]; ok {
			t.Logf("%s %s %s", chain, issue.Status, issue.Category)
		}
	}
	if r.Status != "PASS" {
		t.Errorf("submit probe %s/%s requests=%d", r.Status, r.Category, r.Requests)
	}
}

func runLiveScenario(t *testing.T, deps scenarioDeps) {
	r := runScenario(context.Background(), deps)
	if scenarioOutcome(r) == "SKIP" {
		t.Skip("live scenario disabled")
	}
	required := map[string]bool{}
	for _, c := range r.RequiredChains {
		required[c] = true
	}
	for _, chain := range scenarioChains {
		t.Run(chain, func(t *testing.T) {
			bad := 0
			for _, e := range r.Entries {
				if e.Chain != chain {
					continue
				}
				t.Logf("%s %s", e.ID, e.Status)
				if e.Status == "FAIL" || required[chain] && e.Kind != "submit" && e.Status != "PASS" {
					bad++
				}
			}
			if bad > 0 {
				t.Errorf("%s: %d entries not PASS", chain, bad)
			} else if !required[chain] {
				t.Skip("chain not required")
			}
		})
	}
	if scenarioOutcome(r) != "PASS" {
		t.Errorf("scenario %s/%s requests=%d", r.Status, r.Category, r.Requests)
	}
}
