package integration

import (
	"encoding/json"
	"errors"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func loadHarnessConfig(path, alias string, readFile func(string) ([]byte, error), lookupEnv func(string) (string, bool)) (harnessConfig, error) {
	bad := func() (harnessConfig, error) { return harnessConfig{}, errors.New("configuration") }
	if path == "" || alias == "" || readFile == nil || lookupEnv == nil {
		return bad()
	}
	raw, err := readFile(path)
	if err != nil {
		return bad()
	}
	root, err := strictObject(raw)
	if err != nil || len(root) != 1 {
		return bad()
	}
	targets, ok := root["targets"].(map[string]any)
	if !ok {
		return bad()
	}
	// Validate every target's shape before resolving the selected references.
	for _, target := range targets {
		fields, ok := target.(map[string]any)
		if !ok {
			return bad()
		}
		for key, v := range fields {
			switch key {
			case "endpoint_env", "network", "min_interval", "total_timeout", "dynamic_inputs_path", "submission_material_path":
				if _, ok := v.(string); !ok {
					return bad()
				}
			case "max_requests", "max_concurrent":
				if _, ok := v.(json.Number); !ok {
					return bad()
				}
			case "header_env":
				headers, ok := v.(map[string]any)
				if !ok {
					return bad()
				}
				for _, ref := range headers {
					if _, ok := ref.(string); !ok {
						return bad()
					}
				}
			default:
				return bad()
			}
		}
	}
	fields, ok := targets[alias].(map[string]any)
	if !ok {
		return bad()
	}
	str := func(k string) string { s, _ := fields[k].(string); return s }
	number := func(k string) int {
		n, _ := fields[k].(json.Number)
		v, err := n.Int64()
		if err != nil || int64(int(v)) != v {
			return 0
		}
		return int(v)
	}
	interval, err := time.ParseDuration(str("min_interval"))
	if err != nil {
		return bad()
	}
	timeout, err := time.ParseDuration(str("total_timeout"))
	if err != nil {
		return bad()
	}
	cfg := harnessConfig{Alias: alias, Network: str("network"), MaxRequests: number("max_requests"), MaxConcurrent: number("max_concurrent"), MinInterval: interval, TotalTimeout: timeout, DynamicInputsPath: str("dynamic_inputs_path"), SubmissionMaterialPath: str("submission_material_path"), Headers: map[string]string{}}
	if cfg.Network == "" || cfg.MaxRequests <= 0 || cfg.MaxConcurrent <= 0 || interval <= 0 || timeout <= 0 {
		return bad()
	}
	envName := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	endpointRef := str("endpoint_env")
	if !envName.MatchString(endpointRef) {
		return bad()
	}
	endpoint, present := lookupEnv(endpointRef)
	if !present || !validEndpoint(endpoint) {
		return bad()
	}
	cfg.Endpoint = endpoint
	refs, _ := fields["header_env"].(map[string]any)
	for key, ref := range refs {
		name := textproto.CanonicalMIMEHeaderKey(key)
		if !validHeaderName(key) || !envName.MatchString(ref.(string)) {
			return bad()
		}
		if _, duplicate := cfg.Headers[name]; duplicate {
			return bad()
		}
		v, present := lookupEnv(ref.(string))
		if !present || v == "" || strings.ContainsAny(v, "\r\n") {
			return bad()
		}
		cfg.Headers[name] = v
	}
	return cfg, nil
}

func validEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && !strings.Contains(endpoint, "#")
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c) {
			continue
		}
		return false
	}
	return true
}
