package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func fixtureConfig() harnessConfig {
	return harnessConfig{Alias: "fixture", Endpoint: "https://fixture.invalid/rpc", Network: "fixture", Headers: map[string]string{"Authorization": "SYNTHETIC_SECRET"}, MaxRequests: 8, MaxConcurrent: 1, MinInterval: time.Second, TotalTimeout: time.Minute}
}
func timerWait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func fixtureDeps() harnessDeps {
	return harnessDeps{Load: func() (harnessConfig, error) { return fixtureConfig(), nil }, LoadManifest: func() ([]byte, string, error) { return []byte(fixtureManifest), fixtureDigest, nil }, Now: time.Now, Wait: timerWait}
}
func passingCase(id string) harnessCase {
	return harnessCase{ID: id, Run: func(context.Context, harnessRequest) (string, error) { return "PASS", nil }}
}
func checkStatus(t *testing.T, got harnessReport, want string) {
	t.Helper()
	if got.Status != want {
		t.Errorf("status = %q; want %q", got.Status, want)
	}
}
func assertRedacted(t *testing.T, v any, secrets ...string) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal("diagnostic not JSON serializable")
	}
	text := string(b) + fmt.Sprintf("%v %+v %#v", v, v, v)
	if e, ok := v.(error); ok && e != nil {
		text += e.Error()
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(text, secret) {
			t.Error("diagnostic retained synthetic secret")
		}
	}
}

// HC-01: even clock/material access is forbidden before the switch.
func TestHarnessDisabledReadsNothing(t *testing.T) {
	calls := 0
	touch := func() { calls++ }
	deps := harnessDeps{Load: func() (harnessConfig, error) { touch(); return fixtureConfig(), nil }, LoadManifest: func() ([]byte, string, error) { touch(); return nil, "", nil }, Now: func() time.Time { touch(); return time.Now() }, Wait: func(context.Context, time.Duration) error { touch(); return nil }}
	c := harnessCase{ID: "read-a", Run: func(context.Context, harnessRequest) (string, error) { touch(); return "PASS", nil }}
	got := runHarness(context.Background(), false, deps, []string{"read-a"}, []harnessCase{c})
	checkStatus(t, got, "NOT_RUN")
	if calls != 0 || got.Requests != 0 {
		t.Error("disabled harness touched external dependencies")
	}
}

func TestHarnessInvalidDependenciesAndCatalog(t *testing.T) {
	for _, name := range []string{"nil_context", "nil_load", "nil_manifest", "nil_clock", "nil_wait", "empty_required", "empty_id", "duplicate_required", "duplicate_case", "unknown_case", "child_only", "nil_run"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			deps := fixtureDeps()
			required := []string{"read-a"}
			calls := 0
			c := harnessCase{ID: "read-a", Run: func(context.Context, harnessRequest) (string, error) { calls++; return "PASS", nil }}
			cases := []harnessCase{c}
			switch name {
			case "nil_context":
				ctx = nil
			case "nil_load":
				deps.Load = nil
			case "nil_manifest":
				deps.LoadManifest = nil
			case "nil_clock":
				deps.Now = nil
			case "nil_wait":
				deps.Wait = nil
			case "empty_required":
				required = nil
			case "empty_id":
				required = []string{""}
			case "duplicate_required":
				required = append(required, "read-a")
			case "duplicate_case":
				cases = append(cases, c)
			case "unknown_case":
				cases[0].ID = "unknown"
			case "child_only":
				cases[0].ID = "read-a/child"
			case "nil_run":
				cases[0].Run = nil
			}
			got := runHarness(ctx, true, deps, required, cases)
			checkStatus(t, got, "BLOCKED")
			if calls != 0 || got.Requests != 0 {
				t.Error("invalid prerequisite reached callback")
			}
		})
	}
}

func TestHarnessRequiredLedgerAndSeverity(t *testing.T) {
	for _, tc := range []struct {
		name, a, b, want, caseB string
		omit                    bool
	}{
		{"all_pass", "PASS", "PASS", "PASS", "PASS", false},
		{"missing", "PASS", "", "BLOCKED", "NOT_RUN", true},
		{"not_run", "PASS", "NOT_RUN", "BLOCKED", "NOT_RUN", false},
		{"blocked", "PASS", "BLOCKED", "BLOCKED", "BLOCKED", false},
		{"fail_before_block", "FAIL", "BLOCKED", "FAIL", "BLOCKED", false},
		{"block_before_fail", "BLOCKED", "FAIL", "FAIL", "FAIL", false},
		{"invalid_status", "PASS", "SKIP", "FAIL", "FAIL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			counts := map[string]int{}
			makeCase := func(id, status string) harnessCase {
				return harnessCase{ID: id, Run: func(context.Context, harnessRequest) (string, error) { counts[id]++; return status, nil }}
			}
			cases := []harnessCase{makeCase("read-a", tc.a)}
			if !tc.omit {
				cases = append(cases, makeCase("read-b", tc.b))
			}
			got := runHarness(context.Background(), true, fixtureDeps(), []string{"read-a", "read-b"}, cases)
			checkStatus(t, got, tc.want)
			if !reflect.DeepEqual(got.Cases, map[string]string{"read-a": tc.a, "read-b": tc.caseB}) || counts["read-a"] != 1 || (!tc.omit && counts["read-b"] != 1) {
				t.Error("independent required ledger lost result, missing item, or ran a case twice")
			}
		})
	}
}

// HC-07: ignored operation/budget errors remain sticky even when callback says PASS.
func TestHarnessRequestBudgetAndStickyErrors(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opErr error
		want  string
	}{
		{"budget", nil, "BLOCKED"}, {"failed_attempt_counts", errors.New("SYNTHETIC_SECRET"), "FAIL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				deps := fixtureDeps()
				cfg := fixtureConfig()
				cfg.MaxRequests = 2
				deps.Load = func() (harnessConfig, error) { return cfg, nil }
				calls, denied := 0, 0
				c := harnessCase{ID: "read-a", Run: func(_ context.Context, request harnessRequest) (string, error) {
					for i := 0; i < 3; i++ {
						err := request(func(context.Context) error { calls++; return tc.opErr })
						assertRedacted(t, err, "SYNTHETIC_SECRET")
						if i == 2 && err != nil {
							denied++
						}
					}
					return "PASS", nil
				}}
				got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
				checkStatus(t, got, tc.want)
				if calls != 2 || denied != 1 || got.Requests != 2 || got.Cases["read-a"] != tc.want {
					t.Error("budget or sticky failure was lost")
				}
			})
		})
	}
}

func TestHarnessErrorClassificationAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		name           string
		err            error
		want, category string
	}{
		{"assertion", harnessFailure{"FAIL", "assertion"}, "FAIL", "assertion"},
		{"environment", harnessFailure{"BLOCKED", "environment"}, "BLOCKED", "environment"},
		{"bad_status", harnessFailure{"PASS", "assertion"}, "FAIL", "invalid_result"},
		{"bad_category", harnessFailure{"FAIL", "SYNTHETIC_SECRET"}, "FAIL", "invalid_result"},
		{"arbitrary", errors.New("SYNTHETIC_SECRET"), "FAIL", ""},
		{"cancel", fmt.Errorf("SYNTHETIC_SECRET: %w", context.Canceled), "BLOCKED", ""},
		{"deadline", fmt.Errorf("SYNTHETIC_SECRET: %w", context.DeadlineExceeded), "BLOCKED", ""},
	} {
		for _, stage := range []string{"operation", "case"} {
			t.Run(tc.name+"/"+stage, func(t *testing.T) {
				calls := 0
				c := harnessCase{ID: "read-a", Run: func(_ context.Context, request harnessRequest) (string, error) {
					calls++
					if stage == "case" {
						return "PASS", tc.err
					}
					err := request(func(context.Context) error { return tc.err })
					assertRedacted(t, err, "SYNTHETIC_SECRET")
					return "PASS", nil
				}}
				got := runHarness(context.Background(), true, fixtureDeps(), []string{"read-a"}, []harnessCase{c})
				checkStatus(t, got, tc.want)
				if calls != 1 || got.Cases["read-a"] != tc.want || got.Category == "" || (tc.category != "" && got.Category != tc.category) {
					t.Error("error classification or sticky case severity lost")
				}
				assertRedacted(t, got, "SYNTHETIC_SECRET", "https://fixture.invalid", "Authorization")
			})
		}
	}
	for _, stage := range []string{"loader", "manifest", "waiter"} {
		t.Run(stage, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				deps := fixtureDeps()
				secret := errors.New("SYNTHETIC_SECRET")
				switch stage {
				case "loader":
					deps.Load = func() (harnessConfig, error) { return fixtureConfig(), secret }
				case "manifest":
					deps.LoadManifest = func() ([]byte, string, error) { return []byte("SYNTHETIC_SECRET"), "", secret }
				case "waiter":
					deps.Wait = func(context.Context, time.Duration) error { return secret }
				}
				calls := 0
				c := harnessCase{ID: "read-a", Run: func(_ context.Context, r harnessRequest) (string, error) {
					for i := 0; i < 2; i++ {
						err := r(func(context.Context) error { calls++; return nil })
						assertRedacted(t, err, "SYNTHETIC_SECRET")
					}
					return "PASS", nil
				}}
				got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
				want := "BLOCKED"
				wantCalls := 0
				if stage == "waiter" {
					want = "FAIL"
					wantCalls = 1
				}
				checkStatus(t, got, want)
				if calls != wantCalls || got.Requests != wantCalls {
					t.Error("failed prerequisite or wait admitted operation")
				}
				assertRedacted(t, got, "SYNTHETIC_SECRET")
			})
		})
	}
}

// HC-05/08: real limit+1 goroutines, no recursive limiter call, no FIFO assumption.
func TestHarnessConcurrentCapacityAndCancellation(t *testing.T) {
	for _, limit := range []int{1, 2} {
		for _, cancelQueue := range []bool{false, true} {
			t.Run(fmt.Sprintf("limit%d/cancel%t", limit, cancelQueue), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					deps := fixtureDeps()
					cfg := fixtureConfig()
					cfg.MaxConcurrent = limit
					deps.Load = func() (harnessConfig, error) { return cfg, nil }
					parent, cancel := context.WithCancel(context.Background())
					defer cancel()
					var active, peak, calls, exited atomic.Int32
					var caseCalls int
					c := harnessCase{ID: "read-a", Run: func(ctx context.Context, r harnessRequest) (string, error) {
						caseCalls++
						release := make(chan struct{})
						var wg sync.WaitGroup
						for i := 0; i < limit+1; i++ {
							wg.Add(1)
							go func() {
								defer wg.Done()
								defer exited.Add(1)
								_ = r(func(opCtx context.Context) error {
									n := active.Add(1)
									defer active.Add(-1)
									for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
									}
									calls.Add(1)
									select {
									case <-release:
										return nil
									case <-opCtx.Done():
										return opCtx.Err()
									}
								})
							}()
						}
						time.Sleep(time.Duration(limit+1) * time.Second)
						synctest.Wait()
						if calls.Load() != int32(limit) || active.Load() != int32(limit) || exited.Load() != 0 {
							t.Error("capacity did not block the extra caller")
						}
						if cancelQueue {
							cancel()
						} else {
							close(release)
						}
						wg.Wait()
						return "PASS", nil
					}}
					got := runHarness(parent, true, deps, []string{"read-a"}, []harnessCase{c})
					want := "PASS"
					wantCalls := limit + 1
					if cancelQueue {
						want = "BLOCKED"
						wantCalls = limit
					}
					checkStatus(t, got, want)
					if caseCalls != 1 || int(calls.Load()) != wantCalls || got.Requests != wantCalls || peak.Load() != int32(limit) || active.Load() != 0 || exited.Load() != int32(limit+1) {
						t.Error("concurrent count, peak, cancellation, or slot release incorrect")
					}
				})
			})
		}
	}
}

func TestHarnessConcurrentLedgerTieBreak(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deps := fixtureDeps()
		cfg := fixtureConfig()
		cfg.MaxConcurrent = 2
		deps.Load = func() (harnessConfig, error) { return cfg, nil }
		c := harnessCase{ID: "read-a", Run: func(_ context.Context, r harnessRequest) (string, error) {
			var wg sync.WaitGroup
			for _, category := range []string{"environment", "assertion"} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = r(func(context.Context) error { return harnessFailure{"FAIL", category} })
				}()
			}
			wg.Wait()
			return "PASS", nil
		}}
		got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
		checkStatus(t, got, "FAIL")
		if got.Category != "assertion" || got.Cases["read-a"] != "FAIL" || got.Requests != 2 {
			t.Error("concurrent error summary must choose fixed lexicographic category")
		}
	})
}

func TestHarnessRateDeadlineAndRelease(t *testing.T) {
	for _, timeout := range []time.Duration{2 * time.Second, 4 * time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				deps := fixtureDeps()
				var clockCalls atomic.Int32
				deps.Now = func() time.Time { clockCalls.Add(1); return time.Now() }
				cfg := fixtureConfig()
				cfg.TotalTimeout = timeout
				deps.Load = func() (harnessConfig, error) { time.Sleep(time.Second); return cfg, nil }
				deps.LoadManifest = func() ([]byte, string, error) {
					time.Sleep(time.Second)
					return []byte(fixtureManifest), fixtureDigest, nil
				}
				var starts []time.Duration
				var suiteCtx context.Context
				var start time.Time
				c := harnessCase{ID: "read-a", Run: func(ctx context.Context, r harnessRequest) (string, error) {
					suiteCtx = ctx
					if clockCalls.Load() == 0 {
						t.Error("deadline ignored injected clock")
					}
					start = time.Now()
					deadline, ok := ctx.Deadline()
					if !ok || !deadline.Equal(start.Add(timeout)) {
						t.Error("deadline must begin after local validation")
					}
					for i := 0; i < 3; i++ {
						before := clockCalls.Load()
						_ = r(func(opCtx context.Context) error {
							if clockCalls.Load() <= before {
								t.Error("admission ignored injected clock")
							}
							if opCtx != ctx {
								t.Error("operation must share suite context")
							}
							starts = append(starts, time.Since(start))
							return nil
						})
					}
					return "PASS", nil
				}}
				got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
				want := "PASS"
				n := 3
				if timeout == 2*time.Second {
					want = "BLOCKED"
					n = 2
				}
				checkStatus(t, got, want)
				if clockCalls.Load() == 0 {
					t.Error("enabled execution never consumed injected clock")
				}
				if !reflect.DeepEqual(starts, []time.Duration{0, time.Second, 2 * time.Second}[:n]) || got.Requests != n {
					t.Error("rate spacing or exact deadline admission incorrect")
				}
				if suiteCtx == nil {
					t.Error("case not executed")
				} else {
					select {
					case <-suiteCtx.Done():
					default:
						t.Error("suite context/timer not canceled on return")
					}
				}
			})
		})
	}
}

func TestHarnessContextLifecycle(t *testing.T) {
	for _, mode := range []string{"suite_deadline", "parent_deadline", "parent_cancel", "pre_canceled", "rate_cancel"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				deps := fixtureDeps()
				cfg := fixtureConfig()
				cfg.TotalTimeout = 5 * time.Second
				deps.Load = func() (harnessConfig, error) { return cfg, nil }
				parent, cancel := context.WithCancel(context.Background())
				defer cancel()
				expected := time.Now().Add(5 * time.Second)
				if mode == "parent_deadline" {
					var stop context.CancelFunc
					parent, stop = context.WithDeadline(parent, time.Now().Add(2*time.Second))
					defer stop()
					expected = time.Now().Add(2 * time.Second)
				}
				if mode == "pre_canceled" {
					cancel()
				}
				calls := 0
				var captured context.Context
				var observed error
				c := harnessCase{ID: "read-a", Run: func(ctx context.Context, r harnessRequest) (string, error) {
					captured = ctx
					if mode == "rate_cancel" {
						_ = r(func(context.Context) error { calls++; return nil })
						var wg sync.WaitGroup
						wg.Add(1)
						go func() { defer wg.Done(); time.Sleep(time.Millisecond); cancel() }()
						_ = r(func(context.Context) error { calls++; return nil })
						wg.Wait()
						return "PASS", nil
					}
					_ = r(func(opCtx context.Context) error {
						calls++
						d, ok := opCtx.Deadline()
						if !ok || !d.Equal(expected) {
							t.Error("operation deadline incorrect")
						}
						if mode == "parent_cancel" {
							cancel()
						}
						<-opCtx.Done()
						observed = opCtx.Err()
						return nil
					})
					return "PASS", nil
				}}
				got := runHarness(parent, true, deps, []string{"read-a"}, []harnessCase{c})
				checkStatus(t, got, "BLOCKED")
				wantCalls := 1
				if mode == "pre_canceled" {
					wantCalls = 0
				}
				if calls != wantCalls || got.Requests != wantCalls {
					t.Error("canceled admission consumed budget or in-flight call not executed")
				}
				if mode == "suite_deadline" || mode == "parent_deadline" {
					if observed != context.DeadlineExceeded {
						t.Error("in-flight Done did not signal actual deadline")
					}
				} else if mode == "parent_cancel" && observed != context.Canceled {
					t.Error("parent cancellation detached")
				}
				if captured != nil {
					select {
					case <-captured.Done():
					default:
						t.Error("derived context not released")
					}
				}
			})
		})
	}
}

// A canceled wait does not cancel the suite itself; both rejection paths must
// release any reserved slot so the next request can use the same capacity.
func TestHarnessWaitFailureReleasesCapacity(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{{"failure", errors.New("SYNTHETIC_SECRET"), "FAIL"}, {"cancel", context.Canceled, "BLOCKED"}} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				deps := fixtureDeps()
				var waits, calls, exits atomic.Int32
				deps.Wait = func(ctx context.Context, d time.Duration) error {
					if waits.Add(1) == 1 {
						return tc.err
					}
					return timerWait(ctx, d)
				}
				c := harnessCase{ID: "read-a", Run: func(_ context.Context, r harnessRequest) (string, error) {
					_ = r(func(context.Context) error { calls.Add(1); return nil })
					for i := 0; i < 2; i++ {
						done := make(chan error, 1)
						go func() { defer exits.Add(1); done <- r(func(context.Context) error { calls.Add(1); return nil }) }()
						err := <-done
						synctest.Wait()
						if (i == 0) != (err != nil) {
							t.Error("one-shot wait failure did not recover")
						}
					}
					return "PASS", nil
				}}
				got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
				checkStatus(t, got, tc.want)
				if calls.Load() != 2 || got.Requests != 2 || exits.Load() != 2 || waits.Load() < 2 {
					t.Error("failed wait consumed budget, leaked capacity, or stranded caller")
				}
			})
		})
	}
}

func TestHarnessSummaryOrderIndependence(t *testing.T) {
	for _, tc := range []struct {
		name           string
		a, b           error
		want, category string
	}{
		{"same_severity", harnessFailure{"FAIL", "environment"}, harnessFailure{"FAIL", "assertion"}, "FAIL", "assertion"},
		{"mixed_severity", harnessFailure{"BLOCKED", "environment"}, harnessFailure{"FAIL", "assertion"}, "FAIL", "assertion"},
		{"generic", errors.New("SYNTHETIC_SECRET_A"), errors.New("SYNTHETIC_SECRET_B"), "FAIL", ""},
		{"context", context.Canceled, context.DeadlineExceeded, "BLOCKED", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var category string
			for _, boundary := range []string{"across_cases", "case_and_operation"} {
				for _, reverse := range []bool{false, true} {
					a, b := tc.a, tc.b
					if reverse {
						a, b = b, a
					}
					cases := []harnessCase{{ID: "read-a", Run: func(context.Context, harnessRequest) (string, error) { return "PASS", a }}, {ID: "read-b", Run: func(context.Context, harnessRequest) (string, error) { return "PASS", b }}}
					required := []string{"read-a", "read-b"}
					if boundary == "case_and_operation" {
						required = []string{"read-a"}
						cases = []harnessCase{{ID: "read-a", Run: func(_ context.Context, r harnessRequest) (string, error) {
							_ = r(func(context.Context) error { return a })
							return "PASS", b
						}}}
					}
					got := runHarness(context.Background(), true, fixtureDeps(), required, cases)
					checkStatus(t, got, tc.want)
					if got.Category == "" || (tc.category != "" && got.Category != tc.category) {
						t.Error("summary category missing or wrong")
					}
					if category == "" {
						category = got.Category
					} else if got.Category != category {
						t.Error("error source/order changed category")
					}
					if boundary == "case_and_operation" && got.Cases["read-a"] != tc.want {
						t.Error("case error overwrote sticky operation severity")
					}
					assertRedacted(t, got, "SYNTHETIC_SECRET_A", "SYNTHETIC_SECRET_B")
				}
			}
		})
	}
}

// Both values use the synctest clock domain. Sampling before a synchronous
// delay makes the injected return distinguishable from a fresh time.Now call.
func TestHarnessDeadlineUsesInjectedClockValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deps := fixtureDeps()
		cfg := fixtureConfig()
		cfg.TotalTimeout = 3 * time.Second
		deps.Load = func() (harnessConfig, error) { return cfg, nil }
		var sampled time.Time
		deps.Now = func() time.Time {
			now := time.Now()
			if sampled.IsZero() {
				sampled = now
				time.Sleep(time.Second)
			}
			return now
		}
		cases, operations := 0, 0
		c := harnessCase{ID: "read-a", Run: func(ctx context.Context, r harnessRequest) (string, error) {
			cases++
			want := sampled.Add(cfg.TotalTimeout)
			if d, ok := ctx.Deadline(); !ok || !d.Equal(want) {
				t.Error("suite deadline ignored injected clock return value")
			}
			if sampled.IsZero() || !time.Now().After(sampled) {
				t.Error("injected sample must precede current clock in the same domain")
			}
			err := r(func(opCtx context.Context) error {
				operations++
				if d, ok := opCtx.Deadline(); !ok || !d.Equal(want) {
					t.Error("operation deadline ignored injected clock return value")
				}
				return nil
			})
			return "PASS", err
		}}
		got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
		checkStatus(t, got, "PASS")
		if sampled.IsZero() || cases != 1 || operations != 1 || got.Requests != 1 {
			t.Error("sampled injected clock did not drive enabled execution")
		}
	})
}

// Isolate the injected admission reading from the context timer: the supplied
// reading is the suite's actual deadline in the same synctest clock domain,
// while the timer has not fired. Admission must still apply now >= deadline.
func TestHarnessAdmissionUsesInjectedClockValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		deps := fixtureDeps()
		var boundary time.Time
		var admissionReads, cases, operations int
		deps.Now = func() time.Time {
			if !boundary.IsZero() {
				admissionReads++
				return boundary
			}
			return time.Now()
		}
		c := harnessCase{ID: "read-a", Run: func(ctx context.Context, r harnessRequest) (string, error) {
			cases++
			deadline, ok := ctx.Deadline()
			if !ok || !time.Now().Before(deadline) || ctx.Err() != nil {
				t.Error("admission fixture requires an unexpired suite deadline")
				return "FAIL", nil
			}
			boundary = deadline
			err := r(func(context.Context) error { operations++; return nil })
			if err == nil {
				t.Error("admission ignored injected now equal to deadline")
			}
			return "PASS", nil
		}}
		got := runHarness(context.Background(), true, deps, []string{"read-a"}, []harnessCase{c})
		checkStatus(t, got, "BLOCKED")
		if cases != 1 || admissionReads == 0 || operations != 0 || got.Requests != 0 || got.Cases["read-a"] != "BLOCKED" {
			t.Error("injected deadline boundary did not prevent admission without consuming budget")
		}
	})
}
