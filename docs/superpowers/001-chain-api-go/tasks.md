# chain-api-go 任务清单 / chain-api-go Tasks

**规格文档 / Spec**：[spec.md](spec.md)  
**技术方案 / Plan**：[plan.md](plan.md)  
**任务清单 / Tasks**：[tasks.md](tasks.md)  
**基线版本 / Baseline Version**：PLAN-v6+CONTEXT-CAUSE-v1+HARNESS-CONTRACT-v1+EVM-CONTRACT-v1
**日期 / Date**：2026-11-16  
**状态 / Status**：已批准；T025 已通过，T026 可启动（Approved; T025 Passed / T026 May Start）

## 0. 变更记录 / Change Log

| 版本 / Version | 日期 / Date | 变更 / Change |
|---|---|---|
| EVM-CONTRACT-v1 | 2026-11-16 | Human 已批准：仅细化 T025/T026 的 EC-01–05 契约、T025 原十路径与声明预警；保持 5 Plan / 20 Task、全部 ID/依赖及 T017–T024 内容不变。T025 可恢复，T026 未启动。 |
| HARNESS-CONTRACT-v1 | 2026-11-16 | Human 已批准：冻结 test-only live harness 的显式配置、raw-byte SHA-256、可取消并发等待、总 deadline context、结果优先级与离线并发验证；只重开 T023/T024，保持 5 Plan / 20 Task、任务 ID 与依赖不变。T017–T022 的未受影响证据保持有效。 |
| CONTEXT-CAUSE-v1 | 2026-11-16 | Human 批准在 T017/T018 既有路径内增加 `Error.WithContextCause(error) Error` 的测试与实现；T017/T018 重新打开，T019 已批准证据保持有效。Plan/Task 总数不变。 |
| PLAN-v6 | 2026-11-16 | 将 v5 的 T001–T016 标记为被替代的历史 ID；活动任务改为 T017–T036，共 5 Plan / 20 Task。每个行为实现均依赖更早的失败测试 Task；真实 RPC 与受控提交集中到 T036。Human 已明确批准本修订。 |

历史映射：T001→T017/T018；T002→T019/T020；T003→T021/T022；T004→T023/T024；T005/T006→T025/T026；T008→T027/T028；T010→T029/T030；T012/T013→T031/T032；T007/T009/T011/T014 的离线 runner→T033/T034；其真实验收部分→T036；T015→T035；T016→T036。旧 ID 不再是活动 checkbox，不得执行或复用。

## 1. 任务格式与共同契约 / Task Format and Common Contract

- 活动任务使用 `[ ] T###`；总数精确 20。默认串行执行，不自动授权并行 Agent。
- 测试 Task 必须运行到断言并因目标行为缺失产生 RED；缺 module、编译符号、工具、依赖或网络不算 RED。
- 实现 Task 必须使对应先行测试 GREEN，不得修改或弱化受保护断言。
- 声明骨架仅可定义测试编译所需的已批准类型/签名和显式未实现分支，不得实现协议、网络、解析或链行为。
- 每项 Scope 是默认拒绝列表；花括号按显式文件展开，不代表目录递归写权限。未列路径禁止修改。
- 每项继承 spec C-01–10、constitution 1.1、plan §7.4 Task Contract；新增依赖、公开 API、配置、路径、后台行为或例外必须重新审批。
- 预算格式为总小时；规划/编码/验证按 40%/5%/55% 核算。超过 400 生产行、1,200 测试行或 8 个主要路径是风险预警，不是机械拆分依据。

## 2. PLAN-001 公共基础 / Shared Foundation

### Tests First / 测试先行

- [x] **T017 核心 DI 与安全错误测试 / Test core DI and safe errors**
  - Scope：`apierror/error_test.go`；另允许 `go.mod`、`doc.go`、`transport/http.go`、`apierror/error.go` 仅建立声明骨架。
  - Dependencies：none。
  - Acceptance：AC-01、AC-04、AC-12、AC-13；覆盖安全类别、context 哨兵、常量转换、`RPCCode + HasRPCCode`、值类型 errors.As/Unwrap、敏感字段不存在、`WithContextCause` 的取消优先/清空/不保留包装错误，以及唯一标准库 `HTTPDoer` 指针签名的编译契约。
  - Verification：`go test ./apierror`，产生目标行为缺失的 RED。
  - Budget：2h；≤60 生产行、≤200 测试行、5 文件。
  - Done when：RED 可归因于缺失行为，非编译/环境失败。

### Implementation / 实现

- [x] **T018 实现核心 DI 与安全错误 / Implement core DI and safe errors**
  - Scope：`apierror/error.go`、`transport/http.go`、`doc.go`、`go.mod`。
  - Dependencies：T017。
  - Acceptance：AC-01、AC-04、AC-12、AC-13；实现批准的 `WithContextCause` 值方法，无第三方依赖，T017 GREEN。
  - Verification：`go test ./apierror`；`go list ./...`。
  - Budget：2h；≤160 生产行、0 新测试行、4 文件。
  - Done when：聚焦命令通过且 Diff 未越权。

### Tests First / 测试先行

- [x] **T019 JSON-RPC 协议测试 / Test JSON-RPC contracts**
  - Scope：`jsonrpc/client_test.go`、`jsonrpc/protocol_test.go`；允许 `jsonrpc/client.go` 声明骨架。
  - Dependencies：T018。
  - Acceptance：AC-02–AC-04、AC-12；覆盖 nil/typed-nil、V2/Bitcoin 方言、ID、信封、限长、取消、body、并发和单次请求。
  - Verification：`go test ./jsonrpc`，产生目标行为缺失的 RED。
  - Budget：4h；≤60 生产行、≤700 测试行、3 文件。
  - Done when：每个批准风险均有可归因 RED。

### Implementation / 实现

- [x] **T020 实现 JSON-RPC / Implement JSON-RPC**
  - Scope：`jsonrpc/client.go`、`jsonrpc/protocol.go`。
  - Dependencies：T019。
  - Acceptance：AC-02–AC-04、AC-12；实现 New/Call/RawCall，T019 GREEN。
  - Verification：`go test -race ./jsonrpc`。
  - Budget：6h；约450生产行、0新测试行、2文件；超400行预警已纳入本基线审批。
  - Done when：聚焦 race 检查通过，无隐藏重试或默认客户端。

### Tests First / 测试先行

- [x] **T021 HTTP JSON 协议测试 / Test HTTP JSON contracts**
  - Scope：`httpjson/client_test.go`；允许 `httpjson/client.go` 声明骨架。
  - Dependencies：T018。
  - Acceptance：AC-02–AC-04、AC-12；覆盖 nil/typed-nil、同源及 prefix、路径穿越、HTTP/JSON 错误、取消和 body。
  - Verification：`go test ./httpjson`，产生目标行为缺失的 RED。
  - Budget：3h；≤40生产行、≤450测试行、2文件。
  - Done when：边界和资源风险各有可归因 RED。

### Implementation / 实现

- [x] **T022 实现 HTTP JSON / Implement HTTP JSON**
  - Scope：`httpjson/client.go`。
  - Dependencies：T021。
  - Acceptance：AC-02–AC-04、AC-12；实现 New/Post/RawPost，T021 GREEN。
  - Verification：`go test ./httpjson`。
  - Budget：4h；≤350生产行、0新测试行、1文件。
  - Done when：同源路径、限长、context 与 body 契约通过。

### Tests First / 测试先行

- [x] **T023 live harness 基础测试 / Test live harness foundation**
  - Scope：`internal/test/live/harness_test.go`、`internal/test/live/config_test.go`、`internal/test/live/manifest_test.go`；允许 `internal/test/live/harness.go`、`internal/test/live/config.go`、`internal/test/live/manifest.go` 仅建立 plan §7.2 已批准接口的声明骨架；允许 `internal/test/live/config.example.json` 仅建立不含秘密的合法 JSON 占位 `{}`，使模板测试进入目标内容断言而非文件缺失错误。不得实现 loader、解析、摘要、限流或账本行为。
  - Dependencies：T020、T022。
  - Acceptance：AC-09，继承 AC-04/C-02/05/06/08/09；覆盖 plan §7.2 HC-01–HC-08：关闭开关零配置/材料读取，显式路径与 Header env 引用及缺失 env，模板无秘密，严格 JSON 与 raw-byte SHA-256，独立必需目录和状态优先级，请求预算，真实多 goroutine 并发上限/等待/取消，速率，实际 context deadline、父取消和资源释放，错误脱敏及 callback 不能覆盖基础错误。
  - Verification：`go test ./internal/test/live -run '^TestHarness' -count=1`；`go test -race ./internal/test/live -run '^TestHarness' -count=1`。两者必须完成编译并进入目标断言；目标行为缺失产生 RED，测试自身不得产生 race。缺文件、编译、环境或死锁不是合格 RED。
  - Budget：最多 70 分钟；规划 20 分钟、编码 20 分钟、验证最多 30 分钟。≤100 生产声明骨架行、≤900 测试行、≤10 模板占位行、7 文件；行数仅为范围预警，不是目标额度或延长工期的依据。
  - Done when：每项 HC 风险有可追踪的确定性断言，缺失行为 RED 绑定当前七文件、依赖、环境和配置 fixture 快照；固定 Auditor 只读审核通过。旧 T023 证据标为被替代；未经新门禁不得进入 T024。

### Implementation / 实现

- [x] **T024 实现 live harness 基础 / Implement live harness foundation**
  - Scope：`internal/test/live/config.go`、`manifest.go`、`harness.go`、`config.example.json`。
  - Dependencies：T023。
  - Acceptance：AC-09，继承 AC-04/C-02/05/06/08/09；实现 plan §7.2 HC-01–HC-08，使经审核 T023 测试 GREEN；模板仅 endpoint/Header 环境变量引用和无秘密元数据。无第三方依赖、默认路径、默认客户端、凭据、重试、后台 worker 或真实网络调用。
  - Verification：`go test -race ./internal/test/live -run '^TestHarness' -count=1`；`go vet ./internal/test/live`；检查四文件 Diff、核心依赖方向和无新增依赖。race 命令同时提供全部 Harness 测试 GREEN，不重复运行未受影响 SDK 测试。
  - Budget：最多 70 分钟；规划 20 分钟、编码 20 分钟、验证最多 30 分钟。≤600 生产行、0 新测试行、≤100 模板行、4 文件；预计实现应明显低于上限，行数只触发范围复核，不得据此延长工期或增加抽象。
  - Done when：受保护 T023 断言全部 GREEN，race/vet 通过；普通测试零公网、关闭开关零外部读取，正式缺项不能 PASS；无资源泄露、敏感输出或 Scope 越界；当前快照证据经固定 Auditor 审核通过，依既有 Human 门禁标记完成。

## 3. PLAN-002 Ethereum 与 BNB / Ethereum and BNB

### Tests First / 测试先行

- [x] **T025 完整 EVM 与两具体客户端测试 / Test complete EVM and concrete clients**
  - Scope：恰好十路径：`evm/types_test.go`、`evm/query_test.go`、`evm/compute_test.go`、`evm/submit_test.go`、`ethereum/client_test.go`、`bnb/client_test.go`；仅声明骨架 `evm/types.go`、`evm/client.go`、`ethereum/client.go`、`bnb/client.go`。骨架允许 plan §3.2 的完整字段/签名（含 evm.New），仅返回 Unsupported/零值，不实现 T026 行为。
  - Dependencies：T024。
  - Acceptance：AC-02、AC-05、AC-12、AC-13；精确覆盖 spec §4.2 EC-01–05 与 plan §3.2：公开构造 seam、完整首版 DTO 字段/必需键/null、四个输入金额及全部输出金额的所有权、嵌套 wire 与 FeeHistory NaN/±Inf/维度、两链独立 1/56 身份、非零提交 hash；保留 Has/费用互斥、nullable 三态、unknown 与零公网风险。
  - Verification：`go test ./evm ./ethereum ./bnb`，产生目标行为缺失的 RED。
  - Budget：最多 70 分钟：规划 20 分钟、Coder 20 分钟、验证与固定 Auditor 最多 30 分钟。声明骨架估计 200–240 行、≤240 行预警（当前 151 行，完整字段/签名使原 100 行失实）；测试优先去重保持 ≤1,100 行，十路径不变。行数不是目标，不据此扩大范围或延长时间。
  - Done when：十文件编译并进入目标断言 RED，两链分别有可归因证据；EC 风险逐项绑定当前快照，固定 Auditor 只读审核通过后停在 T025 Human 门禁。旧 T025 RED 为历史；批准本契约不等于 T025 完成，不启动 T026。

### Implementation / 实现

- [ ] **T026 实现完整 EVM 与两具体客户端 / Implement complete EVM and concrete clients**
  - Scope：`evm/client.go`、`types.go`、`query.go`、`compute.go`、`submit.go`、`ethereum/client.go`、`bnb/client.go`。
  - Dependencies：T025。
  - Acceptance：AC-02、AC-05、AC-12、AC-13；实现经 Human 批准后的 spec EC-01–05/plan §3.2 契约，使新 T025 全部断言实际执行并 GREEN；当前未启动，必须先通过 T025 的 RED/固定 Auditor/Human 门禁。
  - Verification：`go test ./evm ./ethereum ./bnb`。
  - Budget：10h；约1,000生产行、0新测试行、7文件；规模预警已纳入审批。
  - Done when：两具体客户端独立通过且无统一分发。

## 4. PLAN-003 Bitcoin

### Tests First / 测试先行

- [ ] **T027 完整 Bitcoin 客户端测试 / Test complete Bitcoin client**
  - Scope：`bitcoin/client_test.go`、`types_test.go`、`query_test.go`、`compute_test.go`、`submit_test.go`；允许 `client.go`、`types.go` 声明骨架。
  - Dependencies：T026。
  - Acceptance：AC-02、AC-06、AC-12、AC-13；覆盖精确 satoshi、脚本/见证、历史前提、null/present 三态、费用预检、载荷和 unknown。
  - Verification：`go test ./bitcoin`，产生目标行为缺失的 RED。
  - Budget：4h；≤80生产行、≤800测试行、7文件。
  - Done when：精度、历史和提交风险均有可归因 RED。

### Implementation / 实现

- [ ] **T028 实现完整 Bitcoin 客户端 / Implement complete Bitcoin client**
  - Scope：`bitcoin/client.go`、`types.go`、`query.go`、`compute.go`、`submit.go`。
  - Dependencies：T027。
  - Acceptance：AC-02、AC-06、AC-12、AC-13；T027 GREEN。
  - Verification：`go test ./bitcoin`。
  - Budget：7h；约700生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：金额无浮点损失、历史限制明确、提交单次调用。

## 5. PLAN-004 Solana

### Tests First / 测试先行

- [ ] **T029 完整 Solana 客户端测试 / Test complete Solana client**
  - Scope：`solana/client_test.go`、`types_test.go`、`query_test.go`、`compute_test.go`、`submit_test.go`；允许 `client.go`、`types.go` 声明骨架。
  - Dependencies：T028。
  - Acceptance：AC-02、AC-07、AC-12、AC-13；覆盖编码、legacy/v0、context、commitment、meta、有效高度、nullable present 三态、模拟和提交选项。
  - Verification：`go test ./solana`，产生目标行为缺失的 RED。
  - Budget：4h；≤80生产行、≤900测试行、7文件。
  - Done when：版本、context 和动态状态风险均有可归因 RED。

### Implementation / 实现

- [ ] **T030 实现完整 Solana 客户端 / Implement complete Solana client**
  - Scope：`solana/client.go`、`types.go`、`query.go`、`compute.go`、`submit.go`。
  - Dependencies：T029。
  - Acceptance：AC-02、AC-07、AC-12、AC-13；T029 GREEN。
  - Verification：`go test ./solana`。
  - Budget：7h；约800生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：commitment、版本、meta 和有效高度均保留。

## 6. PLAN-005 Tron 与最终集成 / Tron and Final Integration

### Tests First / 测试先行

- [ ] **T031 完整 Tron 客户端测试 / Test complete Tron client**
  - Scope：`tron/client_test.go`、`types_test.go`、`query_test.go`、`compute_test.go`、`submit_test.go`；允许 `client.go`、`types.go` 声明骨架。
  - Dependencies：T030。
  - Acceptance：AC-02、AC-08、AC-12、AC-13；覆盖地址/visible、空账户与零余额的 present 三态、资源单位、13条明确查询路径、五个 solidified 查询、计算、无损交易 JSON、HTTP 200 拒绝和 unknown。
  - Verification：`go test ./tron`，产生目标行为缺失的 RED。
  - Budget：5h；≤80生产行、≤1,000测试行、7文件。
  - Done when：地址、状态和提交风险均有可归因 RED。

### Implementation / 实现

- [ ] **T032 实现完整 Tron 客户端 / Implement complete Tron client**
  - Scope：`tron/client.go`、`types.go`、`query.go`、`compute.go`、`submit.go`。
  - Dependencies：T031。
  - Acceptance：AC-02、AC-08、AC-12、AC-13；T031 GREEN。
  - Verification：`go test ./tron`。
  - Budget：8h；约850生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：链错误、solidified 状态和交易 JSON 保持原义。

### Tests First / 测试先行

- [ ] **T033 五链 live runner 离线契约测试 / Test offline contracts for five-chain live runners**
  - Scope：`internal/test/live/read_contract_test.go`、`submission_contract_test.go`；允许 `evm.go`、`bitcoin.go`、`solana.go`、`tron.go` 声明骨架。
  - Dependencies：T024、T026、T028、T030、T032。
  - Acceptance：AC-09–AC-11；合成 fixture 覆盖五链字段、历史/动态分类、独立 preflight、漏跑、双开关和提交材料门禁。
  - Verification：`go test ./internal/test/live -run '^(TestReadContract|TestSubmissionContract)'`，产生目标行为缺失的 RED。
  - Budget：5h；≤80生产行、≤1,000测试行、6文件。
  - Done when：测试不访问公网且所有正式门禁均有可归因 RED。

### Implementation / 实现

- [ ] **T034 实现五链 live runner / Implement five-chain live runners**
  - Scope：`internal/test/live/evm.go`、`bitcoin.go`、`solana.go`、`tron.go`、`live_test.go`。
  - Dependencies：T033。
  - Acceptance：AC-09–AC-11 的离线契约；建立 TestLiveRead/TestLiveBroadcast，本 Task 不采样、不读真实凭据、不联网。
  - Verification：`go test ./internal/test/live -run '^(TestReadContract|TestSubmissionContract)'`。
  - Budget：8h；约800生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：T033 GREEN，真实证据仍明确 deferred。

### Documentation and Offline Verification / 文档与离线核验

- [ ] **T035 文档、治理入口与离线交付核验 / Document and verify the offline delivery**
  - Scope：`README.md`、`AGENTS.md`、`doc.go`、`transport/doc.go`、`apierror/doc.go`、`jsonrpc/doc.go`、`httpjson/doc.go`、`evm/doc.go`、`ethereum/doc.go`、`bnb/doc.go`、`bitcoin/doc.go`、`solana/doc.go`、`tron/doc.go`、`internal/test/live/evidence/offline.md`；不得修改 constitution。
  - Dependencies：T034。
  - Acceptance：AC-01–AC-09、AC-12、AC-13；核对方法、单位、nullable、DI、Raw、提交边界、依赖图、8.5/8.6 和核心构造签名隔离。
  - Verification：`go vet ./...`；`go list -deps ./...`；API/范围核查；只补跑 stale 聚焦证据。
  - Budget：4h；0生产/测试行，约600文档行，14路径单元；路径预警已纳入审批。
  - Done when：有效证据汇总完成，固定 Auditor 可获得完整只读审核包。

### Final Acceptance / 最终验收

- [ ] **T036 五链最终真实验收、最终审计与 Human 门禁 / Run final five-chain acceptance, audit, and Human gate**
  - Scope：`internal/test/live/testdata/ethereum/samples.json`、`bnb/samples.json`、`bitcoin/samples.json`、`solana/samples.json`、`tron/samples.json`、`internal/test/live/evidence/read.json`、`broadcast.json`、`final.md`；外部配置与交易载荷只读。
  - Dependencies：T035，且全部子 Plan 已完成离线验证、审核和批准。
  - Acceptance：AC-10、AC-11；浏览器采样、独立账号 RPC preflight、SDK 字段核对、独立授权提交及按原标识查询。禁止修改 SDK/runner。
  - Verification：`CHAIN_API_GO_LIVE_TEST=1 go test ./internal/test/live -run '^TestLiveRead$' -count=1`；双开关广播命令见 plan §8。
  - Budget：8h；0生产/测试行，约800证据行，8文件。
  - Done when：必需项全部 PASS、固定 Auditor 给出通过建议且 Human 最终批准；缺权限或材料为 BLOCKED。

## 7. 最终验证清单 / Final Validation Checklist

- [ ] 测试 Task 均早于并被对应实现 Task 依赖；RED/GREEN 证据绑定当前快照。
- [ ] T017–T036 依赖无环、Scope 未越界、旧 T001–T016 未被执行或复用。
- [ ] spec C-01–10、AC-01–13、constitution 1.1 保持完整；8.5/8.6 和核心构造签名隔离通过审核。
- [ ] 每个 Plan 的聚焦验证完成；真实数据、外部 RPC 与链上提交仅在 T036 去重执行。
- [ ] 未经批准不提交、不推送、不合并；集成阶段只做结构检查。

Human 已批准 PLAN-v6+CONTEXT-CAUSE-v1+HARNESS-CONTRACT-v1+EVM-CONTRACT-v1 与 T025；T026 可进入 GREEN/Refactor。T017–T025 的有效证据保留；活动清单仍为 5 Plan / 20 Task，全部 Task ID 与依赖不变。尚无真实 RPC 或交易提交授权。
