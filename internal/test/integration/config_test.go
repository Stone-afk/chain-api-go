package integration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const configFixture = `{"targets":{"example":{"endpoint_env":"ENDPOINT","header_env":{"Authorization":"AUTH"},"network":"fixture","max_requests":4,"max_concurrent":2,"min_interval":"1s","total_timeout":"30s","dynamic_inputs_path":"${UNEXPANDED}/dynamic","submission_material_path":"~/material"}}}`

func TestHarnessConfigExplicitSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "synthetic.json")
	if err := os.WriteFile(path, []byte(configFixture), 0600); err != nil {
		t.Fatal(err)
	}
	reads := 0
	lookups := map[string]int{}
	env := map[string]string{"ENDPOINT": "https://fixture.invalid/rpc?key=SYNTHETIC_SECRET", "AUTH": "Bearer ${LITERAL} $(no-shell)"}
	cfg, err := loadHarnessConfig(path, "example", func(p string) ([]byte, error) {
		reads++
		if p != path {
			t.Error("loader searched another path")
			return nil, errors.New("wrong path")
		}
		return os.ReadFile(p)
	}, func(k string) (string, bool) { lookups[k]++; v, ok := env[k]; return v, ok })
	if err != nil {
		t.Error("explicit synthetic config rejected")
	}
	want := harnessConfig{Alias: "example", Endpoint: env["ENDPOINT"], Network: "fixture", Headers: map[string]string{"Authorization": env["AUTH"]}, MaxRequests: 4, MaxConcurrent: 2, MinInterval: time.Second, TotalTimeout: 30 * time.Second, DynamicInputsPath: "${UNEXPANDED}/dynamic", SubmissionMaterialPath: "~/material"}
	if !reflect.DeepEqual(cfg, want) {
		t.Error("config did not preserve explicit values and unexpanded material paths")
	}
	if reads != 1 || !reflect.DeepEqual(lookups, map[string]int{"ENDPOINT": 1, "AUTH": 1}) {
		t.Error("config must read exactly the supplied file and referenced environment names")
	}
	env["AUTH"] = "changed"
	if cfg.Headers["Authorization"] != want.Headers["Authorization"] {
		t.Error("config retained mutable source")
	}
}

func TestHarnessConfigRejectsUnsafeOrAmbiguousInput(t *testing.T) {
	for _, tc := range []struct{ name, old, new string }{
		{"unknown", `"network"`, `"unknown"`},
		{"duplicate", `"network":"fixture"`, `"network":"fixture","network":"other"`},
		{"escaped_duplicate", `"network":"fixture"`, `"network":"fixture","\u006eetwork":"other"`},
		{"duplicate_target", `"targets":`, `"targets":{},"targets":`},
		{"type", `"max_requests":4`, `"max_requests":"4"`},
		{"null_target", configFixture, `{"targets":{"example":null}}`},
		{"null", configFixture, `null`}, {"array", configFixture, `[]`},
		{"trailing", configFixture, configFixture + `{}`},
		{"malformed", configFixture, `{"SYNTHETIC_SECRET":`},
		{"missing_endpoint", `"endpoint_env":"ENDPOINT",`, ``},
		{"bad_env_name", `"ENDPOINT"`, `"BAD-NAME"`},
		{"bad_header_ref", `"AUTH"`, `"${AUTH}"`},
		{"bad_header_name", `"Authorization"`, `"Bad Header"`},
		{"header_duplicate", `"Authorization":"AUTH"`, `"Authorization":"AUTH","authorization":"AUTH"`},
		{"requests_zero", `"max_requests":4`, `"max_requests":0`},
		{"concurrency_negative", `"max_concurrent":2`, `"max_concurrent":-1`},
		{"interval_zero", `"1s"`, `"0s"`}, {"timeout_negative", `"30s"`, `"-1s"`},
		{"duration_unit", `"1s"`, `"1"`}, {"duration_overflow", `"30s"`, `"999999999999999999999h"`},
		{"duration_missing", `"total_timeout":"30s",`, ``},
		{"path_type", `"~/material"`, `42`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(configFixture, tc.old, tc.new, 1)
			cfg, err := loadHarnessConfig("SYNTHETIC_SECRET", "example", func(string) ([]byte, error) { return []byte(raw), nil }, func(string) (string, bool) { return "https://fixture.invalid", true })
			if err == nil || !reflect.DeepEqual(cfg, harnessConfig{}) {
				t.Error("invalid config returned success or partial configuration")
			}
			assertRedacted(t, err, "SYNTHETIC_SECRET", "https://fixture.invalid")
		})
	}
	for _, tc := range []struct {
		name, path, alias, endpoint, auth string
		present                           bool
	}{
		{"empty_path", "", "example", "https://fixture.invalid", "token", true},
		{"empty_alias", "synthetic", "", "https://fixture.invalid", "token", true},
		{"missing_alias", "synthetic", "absent", "https://fixture.invalid", "token", true},
		{"missing_env", "synthetic", "example", "", "", false},
		{"empty_endpoint", "synthetic", "example", "", "token", true},
		{"empty_header", "synthetic", "example", "https://fixture.invalid", "", true},
		{"scheme", "synthetic", "example", "ftp://fixture.invalid", "token", true},
		{"host", "synthetic", "example", "https:///rpc", "token", true},
		{"fragment", "synthetic", "example", "https://fixture.invalid/#secret", "token", true},
		{"header_crlf", "synthetic", "example", "https://fixture.invalid", "token\r\nInjected: yes", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			cfg, err := loadHarnessConfig(tc.path, tc.alias, func(string) ([]byte, error) { reads++; return []byte(configFixture), nil }, func(k string) (string, bool) {
				if k == "ENDPOINT" {
					return tc.endpoint, tc.present
				}
				return tc.auth, tc.present
			})
			if err == nil || !reflect.DeepEqual(cfg, harnessConfig{}) {
				t.Error("invalid source accepted or returned partial config")
			}
			if (tc.path == "" || tc.alias == "") && reads != 0 {
				t.Error("missing explicit source performed file read")
			}
			assertRedacted(t, err, "https://fixture.invalid", "token", "Injected")
		})
	}
	t.Run("reader_error", func(t *testing.T) {
		cfg, err := loadHarnessConfig("SYNTHETIC_SECRET", "example", func(string) ([]byte, error) {
			return nil, &os.PathError{Op: "read", Path: "SYNTHETIC_SECRET", Err: errors.New("PRIVATE_DETAIL")}
		}, func(string) (string, bool) { t.Error("env lookup after failed read"); return "", false })
		if err == nil || !reflect.DeepEqual(cfg, harnessConfig{}) {
			t.Error("read failure returned usable config")
		}
		assertRedacted(t, err, "SYNTHETIC_SECRET", "PRIVATE_DETAIL")
	})
}

func TestHarnessConfigTemplateHasOnlyReferences(t *testing.T) {
	raw, err := os.ReadFile("config.example.json")
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatal("template must be legal JSON")
	}
	const expected = `{"targets":{"example":{"endpoint_env":"CHAIN_API_GO_EXAMPLE_ENDPOINT","header_env":{"Authorization":"CHAIN_API_GO_EXAMPLE_AUTHORIZATION"},"network":"example-network","max_requests":10,"max_concurrent":1,"min_interval":"1s","total_timeout":"30s","dynamic_inputs_path":"","submission_material_path":""}}}`
	if err = json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Error("template must contain only approved reference placeholders and metadata")
	}
}

func TestHarnessConfigMissingEnvIgnoresReturnedValue(t *testing.T) {
	for _, missing := range []string{"ENDPOINT", "AUTH"} {
		t.Run(missing, func(t *testing.T) {
			seenMissing := false
			lookups := map[string]int{}
			reads := 0
			cfg, err := loadHarnessConfig("synthetic", "example", func(p string) ([]byte, error) {
				reads++
				if p != "synthetic" {
					t.Error("unexpected path")
				}
				return []byte(configFixture), nil
			}, func(k string) (string, bool) {
				if seenMissing {
					t.Error("unnecessary lookup after missing reference")
				}
				lookups[k]++
				if k != "ENDPOINT" && k != "AUTH" {
					t.Error("unreferenced environment lookup")
				}
				if k == missing {
					seenMissing = true
				}
				value := "Bearer SYNTHETIC_SECRET"
				if k == "ENDPOINT" {
					value = "https://fixture.invalid/SYNTHETIC_SECRET"
				}
				return value, k != missing
			})
			if err == nil || !reflect.DeepEqual(cfg, harnessConfig{}) {
				t.Error("present=false returned usable config despite nonempty value")
			}
			if reads != 1 || lookups[missing] != 1 {
				t.Error("missing explicit reference was not checked")
			}
			assertRedacted(t, err, "SYNTHETIC_SECRET", "https://fixture.invalid", "Bearer")
		})
	}
}
