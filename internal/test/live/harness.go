package live

import (
	"context"
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

// T023 compile-only declarations: T024 owns all behavior.
func (harnessFailure) Error() string { return "not_implemented" }
func runHarness(context.Context, bool, harnessDeps, []string, []harnessCase) harnessReport {
	return harnessReport{Category: "not_implemented"}
}
