# chain-api-go 需求规格

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
| EVM-CONTRACT-v1 | 2026-11-16 | Human 已批准：补齐 EVM 跨包构造入口、首版查询/费用 DTO 字段与存在性、ownership 和 wire 验收；仅修订 PLAN-002/T025–T026 契约，T025 仍限原 10 路径，T026 未启动。保持 5 Plan / 20 Task；不改 T017–T024、constitution 或既有依赖。 |
| HARNESS-CONTRACT-v1 | 2026-11-16 | Human 已批准：冻结 test-only live harness 的显式配置、raw-byte SHA-256、可取消并发等待、总 deadline context、结果优先级与离线并发验证；只重开 T023/T024，保持 5 Plan / 20 Task、任务 ID 与依赖不变。T017–T022 的未受影响证据保持有效。 |
| CONTEXT-CAUSE-v1 | 2026-11-16 | Human 批准 `Error.WithContextCause(error) Error` 值方法：仅归一化两个 context 哨兵、取消优先、不保存其他底层错误。复用 T017/T018 的既有路径与 TDD/审核门禁，不新增 Plan 或 Task。 |
| PLAN-v6 | 2026-11-16 | 将已批准 v5 的混合 Task 改为先行失败测试与依赖它的实现 Task；活动清单为 5 Plan / 20 Task。真实采样、账号 RPC 与受控提交集中至最终验收。补充默认拒绝路径、预算、双向追踪和证据门禁。链、方法范围、AC-01–13 与 constitution 1.1 不变；Human 已明确批准本修订及值语义、`HTTPDoer` 签名裁决。 |

## 1. 项目目标

`chain-api-go` 是供业务服务、基建服务直接导入的 Go 链节点 API SDK，不是独立服务，也不是业务仓库。它为 Ethereum、BNB Chain、Bitcoin、Solana、Tron 提供清晰、可注入、可测试的原生 RPC 客户端。

## 2. 强制约束

| ID | 约束 |
|---|---|
| C-01 | 纯 Go SDK/库；不得监听端口、启动服务或包含部署单元。 |
| C-02 | 所有运行时依赖显式注入，包括 HTTP 客户端、RPC 调用器、节点地址、Header、超时及验收配置。SDK 不读取环境变量或配置文件。 |
| C-03 | 不提供统一链适配器、注册表、dispatcher 或跨链通用交易模型；调用方直接创建具体链客户端。 |
| C-04 | 保留链原生单位、编码、可空值、错误码以及 RPC 接受、链上执行和最终确认的差异。 |
| C-05 | SDK 默认不重试、不切换节点、不轮询、不缓存、不记录敏感日志。 |
| C-06 | 构造函数只做本地校验，不联网、不创建后台 goroutine；共享 HTTP transport 的生命周期归调用方。 |
| C-07 | 交易提交接口只发送调用方提供的链原生序列化载荷一次，不修改、不重建、不自动补发。 |
| C-08 | 命名常量类型按底层类型提供明确转换方法：`string` → `ToString()`、`int` → `ToInt()`、`int32` → `ToInt32()`、`int64` → `ToInt64()`；其他底层类型遵循同一规则。固定原生依赖仅适用 constitution 8.7 的限定例外。 |
| C-09 | 小字段、小结构体及其参数、返回值和构造器传参使用值语义。仅复杂生命周期实现对象可使用指针；不得借此批量改写既有指针或破坏数据库 `NULL`、第三方 API 契约，冲突必须交由用户决定。固定原生依赖仅适用 constitution 8.7 的限定例外。 |
| C-10 | 核心 SDK 不接收私钥，不构造、编码或签名交易，不调用节点签名或解锁接口。`go-wallet-sdk` 若用于说明组合，只能位于独立示例 module，且示例默认离线、不得默认广播或包含真实凭据。本约束是技术隔离边界，不是钱包业务职责。 |

## 3. 功能范围

### 3.1 公共协议能力

- 注入式 JSON-RPC 客户端，支持 JSON-RPC 2.0 及 Bitcoin Core 所需方言。
- 注入式 HTTP JSON POST 客户端，供 Tron 原生路径调用。
- context 取消、响应大小限制、响应 body 关闭、并发安全请求 ID。
- 结构化错误：保留配置、参数、传输、HTTP、协议、节点 RPC、不支持和网络不匹配类别，并将节点原始 Message/Data、受响应限长约束的错误正文、原始底层 Cause 提供给调用层。SDK 不承担业务映射、重试或脱敏策略。
- RawCall/RawPost 只是协议逃生口，不是统一链业务入口。

### 3.2 链能力矩阵

| 链 | 查询 | 节点计算/预检 | 交易提交 |
|---|---|---|---|
| Ethereum、BNB | ChainID、BlockNumber、Block、Balance、Nonce、Code、Transaction、Receipt、Logs | Call、EstimateGas、GasPrice、FeeHistory | SendRawTransaction |
| Bitcoin | BlockchainInfo、BlockCount、BlockHash、Block、TxOut、RawTransaction、MempoolEntry | EstimateSmartFee、TestMempoolAccept | SendRawTransaction |
| Solana | GenesisHash、Slot、Block、Balance、AccountInfo、LatestBlockhash、SignatureStatuses、Transaction | GetFeeForMessage、SimulateTransaction | SendTransaction |
| Tron | NodeInfo、Block、Account、AccountResource、ChainParameters、Transaction、TransactionInfo，以及五个明确 solidified 查询 | TriggerConstantContract、EstimateEnergy | BroadcastTransaction |

Tron 的五个 solidified 查询为 Account、NowBlock、BlockByNum、TransactionByID、TransactionInfoByID。不得通过通用路径替换推测更多接口。

## 4. 验收要求

| ID | 验收条件 |
|---|---|
| AC-01 | 五链包可被直接导入；核心没有服务端、统一适配器或 dispatcher。 |
| AC-02 | 构造函数在单一注入边界拒绝 nil 依赖和本契约明确列出的非法配置，零网络 I/O；不反射探测接口内部 typed-nil。方法仅验证各链契约明确列出的请求输入。调用方须提供有效非 nil context；链层不重复协议层 context 检查，不保护未经构造的零值 Client。 |
| AC-03 | JSON-RPC/HTTP JSON 正确处理 ID、HTTP 状态、错误信封、超限响应、取消和 body 关闭；没有隐式重试。 |
| AC-04 | SDK 返回的结构化错误保留原始节点 Message/Data、数字或字符串错误码、HTTP 错误响应正文和底层 Cause；`errors.Is/As` 不因 SDK 包装丢失原错误语义。SDK 不主动附加 Header、凭据或完整请求/交易载荷，不按英文 message 猜测分类；调用层负责业务映射、返回、重试及脱敏。live 报告、fixture 和证据仍按安全输出契约脱敏。精确验收见 RE-01–RE-08。 |
| AC-05 | Ethereum 与 BNB 通过 §8 和 plan §10 的固定 go-ethereum 原生查询、计算、已编码载荷提交与显式网络身份检查；旧 EVM 自研 DTO/API 不再是活动契约。 |
| AC-06 | Bitcoin 金额不经过浮点，历史查询前提、null 和节点拒绝保持原义。 |
| AC-07 | Solana 通过 §8 和 plan §10 的固定 solana-go/v2 原生 API；保留上游 commitment、context、legacy/v0、meta、blockhash 有效高度、nullable 和提交选项语义，不添加旧 DTO 兼容层。 |
| AC-08 | Tron 正确处理地址编码、visible、空账户、HTTP 200 内链错误、资源单位与 solidified 状态。 |
| AC-09 | 普通测试零公网；真实 live 入口仅在显式启用后读取指定外部配置与材料。测试装配按 plan §7.2 HARNESS-CONTRACT-v1 实现严格路径/env 引用解析、原始字节 SHA-256、请求数、可取消并发等待、开始间隔和真实 deadline context；独立必需目录与 FAIL 优先账本禁止漏跑、阻塞或错误被 PASS 掩盖。关闭开关零配置读取、零材料读取、零请求 I/O。 |
| AC-10 | 每链至少采集 3 笔区块链浏览器真实交易，先以用户账号 RPC 独立预检查，再通过 SDK 逐字段核对；不得用公共 RPC 冒充用户账号或用 SDK 自己生成全部预期。 |
| AC-11 | 五链提交验收必须独立授权并使用测试网或受控节点及专用交易载荷；任一材料缺失均记为 BLOCKED。 |
| AC-12 | 项目自有类型遵守命名转换和值语义规则；固定原生依赖仅按 constitution 8.7 保留第三方契约。任何未覆盖冲突必须交 Human 决定。 |
| AC-13 | 审查项目自行声明的公开 API 和实际调用路径：不存在私钥输入、交易构造、交易对象编码、签名、节点解锁或完整上游 client 暴露。固定依赖包含的未调用能力及返回原生 DTO 在上游定义的方法按 constitution 5.4 处理。说明性 go-wallet-sdk 示例仍须 module 隔离、默认离线且无真实凭据。 |

### 4.1 值语义与契约冲突裁决 / Value Semantics and Contract Decisions

PLAN-v7-REMAINING-RAW-v1 对 Bitcoin、Solana、Tron 的响应 DTO 使用原始字段与必要转换，精确规则以 plan §3.3–§3.5 为准。可选请求的 Has 标记只控制 wire 省略；可选响应字段优先使用 `json.RawMessage`，禁止由通用 Has 规则重新引入全字段 schema。

DTO 成功解码不承诺节点响应满足完整官方 schema。普通字段的缺失/null 按 `encoding/json` 处理；仅本方案逐项列明的顶层 nullable、金额转换、结果存在性和提交判定边界允许必要校验。`RawMessage` 的 nil、JSON null 与显式零/false 不合并。

EVM-RAW-DTO-v1 获批后，EVM 的可选请求字段、响应 DTO 和校验范围以 §4.2 及 plan §3.2 为准，替代本节对 EVM 的统一 Has 规则；其他链未被本修订替代的值语义契约不变，所有链错误遵循 §4.4。Human 当前明确要求遵循 constitution 1.4 中的 §2.4–§2.5 防御分支与注入依赖约束，以及 §7 原始错误边界。

`PLAN-v6` 依据 constitution 8.5/8.6，将 SDK 自有的小字段、请求和响应 DTO 统一定义为值语义。可选请求字段使用字段值加同名 `Has<Field> bool`；nullable 方法使用链原生 DTO 值加 `present bool`，不得使用小 DTO 指针表示缺失。

`Has<Field>=false` 时该字段不进入 RPC 请求，其对应值不参与业务校验；为 true 时保留并校验实际值，包括显式零。费用字段互斥依据 Has 标记判断，不依据数值是否为零。

合法空结果表示为 `(零值, false, nil)`；有效结果表示为 `(值, true, nil)`；失败表示为 `(零值, false, error)`。`present` 只表示该方法结果存在，不表示交易成功、已执行或已确认。错误信封、缺失必需字段和不合法结果不能借 `present=false` 转换为成功；Tron 空账户仍须与存在但余额为零的账户区分。

Bitcoin `EstimateSmartFee` 除外：其 bool 表示 `hasEstimate`；收到无估值结果时保留节点 Blocks/Errors 并返回 false，见 BC-03。

错误保持值语义；可选数字码继续使用 `RPCCode int + HasRPCCode bool`，包括显式零。原始错误信息、格式与解包契约以 §4.4 RAW-ERROR-v1 为准，替代 CONTEXT-CAUSE-v1 中“只保存两个哨兵、其他输入清空”的限制。保留已有 `WithContextCause(error) Error` 签名，但它改为原样设置 Cause 的值副本方法，不归一化、不筛选错误。

`errors.As` 提取 `apierror.Error` 值；底层第三方 error 通过 interface 原样保留，不因此授权 SDK 新增小 DTO 指针或批量改写既有指针。

保留复杂 Client 实现对象的指针。`HTTPDoer.Do(*http.Request) (*http.Response, error)` 是本次提交 Human 明确裁决的唯一标准库指针签名；批准该签名不授权其他 SDK DTO 使用指针，也不构成“第三方或标准库类型均可使用指针”的通用例外。

上述规则是历史 `PLAN-v6` 的值语义裁决；RAW-ERROR-v1 对错误字段、Error() 格式和 Cause 的明确变化以 §4.4 为准。Human 批准 `PLAN-v6` 时对返回形式及 HTTPDoer 签名的裁决继续有效；新增冲突仍按 constitution 8.6/9.4 停止并提交 Human。

### 4.2 EVM-RAW-DTO-v1 验收 / Acceptance（候选）

Human 已选择“原始 RPC DTO + 必要转换”。本修订保留全部 14 个方法、显式 Caller 注入、独立 Ethereum/BNB 客户端及原始信息可达的结构化错误/单次提交语义；精确类型和签名见 [plan §3.2](plan.md#32-evm-raw-dto-v1-首版契约--first-release-contract)。

| 条款 | 可观察验收 | 追踪 |
|---|---|---|
| ER-01 API/注入 | Address/Hash/Data 改为命名 string 并提供 ToString；保留 New、BlockRef 构造入口、14 方法及显式 ValidateNetwork。构造只拒绝 nil Caller，不反射探测 typed-nil，零 I/O，无默认配置或客户端。 | AC-02/05/12；T025→T026 |
| ER-02 原始 DTO | 完整保留 plan 字段表；数量/金额保持 hex string，切片保持顺序。可选/nullable 响应字段用 json.RawMessage 保留缺失、null、显式零/false；不使用 *bool 或 Has 字段。顶层 nullable 三态不变。 | AC-05/12；T025→T026 |
| ER-03 必要转换 | 既有标量方法继续返回 uint64 或 big.Int；转换无浮点、无截断，Config 大整数深拷贝。DTO 数值解释和单位展示归调用方；无整份 DTO 数值模型转换。 | AC-05/12；T025→T026 |
| ER-04 简化边界 | EVM 使用标准 json.Unmarshal 解码，不做全字段 schema、duplicate-key、响应嵌套 hex/quantity 或 FeeHistory 维度验证。保留请求 wire 校验、必要标量转换、顶层 null/JSON 类型与提交 hash 检查。删除 typed-nil context 反射及无依据 fallback/default/recover。 | AC-02/04/05；T025→T026 |
| ER-05 两链/提交 | 两链分别证明 14 方法接线、1/56 身份样例与 mismatch；提交 payload 原样一次发送，非零 hash 原样返回，明确拒绝与未知结果不混淆；节点字段和底层错误链按 RE-01–RE-07 保留，业务映射与脱敏归调用层。 | AC-02/04/05/13；T025→T026 |

ER-04 明确替代旧 EC-02/04 的“所有必需字段缺失或坏格式一律 Protocol”与全 DTO 严格校验承诺：成功解码表示获得原始 DTO，不保证节点返回完整或符合执行客户端完整 schema。普通 string 字段缺失/null 为 `""`，不会合成 `"0x0"`；具有原生缺失/null 区别的字段按表使用 RawMessage。未知字段忽略；重复键及大小写匹配采用 encoding/json 行为。此验收变化须包含在 Human 批准中，Coder 不得自行决定。

是否已发布或已有外部使用者未知。类型底层表示、DTO 字段及 Has/Parse API 删除属于破坏性变化；不假定“尚未发布所以安全”，不自动添加兼容层。

本段关于 stale/reopened 的描述是 EVM-RAW-DTO-v1 与 RAW-ERROR-v1 提出时的历史过程。当前以 PLAN-v7-REMAINING-RAW-v1+PARALLEL-v1 状态为准：T017–T035 已完成离线实现、验证与独立审计；T036 真实验收因缺少授权和材料保持 `BLOCKED`。

### 4.3 剩余链原始 DTO 契约 / Remaining Native DTO Contracts（候选）

BITCOIN-CONTRACT-v1 从未获批，其严格解析设计由本候选替代。Bitcoin、Solana、Tron 与 test-only runner 的精确 Go 声明和 wire 契约见 plan §3.3–§3.5、§7.8。候选落盘并不授权实施；Human 批准后仅恢复 T027 RED。

| 条款 / Clause | 可观察验收 / Observable acceptance | 追踪 / Trace |
|---|---|---|
| BC-01 | Bitcoin 十方法、固定 verbosity、精确位置参数及单次调用；构造注入验证仅在 New，context 原样传递。 | AC-02/06/12/13；T027→T028 |
| BC-02 金额 / Exact amounts | Satoshi、SatoshiPerKVByte 为 int64 且有 ToInt64；BTC/BTC-kvB 原始数字精确乘10^8，不经浮点、不舍入。1e-8 BTC=1sat，0.00001000 BTC/kvB=1000sat/kvB；非整数sat、负数、溢出失败。 | AC-06/12；T027→T028 |
| BC-03 | 原始 DTO 按 plan §3.3 字段表；可选字段 `RawMessage` 保留缺失/null/零；UTXO 顶层 null 三态；无估值保留 Blocks/Errors。 | AC-06/12；T027→T028 |
| BC-04 历史与拒绝 / History | blockhash 可选，历史依赖 mempool/txindex/区块可用性；不探测、不 fallback，不吞节点错误。预检 allowed=false、提交明确 RPC 拒绝与结果未知分别保留。 | AC-02/06/13；T027→T028 |
| BC-05 | 标准 JSON 解码；不做 duplicate-key 或全字段 schema；必要错误、预检 allowed、提交 txid/unknown 按 plan §3.3/§4。 | AC-02/04/06；T027→T028 |

| SOL-01 | Solana 十一方法、显式选项、固定 wire；原始 encoded payload 不改写，构造零 I/O。 | AC-02/07/12/13；T029→T030 |
| SOL-02 | 保留 context slot、legacy/v0、meta、账户 data、nullable value 和 lastValidBlockHeight；整数金额无浮点。 | AC-07/12；T029→T030 |
| SOL-03 | GetBlock/GetTransaction 的顶层 null 表示不存在；账户/费率 envelope 的 value:null 保留原 context，不伪造零。 | AC-07；T029→T030 |
| SOL-04 | 模拟 err 是原生结果；send preflight RPC error 保留 code/message/data/Cause；节点 maxRetries 与 SDK 单次请求区分。 | AC-04/07；T029→T030 |
| SOL-05 | 标准解码、明确请求校验、提交签名校验；不解析/构造 message，不刷新 blockhash，不签名或轮询。 | AC-02/07/13；T029→T030 |
| TR-01 | Tron 十三查询、两计算、一提交的精确 POST 路径；五个 solidified 方法独立接线。 | AC-02/08/12/13；T031→T032 |
| TR-02 | 地址编码与 visible 原样传递；空对象账户与 address 存在的零余额账户区分；sun、Energy、Bandwidth 不混为金额。 | AC-08/12；T031→T032 |
| TR-03 | HTTP 200 内 Error、计算 result 与广播拒绝保留原始 code/message/JSON；既有依赖 error 完整可达。 | AC-04/08；T031→T032 |
| TR-04 | 原始交易 JSON 作为 `RawMessage` 提交一次，不重建 raw_data、签名、TAPOS 或 txID；RPC 接受不等于执行或 solidified。 | AC-08/13；T031→T032 |
| TR-05 | 标准 JSON 解码，仅顶层对象、明确存在性及提交判定检查；不做完整 protobuf/schema 校验。 | AC-02/08；T031→T032 |
| LR-01 | 五链 runner 使用固定方法目录和具体客户端；关闭开关零配置/材料/网络读取。 | AC-09；T033→T034 |
| LR-02 | manifest 摘要独立冻结；真实预期来自浏览器与独立账号 RPC，runner 不用被测 SDK 生成预期。 | AC-09/10；T033→T036 |
| LR-03 | 历史字段等值与动态状态分开；漏方法、漏样本、缺预检查不能 PASS。 | AC-09/10；T033→T036 |
| LR-04 | 提交须双开关、逐链授权、专用材料和身份检查；身份失败禁止提交；在已有 evidence/broadcast.json 追加并同步成功的 claim 先于发送，同一授权或 payload 已消费时禁止再发；未知结果仅按原 ID 查询一次，不补发。账本单写者由 T036 Controller 门禁保证，损坏或无法持久化时 BLOCKED。 | AC-09/11；T033→T036 |
| LR-05 | 原始 SDK error 仅在内存断言；证据不序列化原始错误、配置或载荷。read.json 为固定 suiteReport JSON，broadcast.json 为追加式 claim/report JSON Lines；精确字段见 plan §7.8，不新增证据路径。 | AC-04/09，RE-08；T033→T036 |
| LR-06 | harness 的限额、deadline、账本原样复用，不新增调度、重试或后台 worker。 | AC-09；T033→T034 |

本候选不增加链级 Raw 入口、网络探测、钱包/签名能力、依赖、默认客户端或重试。每个测试 Task 完成后仍须固定 Auditor 与 Human 门禁，对应实现 Task 不自动启动。PARALLEL-v1 只调整独立任务的执行波次，具体依赖与授权来源见 plan §5/§9，不改变本节任何行为验收。五链真实验收只在另行授权的 T036 执行。

### 4.4 RAW-ERROR-v1 验收 / Raw Error Acceptance（已批准）

| 条款 | 可观察验收 | 任务 |
|---|---|---|
| RE-01 原始字段 | JSON-RPC Message 保留解码后的原字符串，不 trim、改写或分类；Data 为原始 JSON value，缺失为 nil、显式 null 为 `null`，数字不经浮点转换；数字码包括零，链字符串码原样保留。 | T017–T020、T025–T032 |
| RE-02 底层错误 | 实际获得的底层 error 原样保存为 Cause；Unwrap 返回它；普通 sentinel、包装 error、类型化 error 和已有多错误链均可由 `errors.Is/As` 到达。 | T017–T022、T025–T032 |
| RE-03 HTTP 正文 | HTTP 错误的原始响应字节可读取，允许非 JSON；不把字节强制塞入 JSON Data。正文最多保留 MaxResponseBytes，检测到超限须标识截断；部分读取失败保留已读字节及读取错误。 | T019–T022 |
| RE-04 格式与责任 | `Error()` 按 plan §4 固定规则提供原始 Message 或 Cause 文本；Data/正文保留在结构化字段，不自动拼进字符串。SDK 错误不是已脱敏日志对象。 | T017/T018、T035 |
| RE-05 不主动追加 | SDK 不将 endpoint/Header/request/payload 另行拼接或存入错误；节点或底层 error 自带的相同合成内容仍完整可达，不做扫描、正则替换或内容清空。 | T019–T022、T025/T026 |
| RE-06 分类与 context | Kind 依据发生阶段、HTTP 状态及有效协议结构确定；不得按英文 message 推断 Unsupported、未找到、拒绝或可重试。已获得错误的原始 context 链不折叠，不设置“取消优先”来丢弃另一分支。 | T017–T022、T025–T032 |
| RE-07 提交未知 | 本地拒绝未调用依赖时 false；进入依赖后的 Transport/HTTP/Protocol、取消及畸形成功结果为 true；可信节点明确拒绝为 false。信息保留不改变一次提交与禁止自动重发。 | T025–T032 |
| RE-08 输出隔离 | live callback 可以在内存中检查原始 SDK 错误；harness/runner 输出只保留批准的类别、ID、摘要等字段。报告、fixture、诊断不直接格式化或序列化整个 SDK error。 | T033–T036；既有 HC-07/08 |

具体 Case：

```text
输入：合法 RPC error，code=0，message="synthetic endpoint/token marker"，
data={"reason":"synthetic detail","n":9007199254740993}
结果：Kind=RPC，HasRPCCode=true、RPCCode=0；Message 原样，Data 原始 JSON
不经 float64，errors.As 可提取 Error 值。

输入：注入 Doer 返回包装 io.ErrUnexpectedEOF 的类型化 error。
结果：SDK Cause 为收到的原 error；errors.Is 命中 io.ErrUnexpectedEOF，
errors.As 可提取该类型；不主动追加配置 Header 或请求正文。
```

### 4.5 具体行为样例 / Concrete Cases

- Bitcoin `value:0.00000001` → 1 sat；`value:1e-9` → Protocol；未知 DTO 字段与可选响应字段缺失不触发全 schema 失败。
- Bitcoin `gettxout` 的 `null` → `(TxOut{}, false, nil)`；`estimatesmartfee` 无 feerate、含 blocks/errors → 保留节点对象且 `hasEstimate=false`。
- Solana `getAccountInfo` 的 `{"context":{"slot":9},"value":null}` → `Context.Slot=9`、Value 原始 null；不能返回虚构的零余额账户。
- Solana `getTransaction` 的 `version:"legacy"` 与 `version:0` 分别原样保留；`meta.err` 非 null 表示链上执行结果，不转成传输错误。
- Tron `getaccount {}` → `(Account{}, false, nil)`；`{"address":"41...","balance":0}` 与仅有 address 的合法账户均 `present=true`。
- Tron 广播 `{"result":false,"code":"CONTRACT_VALIDATE_ERROR","message":"synthetic"}` → RPC error，NodeCode/Message/Data 原样，`SubmissionUnknown=false`。
- 提交进入依赖后返回 HTTP/Transport/Protocol，或成功结果无法确定接受身份 → `SubmissionUnknown=true`，调用次数仍为一。
- live 关闭时即使注入的配置/材料 reader 会失败，也不得调用 reader。
- 正式验收缺一链样本、预检查、授权或材料 → BLOCKED；不能以四链通过或离线 fixture 代替。

## 5. 真实数据验收规则

- 浏览器候选：Etherscan、BscScan、mempool.space/Blockstream、Solana Explorer/Solscan、TRONSCAN。
- 每链至少三笔不同交易，覆盖原生转账、代币/脚本或合约交互、失败或特殊状态等代表性场景。
- 样本清单保存来源 URL、网络、交易标识、关联区块、地址、采集时间、独立 RPC 预检查结果和内容摘要；不保存 endpoint、Token 或完整交易载荷。
- 测试 RPC 由调用方通过外部配置注入，可参考 `/Users/stone/the_web3/evm-scanner-service` 的测试账号装配方式；SDK 本身不得读取该项目配置。
- 节点权限、历史能力或样本不可用时记录 BLOCKED，不静默换公共节点或随机替换样本。

### 5.1 HARNESS-CONTRACT-v1 验收边界

T023/T024 仅建立测试装配基础，不读取真实凭据、不执行真实 RPC 或广播。核心 SDK 不读取环境变量或文件，测试装配通过显式参数和注入函数读取外部配置。

冻结身份使用 SHA-256 覆盖 manifest 原始文件全部字节，不规范化、不忽略时间元数据；任意字节变化必须重新冻结并经既有样本审批流程审核。该条替代 architecture §12.5 中有关规范化摘要的草案，不修改 constitution。

并发上限采用可取消的阻塞等待；总执行时限必须体现为传给 case/operation 的 context deadline。正式总体状态采用 FAIL 优先，其次 BLOCKED；只有全部独立 required 项 PASS 且无基础错误才可 PASS。精确配置、接口、预算、错误和测试契约以 plan §7.2 HC-01–HC-08 为准。

## 6. 非目标

- 地址交易历史索引、任意地址 UTXO 索引、数据库和缓存。
- WebSocket 订阅、多节点路由、自动容灾、自动费用策略和交易生命周期编排。
- HTTP 服务、CLI 服务、部署与运维平台。

## 7. 追踪关系

| Plan | 活动任务 / Active Tasks | 覆盖 / Coverage |
|---|---|---|
| PLAN-001 | T017–T024 | 公共协议与离线 harness；AC-01–04、09、12、13 |
| PLAN-002 | T025–T026 | Ethereum/BNB 全部既定方法；AC-02、05、12、13；真实验收进入 T036 |
| PLAN-003 | T027–T028 | Bitcoin 全部既定方法；AC-02、06、12、13；真实验收进入 T036 |
| PLAN-004 | T029–T030 | Solana 全部既定方法；AC-02、07、12、13；真实验收进入 T036 |
| PLAN-005 | T031–T036 | Tron、五链 runner、文档与最终验收；汇总 AC-01–13 最终证据 |

C-01–C-10 是所有活动 Task 的不变量；C-08/C-09/C-10 分别由 AC-12/AC-12/AC-13 验收。真实验收只在 T036 执行，离线通过不得表述为 AC-10/AC-11 已通过。

历史状态（PLAN-v7）：T017–T035 已完成对应离线 RED/GREEN、验证和独立 Auditor 门禁。BITCOIN-CONTRACT-v1 从未获批，其内容已由 PLAN-v7 替代。T036 的真实数据、浏览器采集、账号 RPC 和受控提交仍须另获具体授权并提供完整材料，当前为 `BLOCKED`；推送、PR 与真实 RPC/广播仍未授权。当前权威状态见文首及 §8。

## 8. PLAN-v8-NATIVE-SDK-v1 原生 SDK 迁移

### 8.1 输入、目标与兼容性

Human 已批准 NATIVE-SDK-GOVERNANCE-v1，要求 EVM/Ethereum/BNB 改用 go-ethereum，Solana 直接改用 solana-go/v2，不做试点、不保留兼容层。迁移目标是删除项目重复维护的高层请求/响应类型和 RPC 组装逻辑，同时保留库形态、显式 DI、原始错误和已编码载荷提交边界。

固定版本为 go-ethereum v1.17.10、solana-go/v2 v2.3.0；go.mod 使用 go 1.26.0。不追随 main/latest，不添加 replace/fork。升级或增加直接依赖须另行审批。

场景：
1. 宿主注入已有 jsonrpc.Caller，直接创建 Ethereum/BNB/Solana 客户端；构造不发请求，链查询返回固定上游原生类型。
2. 宿主提供已签名且已编码的载荷；客户端只提交一次，返回原生 hash/signature，保留拒绝和结果未知差异。
3. Caller 返回原始节点错误或底层错误；调用方通过 errors.As/Is 仍能取得原始字段和 Cause。
4. 普通测试用 fake 或本地 httptest 验证 native→bridge→Caller→jsonrpc 的路径，不访问公网。
5. 现有 integration 入口继续默认关闭；材料字典迁移不构成真实执行授权。

破坏性变化：旧 EVM/Solana 自研 DTO、枚举、Has 字段、present 返回值、旧方法名及其测试预期删除；采用 plan §10 的唯一 API。第三方原生 pointer/null/default/NotFound 语义生效。Bitcoin、Tron 和公共协议 API 不变。

### 8.2 新增验收

| ID | 需求与具体 Case | 主要证据 |
|---|---|---|
| NS-01 | 固定直接依赖和 Go 1.26；公开方法、参数和返回类型逐项符合 plan §10；不存在旧 DTO 别名、旧方法转发或完整上游 client 暴露。 | go.mod/go.sum、原生接口断言、API/调用路径审查 |
| NS-02 | evm.New、ethereum.New、bnb.New、solana.New 仅直接检查 nil 接口及既有明确配置；零 Caller/HTTP/节点调用，无 env/file/config 读取、默认网络 transport 或后台 goroutine。geth 内部 crypto/rand 熵读取单独披露，不宣称绝对零 I/O。 | fake Caller 计数、构造路径审查、无默认 transport 测试 |
| NS-03 | EVM 十三个高层方法委托固定 ethclient；原生 hex quantity、CallMsg、FilterQuery、FeeHistory、完整区块、pending bool、ethereum.NotFound 和原生类型保持上游语义。完整区块含叔块时允许一次块查询加按序叔块查询。 | 表格单元测试、完整块/叔块 fixture、原生错误断言 |
| NS-04 | EVM SendRawTransaction 原样发送合法非空十六进制载荷一次，返回 common.Hash；本地非法载荷零请求；节点拒绝不是结果未知；进入提交后的非节点拒绝错误及无效/零 hash 标记结果未知。 | 载荷、次数、拒绝/传输/解码/零 hash 测试 |
| NS-05 | Solana 十一个批准方法使用原生参数和结果；保留 Get*WithOpts、显式零指针、context、legacy/v0、nullable、ErrNotFound/ErrNotConfirmed 和上游默认值。模拟只接收已序列化字节。 | 表格测试、legacy/v0/null fixture、精确 wire 断言 |
| NS-06 | Solana SendEncodedTransactionWithOpts 原样提交调用方编码字符串一次；编码为空按上游 base64 默认，支持 base64/base58；保留 SkipPreflight/PreflightCommitment/MaxRetries/MinContextSlot。非法输入零请求，无效/零 signature 与提交后不确定错误标记结果未知。 | 载荷、选项、一次调用、拒绝/未知测试 |
| NS-07 | 两种桥接私有且不联网；原始 apierror 字段和底层 errors.Is/As 可达，合成 URL 不含配置；未支持的 Solana callback/batch 返回 Unsupported 且零调用。不得通过错误信封重编码丢失 raw data、HTTP body 或 Cause。 | 原始错误合成标记、body/data 字节、cause 链、unsupported 测试 |
| NS-08 | integration 的 EVM/Solana native 方法目录、材料字典和断言同步迁移；每次真实 Caller 调用恰好消耗一次请求预算，包括 EVM 叔块读取；预算不足为 BLOCKED 且不额外发请求。开关关闭仍零读取/零请求。Bitcoin/Tron 行为不变。 | read/submission contract 测试、三调用/二预算 fixture、默认关闭回归 |

NS-01/02/07 同时服务两个功能 Plan。原 AC-01–04、AC-06、AC-08–11、RAW-ERROR-v1 仍有效；AC-05/07/12/13 按本次修订生效。原 ER 和 Solana SC 条款仅在被本章替代部分成为历史，不扩大其他链变更范围。

### 8.3 最小使用方式

EVM：宿主先构造 jsonrpc.Caller，再用 ethereum.New 或 bnb.New 注入 ExpectedChainID；显式调用 ValidateNetwork；使用 common.Address、*big.Int、ethereum.CallMsg 等原生参数。SendRawTransaction 接收业务已编码的字符串。

Solana：宿主通过 solana.New(Config{}, caller) 构造受限客户端；读取方法使用原生 rpc options；模拟传入已序列化字节；提交传入已编码字符串及 rpc.TransactionOpts。项目不提供 Transaction 对象构造、签名或编码入口。

### 8.4 非功能要求与非目标

继续保持无后台、无重试、无轮询、无 failover、无日志实例、无统一适配器。桥接只承担本轮批准方法所需的协议连接，不导出通用设施。

非目标：迁移 Bitcoin/Tron 或公共协议、提供旧 API 兼容层、公开完整原生 client、添加新链、新方法、订阅、批处理公共 API、交易构造/签名、钱包 module、部署、真实 RPC、广播和 Git 操作。

### 8.5 双向追踪

| 功能 Plan | 需求/AC | 活动任务 | 验证贡献 |
|---|---|---|---|
| PLAN-002 原生 EVM/Ethereum/BNB | NS-01/02/03/04/07/08，AC-01–05/09/12/13 | T037→T038→T039 | 原生 EVM、桥接、身份、提交、EVM integration |
| PLAN-004 原生 Solana | NS-01/02/05/06/07/08，AC-01–04/07/09/12/13 | T040→T041→T042 | 原生 Solana、桥接、提交、Solana integration、最终离线验收 |
| 保留真实验收 | AC-10/11 | T036（BLOCKED） | 不属于本轮已授权执行范围 |

每个 Task 必须反向列出上表需求；不存在无需求 Task、无 Task 的新增需求或通过删除旧验证降低未变更链验收的情况。
