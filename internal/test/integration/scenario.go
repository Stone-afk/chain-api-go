package integration

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Stone-afk/chain-api-go/apierror"
	"github.com/Stone-afk/chain-api-go/transport"
)

const (
	scenarioSwitch      = "CHAIN_API_GO_LIVE_SCENARIO"
	scenarioProbeSwitch = "CHAIN_API_GO_LIVE_SCENARIO_SUBMIT_PROBE"
	scenarioConfigPath  = "config.local.yaml"
)

var (
	errScenarioYAML   = errors.New("scenario_yaml")
	errScenarioConfig = errors.New("scenario_config")
	errScenarioBudget = errors.New("scenario_budget")
)

// scenarioChains is the fixed execution and reporting order.
var scenarioChains = []string{"ethereum", "bnb", "bitcoin", "solana", "tron"}

type scenarioChainConfig struct {
	Endpoint string
	Headers  map[string]string
	Params   map[string]string
	// RequestTimeout is resolved by runScenario before NewDoer; parsed configs leave it zero.
	RequestTimeout time.Duration
}

type scenarioConfig struct {
	MaxRequests               int
	MinInterval, TotalTimeout time.Duration
	// RequestTimeout is optional; zero means TotalTimeout bounds each HTTP request.
	RequestTimeout           time.Duration
	RequiredChains           []string
	KnownUnsupported         []string
	SubmitProbeAuthorization string
	Chains                   map[string]scenarioChainConfig
}

// scenarioRunner executes one chain through the budgeted doer and returns status by catalog ID.
type scenarioRunner func(ctx context.Context, cfg scenarioConfig, chain scenarioChainConfig, doer transport.HTTPDoer) map[string]string

type scenarioDeps struct {
	LookupEnv func(string) (string, bool)
	ReadFile  func(string) ([]byte, error)
	NewDoer   func(scenarioChainConfig) (transport.HTTPDoer, func(), error)
	Now       func() time.Time
	Wait      func(context.Context, time.Duration) error
	Runners   map[string]scenarioRunner
}

// Kind is one of read, submit or raw.
type scenarioEntry struct{ ID, Chain, Method, Kind, Status string }

type scenarioIssue struct{ Status, Category string }

type scenarioReport struct {
	Status, Category string
	Requests         int
	RequiredChains   []string
	Entries          []scenarioEntry
}

type scenarioBudget struct {
	mu       sync.Mutex
	next     transport.HTTPDoer
	limit    int
	interval time.Duration
	wait     func(context.Context, time.Duration) error
	count    int
}

type scenarioLine struct {
	no, indent int
	text       string
}

func scenarioYAMLError(line int) error { return fmt.Errorf("%w: line %d", errScenarioYAML, line) }

// parseScenarioYAML accepts only indented mappings, plain scalars, scalar sequences and comments.
func parseScenarioYAML(raw []byte) (map[string]any, error) {
	lines, err := scanScenarioYAML(raw)
	if err != nil {
		return nil, err
	}
	m, next, err := parseScenarioMapping(lines, 0, 0)
	if err != nil {
		return nil, err
	}
	if next != len(lines) {
		return nil, scenarioYAMLError(lines[next].no)
	}
	return m, nil
}

func scanScenarioYAML(raw []byte) ([]scenarioLine, error) {
	var lines []scenarioLine
	for i, l := range strings.Split(string(raw), "\n") {
		l = strings.TrimSuffix(l, "\r")
		if strings.ContainsRune(l, '\t') {
			return nil, scenarioYAMLError(i + 1)
		}
		text := strings.TrimLeft(l, " ")
		indent := len(l) - len(text)
		text = strings.TrimRight(stripScenarioComment(text), " ")
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "---") || strings.HasPrefix(text, "...") || strings.HasPrefix(text, "%") {
			return nil, scenarioYAMLError(i + 1)
		}
		lines = append(lines, scenarioLine{no: i + 1, indent: indent, text: text})
	}
	if len(lines) == 0 || lines[0].indent != 0 || isScenarioSeqItem(lines[0].text) {
		return nil, scenarioYAMLError(1)
	}
	return lines, nil
}

func stripScenarioComment(text string) string {
	for i := 0; i < len(text); i++ {
		if text[i] == '#' && (i == 0 || text[i-1] == ' ') {
			return text[:i]
		}
	}
	return text
}

func isScenarioSeqItem(text string) bool { return text == "-" || strings.HasPrefix(text, "- ") }

func splitScenarioKey(text string) (string, string, bool) {
	for i := 0; i < len(text); i++ {
		if text[i] == ':' && (i+1 == len(text) || text[i+1] == ' ') {
			key := text[:i]
			if key == "" || strings.ContainsAny(key, " \"'") || strings.ContainsRune("-?:,[]{}#&*!|>%@`", rune(key[0])) {
				return "", "", false
			}
			return key, strings.TrimLeft(text[i+1:], " "), true
		}
	}
	return "", "", false
}

func isScenarioPlainScalar(v string) bool {
	if v == "" || strings.ContainsRune("&*[]{}|>!%@`'\"#,?", rune(v[0])) || isScenarioSeqItem(v) {
		return false
	}
	return !strings.Contains(v, ": ") && !strings.HasSuffix(v, ":")
}

func parseScenarioMapping(lines []scenarioLine, pos, indent int) (map[string]any, int, error) {
	m := map[string]any{}
	for pos < len(lines) {
		l := lines[pos]
		if l.indent < indent {
			break
		}
		if l.indent > indent {
			return nil, 0, scenarioYAMLError(l.no)
		}
		key, value, ok := splitScenarioKey(l.text)
		if !ok {
			return nil, 0, scenarioYAMLError(l.no)
		}
		if _, dup := m[key]; dup {
			return nil, 0, scenarioYAMLError(l.no)
		}
		pos++
		if value != "" {
			if !isScenarioPlainScalar(value) {
				return nil, 0, scenarioYAMLError(l.no)
			}
			m[key] = value
			continue
		}
		if pos >= len(lines) || lines[pos].indent <= indent {
			return nil, 0, scenarioYAMLError(l.no)
		}
		childIndent := lines[pos].indent
		if isScenarioSeqItem(lines[pos].text) {
			items, next, err := parseScenarioSequence(lines, pos, childIndent)
			if err != nil {
				return nil, 0, err
			}
			m[key], pos = items, next
			continue
		}
		child, next, err := parseScenarioMapping(lines, pos, childIndent)
		if err != nil {
			return nil, 0, err
		}
		m[key], pos = child, next
	}
	return m, pos, nil
}

func parseScenarioSequence(lines []scenarioLine, pos, indent int) ([]string, int, error) {
	var items []string
	for pos < len(lines) && lines[pos].indent >= indent {
		l := lines[pos]
		if l.indent > indent || !isScenarioSeqItem(l.text) {
			return nil, 0, scenarioYAMLError(l.no)
		}
		item := strings.TrimLeft(l.text[1:], " ")
		if _, _, isKey := splitScenarioKey(item); isKey || !isScenarioPlainScalar(item) {
			return nil, 0, scenarioYAMLError(l.no)
		}
		items = append(items, item)
		pos++
	}
	return items, pos, nil
}

func scenarioConfigError(field string) error { return fmt.Errorf("%w: %s", errScenarioConfig, field) }

// checkScenarioBlockKinds rejects a top-level block whose first child fixes the wrong node kind.
func checkScenarioBlockKinds(lines []scenarioLine) error {
	for i, l := range lines {
		if l.indent != 0 || i+1 == len(lines) || lines[i+1].indent == 0 {
			continue
		}
		key, value, ok := splitScenarioKey(l.text)
		if !ok || value != "" {
			continue
		}
		isSeq := isScenarioSeqItem(lines[i+1].text)
		switch key {
		case "required_chains", "known_unsupported":
			if !isSeq {
				return scenarioConfigError(key)
			}
		case "chains":
			if isSeq {
				return scenarioConfigError(key)
			}
		}
	}
	return nil
}

func parseScenarioConfig(raw []byte) (scenarioConfig, error) {
	lines, err := scanScenarioYAML(raw)
	if err != nil {
		return scenarioConfig{}, err
	}
	if err := checkScenarioBlockKinds(lines); err != nil {
		return scenarioConfig{}, err
	}
	doc, err := parseScenarioYAML(raw)
	if err != nil {
		return scenarioConfig{}, err
	}
	for k := range doc {
		switch k {
		case "max_requests", "min_interval", "total_timeout", "request_timeout", "required_chains", "known_unsupported", "submit_probe_authorization", "chains":
		default:
			return scenarioConfig{}, scenarioConfigError("unknown_key")
		}
	}
	var cfg scenarioConfig
	var ok bool
	if cfg.MaxRequests, ok = scenarioPositiveInt(doc["max_requests"]); !ok {
		return scenarioConfig{}, scenarioConfigError("max_requests")
	}
	if cfg.MinInterval, ok = scenarioPositiveDuration(doc["min_interval"]); !ok {
		return scenarioConfig{}, scenarioConfigError("min_interval")
	}
	if cfg.TotalTimeout, ok = scenarioPositiveDuration(doc["total_timeout"]); !ok {
		return scenarioConfig{}, scenarioConfigError("total_timeout")
	}
	if v, present := doc["request_timeout"]; present {
		if cfg.RequestTimeout, ok = scenarioPositiveDuration(v); !ok {
			return scenarioConfig{}, scenarioConfigError("request_timeout")
		}
	}
	if v, present := doc["submit_probe_authorization"]; present {
		if cfg.SubmitProbeAuthorization, ok = v.(string); !ok {
			return scenarioConfig{}, scenarioConfigError("submit_probe_authorization")
		}
	}
	chains, ok := doc["chains"].(map[string]any)
	if !ok {
		return scenarioConfig{}, scenarioConfigError("chains")
	}
	cfg.Chains = map[string]scenarioChainConfig{}
	for name, v := range chains {
		if !isScenarioChain(name) {
			return scenarioConfig{}, scenarioConfigError("chains")
		}
		c, err := parseScenarioChain(v)
		if err != nil {
			return scenarioConfig{}, err
		}
		cfg.Chains[name] = c
	}
	if cfg.RequiredChains, ok = scenarioUniqueList(doc["required_chains"]); !ok {
		return scenarioConfig{}, scenarioConfigError("required_chains")
	}
	for _, name := range cfg.RequiredChains {
		if _, configured := cfg.Chains[name]; !configured {
			return scenarioConfig{}, scenarioConfigError("required_chains")
		}
	}
	if v, present := doc["known_unsupported"]; present {
		if cfg.KnownUnsupported, ok = scenarioUniqueList(v); !ok {
			return scenarioConfig{}, scenarioConfigError("known_unsupported")
		}
		kinds := map[string]string{}
		for _, e := range scenarioCatalog() {
			kinds[e.ID] = e.Kind
		}
		for _, id := range cfg.KnownUnsupported {
			if kind, known := kinds[id]; !known || kind == "submit" {
				return scenarioConfig{}, scenarioConfigError("known_unsupported")
			}
		}
	}
	return cfg, nil
}

func parseScenarioChain(v any) (scenarioChainConfig, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return scenarioChainConfig{}, scenarioConfigError("chain")
	}
	var c scenarioChainConfig
	for k, raw := range m {
		switch k {
		case "endpoint":
			s, isString := raw.(string)
			if !isString || !isScenarioEndpoint(s) {
				return scenarioChainConfig{}, scenarioConfigError("endpoint")
			}
			c.Endpoint = s
		case "headers":
			h, isMap := raw.(map[string]any)
			if !isMap {
				return scenarioChainConfig{}, scenarioConfigError("headers")
			}
			c.Headers = map[string]string{}
			seen := map[string]bool{}
			for name, value := range h {
				s, isString := value.(string)
				canonical := http.CanonicalHeaderKey(name)
				if !validHeaderName(name) || !isString || !isScenarioHeaderValue(s) || seen[canonical] {
					return scenarioChainConfig{}, scenarioConfigError("headers")
				}
				seen[canonical] = true
				c.Headers[name] = s
			}
		case "params":
			p, isMap := raw.(map[string]any)
			if !isMap {
				return scenarioChainConfig{}, scenarioConfigError("params")
			}
			c.Params = map[string]string{}
			for name, value := range p {
				s, isString := value.(string)
				if !isString {
					return scenarioChainConfig{}, scenarioConfigError("params")
				}
				c.Params[name] = s
			}
		default:
			return scenarioChainConfig{}, scenarioConfigError("chain_key")
		}
	}
	if c.Endpoint == "" {
		return scenarioChainConfig{}, scenarioConfigError("endpoint")
	}
	return c, nil
}

func isScenarioChain(name string) bool {
	for _, c := range scenarioChains {
		if c == name {
			return true
		}
	}
	return false
}

func isScenarioEndpoint(s string) bool {
	u, err := url.Parse(s)
	return err == nil && validEndpoint(s) && u.User == nil
}

func isScenarioHeaderValue(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f {
			return false
		}
	}
	return true
}

func scenarioPositiveInt(v any) (int, bool) {
	s, ok := v.(string)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil && n > 0
}

func scenarioPositiveDuration(v any) (time.Duration, bool) {
	s, ok := v.(string)
	if !ok {
		return 0, false
	}
	d, err := time.ParseDuration(s)
	return d, err == nil && d > 0
}

func scenarioUniqueList(v any) ([]string, bool) {
	items, ok := v.([]string)
	if !ok || len(items) == 0 {
		return nil, false
	}
	seen := map[string]bool{}
	for _, item := range items {
		if seen[item] {
			return nil, false
		}
		seen[item] = true
	}
	return items, true
}

func newScenarioBudget(next transport.HTTPDoer, maxRequests int, interval time.Duration, wait func(context.Context, time.Duration) error) *scenarioBudget {
	return &scenarioBudget{next: next, limit: maxRequests, interval: interval, wait: wait}
}

// Do serializes requests so pacing and the request count stay exact.
func (b *scenarioBudget) Do(req *http.Request) (*http.Response, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ctx := req.Context()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b.count >= b.limit {
		return nil, errScenarioBudget
	}
	if b.count > 0 {
		if err := b.wait(ctx, b.interval); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	b.count++
	return b.next.Do(req)
}

func (b *scenarioBudget) Requests() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// classifyScenarioError maps errors to fixed categories and never retains error text.
func classifyScenarioError(err error) scenarioIssue {
	switch {
	case err == nil:
		return scenarioIssue{"PASS", ""}
	case errors.Is(err, errScenarioBudget):
		return scenarioIssue{"BLOCKED", "budget"}
	case errors.Is(err, context.Canceled):
		return scenarioIssue{"BLOCKED", "canceled"}
	case errors.Is(err, context.DeadlineExceeded):
		return scenarioIssue{"BLOCKED", "deadline"}
	}
	var apiErr apierror.Error
	if errors.As(err, &apiErr) {
		switch apiErr.Kind {
		case apierror.Transport, apierror.HTTP:
			return scenarioIssue{"BLOCKED", "environment"}
		case apierror.RPC:
			return scenarioIssue{"FAIL", "rpc"}
		case apierror.Protocol:
			return scenarioIssue{"FAIL", "protocol"}
		case apierror.NetworkMismatch:
			return scenarioIssue{"FAIL", "network_mismatch"}
		case apierror.Unsupported:
			return scenarioIssue{"FAIL", "unsupported"}
		case apierror.Configuration:
			return scenarioIssue{"FAIL", "configuration"}
		case apierror.InvalidArgument:
			return scenarioIssue{"FAIL", "invalid_argument"}
		}
	}
	return scenarioIssue{"FAIL", "unclassified"}
}

func scenarioDiagnostic(id string, err error) string {
	issue := classifyScenarioError(err)
	return fmt.Sprintf("%s %s %s", id, issue.Status, issue.Category)
}

var scenarioEVMMethods = []string{"BalanceAt", "BlockByNumber", "BlockNumber", "CallContract", "ChainID", "CodeAt", "EstimateGas", "FeeHistory", "FilterLogs", "NonceAt", "SendRawTransaction", "SuggestGasPrice", "TransactionByHash", "TransactionReceipt", "ValidateNetwork"}

var scenarioCatalogChains = []struct {
	chain   string
	methods []string
	raw     string
}{
	{"ethereum", scenarioEVMMethods, "jsonrpc.RawCall"},
	{"bnb", scenarioEVMMethods, ""},
	{"bitcoin", []string{"EstimateSmartFee", "GetBlock", "GetBlockCount", "GetBlockHash", "GetBlockchainInfo", "GetMempoolEntry", "GetRawTransaction", "GetTxOut", "SendRawTransaction", "TestMempoolAccept"}, ""},
	{"solana", []string{"GetAccountInfoWithOpts", "GetBalanceWithOpts", "GetBlockWithOpts", "GetBlocks", "GetFeeForMessageWithOpts", "GetFirstAvailableBlock", "GetGenesisHash", "GetLatestBlockhashWithOpts", "GetSignatureStatuses", "GetSlotWithOpts", "GetTransaction", "SendEncodedTransactionWithOpts", "SimulateRawTransactionWithOpts"}, ""},
	{"tron", []string{"BroadcastTransaction", "EstimateEnergy", "GetAccount", "GetAccountResource", "GetBlockByNum", "GetChainParameters", "GetNodeInfo", "GetNowBlock", "GetSolidifiedAccount", "GetSolidifiedBlockByNum", "GetSolidifiedNowBlock", "GetSolidifiedTransactionByID", "GetSolidifiedTransactionInfoByID", "GetTransactionByID", "GetTransactionInfoByID", "TriggerConstantContract"}, "httpjson.RawPost"},
}

var scenarioSubmitMethods = map[string]bool{"SendRawTransaction": true, "SendEncodedTransactionWithOpts": true, "BroadcastTransaction": true}

func scenarioCatalog() []scenarioEntry {
	var entries []scenarioEntry
	for _, c := range scenarioCatalogChains {
		for _, m := range c.methods {
			kind, status := "read", "NOT_RUN"
			if scenarioSubmitMethods[m] {
				kind, status = "submit", "NOT_AUTHORIZED"
			}
			entries = append(entries, scenarioEntry{ID: c.chain + "/" + m, Chain: c.chain, Method: m, Kind: kind, Status: status})
		}
		if c.raw != "" {
			entries = append(entries, scenarioEntry{ID: c.chain + "/" + c.raw, Chain: c.chain, Method: c.raw, Kind: "raw", Status: "NOT_RUN"})
		}
	}
	return entries
}

// summarizeScenario ranks FAIL > BLOCKED > NOT_RUN > PASS; a FAIL counts on any chain.
func summarizeScenario(requiredChains []string, results map[string]string) string {
	for _, status := range results {
		if status == "FAIL" {
			return "FAIL"
		}
	}
	if len(requiredChains) == 0 {
		return "BLOCKED"
	}
	required := map[string]bool{}
	for _, c := range requiredChains {
		if !isScenarioChain(c) {
			return "BLOCKED"
		}
		required[c] = true
	}
	summary := "PASS"
	for _, e := range scenarioCatalog() {
		if e.Kind == "submit" || !required[e.Chain] {
			continue
		}
		switch results[e.ID] {
		case "PASS":
		case "", "NOT_RUN":
			if summary == "PASS" {
				summary = "NOT_RUN"
			}
		default:
			summary = "BLOCKED"
		}
	}
	return summary
}

func runScenario(ctx context.Context, deps scenarioDeps) scenarioReport {
	if v, _ := deps.LookupEnv(scenarioSwitch); v != "1" {
		return scenarioReport{Status: "NOT_RUN", Category: "disabled"}
	}
	raw, err := deps.ReadFile(scenarioConfigPath)
	if errors.Is(err, fs.ErrNotExist) {
		return scenarioReport{Status: "BLOCKED", Category: "config_missing"}
	}
	if err != nil {
		return scenarioReport{Status: "BLOCKED", Category: "config_read"}
	}
	cfg, err := parseScenarioConfig(raw)
	if errors.Is(err, errScenarioYAML) {
		return scenarioReport{Status: "BLOCKED", Category: "config_yaml"}
	}
	if err != nil {
		return scenarioReport{Status: "BLOCKED", Category: "config_invalid"}
	}

	ctx, cancel := context.WithDeadline(ctx, deps.Now().Add(cfg.TotalTimeout))
	defer cancel()
	catalog := scenarioCatalog()
	required := map[string]bool{}
	for _, c := range cfg.RequiredChains {
		required[c] = true
	}
	results := map[string]string{}
	requests := 0
	for _, chain := range scenarioChains {
		run, wired := deps.Runners[chain]
		if !required[chain] || !wired {
			continue
		}
		chainCfg := cfg.Chains[chain]
		chainCfg.RequestTimeout = cfg.RequestTimeout
		if chainCfg.RequestTimeout == 0 {
			chainCfg.RequestTimeout = cfg.TotalTimeout
		}
		doer, release, err := deps.NewDoer(chainCfg)
		if err != nil {
			for _, e := range catalog {
				if e.Chain == chain && e.Kind != "submit" {
					results[e.ID] = "BLOCKED"
				}
			}
			continue
		}
		budget := newScenarioBudget(doer, cfg.MaxRequests-requests, cfg.MinInterval, deps.Wait)
		got := run(ctx, cfg, chainCfg, budget)
		if release != nil {
			release()
		}
		requests += budget.Requests()
		for _, e := range catalog {
			if status, ok := got[e.ID]; ok && e.Chain == chain && e.Kind != "submit" {
				results[e.ID] = status
			}
		}
	}

	for i := range catalog {
		if status, ok := results[catalog[i].ID]; ok {
			catalog[i].Status = status
		}
	}
	report := scenarioReport{Requests: requests, RequiredChains: cfg.RequiredChains, Entries: catalog}
	switch summarizeScenario(cfg.RequiredChains, results) {
	case "PASS":
		report.Status = "PASS"
	case "FAIL":
		report.Status, report.Category = "FAIL", "chain_result"
	case "NOT_RUN":
		report.Status, report.Category = "BLOCKED", "not_run"
	default:
		report.Status, report.Category = "BLOCKED", "chain_result"
	}
	return report
}

func scenarioOutcome(r scenarioReport) string {
	switch r.Status {
	case "NOT_RUN":
		return "SKIP"
	case "PASS":
		return "PASS"
	default:
		return "FAIL"
	}
}
