# chain-api-go 任务清单 / chain-api-go Tasks

**规格文档 / Spec**：[spec.md](spec.md)  
**技术方案 / Plan**：[plan.md](plan.md)  
**任务清单 / Tasks**：[tasks.md](tasks.md)  
**基线版本 / Baseline Version**：PLAN-v8-NATIVE-SDK-v1
**日期 / Date**：2026-11-17
**状态 / Status**：T037–T042 已完成离线实施、验证与独立审计；T036 保持 `BLOCKED`；等待 Human 最终验收

> 当前权威范围：本轮迁移以 spec §8、plan §10、tasks §8 为准。它们替代旧文档中 EVM/Solana 自研 API、DTO、Has/present 规则、依赖禁令、相关 runner 字典和逐 Task 人工停顿要求；未涉及的 Bitcoin、Tron、jsonrpc、httpjson、原始错误、安全输出和真实验收规则继续有效。旧版本条目及 T017–T035 是历史证据，不代表新原生依赖基线已经通过验证。历史已完成任务不计入本轮活动清单。活动工作为两个功能 Plan、六个迁移 Task，另有未获真实执行授权的 T036。

## 0. 变更记录 / Change Log

| 版本 / Version | 日期 / Date | 变更 / Change |
|---|---|---|
| PLAN-v8-NATIVE-SDK-v1 | 2026-11-17 | Human 明确批准 NATIVE-SDK-GOVERNANCE-v1，授权依本候选连续规划、实施和独立审计。固定 go-ethereum v1.17.10、solana-go/v2 v2.3.0、Go 1.26；以私有持有上游 client 的受限封装替换 EVM/Ethereum/BNB 和 Solana 自研高层方法及类型，不提供兼容层。保持原始错误契约，批准两种私有最小协议桥接及仅限固定依赖的原生指针/转换例外。constitution 升至 1.5；AC-13 改为项目公开 API 和实际调用路径审查。本轮复用 PLAN-002、PLAN-004，新增稳定 Task T037–T042；原已完成 Task 仅作历史，T036 继续 BLOCKED。保留现有 internal/test/integration 迁移，不授权真实 RPC、广播、Git 操作或发布。 |
| PARALLEL-v1 | 2026-11-17 | Human 已授权：仅把 T031 的调度依赖由 T030 改为 T028，允许 T029∥T031 RED 及其独立门禁后的 T030∥T032 GREEN；T033 的全链依赖与 T033→T034→T035→T036 保持不变。剩余审计统一使用 `gpt-6.1-sol / medium` 全新只读上下文。保持 5 Plan/20 Task、全部 Scope/API/验收/预算与 RED/GREEN 门禁；不授权真实 RPC、浏览器采集、广播或 Git 发布/集成。 |
| PLAN-v7-REMAINING-RAW-v1 | 2026-11-17 | Human 已批准：同步 T017–T026 完成事实；替代未批准 BITCOIN-CONTRACT-v1 的严格解析设计；冻结 T027–T036 全部接口、DTO、wire、runner、文档与最终验收。保持 5 Plan / 20 Task、全部稳定 ID 和依赖，无新增依赖或核心配置来源。T027/T028 已完成 RED/GREEN、独立 Auditor 与 Human 门禁并获准集成；T029 明确暂停。真实 RPC 与广播仍未授权。 |
| RAW-ERROR-v1 | 2026-11-17 | Human 已批准：恢复原始节点 Message/Data、HTTP 错误正文及底层 Cause；区分 SDK 返回与 live 脱敏输出。重开 T017–T022，修订 T025/T026；T025 当前 RED/审核不能作为新基线门禁。T023/T024 未受影响证据保留，SDK→harness 输出隔离在 T033/T034 验证。保持 5 Plan / 20 Task、T017–T036 ID 与原路径；T027 继续暂停。constitution 1.3 同时获批；T017/T018 已完成门禁并获准集成，T019 未授权。 |
| EVM-RAW-DTO-v1 | 2026-11-17 | Human 已批准：将 EVM 收敛为原始 RPC DTO + 必要转换，接受破坏性 API 变更且不提供兼容层；重开 T025/T026，旧证据 stale。T025 可进入 RED，T026 仍等待审核和 Human 门禁；Bitcoin 候选暂停。保持 5 Plan / 20 Task、路径和依赖，不改 constitution。 |
| BITCOIN-CONTRACT-v1 | 2026-11-17 | 候选：补齐 PLAN-003 十方法、DTO、金额/费率、历史和 wire 契约；T027 暂停待批准。只改三件套；保持 5 Plan / 20 Task、全部依赖、T027 七路径、T028 五路径和 T017–T026 有效证据。 |
| EVM-CONTRACT-v1 | 2026-11-16 | Human 已批准：仅细化 T025/T026 的 EC-01–05 契约、T025 原十路径与声明预警；保持 5 Plan / 20 Task、全部 ID/依赖及 T017–T024 内容不变。T025 可恢复，T026 未启动。 |
| HARNESS-CONTRACT-v1 | 2026-11-16 | Human 已批准：冻结 test-only live harness 的显式配置、raw-byte SHA-256、可取消并发等待、总 deadline context、结果优先级与离线并发验证；只重开 T023/T024，保持 5 Plan / 20 Task、任务 ID 与依赖不变。T017–T022 的未受影响证据保持有效。 |
| CONTEXT-CAUSE-v1 | 2026-11-16 | Human 批准在 T017/T018 既有路径内增加 `Error.WithContextCause(error) Error` 的测试与实现；T017/T018 重新打开，T019 已批准证据保持有效。Plan/Task 总数不变。 |
| PLAN-v6 | 2026-11-16 | 将 v5 的 T001–T016 标记为被替代的历史 ID；活动任务改为 T017–T036，共 5 Plan / 20 Task。每个行为实现均依赖更早的失败测试 Task；真实 RPC 与受控提交集中到 T036。Human 已明确批准本修订。 |

历史映射：T001→T017/T018；T002→T019/T020；T003→T021/T022；T004→T023/T024；T005/T006→T025/T026；T008→T027/T028；T010→T029/T030；T012/T013→T031/T032；T007/T009/T011/T014 的离线 runner→T033/T034；其真实验收部分→T036；T015→T035；T016→T036。旧 ID 不再是活动 checkbox，不得执行或复用。

## 1. 任务格式与共同契约 / Task Format and Common Contract

当前活动迁移清单是 §8 的 T037–T042，共两个功能 Plan、六个 Task；历史五 Plan 目录不增加。T017–T035 已完成记录保留，T036 继续 BLOCKED。旧 Task 的 Scope、预算和人工停顿只描述历史，不控制本轮新任务。

每项 Task 继承 constitution 1.5、spec §8、plan §10 完整 Task Contract。Scope 默认拒绝，花括号仅按明确文件展开，不授权递归目录。先取得可运行目标断言 RED，再实施 GREEN；缺依赖、缺符号、编译或环境错误不算 RED。实现 Task 不修改受保护测试。

Human 已授权当前冻结范围内连续规划、实施与独立审计。Controller 必须验证 document gate、RED 归因、依赖、Scope、证据和角色；无需重复询问已授权的逐 Task 许可。新范围/冲突按 plan §10.11 停止。无真实 RPC、广播或 Git 授权。

每次交接记录独立 pre-task tree、代码/测试摘要、工具链与依赖、配置/fixture 身份、真实命令结果和状态；不得提前宣布未来测试或审计通过。任务的完成判断与 T036 真实验收状态严格分开。

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

- [x] **T019 JSON-RPC 原始错误协议测试 / Test JSON-RPC raw-error contracts**
  - Scope：`jsonrpc/client_test.go`、`jsonrpc/protocol_test.go`；不修改生产文件。
  - Dependencies：T018。
  - Acceptance：AC-02–AC-04、AC-12，RE-01–06；保留普通 nil、V2/Bitcoin 方言、ID、信封、限长、取消、body、并发和单次请求风险，不反射探测 typed-nil 依赖；增加 code=0、Message 原样、Data 缺失/null/大数字/对象、HTTP/RPC 正文、截断标志、部分读错误、原始 Cause/context Is/As 及“不主动附加请求秘密”。
  - Verification：`go test ./jsonrpc`，产生目标行为缺失的 RED。
  - Budget：计入 RAW-ERROR-v1 共享时箱；0生产行、约100–180测试增量行、2文件。
  - Done when：RAW-ERROR-v1 新增或改变的目标行为具有可归因 RED；未改变的既有断言保留，不要求重新产生 RED。

### Implementation / 实现

- [x] **T020 实现 JSON-RPC 原始错误保留 / Implement JSON-RPC raw-error preservation**
  - Scope：`jsonrpc/client.go`、`jsonrpc/protocol.go`。
  - Dependencies：T019。
  - Acceptance：AC-02–AC-04、AC-12，RE-01–06；实现 New/Call/RawCall 的原始节点字段、受限正文与完整 Cause 保留，T019 GREEN。
  - Verification：`go test -race ./jsonrpc`。
  - Budget：计入 RAW-ERROR-v1 共享时箱；约60–110生产增量行、0新测试行、2文件。
  - Done when：聚焦 race 检查通过，无隐藏重试或默认客户端。

### Tests First / 测试先行

- [x] **T021 HTTP JSON 原始错误协议测试 / Test HTTP JSON raw-error contracts**
  - Scope：`httpjson/client_test.go`；不修改生产文件。
  - Dependencies：T018。
  - Acceptance：AC-02–AC-04、AC-12，RE-02–06；保留普通 nil、同源及 prefix、路径穿越、取消和 body 风险，不反射探测 typed-nil 依赖；增加非 JSON HTTP 正文、刚好限长/超限、部分读取、原始 Cause/context Is/As 及“不主动附加请求秘密”。默认串行在 T020 后执行。
  - Verification：`go test ./httpjson`，产生目标行为缺失的 RED。
  - Budget：计入 RAW-ERROR-v1 共享时箱；0生产行、约60–120测试增量行、1文件。
  - Done when：RAW-ERROR-v1 新增或改变的目标行为具有可归因 RED；未改变的路径与资源断言保留，不要求重新产生 RED。

### Implementation / 实现

- [x] **T022 实现 HTTP JSON 原始错误保留 / Implement HTTP JSON raw-error preservation**
  - Scope：`httpjson/client.go`。
  - Dependencies：T021。
  - Acceptance：AC-02–AC-04、AC-12，RE-02–06；实现 New/Post/RawPost 的受限错误正文与完整 Cause 保留，T021 GREEN。
  - Verification：`go test ./httpjson`。
  - Budget：计入 RAW-ERROR-v1 共享时箱；约30–50生产增量行、0新测试行、1文件。
  - Done when：同源路径、限长、context 与 body 契约通过。

### Tests First / 测试先行

- [x] **T023 live harness 基础测试 / Test live harness foundation**
  - Scope：`internal/test/integration/harness_test.go`、`internal/test/integration/config_test.go`、`internal/test/integration/manifest_test.go`；允许 `internal/test/integration/harness.go`、`internal/test/integration/config.go`、`internal/test/integration/manifest.go` 仅建立 plan §7.2 已批准接口的声明骨架；允许 `internal/test/integration/config.example.json` 仅建立不含秘密的合法 JSON 占位 `{}`，使模板测试进入目标内容断言而非文件缺失错误。不得实现 loader、解析、摘要、限流或账本行为。
  - Dependencies：T020、T022。
  - Acceptance：AC-09，继承 AC-04/C-02/05/06/08/09；覆盖 plan §7.2 HC-01–HC-08：关闭开关零配置/材料读取，显式路径与 Header env 引用及缺失 env，模板无秘密，严格 JSON 与 raw-byte SHA-256，独立必需目录和状态优先级，请求预算，真实多 goroutine 并发上限/等待/取消，速率，实际 context deadline、父取消和资源释放，错误脱敏及 callback 不能覆盖基础错误。
  - Verification：`go test ./internal/test/integration -run '^TestHarness' -count=1`；`go test -race ./internal/test/integration -run '^TestHarness' -count=1`。两者必须完成编译并进入目标断言；目标行为缺失产生 RED，测试自身不得产生 race。缺文件、编译、环境或死锁不是合格 RED。
  - Budget：最多 70 分钟；规划 20 分钟、编码 20 分钟、验证最多 30 分钟。≤100 生产声明骨架行、≤900 测试行、≤10 模板占位行、7 文件；行数仅为范围预警，不是目标额度或延长工期的依据。
  - Done when：每项 HC 风险已有可追踪的确定性断言，HARNESS-CONTRACT-v1 的 RED、固定 Auditor 与 Human 门禁已完成。其替代记录只针对更早历史证据；RAW-ERROR-v1 保留 T023/T024 已完成状态，不重开、不重新要求 RED。

### Implementation / 实现

- [x] **T024 实现 live harness 基础 / Implement live harness foundation**
  - Scope：`internal/test/integration/config.go`、`manifest.go`、`harness.go`、`config.example.json`。
  - Dependencies：T023。
  - Acceptance：AC-09，继承 AC-04/C-02/05/06/08/09；实现 plan §7.2 HC-01–HC-08，使经审核 T023 测试 GREEN；模板仅 endpoint/Header 环境变量引用和无秘密元数据。无第三方依赖、默认路径、默认客户端、凭据、重试、后台 worker 或真实网络调用。
  - Verification：`go test -race ./internal/test/integration -run '^TestHarness' -count=1`；`go vet ./internal/test/integration`；检查四文件 Diff、核心依赖方向和无新增依赖。race 命令同时提供全部 Harness 测试 GREEN，不重复运行未受影响 SDK 测试。
  - Budget：最多 70 分钟；规划 20 分钟、编码 20 分钟、验证最多 30 分钟。≤600 生产行、0 新测试行、≤100 模板行、4 文件；预计实现应明显低于上限，行数只触发范围复核，不得据此延长工期或增加抽象。
  - Done when：受保护 T023 断言全部 GREEN，race/vet 通过；普通测试零公网、关闭开关零外部读取，正式缺项不能 PASS；无资源泄露、敏感输出或 Scope 越界；当前快照证据经固定 Auditor 审核通过，依既有 Human 门禁标记完成。

## 3. PLAN-002 Ethereum 与 BNB / Ethereum and BNB

T025/T026 的 reopened/stale 说明是历史过程；二者当前已完成对应 RED/GREEN、Auditor、Human 门禁并集成。本轮不重开，数量仍为 5 Plan / 20 Task。

### Tests First / 测试先行

- [x] **T025 重建原始 EVM DTO 契约测试 / Reopen raw EVM DTO contract tests**
  - Scope：沿用原十路径：`evm/types_test.go`、`evm/query_test.go`、`evm/compute_test.go`、`evm/submit_test.go`、`ethereum/client_test.go`、`bnb/client_test.go`、`evm/types.go`、`evm/client.go`、`ethereum/client.go`、`bnb/client.go`。本次仅修改前六测试文件，后四生产文件不得修改；不新增路径。
  - Dependencies：当前 T020、当前 T022、T024；EVM-RAW-DTO-v1 与 RAW-ERROR-v1 明确批准。
  - Acceptance：AC-02/04/05/12/13，RE-01/02/05–07，spec ER-01–05/plan §3.2；仅撤换明确列出的被替代断言，保留 DI/14 方法/nullable/单次提交/网络身份保护，并证明链包装保留原始节点字段与完整依赖错误链、不主动附加请求秘密。
  - Verification：`go test ./evm ./ethereum ./bnb`；旧实现编译成功，在原始字段/底层类型/新增 ToString 等目标断言处 RED。测试侧用 JSON fixture、有限反射及本地接口断言避免新 API 未定义导致编译失败。
  - Budget：与 T026 共享规划 20 分钟、编码 20 分钟、验证含审核最多 30 分钟；生产变更 0，六测试文件最终约 600–900 行仅为预警。
  - Done when：DTO 与原始错误新契约风险均有可追踪 RED，当前旧整包证据明确 stale，证据绑定当前代码/测试/依赖/配置/环境快照；独立只读 Auditor 后停在 T025 Human 门禁，不启动 T026。

### Implementation / 实现

- [x] **T026 简化 EVM 与两具体客户端 / Simplify EVM and concrete clients**
  - Scope：`evm/client.go`、`evm/types.go`、`evm/query.go`、`evm/compute.go`、`evm/submit.go`、`ethereum/client.go`、`bnb/client.go`；不改测试、其他包、依赖或 constitution。
  - Dependencies：T025 的新 RED、独立审核及 Human 门禁。
  - Acceptance：AC-02/04/05/12/13，RE-01/02/05–07；严格实现 spec ER-01–05、RE-01–07 与 plan §3.2/§4，标准解码替换手写 schema，保留完整 Caller 错误链；全部 T025 断言实际执行并 GREEN。保留现有未提交修改的快照与来源，只按批准候选授权处理相关 EVM 行为。
  - Verification：`go test ./evm ./ethereum ./bnb`；`go vet ./evm ./ethereum ./bnb`；核对七路径、API 删除/变更及防御分支依据。只复验因修复而 stale 的检查。
  - Budget：使用 T025 所列共享 20/20/30 分钟时箱，不另计 10h 或叠加；七生产文件最终约 350–500 行仅为预警，0 新测试行。
  - Done when：T025 全部受保护断言 GREEN、vet 通过、无生产 schema/重复 context 防御、无越权；当前快照经独立 Auditor 和 Human 门禁。真实数据未验证，不自动恢复 T027。

## 4. PLAN-003 Bitcoin

BITCOIN-CONTRACT-v1 从未获批并已由已批准的 PLAN-v7-REMAINING-RAW-v1 替代。T027–T035 已完成对应离线 RED/GREEN、验证与独立 Auditor 门禁；T036 真实验收因授权和材料缺失为 `BLOCKED`。不得把旧严格解析或无证据防御复制到其他链。

### Tests First / 测试先行

- [x] **T027 完整 Bitcoin 客户端测试 / Test complete Bitcoin client**
  - Scope：恰好七路径：`bitcoin/client_test.go`、`bitcoin/types_test.go`、`bitcoin/query_test.go`、`bitcoin/compute_test.go`、`bitcoin/submit_test.go`、`bitcoin/client.go`、`bitcoin/types.go`；后两文件仅含 plan §3.3 完整声明骨架和 Unsupported/零值分支，不实现解析、校验或请求行为。
  - Dependencies：T026。
  - Acceptance：AC-02/04/06/12/13、BC-01–05、RE-01/02/06/07；覆盖 plan §3.3 的十方法 wire、原始 DTO、集中精确金额转换、历史/null/估费、预检拒绝、原始错误、单次提交 unknown；标准解码正例证明未知/非必要字段不触发全 schema。
  - Verification：`go test ./bitcoin`，产生目标行为缺失的 RED。
  - Budget：与 T028 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；180–260 声明、500–750 测试行预警、7路径。
  - Done when：七文件编译并进入可归因 RED，BC-01–05 证据绑定当前快照，经固定只读 Auditor 后停在 T027 Human 门禁；骨架遮蔽的细部断言须在 T028 实际执行。

### Implementation / 实现

- [x] **T028 实现完整 Bitcoin 客户端 / Implement complete Bitcoin client**
  - Scope：`bitcoin/client.go`、`bitcoin/types.go`、`bitcoin/query.go`、`bitcoin/compute.go`、`bitcoin/submit.go`。
  - Dependencies：T027。
  - Acceptance：AC-02/04/06/12/13、BC-01–05、RE-01/02/06/07；实现批准的 plan §3.3/§4，T027 全部受保护断言实际执行且 GREEN，不改测试。
  - Verification：`go test ./bitcoin`；`go vet ./bitcoin`。
  - Budget：与 T027 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；350–500生产行预警、0新测试行、5文件。
  - Done when：金额无浮点、历史限制不被吞、单次提交语义通过；聚焦测试/vet、独立 Auditor 和 Human 门禁完成。

## 5. PLAN-004 Solana

### Tests First / 测试先行

- [x] **T029 完整 Solana 客户端测试 / Test complete Solana client**
  - Scope：`solana/client_test.go`、`solana/types_test.go`、`solana/query_test.go`、`solana/compute_test.go`、`solana/submit_test.go`、`solana/client.go`、`solana/types.go`；后两文件仅声明骨架。
  - Dependencies：T028。
  - Acceptance：AC-02/04/07/12/13、SOL-01–05、RE-01/02/06/07；覆盖十一方法 wire、选项省略/显式零、legacy/v0/meta、context/value null、编码边界、模拟/preflight、一次提交 unknown 和原始错误链。
  - Verification：`go test ./solana`，产生目标行为缺失的 RED。
  - Budget：与 T030 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；180–260声明、550–800测试行预警、7文件。
  - Done when：版本、context、nullable、选项和提交风险均有可归因 RED，经固定 Auditor 后停在 Human 门禁。

### Implementation / 实现

- [x] **T030 实现完整 Solana 客户端 / Implement complete Solana client**
  - Scope：`solana/client.go`、`solana/types.go`、`solana/query.go`、`solana/compute.go`、`solana/submit.go`。
  - Dependencies：T029。
  - Acceptance：AC-02/04/07/12/13、SOL-01–05、RE-01/02/06/07；T029 全部受保护断言 GREEN，不解析/构造 message，不补默认选项、不重试。
  - Verification：`go test ./solana`；`go vet ./solana`。
  - Budget：与 T029 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；350–500生产行预警、0新测试行、5文件。
  - Done when：commitment、version、meta、context/value 与有效高度保留，聚焦测试/vet、Auditor、Human 门禁完成。

## 6. PLAN-005 Tron 与最终集成 / Tron and Final Integration

### Tests First / 测试先行

- [x] **T031 完整 Tron 客户端测试 / Test complete Tron client**
  - Scope：`tron/client_test.go`、`tron/types_test.go`、`tron/query_test.go`、`tron/compute_test.go`、`tron/submit_test.go`、`tron/client.go`、`tron/types.go`；后两文件仅声明骨架。
  - Dependencies：T028（PARALLEL-v1：替代原 T030 串行调度边；既有 T022 HTTP JSON 基础须保持已完成且只读）。
  - Acceptance：AC-02/04/08/12/13、TR-01–05、RE-01–07；覆盖十三查询、两计算、一提交的路径/body/调用次数、visible 地址、五 solidified 独立接线、`{}` 与零余额、proto3 默认值、HTTP 200 Error/result、原始交易 JSON、Cause 与 unknown。
  - Verification：`go test ./tron`，产生目标行为缺失的 RED。
  - Budget：与 T032 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；180–260声明、550–850测试行预警、7文件。
  - Done when：地址、present、资源、solidified、拒绝和提交风险均有可归因 RED，经固定 Auditor 后停在 Human 门禁。

### Implementation / 实现

- [x] **T032 实现完整 Tron 客户端 / Implement complete Tron client**
  - Scope：`tron/client.go`、`tron/types.go`、`tron/query.go`、`tron/compute.go`、`tron/submit.go`。
  - Dependencies：T031。
  - Acceptance：AC-02/04/08/12/13、TR-01–05、RE-01–07；T031 全部受保护断言 GREEN，不改测试，不构造交易/TAPOS/签名，不按 message 分类。
  - Verification：`go test ./tron`；`go vet ./tron`。
  - Budget：与 T031 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；350–500生产行预警、0新测试行、5文件。
  - Done when：十六方法、原始链错误、solidified 状态和交易 JSON 保持原义，聚焦测试/vet、Auditor、Human 门禁完成。

### Tests First / 测试先行

- [x] **T033 五链 live runner 离线契约测试 / Test offline contracts for five-chain live runners**
  - Scope：`internal/test/integration/read_contract_test.go`、`internal/test/integration/submission_contract_test.go`、`internal/test/integration/evm.go`、`internal/test/integration/bitcoin.go`、`internal/test/integration/solana.go`、`internal/test/integration/tron.go`；后四文件仅声明骨架。
  - Dependencies：T024、T026、T028、T030、T032。
  - Acceptance：AC-09–11、LR-01–06、RE-08；按 plan §7.8.1–§7.8.7 的精确声明和固定字典，以合成依赖覆盖全部方法目录、manifest/dynamic/material、历史/动态、独立 preflight、漏项、双开关零读取、身份失败阻止提交、claim 持久化先于 send、已有 claim 禁止重发、unknown 仅查原 ID 一次及报告输出隔离；不新增文件或修改 harness。
  - Verification：`go test ./internal/test/integration -run '^(TestReadContract|TestSubmissionContract)' -count=1`，产生目标行为缺失的 RED；不得外部读取。
  - Budget：与 T034 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；≤120声明、500–800测试行预警、6文件。
  - Done when：LR-01–06 的离线门禁均有可归因 RED，经固定 Auditor 后停在 Human 门禁。

### Implementation / 实现

- [x] **T034 实现五链 live runner / Implement five-chain live runners**
  - Scope：`internal/test/integration/evm.go`、`internal/test/integration/bitcoin.go`、`internal/test/integration/solana.go`、`internal/test/integration/tron.go`、`internal/test/integration/live_test.go`。
  - Dependencies：T033。
  - Acceptance：AC-09–11、LR-01–06、RE-08；实现 plan §7.8 的具体客户端装配、固定 catalog、材料解析与输出隔离；建立 TestLiveRead/TestLiveBroadcast，但本 Task 不采样、不读真实凭据、不联网。
  - Verification：`go test ./internal/test/integration -run '^(TestReadContract|TestSubmissionContract)' -count=1`；`CHAIN_API_GO_LIVE_TEST=0 CHAIN_API_GO_LIVE_BROADCAST=0 go test ./internal/test/integration -run '^TestLive(Read|Broadcast)$' -count=1`；`go vet ./internal/test/integration`。关闭开关零读取由 T033 的可注入入口断言证明，不能仅以 Skip 当作证据；禁止继承宿主开启值执行真实装配。
  - Budget：与 T033 共享 plan §7.7 的 20/20/30；本 Task 资源估算 25 分钟（规划10、编码1.25、验证13.75）；450–650生产行预警、5文件。
  - Done when：T033 GREEN，关闭开关零读取，聚焦 vet、Auditor、Human 门禁完成；真实证据保持 deferred。

### Documentation and Offline Verification / 文档与离线核验

- [x] **T035 文档与离线交付核验 / Document and verify the offline delivery**
  - Scope：`README.md`、`doc.go`、`transport/doc.go`、`apierror/doc.go`、`jsonrpc/doc.go`、`httpjson/doc.go`、`evm/doc.go`、`ethereum/doc.go`、`bnb/doc.go`、`bitcoin/doc.go`、`solana/doc.go`、`tron/doc.go`、`internal/test/integration/evidence/offline.md`；`AGENTS.md`、`constitution.md`、`docs/architecture.md` 仅只读核对，不得修改。
  - Dependencies：T034。
  - Acceptance：AC-01–09/12/13、RE-01–08；文档覆盖五链构造器/全部首版方法、Raw 协议入口、单位、RawMessage 缺失/null、Tron proto3 零省略、原始 error 非安全日志、提交 once/unknown/接受-执行-确认差异、test-only schema/alias/材料单次使用；准确标明真实验收未执行。
  - Verification：`go vet ./...`；`go list -deps ./...`；API/范围核查；只补跑 stale 聚焦证据。
  - Budget：最多70分钟，plan §7.7 的 20/20/30 分段；资源估算 25 分钟（规划10、编码1.25、验证13.75），0生产/测试行、350–550文档行预警、13路径。
  - Done when：离线证据汇总完成，公开 API/依赖/边界与批准基线一致，固定 Auditor 与 Human 门禁完成。

### Final Acceptance / 最终验收

- [ ] **T036 五链最终真实验收、最终审计与 Human 门禁 / Run final five-chain acceptance, audit, and Human gate**
  - Scope：`internal/test/integration/testdata/ethereum/samples.json`、`internal/test/integration/testdata/bnb/samples.json`、`internal/test/integration/testdata/bitcoin/samples.json`、`internal/test/integration/testdata/solana/samples.json`、`internal/test/integration/testdata/tron/samples.json`、`internal/test/integration/evidence/read.json`、`internal/test/integration/evidence/broadcast.json`、`internal/test/integration/evidence/final.md`；外部配置与交易载荷只读。
  - Dependencies：T035，且全部子 Plan 已完成离线验证、审核和批准。
  - Acceptance：AC-10/11、RE-08、FA-01–06；每链三笔浏览器样本、非 SDK 账号 RPC preflight、原始字节摘要冻结、五链固定 catalog、逐链独立授权提交与原 ID 单次查询；启动前 Controller 确认广播工作区单写者及材料未消费，broadcast.json 以追加 JSON Lines 保存提交前 claim 和最终报告；已有 claim 不得重播，截断账本为 BLOCKED。最终 evidence 脱敏。禁止修改 SDK/runner。
  - Verification：`CHAIN_API_GO_LIVE_TEST=1 go test ./internal/test/integration -run '^TestLiveRead$' -count=1`；双开关广播命令见 plan §8。
  - Budget：最多70分钟，plan §7.7 的 20/20/30 分段；资源估算 50 分钟（规划20、编码2.5、验证27.5），0生产/测试行，数据/证据按实际样本、8文件；等待授权/材料不构成自动延时。
  - Done when：FA-01–06 必需项全部 PASS，固定 Auditor 给出通过建议且 Human 最终批准；缺权限、材料、网络身份或任一链必需项为 BLOCKED。

## 7. 最终验证清单 / Final Validation Checklist

- [ ] 测试 Task 均早于并被对应实现 Task 依赖；RED/GREEN 证据绑定当前快照。
- [ ] T017–T036 依赖无环、Scope 未越界、旧 T001–T016 未被执行或复用。
- [ ] spec C-01–10、AC-01–13、RE-01–08 与 Human 批准后的 constitution 1.4 保持完整；8.5/8.6、原始错误边界和核心构造签名隔离通过审核。
- [ ] 每个 Plan 的聚焦验证完成；真实数据、外部 RPC 与链上提交仅在 T036 去重执行。
- [ ] 未经批准不提交、不推送、不合并；集成阶段只做结构检查。

历史状态（PLAN-v7）：T017–T035 的旧基线工作已完成并保留为历史证据；其旧波次、时箱和人工门禁不控制 §8 的原生迁移。当前权威状态见文首及 §8；T036 仍不授权真实 RPC、浏览器采集、广播、推送或 PR。

## 8. PLAN-v8-NATIVE-SDK-v1 活动任务

共同契约见 plan §10.8–10.11；下面逐项字段与共同契约合并为完整执行合同。所有路径相对仓库根目录，未列路径禁止修改。依赖总览：document gate → T037 → T038 → T039 → T040 → T041 → T042。T036 不在本轮执行链内，继续 BLOCKED。本轮不并行修改共享 go.mod、integration 或 evidence。

### PLAN-002：原生 EVM/Ethereum/BNB

- [x] **T037 原生 EVM 先行契约测试 / Test native EVM contracts first**
  - Action：加入固定 go-ethereum 元数据并升级 go directive；以原生接口和 NS 风险测试替换旧 EVM 自研 API 测试，迁移 integration 的 EVM 原生目录和请求计数预期；运行时接口断言取得行为 RED，不实现生产 bridge。
  - Dependencies：六文件冻结、document gate、Go 1.26 可用。
  - Scope：`go.mod`、`go.sum`、`evm/types_test.go`、`evm/query_test.go`、`evm/compute_test.go`、`evm/submit_test.go`、`evm/bridge_test.go`、`ethereum/client_test.go`、`bnb/client_test.go`、`internal/test/integration/read_contract_test.go`、`internal/test/integration/submission_contract_test.go`。
  - Restrictions：integration 两文件只改 EVM/BNB fixture、目录、断言和本地 helper；保留 Bitcoin/Solana/Tron 风险。go.mod 只改 Go 指令、固定 geth 及必要 indirect；不加入 Solana。
  - Acceptance：NS-01/02/03/04/07/08，AC-01–05/09/12/13；RED 必须是目标断言失败而非编译/环境错误。
  - Budget：30 分钟；生产行为 0 行，测试预计 500–1000 行，依赖生成单列。

- [x] **T038 实现原生 EVM 及私有桥接 / Implement native EVM and private bridge**
  - Action：私有持有固定 ethclient/rpc；实现内存 RoundTripper、十三个原生方法及一次 raw seam；删除自研类型和旧方法；保留 Ethereum/BNB ExpectedChainID/ValidateNetwork；迁移 EVM integration 与 Caller 边界预算。
  - Dependencies：T037 有效 RED 和受保护测试摘要。
  - Scope：`evm/client.go`、`evm/types.go`、`evm/query.go`、`evm/compute.go`、`evm/submit.go`、`evm/bridge.go`、`ethereum/client.go`、`bnb/client.go`、`internal/test/integration/evm.go`。
  - Restrictions：types.go 可删除；evm.go 仅 EVM/BNB 分支、专用预算 wrapper 和必要共享声明；不得修改测试、公共协议或依赖版本。
  - Acceptance：同 T037；完整块叔块计数和原始错误必须 GREEN；聚焦单元、TestNativeEVM 与 vet 通过。
  - Budget：40 分钟；生产修改预计 350–650 行（含删除/runner），测试 0 行。

- [x] **T039 EVM 文档与功能审计 / Document and audit native EVM**
  - Action：更新 EVM/Ethereum/BNB 包文档、README、破坏性变化和离线证据；由全新只读 Auditor 审核 PLAN-002。
  - Dependencies：T038 LOCALLY_VERIFIED。
  - Scope：`evm/doc.go`、`ethereum/doc.go`、`bnb/doc.go`、`README.md`、`internal/test/integration/evidence/offline.md`。
  - Restrictions：只改文档/证据，不实现行为、不修改测试、不伪造真实链结果。
  - Acceptance：NS-01–04/07/08，AC-01–05/09/12/13；公开 API/调用路径审查无未解决高优先级问题。
  - Budget：30 分钟；生产/测试行为 0 行。

### PLAN-004：原生 Solana

- [x] **T040 原生 Solana 先行契约测试 / Test native Solana contracts first**
  - Action：加入固定 solana-go/v2 元数据；以原生接口、nullable/options/wire、错误、一次 encoded submit 和 unsupported bridge 测试替换旧 API 测试；迁移 integration 的 Solana 原生材料与断言，取得行为 RED。
  - Dependencies：T039 有效审计门禁。
  - Scope：`go.mod`、`go.sum`、`solana/client_test.go`、`solana/types_test.go`、`solana/query_test.go`、`solana/compute_test.go`、`solana/submit_test.go`、`solana/bridge_test.go`、`internal/test/integration/read_contract_test.go`、`internal/test/integration/submission_contract_test.go`。
  - Restrictions：只改 Solana 测试和必要 helper；EVM 受保护断言及 Bitcoin/Tron 风险不降低；不改 geth 版本。
  - Acceptance：NS-01/02/05/06/07/08，AC-01–04/07/09/12/13；RED 必须是目标断言失败。
  - Budget：30 分钟；生产行为 0 行，测试预计 500–1000 行。

- [x] **T041 实现原生 Solana 及私有桥接 / Implement native Solana and private bridge**
  - Action：私有持有 `NewWithCustomRPCClient` 创建的上游 client；实现批准的十一个方法和 CallForInto bridge，其余接口 Unsupported；删除旧自研 DTO/API；迁移 Solana integration 和共享文件中的类型引用。
  - Dependencies：T040 有效 RED。
  - Scope：`solana/client.go`、`solana/types.go`、`solana/query.go`、`solana/compute.go`、`solana/submit.go`、`solana/bridge.go`、`internal/test/integration/solana.go`、`internal/test/integration/evm.go`。
  - Restrictions：types.go 可删除，空 Config 移 client.go；共享 evm.go 只改 Solana 装配/版本/类型引用；不修改测试或依赖版本。
  - Acceptance：同 T040；批准方法只走 CallForInto，callback/batch 零调用；单元、TestNativeSolana 和 vet 通过。
  - Budget：40 分钟；生产修改预计 350–650 行（含删除/runner），测试 0 行。

- [x] **T042 最终离线验收与独立审计 / Verify and audit the native SDK migration**
  - Action：同步 Solana/根包/README 和材料版本说明；执行全仓离线测试、vet、依赖/API/路径检查；由全新只读 Auditor 审核 PLAN-004 与跨 Plan 最终状态；记录 T036 仍 BLOCKED。
  - Dependencies：T041 LOCALLY_VERIFIED、T039 有效审计。
  - Scope：`solana/doc.go`、`doc.go`、`README.md`、`internal/test/integration/config.example.json`、`internal/test/integration/evidence/offline.md`、`internal/test/integration/evidence/final.md`。
  - Restrictions：仅文档、合成示例配置和证据；不改变 config schema/来源或启用开关，不填真实 endpoint/凭据，不写真实样本/journal，不修改生产行为和测试。若最终失败需生产修复，返回对应 T038/T041 Scope。
  - Acceptance：NS-01–08，AC-01–09/12/13；AC-10/11 明确由未执行 T036 负责。最终关闭开关运行 plan §10.10 命令集。
  - Budget：30 分钟；生产/测试行为 0 行。

### 8.1 活动任务反向追踪与最终检查

| Task | Plan | 需求 | 输出门禁 |
|---|---|---|---|
| T037 | PLAN-002 | NS-01/02/03/04/07/08 | 受保护 RED |
| T038 | PLAN-002 | NS-01/02/03/04/07/08 | 实现与局部 GREEN |
| T039 | PLAN-002 | NS-01/02/03/04/07/08 | 文档及功能审计 |
| T040 | PLAN-004 | NS-01/02/05/06/07/08 | 受保护 RED |
| T041 | PLAN-004 | NS-01/02/05/06/07/08 | 实现与局部 GREEN |
| T042 | PLAN-004 | NS-01–08 | 最终离线验收及独立审计 |

- [x] 六文件冻结，三件套元数据/交叉链接/完整组成/追踪通过。
- [x] T037、T040 的 RED 均为目标断言失败，非环境或编译错误。
- [x] T038、T041 未修改受保护测试，所有目标断言 GREEN。
- [x] 固定原生依赖与 Go 1.26、公开 API、原始错误、一次提交和 DI 经核对。
- [x] 用户已有 integration 迁移及未迁移链行为保留。
- [x] 两个功能 Plan 和最终审计证据有效，没有提前声明通过。
- [x] 无真实 RPC/广播/Git 操作；T036 明确 BLOCKED。
