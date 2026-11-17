package integration

import (
	"context"
	"errors"
	"github.com/Stone-afk/chain-api-go/transport"
	"net/http"
	"os"
	"testing"
	"time"
)

func liveDependencies() runnerDeps {
	return runnerDeps{
		LookupEnv: os.LookupEnv, ReadFile: os.ReadFile, Now: time.Now,
		Wait: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
		NewDoer: func(cfg harnessConfig) (transport.HTTPDoer, func(), error) {
			tr := &http.Transport{}
			client := &http.Client{Transport: tr, Timeout: cfg.TotalTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			return client, tr.CloseIdleConnections, nil
		},
		WriteReadReport: func(b []byte) error { return os.WriteFile("evidence/read.json", b, 0600) },
		ReadBroadcastJournal: func() ([]byte, error) {
			b, e := os.ReadFile("evidence/broadcast.json")
			if errors.Is(e, os.ErrNotExist) {
				return nil, nil
			}
			return b, e
		},
		AppendBroadcastRecord: func(b []byte) error {
			f, e := os.OpenFile("evidence/broadcast.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if e != nil {
				return e
			}
			_, e = f.Write(b)
			if e == nil {
				e = f.Sync()
			}
			closeErr := f.Close()
			if e != nil {
				return e
			}
			return closeErr
		},
	}
}
func TestLiveRead(t *testing.T) {
	r := runLiveRead(context.Background(), liveDependencies())
	if r.Status == "NOT_RUN" {
		t.Skip("live read disabled")
	}
	if r.Status != "PASS" {
		t.Fail()
	}
}
func TestLiveBroadcast(t *testing.T) {
	r := runLiveBroadcast(context.Background(), liveDependencies())
	if r.Status == "NOT_RUN" {
		t.Skip("live broadcast disabled")
	}
	if r.Status != "PASS" {
		t.Fail()
	}
}
