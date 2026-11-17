# chain-api-go 任务清单 / chain-api-go Tasks

**规格文档 / Spec**：[spec.md](spec.md)  
**技术方案 / Plan**：[plan.md](plan.md)  
**任务清单 / Tasks**：[tasks.md](tasks.md)  
**基线版本 / Baseline Version**：PLAN-v6+CONTEXT-CAUSE-v1+HARNESS-CONTRACT-v1+EVM-CONTRACT-v1+BITCOIN-CONTRACT-v1+EVM-RAW-DTO-v1+RAW-ERROR-v1
**日期 / Date**：2026-11-17
**状态 / Status**：T017/T018 已通过 RED、Auditor 与 Human 门禁并获准集成；T019 及以后任务未授权（T017/T018 Complete）

## 0. 变更记录 / Change Log

| 版本 / Version | 日期 / Date | 变更 / Change |
|---|---|---|
| RAW-ERROR-v1 | 2026-11-17 | Human 已批准：恢复原始节点 Message/Data、HTTP 错误正文及底层 Cause；区分 SDK 返回与 live 脱敏输出。重开 T017–T022，修订 T025/T026；T025 当前 RED/审核不能作为新基线门禁。T023/T024 未受影响证据保留，SDK→harness 输出隔离在 T033/T034 验证。保持 5 Plan / 20 Task、T017–T036 ID 与原路径；T027 继续暂停。constitution 1.3 同时获批；T017/T018 已完成门禁并获准集成，T019 未授权。 |
| EVM-RAW-DTO-v1 | 2026-11-17 | Human 已批准：将 EVM 收敛为原始 RPC DTO + 必要转换，接受破坏性 API 变更且不提供兼容层；重开 T025/T026，旧证据 stale。T025 可进入 RED，T026 仍等待审核和 Human 门禁；Bitcoin 候选暂停。保持 5 Plan / 20 Task、路径和依赖，不改 constitution。 |
| BITCOIN-CONTRACT-v1 | 2026-11-17 | 候选：补齐 PLAN-003 十方法、DTO、金额/费率、历史和 wire 契约；T027 暂停待批准。只改三件套；保持 5 Plan / 20 Task、全部依赖、T027 七路径、T028 五路径和 T017–T026 有效证据。 |
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
- 每项继承 spec C-01–10、Human 批准后的 constitution 1.3 及 plan §7.4 Task Contract；新增依赖、公开 API、配置、路径、后台行为或例外必须重新审批。
- 未修订任务保留原预算；EVM-RAW-DTO-v1/T025–T026 使用 plan §3.2 的共享 20/20/30 分钟时箱，不按 40%/5%/55% 换算。行数及路径数量只作风险预警。
- RAW-ERROR-v1 错误增量在 T017–T022 和 T025/T026 间共享规划 20 分钟、生产编码 20 分钟、测试执行与独立审核 30 分钟，不按 Task 重置；T023/T024 不重开，T027 继续暂停。

## 2. PLAN-001 公共基础 / Shared Foundation

### Tests First / 测试先行

- [x] **T017 核心 DI 与原始错误测试 / Test core DI and raw errors**
  - Scope：`apierror/error_test.go`；仅允许 `apierror/error.go` 增加测试编译所需的已批准字段声明，不改 Error/Unwrap/WithContextCause 行为；其他原授权路径不得无关修改。
  - Dependencies：RAW-ERROR-v1 四文件及 constitution 1.3 已获 Human 批准。
  - Acceptance：AC-01、AC-04、AC-12、AC-13，RE-01/02/04/06；撤换字段禁止、私有 cause、取消优先归一化及 Error 全格式脱敏断言。覆盖 Message/Data/ResponseBody/Cause 精确字段、`RPCCode + HasRPCCode`、值类型 `errors.As`、原始 `errors.Is/As` 链、Message/Cause 格式优先级、值副本和 `Kind.ToString()`；保留唯一标准库 `HTTPDoer` 指针签名编译契约。不得以缺符号、Error 不可比较或环境失败作为 RED。
  - Verification：`go test ./apierror`，产生目标行为缺失的 RED。
  - Budget：计入 RAW-ERROR-v1 共享 20/20/30 时箱；≤30 声明行、约60–100测试增量行、2文件。
  - Done when：RED 可归因于缺失行为，非编译/环境失败。

### Implementation / 实现

- [x] **T018 实现核心 DI 与原始错误 / Implement core DI and raw errors**
  - Scope：`apierror/error.go`；其他原路径不得无关修改。
  - Dependencies：T017。
  - Acceptance：AC-01、AC-04、AC-12、AC-13，RE-01/02/04/06；实现已批准字段、Error/Unwrap 与 `WithContextCause` 原样设置 Cause 的值副本语义，无第三方依赖，T017 GREEN。
  - Verification：`go test ./apierror`；`go list ./...`。
  - Budget：计入 RAW-ERROR-v1 共享时箱；约30–60生产增量行、0新测试行、1文件。
  - Done when：聚焦命令通过且 Diff 未越权。

### Tests First / 测试先行

- [ ] **T019 JSON-RPC 原始错误协议测试 / Test JSON-RPC raw-error contracts**
  - Scope：`jsonrpc/client_test.go`、`jsonrpc/protocol_test.go`；不修改生产文件。
  - Dependencies：T018。
  - Acceptance：AC-02–AC-04、AC-12，RE-01–06；在保留 nil/typed-nil、V2/Bitcoin 方言、ID、信封、限长、取消、body、并发和单次请求风险的基础上，增加 code=0、Message 原样、Data 缺失/null/大数字/对象、HTTP/RPC 正文、截断标志、部分读错误、原始 Cause/context Is/As 及“不主动附加请求秘密”。
  - Verification：`go test ./jsonrpc`，产生目标行为缺失的 RED。
  - Budget：计入 RAW-ERROR-v1 共享时箱；0生产行、约100–180测试增量行、2文件。
  - Done when：RAW-ERROR-v1 新增或改变的目标行为具有可归因 RED；未改变的既有断言保留，不要求重新产生 RED。

### Implementation / 实现

- [ ] **T020 实现 JSON-RPC 原始错误保留 / Implement JSON-RPC raw-error preservation**
  - Scope：`jsonrpc/client.go`、`jsonrpc/protocol.go`。
  - Dependencies：T019。
  - Acceptance：AC-02–AC-04、AC-12，RE-01–06；实现 New/Call/RawCall 的原始节点字段、受限正文与完整 Cause 保留，T019 GREEN。
  - Verification：`go test -race ./jsonrpc`。
  - Budget：计入 RAW-ERROR-v1 共享时箱；约60–110生产增量行、0新测试行、2文件。
  - Done when：聚焦 race 检查通过，无隐藏重试或默认客户端。

### Tests First / 测试先行

- [ ] **T021 HTTP JSON 原始错误协议测试 / Test HTTP JSON raw-error contracts**
  - Scope：`httpjson/client_test.go`；不修改生产文件。
  - Dependencies：T018。
  - Acceptance：AC-02–AC-04、AC-12，RE-02–06；保留 nil/typed-nil、同源及 prefix、路径穿越、取消和 body 风险，增加非 JSON HTTP 正文、刚好限长/超限、部分读取、原始 Cause/context Is/As 及“不主动附加请求秘密”。默认串行在 T020 后执行。
  - Verification：`go test ./httpjson`，产生目标行为缺失的 RED。
  - Budget：计入 RAW-ERROR-v1 共享时箱；0生产行、约60–120测试增量行、1文件。
  - Done when：RAW-ERROR-v1 新增或改变的目标行为具有可归因 RED；未改变的路径与资源断言保留，不要求重新产生 RED。

### Implementation / 实现

- [ ] **T022 实现 HTTP JSON 原始错误保留 / Implement HTTP JSON raw-error preservation**
  - Scope：`httpjson/client.go`。
  - Dependencies：T021。
  - Acceptance：AC-02–AC-04、AC-12，RE-02–06；实现 New/Post/RawPost 的受限错误正文与完整 Cause 保留，T021 GREEN。
  - Verification：`go test ./httpjson`。
  - Budget：计入 RAW-ERROR-v1 共享时箱；约30–50生产增量行、0新测试行、1文件。
  - Done when：同源路径、限长、context 与 body 契约通过。

### Tests First / 测试先行

- [x] **T023 live harness 基础测试 / Test live harness foundation**
  - Scope：`internal/test/live/harness_test.go`、`internal/test/live/config_test.go`、`internal/test/live/manifest_test.go`；允许 `internal/test/live/harness.go`、`internal/test/live/config.go`、`internal/test/live/manifest.go` 仅建立 plan §7.2 已批准接口的声明骨架；允许 `internal/test/live/config.example.json` 仅建立不含秘密的合法 JSON 占位 `{}`，使模板测试进入目标内容断言而非文件缺失错误。不得实现 loader、解析、摘要、限流或账本行为。
  - Dependencies：T020、T022。
  - Acceptance：AC-09，继承 AC-04/C-02/05/06/08/09；覆盖 plan §7.2 HC-01–HC-08：关闭开关零配置/材料读取，显式路径与 Header env 引用及缺失 env，模板无秘密，严格 JSON 与 raw-byte SHA-256，独立必需目录和状态优先级，请求预算，真实多 goroutine 并发上限/等待/取消，速率，实际 context deadline、父取消和资源释放，错误脱敏及 callback 不能覆盖基础错误。
  - Verification：`go test ./internal/test/live -run '^TestHarness' -count=1`；`go test -race ./internal/test/live -run '^TestHarness' -count=1`。两者必须完成编译并进入目标断言；目标行为缺失产生 RED，测试自身不得产生 race。缺文件、编译、环境或死锁不是合格 RED。
  - Budget：最多 70 分钟；规划 20 分钟、编码 20 分钟、验证最多 30 分钟。≤100 生产声明骨架行、≤900 测试行、≤10 模板占位行、7 文件；行数仅为范围预警，不是目标额度或延长工期的依据。
  - Done when：每项 HC 风险已有可追踪的确定性断言，HARNESS-CONTRACT-v1 的 RED、固定 Auditor 与 Human 门禁已完成。其替代记录只针对更早历史证据；RAW-ERROR-v1 保留 T023/T024 已完成状态，不重开、不重新要求 RED。

### Implementation / 实现

- [x] **T024 实现 live harness 基础 / Implement live harness foundation**
  - Scope：`internal/test/live/config.go`、`manifest.go`、`harness.go`、`config.example.json`。
  - Dependencies：T023。
  - Acceptance：AC-09，继承 AC-04/C-02/05/06/08/09；实现 plan §7.2 HC-01–HC-08，使经审核 T023 测试 GREEN；模板仅 endpoint/Header 环境变量引用和无秘密元数据。无第三方依赖、默认路径、默认客户端、凭据、重试、后台 worker 或真实网络调用。
  - Verification：`go test -race ./internal/test/live -run '^TestHarness' -count=1`；`go vet ./internal/test/live`；检查四文件 Diff、核心依赖方向和无新增依赖。race 命令同时提供全部 Harness 测试 GREEN，不重复运行未受影响 SDK 测试。
  - Budget：最多 70 分钟；规划 20 分钟、编码 20 分钟、验证最多 30 分钟。≤600 生产行、0 新测试行、≤100 模板行、4 文件；预计实现应明显低于上限，行数只触发范围复核，不得据此延长工期或增加抽象。
  - Done when：受保护 T023 断言全部 GREEN，race/vet 通过；普通测试零公网、关闭开关零外部读取，正式缺项不能 PASS；无资源泄露、敏感输出或 Scope 越界；当前快照证据经固定 Auditor 审核通过，依既有 Human 门禁标记完成。

## 3. PLAN-002 Ethereum 与 BNB / Ethereum and BNB

T025/T026 曾完成并集成的记录保留为历史；EVM-RAW-DTO-v1 与 RAW-ERROR-v1 改变契约，二者当前状态 reopened。当前 T025 整包 RED/审核对新错误基线 stale；先完成 T017–T022，再重建 T025 受影响证据。数量仍为 5 Plan / 20 Task。

### Tests First / 测试先行

- [ ] **T025 重建原始 EVM DTO 契约测试 / Reopen raw EVM DTO contract tests**
  - Scope：沿用原十路径：`evm/types_test.go`、`evm/query_test.go`、`evm/compute_test.go`、`evm/submit_test.go`、`ethereum/client_test.go`、`bnb/client_test.go`、`evm/types.go`、`evm/client.go`、`ethereum/client.go`、`bnb/client.go`。本次仅修改前六测试文件，后四生产文件不得修改；不新增路径。
  - Dependencies：当前 T020、当前 T022、T024；EVM-RAW-DTO-v1 与 RAW-ERROR-v1 明确批准。
  - Acceptance：AC-02/04/05/12/13，RE-01/02/05–07，spec ER-01–05/plan §3.2；仅撤换明确列出的被替代断言，保留 DI/14 方法/nullable/单次提交/网络身份保护，并证明链包装保留原始节点字段与完整依赖错误链、不主动附加请求秘密。
  - Verification：`go test ./evm ./ethereum ./bnb`；旧实现编译成功，在原始字段/底层类型/新增 ToString 等目标断言处 RED。测试侧用 JSON fixture、有限反射及本地接口断言避免新 API 未定义导致编译失败。
  - Budget：与 T026 共享规划 20 分钟、编码 20 分钟、验证含审核最多 30 分钟；生产变更 0，六测试文件最终约 600–900 行仅为预警。
  - Done when：DTO 与原始错误新契约风险均有可追踪 RED，当前旧整包证据明确 stale，证据绑定当前代码/测试/依赖/配置/环境快照；独立只读 Auditor 后停在 T025 Human 门禁，不启动 T026。

### Implementation / 实现

- [ ] **T026 简化 EVM 与两具体客户端 / Simplify EVM and concrete clients**
  - Scope：`evm/client.go`、`evm/types.go`、`evm/query.go`、`evm/compute.go`、`evm/submit.go`、`ethereum/client.go`、`bnb/client.go`；不改测试、其他包、依赖或 constitution。
  - Dependencies：T025 的新 RED、独立审核及 Human 门禁。
  - Acceptance：AC-02/04/05/12/13，RE-01/02/05–07；严格实现 spec ER-01–05、RE-01–07 与 plan §3.2/§4，标准解码替换手写 schema，保留完整 Caller 错误链；全部 T025 断言实际执行并 GREEN。保留现有未提交修改的快照与来源，只按批准候选授权处理相关 EVM 行为。
  - Verification：`go test ./evm ./ethereum ./bnb`；`go vet ./evm ./ethereum ./bnb`；核对七路径、API 删除/变更及防御分支依据。只复验因修复而 stale 的检查。
  - Budget：使用 T025 所列共享 20/20/30 分钟时箱，不另计 10h 或叠加；七生产文件最终约 350–500 行仅为预警，0 新测试行。
  - Done when：T025 全部受保护断言 GREEN、vet 通过、无生产 schema/重复 context 防御、无越权；当前快照经独立 Auditor 和 Human 门禁。真实数据未验证，不自动恢复 T027。

## 4. PLAN-003 Bitcoin

BITCOIN-CONTRACT-v1 保留为未批准且暂停的候选，T027 不得实施。T027 仍依赖重新完成的 T026 及另行批准的 Bitcoin 契约；T028–T036 保持原依赖链阻塞。不得把旧 EVM 解析模式复制到其他链。

### Tests First / 测试先行

- [ ] **T027 完整 Bitcoin 客户端测试 / Test complete Bitcoin client**
  - Scope：恰好原七路径：`bitcoin/client_test.go`、`bitcoin/types_test.go`、`bitcoin/query_test.go`、`bitcoin/compute_test.go`、`bitcoin/submit_test.go`；仅声明骨架 `bitcoin/client.go`、`bitcoin/types.go`，只含plan §3.3完整类型/字段/签名与Unsupported/零值分支，不实现解析/校验/请求行为。
  - Dependencies：T026。
  - Acceptance：AC-02、AC-04、AC-06、AC-12、AC-13，RE-01/02/06/07；覆盖spec §4.3/§4.4、plan §3.3/§4 的十方法/DTO、精确金额费率、script/witness、历史blockhash/txindex、nullable、原始 errors/reject 字段、所有权、预检拒绝、单次提交unknown；按风险矩阵去重。BITCOIN-CONTRACT-v1 仍须另获批准。
  - Verification：`go test ./bitcoin`，产生目标行为缺失的 RED。
  - Budget：最多70分钟：规划20分钟、Coder20分钟、验证及固定Auditor最多30分钟。声明预计160–220行、测试650–900行；220/900仅现实预警，不是目标或延时依据，原7路径不变。
  - Done when：候选获Human批准后七文件编译并进入目标断言RED；BC-01–05证据绑定代码/测试/依赖/配置fixture/环境快照，经固定只读Auditor后停在T027 Human门禁。构造骨架遮蔽的细部断言须在T028实际执行，不得把候选视为T027完成。

### Implementation / 实现

- [ ] **T028 实现完整 Bitcoin 客户端 / Implement complete Bitcoin client**
  - Scope：`bitcoin/client.go`、`types.go`、`query.go`、`compute.go`、`submit.go`。
  - Dependencies：T027。
  - Acceptance：AC-02、AC-04、AC-06、AC-12、AC-13，RE-01/02/06/07；实现 Human 批准的 spec BC-01–05/RE-01–07 与 plan §3.3/§4，T027全部受保护断言实际执行且GREEN；先通过候选批准与T027 RED/固定Auditor/Human门禁。
  - Verification：`go test ./bitcoin`。
  - Budget：7h；约700生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：金额无浮点损失、历史限制明确、提交单次调用。

## 5. PLAN-004 Solana

### Tests First / 测试先行

- [ ] **T029 完整 Solana 客户端测试 / Test complete Solana client**
  - Scope：`solana/client_test.go`、`types_test.go`、`query_test.go`、`compute_test.go`、`submit_test.go`；允许 `client.go`、`types.go` 声明骨架。
  - Dependencies：T028。
  - Acceptance：AC-02、AC-04、AC-07、AC-12、AC-13，RE-01/02/06/07；覆盖编码、legacy/v0、context、commitment、meta、有效高度、nullable present 三态、模拟和提交选项，并保留 Solana RPC/preflight 原始 message/data/Cause。
  - Verification：`go test ./solana`，产生目标行为缺失的 RED。
  - Budget：4h；≤80生产行、≤900测试行、7文件。
  - Done when：版本、context 和动态状态风险均有可归因 RED。

### Implementation / 实现

- [ ] **T030 实现完整 Solana 客户端 / Implement complete Solana client**
  - Scope：`solana/client.go`、`types.go`、`query.go`、`compute.go`、`submit.go`。
  - Dependencies：T029。
  - Acceptance：AC-02、AC-04、AC-07、AC-12、AC-13，RE-01/02/06/07；T029 GREEN。
  - Verification：`go test ./solana`。
  - Budget：7h；约800生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：commitment、版本、meta 和有效高度均保留。

## 6. PLAN-005 Tron 与最终集成 / Tron and Final Integration

### Tests First / 测试先行

- [ ] **T031 完整 Tron 客户端测试 / Test complete Tron client**
  - Scope：`tron/client_test.go`、`types_test.go`、`query_test.go`、`compute_test.go`、`submit_test.go`；允许 `client.go`、`types.go` 声明骨架。
  - Dependencies：T030。
  - Acceptance：AC-02、AC-04、AC-08、AC-12、AC-13，RE-01–07；覆盖地址/visible、空账户与零余额的 present 三态、资源单位、13条明确查询路径、五个 solidified 查询、计算、无损交易 JSON、HTTP 200 拒绝原始 code/message/JSON、HTTP 正文/Cause 和 unknown。
  - Verification：`go test ./tron`，产生目标行为缺失的 RED。
  - Budget：5h；≤80生产行、≤1,000测试行、7文件。
  - Done when：地址、状态和提交风险均有可归因 RED。

### Implementation / 实现

- [ ] **T032 实现完整 Tron 客户端 / Implement complete Tron client**
  - Scope：`tron/client.go`、`types.go`、`query.go`、`compute.go`、`submit.go`。
  - Dependencies：T031。
  - Acceptance：AC-02、AC-04、AC-08、AC-12、AC-13，RE-01–07；T031 GREEN。
  - Verification：`go test ./tron`。
  - Budget：8h；约850生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：链错误、solidified 状态和交易 JSON 保持原义。

### Tests First / 测试先行

- [ ] **T033 五链 live runner 离线契约测试 / Test offline contracts for five-chain live runners**
  - Scope：`internal/test/live/read_contract_test.go`、`submission_contract_test.go`；允许 `evm.go`、`bitcoin.go`、`solana.go`、`tron.go` 声明骨架。
  - Dependencies：T024、T026、T028、T030、T032。
  - Acceptance：AC-09–AC-11，RE-08；合成 fixture 覆盖五链字段、历史/动态分类、独立 preflight、漏跑、双开关和提交材料门禁；callback 可在内存检查原始 Message/Data/ResponseBody/Cause，runner/harness 输出仅保留固定类别且不泄露合成标记。
  - Verification：`go test ./internal/test/live -run '^(TestReadContract|TestSubmissionContract)'`，产生目标行为缺失的 RED。
  - Budget：5h；≤80生产行、≤1,000测试行、6文件。
  - Done when：测试不访问公网且所有正式门禁均有可归因 RED。

### Implementation / 实现

- [ ] **T034 实现五链 live runner / Implement five-chain live runners**
  - Scope：`internal/test/live/evm.go`、`bitcoin.go`、`solana.go`、`tron.go`、`live_test.go`。
  - Dependencies：T033。
  - Acceptance：AC-09–AC-11、RE-08 的离线契约；建立 TestLiveRead/TestLiveBroadcast，并在原 runner 路径内实现原始 SDK 错误到固定报告类别的输出隔离；本 Task 不采样、不读真实凭据、不联网。
  - Verification：`go test ./internal/test/live -run '^(TestReadContract|TestSubmissionContract)'`。
  - Budget：8h；约800生产行、0新测试行、5文件；规模预警已纳入审批。
  - Done when：T033 GREEN，真实证据仍明确 deferred。

### Documentation and Offline Verification / 文档与离线核验

- [ ] **T035 文档、治理入口与离线交付核验 / Document and verify the offline delivery**
  - Scope：`README.md`、`AGENTS.md`、`doc.go`、`transport/doc.go`、`apierror/doc.go`、`jsonrpc/doc.go`、`httpjson/doc.go`、`evm/doc.go`、`ethereum/doc.go`、`bnb/doc.go`、`bitcoin/doc.go`、`solana/doc.go`、`tron/doc.go`、`internal/test/live/evidence/offline.md`；不得修改 constitution。
  - Dependencies：T034。
  - Acceptance：AC-01–AC-09、AC-12、AC-13，RE-01–08；核对方法、单位、nullable、DI、Raw、原始错误字段、Error() 非脱敏语义、调用层责任、兼容性、提交边界、依赖图、8.5/8.6 和核心构造签名隔离。
  - Verification：`go vet ./...`；`go list -deps ./...`；API/范围核查；只补跑 stale 聚焦证据。
  - Budget：4h；0生产/测试行，约600文档行，14路径单元；路径预警已纳入审批。
  - Done when：有效证据汇总完成，固定 Auditor 可获得完整只读审核包。

### Final Acceptance / 最终验收

- [ ] **T036 五链最终真实验收、最终审计与 Human 门禁 / Run final five-chain acceptance, audit, and Human gate**
  - Scope：`internal/test/live/testdata/ethereum/samples.json`、`bnb/samples.json`、`bitcoin/samples.json`、`solana/samples.json`、`tron/samples.json`、`internal/test/live/evidence/read.json`、`broadcast.json`、`final.md`；外部配置与交易载荷只读。
  - Dependencies：T035，且全部子 Plan 已完成离线验证、审核和批准。
  - Acceptance：AC-10、AC-11，RE-08；浏览器采样、独立账号 RPC preflight、SDK 字段核对、独立授权提交及按原标识查询；SDK 原始错误可供断言，最终 evidence 仍脱敏。禁止修改 SDK/runner。
  - Verification：`CHAIN_API_GO_LIVE_TEST=1 go test ./internal/test/live -run '^TestLiveRead$' -count=1`；双开关广播命令见 plan §8。
  - Budget：8h；0生产/测试行，约800证据行，8文件。
  - Done when：必需项全部 PASS、固定 Auditor 给出通过建议且 Human 最终批准；缺权限或材料为 BLOCKED。

## 7. 最终验证清单 / Final Validation Checklist

- [ ] 测试 Task 均早于并被对应实现 Task 依赖；RED/GREEN 证据绑定当前快照。
- [ ] T017–T036 依赖无环、Scope 未越界、旧 T001–T016 未被执行或复用。
- [ ] spec C-01–10、AC-01–13、RE-01–08 与 Human 批准后的 constitution 1.3 保持完整；8.5/8.6、原始错误边界和核心构造签名隔离通过审核。
- [ ] 每个 Plan 的聚焦验证完成；真实数据、外部 RPC 与链上提交仅在 T036 去重执行。
- [ ] 未经批准不提交、不推送、不合并；集成阶段只做结构检查。

既有批准和集成事实保留。RAW-ERROR-v1 已批准并重开 T017–T022；T017/T018 已完成门禁并获准集成，T019 未授权。T023/T024 保持完成；当前 T025 整包 RED/审核对新基线 stale，T025/T026 等待 T019–T022 完成后重建受影响证据。BITCOIN-CONTRACT-v1 与 T027 继续暂停，T028–T036 维持依赖并阻塞。仍为 5 Plan / 20 Task，无真实 RPC、广播、提交、推送或 PR 授权。
