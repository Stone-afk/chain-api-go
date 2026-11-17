package integration

import (
	"context"
	"errors"
	"sync"
	"time"
)

type harnessConfig struct {
	Alias, Endpoint, Network                  string
	Headers                                   map[string]string
	MaxRequests, MaxConcurrent                int
	MinInterval, TotalTimeout                 time.Duration
	DynamicInputsPath, SubmissionMaterialPath string
}
type harnessDeps struct {
	Load         func() (harnessConfig, error)
	LoadManifest func() ([]byte, string, error)
	Now          func() time.Time
	Wait         func(context.Context, time.Duration) error
}
type harnessRequest func(func(context.Context) error) error
type harnessCase struct {
	ID  string
	Run func(context.Context, harnessRequest) (string, error)
}
type harnessReport struct {
	Status, Category, Digest string
	Requests                 int
	Cases                    map[string]string
}
type harnessFailure struct{ Status, Category string }

func (f harnessFailure) Error() string {
	if (f.Status == "FAIL" || f.Status == "BLOCKED") && (f.Category == "assertion" || f.Category == "environment") {
		return f.Category
	}
	return "invalid_result"
}

// Only fixed, local categories are stored or returned by the limiter.
type harnessIssue struct{ status, category string }

func (i harnessIssue) Error() string {
	switch i.status {
	case "FAIL":
		switch i.category {
		case "assertion", "environment", "invalid_result", "unclassified":
			return i.category
		}
	case "BLOCKED":
		switch i.category {
		case "assertion", "environment", "budget", "canceled", "deadline":
			return i.category
		}
	}
	return "invalid_result"
}

func classifyHarnessError(err error) harnessIssue {
	if err == nil {
		return harnessIssue{"PASS", ""}
	}
	if errors.Is(err, context.Canceled) {
		return harnessIssue{"BLOCKED", "canceled"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return harnessIssue{"BLOCKED", "deadline"}
	}
	var issue harnessIssue
	if errors.As(err, &issue) {
		if issue.Error() == "invalid_result" {
			return harnessIssue{"FAIL", "invalid_result"}
		}
		return issue
	}
	var failure harnessFailure
	if errors.As(err, &failure) {
		if failure.Error() == "invalid_result" {
			return harnessIssue{"FAIL", "invalid_result"}
		}
		return harnessIssue{failure.Status, failure.Category}
	}
	return harnessIssue{"FAIL", "unclassified"}
}

func harnessStatus(status string) harnessIssue {
	switch status {
	case "PASS":
		return harnessIssue{"PASS", ""}
	case "FAIL":
		return harnessIssue{"FAIL", "assertion"}
	case "BLOCKED":
		return harnessIssue{"BLOCKED", "environment"}
	case "NOT_RUN":
		return harnessIssue{"NOT_RUN", "not_run"}
	default:
		return harnessIssue{"FAIL", "invalid_result"}
	}
}

func mergeHarnessIssue(a, b harnessIssue) harnessIssue {
	rank := func(s string) int {
		if s == "FAIL" {
			return 2
		}
		if s == "BLOCKED" || s == "NOT_RUN" {
			return 1
		}
		return 0
	}
	if rank(b.status) > rank(a.status) || rank(b.status) == rank(a.status) && b.category < a.category {
		return b
	}
	return a
}

func runHarness(parent context.Context, enabled bool, deps harnessDeps, required []string, cases []harnessCase) harnessReport {
	report := harnessReport{Status: "NOT_RUN"}
	if !enabled {
		return report
	}
	report.Status, report.Category = "BLOCKED", "configuration"
	if parent == nil || deps.Load == nil || deps.LoadManifest == nil || deps.Now == nil || deps.Wait == nil {
		return report
	}
	report.Category = "catalog"
	if len(required) == 0 {
		return report
	}
	ledger := make(map[string]string, len(required))
	for _, id := range required {
		if id == "" {
			return report
		}
		if _, exists := ledger[id]; exists {
			return report
		}
		ledger[id] = "NOT_RUN"
	}
	seen := make(map[string]bool, len(cases))
	for _, c := range cases {
		if _, exists := ledger[c.ID]; !exists || seen[c.ID] || c.Run == nil {
			return report
		}
		seen[c.ID] = true
	}
	report.Cases = ledger
	report.Category = "configuration"
	cfg, err := deps.Load()
	if err != nil || cfg.Alias == "" || cfg.Network == "" || !validEndpoint(cfg.Endpoint) || cfg.MaxRequests <= 0 || cfg.MaxConcurrent <= 0 || cfg.MinInterval <= 0 || cfg.TotalTimeout <= 0 {
		return report
	}
	headers := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	cfg.Headers = headers
	report.Category = "manifest"
	raw, expected, err := deps.LoadManifest()
	if err != nil {
		return report
	}
	digest, err := verifyManifest(raw, expected)
	if err != nil {
		return report
	}
	report.Digest = digest
	ctx, cancel := context.WithDeadline(parent, deps.Now().Add(cfg.TotalTimeout))
	defer cancel()
	deadline, _ := ctx.Deadline()
	checkContext := func() harnessIssue {
		if err := ctx.Err(); err != nil {
			return classifyHarnessError(err)
		}
		if !deps.Now().Before(deadline) {
			return harnessIssue{"BLOCKED", "deadline"}
		}
		return harnessIssue{"PASS", ""}
	}
	var mu sync.Mutex
	requests := 0
	var lastStart time.Time
	slots := make(chan struct{}, cfg.MaxConcurrent)
	// The admission token serializes start-time decisions, not operations.
	admission := make(chan struct{}, 1)
	overall := harnessIssue{"PASS", ""}
	for _, c := range cases {
		if issue := checkContext(); issue.status != "PASS" {
			overall = mergeHarnessIssue(overall, issue)
			break
		}
		sticky := harnessIssue{"PASS", ""}
		record := func(issue harnessIssue) error {
			if issue.status == "PASS" {
				return nil
			}
			mu.Lock()
			sticky = mergeHarnessIssue(sticky, issue)
			mu.Unlock()
			return issue
		}
		budgetReached := func() bool { mu.Lock(); defer mu.Unlock(); return requests >= cfg.MaxRequests }
		request := func(operation func(context.Context) error) error {
			if operation == nil {
				return record(harnessIssue{"FAIL", "invalid_result"})
			}
			if issue := checkContext(); issue.status != "PASS" {
				return record(issue)
			}
			if budgetReached() {
				return record(harnessIssue{"BLOCKED", "budget"})
			}
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				return record(classifyHarnessError(ctx.Err()))
			}
			defer func() { <-slots }()
			select {
			case admission <- struct{}{}:
			case <-ctx.Done():
				return record(classifyHarnessError(ctx.Err()))
			}
			// Every rejection releases both admission and capacity without charging.
			admit := func() harnessIssue {
				defer func() { <-admission }()
				for {
					if issue := checkContext(); issue.status != "PASS" {
						return issue
					}
					if budgetReached() {
						return harnessIssue{"BLOCKED", "budget"}
					}
					now := deps.Now()
					if !lastStart.IsZero() {
						if delay := lastStart.Add(cfg.MinInterval).Sub(now); delay > 0 {
							if err := deps.Wait(ctx, delay); err != nil {
								return classifyHarnessError(err)
							}
							continue
						}
					}
					// The gate owns lastStart; the mutex protects the shared request ledger.
					lastStart = now
					mu.Lock()
					requests++
					mu.Unlock()
					return harnessIssue{"PASS", ""}
				}
			}
			if issue := admit(); issue.status != "PASS" {
				return record(issue)
			}
			issue := classifyHarnessError(operation(ctx))
			issue = mergeHarnessIssue(issue, checkContext())
			return record(issue)
		}
		status, err := c.Run(ctx, request)
		result := mergeHarnessIssue(harnessStatus(status), classifyHarnessError(err))
		result = mergeHarnessIssue(result, checkContext())
		mu.Lock()
		result = mergeHarnessIssue(result, sticky)
		mu.Unlock()
		ledger[c.ID] = result.status
		overall = mergeHarnessIssue(overall, result)
	}
	for _, status := range ledger {
		if status == "NOT_RUN" {
			overall = mergeHarnessIssue(overall, harnessStatus(status))
		}
	}
	if overall.status == "NOT_RUN" {
		overall.status = "BLOCKED"
	}
	report.Status, report.Category = overall.status, overall.category
	mu.Lock()
	report.Requests = requests
	mu.Unlock()
	return report
}
