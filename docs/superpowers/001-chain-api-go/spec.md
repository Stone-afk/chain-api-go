# chain-api-go 需求规格

**规格文档 / Spec**：[spec.md](spec.md)  
**技术方案 / Plan**：[plan.md](plan.md)  
**任务清单 / Tasks**：[tasks.md](tasks.md)  
**基线版本 / Baseline Version**：PLAN-v6+CONTEXT-CAUSE-v1+HARNESS-CONTRACT-v1+EVM-CONTRACT-v1
**日期 / Date**：2026-11-16  
**状态 / Status**：已批准；T026 已通过，T027 可启动（Approved; T026 Passed / T027 May Start）

## 0. 变更记录 / Change Log

| 版本 / Version | 日期 / Date | 变更 / Change |
|---|---|---|
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
| C-08 | 命名常量类型按底层类型提供明确转换方法：`string` → `ToString()`、`int` → `ToInt()`、`int32` → `ToInt32()`、`int64` → `ToInt64()`；其他底层类型遵循同一规则。 |
| C-09 | 小字段、小结构体及其参数、返回值和构造器传参使用值语义。仅复杂生命周期实现对象可使用指针；不得借此批量改写既有指针或破坏数据库 `NULL`、第三方 API 契约，冲突必须交由用户决定。 |
| C-10 | 核心 SDK 不接收私钥，不构造、编码或签名交易，不调用节点签名或解锁接口。`go-wallet-sdk` 若用于说明组合，只能位于独立示例 module，且示例默认离线、不得默认广播或包含真实凭据。本约束是技术隔离边界，不是钱包业务职责。 |

## 3. 功能范围

### 3.1 公共协议能力

- 注入式 JSON-RPC 客户端，支持 JSON-RPC 2.0 及 Bitcoin Core 所需方言。
- 注入式 HTTP JSON POST 客户端，供 Tron 原生路径调用。
- context 取消、响应大小限制、响应 body 关闭、并发安全请求 ID。
- 结构化安全错误：配置、参数、传输、HTTP、协议、节点 RPC、不支持、网络不匹配。
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
| AC-02 | nil/typed-nil 依赖、非法配置和非法参数在请求前返回明确错误；构造过程零网络 I/O。 |
| AC-03 | JSON-RPC/HTTP JSON 正确处理 ID、HTTP 状态、错误信封、超限响应、取消和 body 关闭；没有隐式重试。 |
| AC-04 | 错误对象只保留安全的机器可读分类与代码，不包含节点 URL、鉴权 Header、完整正文或任意底层敏感文本。 |
| AC-05 | Ethereum 与 BNB 分别完成全部查询、计算、提交契约及网络身份检查。 |
| AC-06 | Bitcoin 金额不经过浮点，历史查询前提、null 和节点拒绝保持原义。 |
| AC-07 | Solana 保留 commitment、context slot、legacy/v0、meta、blockhash 有效高度及提交选项。 |
| AC-08 | Tron 正确处理地址编码、visible、空账户、HTTP 200 内链错误、资源单位与 solidified 状态。 |
| AC-09 | 普通测试零公网；真实 live 入口仅在显式启用后读取指定外部配置与材料。测试装配按 plan §7.2 HARNESS-CONTRACT-v1 实现严格路径/env 引用解析、原始字节 SHA-256、请求数、可取消并发等待、开始间隔和真实 deadline context；独立必需目录与 FAIL 优先账本禁止漏跑、阻塞或错误被 PASS 掩盖。关闭开关零配置读取、零材料读取、零请求 I/O。 |
| AC-10 | 每链至少采集 3 笔区块链浏览器真实交易，先以用户账号 RPC 独立预检查，再通过 SDK 逐字段核对；不得用公共 RPC 冒充用户账号或用 SDK 自己生成全部预期。 |
| AC-11 | 五链提交验收必须独立授权并使用测试网或受控节点及专用交易载荷；任一材料缺失均记为 BLOCKED。 |
| AC-12 | API 审核确认命名常量具有匹配底层类型的 `To<Type>()` 方法，新增或触及的小字段/小结构体遵循值语义；契约冲突必须有用户决定。 |
| AC-13 | 核心公开 API 和依赖图不存在私钥、交易构造、编码、签名或节点解锁能力；若存在说明性 `go-wallet-sdk` 示例，其 module 隔离、默认离线且无真实凭据。 |

### 4.1 值语义与契约冲突裁决 / Value Semantics and Contract Decisions

`PLAN-v6` 依据 constitution 8.5/8.6，将 SDK 自有的小字段、请求和响应 DTO 统一定义为值语义。可选请求字段使用字段值加同名 `Has<Field> bool`；nullable 方法使用链原生 DTO 值加 `present bool`，不得使用小 DTO 指针表示缺失。

`Has<Field>=false` 时该字段不进入 RPC 请求，其对应值不参与业务校验；为 true 时保留并校验实际值，包括显式零。费用字段互斥依据 Has 标记判断，不依据数值是否为零。

合法空结果表示为 `(零值, false, nil)`；有效结果表示为 `(值, true, nil)`；失败表示为 `(零值, false, error)`。`present` 只表示该方法结果存在，不表示交易成功、已执行或已确认。错误信封、缺失必需字段和不合法结果不能借 `present=false` 转换为成功；Tron 空账户仍须与存在但余额为零的账户区分。

安全错误中的可选数字 RPC code 使用 `RPCCode int` 与 `HasRPCCode bool`。SDK 不保存或返回 apierror 的原始 Message、Data 或任意底层 Cause；仅保留安全分类、方法、状态、节点代码和提交未知标志。取消/超时链只能保存精确的 `context.Canceled` 或 `context.DeadlineExceeded` 哨兵。

`func (e Error) WithContextCause(err error) Error` 是跨包设置上述安全哨兵的唯一批准入口。它返回 `Error` 值副本：`errors.Is(err, context.Canceled)` 为真时保存精确 `context.Canceled`；否则匹配 deadline 时保存精确 `context.DeadlineExceeded`；其他输入清空 cause。两者同时匹配时取消优先，不得保存传入包装错误或其文本。

保留复杂 Client 实现对象的指针。`HTTPDoer.Do(*http.Request) (*http.Response, error)` 是本次提交 Human 明确裁决的唯一标准库指针签名；批准该签名不授权其他 SDK DTO 使用指针，也不构成“第三方或标准库类型均可使用指针”的通用例外。

上述规则只改变 Go 表达方式，不改变 v5 的链原生结果、合法 null、错误、提交未知和确认语义。Human 批准 `PLAN-v6` 时须明确接受这些字段、返回形式及 HTTPDoer 签名裁决；新增冲突仍按 constitution 8.6/9.4 停止并提交 Human。

### 4.2 EVM-CONTRACT-v1 验收 / Acceptance

本修订响应 T025 固定审计的两项 P1（跨包构造缺口、DTO/字段断言不完整）和两项 P2（大整数所有权、wire/非有限浮点覆盖不足）。AC-05 的“全部查询”指全部既定方法及下列首版类型化字段，未知扩展字段仍可忽略，不等于复制完整执行客户端 Schema。精确 Go 字段、wire 必需性和方法签名冻结在 [plan §3.2](plan.md#32-evm-contract-v1-首版契约--first-release-contract)。

| 条款 / Clause | 可观察验收 / Observable acceptance | 追踪 / Trace |
|---|---|---|
| EC-01 构造 / Construction | `evm.New(jsonrpc.Caller) (*evm.Client,error)` 是最小跨包构造入口；拒绝 nil/typed-nil，零 I/O。Ethereum/BNB 仍由各自 New 创建、保存独立预期网络身份并显式 ValidateNetwork；不暴露 rpc 字段或建立统一链入口。 | AC-02/05/12；T025→T026 |
| EC-02 查询 / Queries | 已确认区块保留父 hash、时间、gas 与交易 hash 列表；交易保留 from/to、nonce、value、input、gas/费用、区块与交易索引；回执保留执行状态或历史 root、gas/费用、合约地址与日志；日志保留完整区块/交易/日志索引关联。完整字段值由 plan §3.2 表冻结，不能只检查 present 或长度。合法 pending/null 与明确零分别表达；缺必需字段或坏格式为 Protocol。 | AC-05/12；T025→T026 |
| EC-03 所有权 / Ownership | 输入 Config/CallRequest 的大整数和切片不被 SDK 修改；保存的配置深拷贝。所有大整数结果及 FeeHistory 的嵌套金额元素独立拥有存储；修改一个结果不得污染其他字段、结果、fake 保存数据或后续调用。 | AC-05/12；T025→T026 |
| EC-04 编码与费用 / Wire and fees | Quantity 最短十六进制、Data 偶数位、地址/hash 定长；嵌套 DTO 与列表元素同样校验。FeeHistory 百分位只接受有限、0–100、非递减数值，NaN/±Inf 在 Caller 前拒绝；返回维度与金额/比例合法，允许节点返回少于请求的历史范围。 | AC-02/05；T025→T026 |
| EC-05 两链与提交 / Concrete clients and submission | 两链分别验证完整方法接线与一次调用；Ethereum 身份样例 1、BNB 身份样例 56（`0x38`），不将这两个值写成网络限制。提交成功 fixture 必须使用非零 hash 并逐字节断言，未知结果/本地拒绝/节点拒绝保持既有分类。 | AC-02/05/13；T025→T026 |

审批只允许恢复 T025 测试与声明骨架修订；仍需新的 RED、固定 Auditor 和 T025 Human 门禁，不能因此启动 T026。T017–T024 已批准且输入未变的证据继续有效；真实数据仍只在 T036 验收。

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

Human 已批准 PLAN-v6、CONTEXT-CAUSE-v1、HARNESS-CONTRACT-v1、EVM-CONTRACT-v1 与 T025/T026。T017–T026 已通过且相关输入未变的证据继续有效；T027 可进入 Bitcoin 测试 RED。本次无真实 RPC 或广播授权，不新增 Plan/Task，不修改 constitution 1.1。
