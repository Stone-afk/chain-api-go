# chain-api-go 技术方案

**规格文档 / Spec**：[spec.md](spec.md)  
**技术方案 / Plan**：[plan.md](plan.md)  
**任务清单 / Tasks**：[tasks.md](tasks.md)  
**基线版本 / Baseline Version**：PLAN-v7-REMAINING-RAW-v1
**日期 / Date**：2026-11-17
**状态 / Status**：PLAN-v7 已获 Human 批准；T017–T027 已完成并集成；T028 已授权进入实现，T029–T036 仍阻塞 / T028 authorized

## 0. 变更记录 / Change Log

| 版本 / Version | 日期 / Date | 变更 / Change |
|---|---|---|
| PLAN-v7-REMAINING-RAW-v1 | 2026-11-17 | Human 已批准：同步 T017–T026 完成事实；替代未批准 BITCOIN-CONTRACT-v1 的严格解析设计；冻结 T027–T036 全部接口、DTO、wire、runner、文档与最终验收。保持 5 Plan / 20 Task、全部稳定 ID 和依赖，无新增依赖或核心配置来源。T027 已完成 RED、独立 Auditor 与 Human 门禁并获准集成；T028 已单独授权。真实 RPC 与广播仍未授权。 |
| RAW-ERROR-v1 | 2026-11-17 | Human 已批准：恢复原始节点 Message/Data、HTTP 错误正文及底层 Cause；区分 SDK 返回与 live 脱敏输出。重开 T017–T022，修订 T025/T026；T025 当前 RED/审核不能作为新基线门禁。T023/T024 未受影响证据保留，SDK→harness 输出隔离在 T033/T034 验证。保持 5 Plan / 20 Task、T017–T036 ID 与原路径；T027 继续暂停。constitution 1.3 同时获批；T017/T018 已完成门禁并获准集成，T019 未授权。 |
| EVM-RAW-DTO-v1 | 2026-11-17 | Human 已批准：将 EVM 收敛为原始 RPC DTO + 必要转换，接受破坏性 API 变更且不提供兼容层；重开 T025/T026，旧证据 stale。T025 可进入 RED，T026 仍等待审核和 Human 门禁；Bitcoin 候选暂停。保持 5 Plan / 20 Task、路径和依赖，不改 constitution。 |
| BITCOIN-CONTRACT-v1 | 2026-11-17 | 候选：补齐 PLAN-003 十方法、DTO、金额/费率、历史和 wire 契约；T027 暂停待批准。只改三件套；保持 5 Plan / 20 Task、全部依赖、T027 七路径、T028 五路径和 T017–T026 有效证据。 |
| EVM-CONTRACT-v1 | 2026-11-16 | Human 已批准：新增最小 `evm.New` seam，冻结首版 EVM DTO 与 EC-01–05 的精确验收；修订 T025 的声明预算及 20/20/30 分钟时箱。仅影响 PLAN-002 契约；T025 可恢复、T026 未启动，既有 5 Plan / 20 Task 和 T017–T024 不变。 |
| HARNESS-CONTRACT-v1 | 2026-11-16 | Human 已批准：冻结 test-only live harness 的显式配置、raw-byte SHA-256、可取消并发等待、总 deadline context、结果优先级与离线并发验证；只重开 T023/T024，保持 5 Plan / 20 Task、任务 ID 与依赖不变。T017–T022 的未受影响证据保持有效。 |
| CONTEXT-CAUSE-v1 | 2026-11-16 | Human 批准安全的跨包 context cause 值方法；重新执行 T017 RED、T018 GREEN 及固定 Auditor 门禁，不增加 Task。 |
| PLAN-v6 | 2026-11-16 | 将 v5 的混合 Task 改为独立测试/实现依赖，活动任务为 T017–T036；补充默认拒绝路径、预算、证据门禁和 constitution 8.6 冲突裁决。功能范围不变；Human 已明确批准本修订。 |

## 1. 设计结论

采用“共享协议层 + 五链原生客户端 + 测试专用 live harness”的结构。共享只发生在传输、错误和 EVM 原生能力层，不引入链适配器、统一访问入口或业务流程。

```mermaid
flowchart LR
    Host[业务服务 / 基建服务]
    ETH[Ethereum Client]
    BNB[BNB Client]
    BTC[Bitcoin Client]
    SOL[Solana Client]
    TRON[Tron Client]
    RPC[注入的 HTTPDoer / Caller / Poster]
    Nodes[各链节点 RPC]

    Host --> ETH
    Host --> BNB
    Host --> BTC
    Host --> SOL
    Host --> TRON
    ETH --> RPC
    BNB --> RPC
    BTC --> RPC
    SOL --> RPC
    TRON --> RPC
    RPC --> Nodes
```

## 2. 包结构

```text
transport/                 # HTTPDoer 接口
apierror/                  # 保留原始信息的结构化错误
jsonrpc/                   # JSON-RPC 2.0 / Bitcoin 方言
httpjson/                  # Tron 同 endpoint HTTP JSON
evm/                       # Ethereum/BNB 共用原生类型与方法
ethereum/                  # Ethereum 具体客户端
bnb/                       # BNB 具体客户端
bitcoin/                   # Bitcoin Core 客户端
solana/                    # Solana 客户端
tron/                      # Tron 客户端
internal/test/live/        # 测试专用真实数据装配与证据
```

依赖方向固定为：具体链包 → 协议包 → `transport`；`ethereum`/`bnb` → `evm` → `jsonrpc`；`tron` → `httpjson`。核心包不得反向依赖 live 测试。

## 3. 公共代码契约

```go
type HTTPDoer interface {
    Do(*http.Request) (*http.Response, error)
}

type Caller interface {
    Call(context.Context, string, any, any) error
}

type Poster interface {
    Post(context.Context, string, any, any) error
}

jsonrpc.New(jsonrpc.Config, transport.HTTPDoer) (*jsonrpc.Client, error)
httpjson.New(httpjson.Config, transport.HTTPDoer) (*httpjson.Client, error)

ethereum.New(ethereum.Config, jsonrpc.Caller) (*ethereum.Client, error)
bnb.New(bnb.Config, jsonrpc.Caller) (*bnb.Client, error)
evm.New(jsonrpc.Caller) (*evm.Client, error) // EVM-CONTRACT-v1 跨包构造 seam
bitcoin.New(bitcoin.Config, jsonrpc.Caller) (*bitcoin.Client, error)
solana.New(solana.Config, jsonrpc.Caller) (*solana.Client, error)
tron.New(tron.Config, httpjson.Poster) (*tron.Client, error)
```

规则：

- 构造函数复制可变配置并拒绝普通 nil；不反射探测接口内部 typed-nil，只做有契约依据的本地校验。
- `context.Context` 是每个 I/O 方法首参；调用方必须提供有效 context，公共协议层负责验证与取消。EVM 不重复该检查。
- 金额不使用浮点；EVM 原始 DTO 保留原生 hex string，必要标量转换使用整数或 `big.Int`。
- nullable 字段必须保留协议语义，不得把缺失解释为零。
- RawCall/RawPost 复用同一限长、context 和原始错误保留规则。
- 命名常量类型按底层类型提供 `ToString()`、`ToInt()`、`ToInt32()`、`ToInt64()` 等明确转换方法，不提供含糊的通用 `Value()`。
- 小字段、小结构体及其参数、返回值、构造器传参默认使用值语义。仅 `Client` 等包含大量方法或持有大量依赖并负责复杂生命周期的实现对象允许使用指针。
- 指针规则不授权批量重写既有指针。数据库 `NULL` 或第三方 API 契约需要指针时，记录准确冲突并等待用户决定。
- 核心 SDK 不接收私钥，不提供交易构造、编码、签名或节点解锁能力。说明性 `go-wallet-sdk` 示例若未来获批，只能位于独立 module，并保持默认离线、无默认广播和无真实凭据；本期不要求交付该示例。

### 3.1 值语义接口裁决 / Value-Semantics API Decisions

```go
// bitcoin
func (c *Client) GetTxOut(ctx context.Context, req GetTxOutRequest) (TxOut, bool, error)
func (c *Client) EstimateSmartFee(ctx context.Context, req EstimateSmartFeeRequest) (FeeEstimate, bool, error)

// solana
func (c *Client) GetLatestBlockhash(ctx context.Context, opts BlockhashOptions) (BlockhashResult, bool, error)
func (c *Client) SimulateTransaction(ctx context.Context, tx EncodedTransaction, opts SimulateOptions) (SimulationResult, bool, error)

// tron
func (c *Client) GetAccount(ctx context.Context, req GetAccountRequest) (Account, bool, error)
func (c *Client) BroadcastTransaction(ctx context.Context, payload TransactionPayload) (BroadcastResult, bool, error)
func (c *Client) GetTransactionInfoByID(ctx context.Context, id TxID) (TransactionInfo, bool, error)
```

EVM 的请求、DTO 和必要数值转换以 §3.2 为准。其他链继续使用已批准的值语义与存在性契约；本修订不迁移其他链、不引入统一 Optional。

上述 nullable 签名的第二个返回值统一表示 `present`。合法空结果为 `(零值,false,nil)`，有效结果为 `(值,true,nil)`，错误为 `(零值,false,error)`。成功时协议要求必有结果的方法必须返回 true，不能因增加 bool 而放宽必需字段或响应合法性。

Bitcoin `EstimateSmartFee` 除外：其 bool 表示 `hasEstimate`；收到无估值结果时保留节点 Blocks/Errors 并返回 false，见 §3.3 BC-03。

具体链与协议 `*Client` 保持指针构造器和方法接收者，因为这些对象包含大量方法并持有通信依赖。`HTTPDoer.Do(*http.Request) (*http.Response,error)` 仅因兼容标准库 `http.Client.Do` 契约而保留，是本次要求 Human 明确批准的唯一标准库指针签名；不授权其他 SDK DTO 使用指针。

### 3.2 EVM-RAW-DTO-v1 首版契约 / First-Release Contract

**依据与选择。** 2026-11-17 只读观察：当前 HEAD 为 `d3502b8c151de849bfc545656ce2dd81d87d1e86`，另有未提交 `evm/client.go`；`types.go`/`query.go` 合计 632 行。参考 `/Users/stone/cndt/chain-scanner-core-go/rpc/evm/{types,client,hex}.go` 合计 216 行，但只覆盖四方法，不能直接作为本 SDK 行数目标。只借鉴原始字段与按需转换，不复制配置、logger、默认客户端、重试或 dispatcher。

| 方案 | 复杂度/性能/扩展性 | 兼容性/风险/维护 | 决定 |
|---|---|---|---|
| 保留强类型领域模型 | 立即转换全部字段，解析分支多，新字段需扩展解析器；无已测性能优势 | API 变化较少，但继续维护 Has、所有权与 schema 分支 | 拒绝 |
| 原始 RPC DTO + 必要转换 | 标准解码，业务需要时转换；新增字段只增加 DTO 字段 | 有公开 API 破坏；减少解析维护，调用方负责 DTO 数值解释 | Human 已选 |
| 完整复刻 scanner 客户端 | 当前四方法实现较小，但扩至 14 方法仍需设计 | 配置/路由/指针/默认行为不符合本 SDK 边界 | 拒绝 |

[官方 JSON-RPC 文档](https://ethereum.org/developers/docs/apis/json-rpc/)支持 Quantity/Data、pending 与合约创建 null 语义；[Go encoding/json](https://pkg.go.dev/encoding/json)定义标准解码、RawMessage、未知字段和重复键行为。访问日期 2026-11-17；公开 Go 形状及校验范围是本项目候选决定。无需新增 PoC、依赖或 benchmark，T025 的离线 fixture 提供可行性证据。

#### ER-01：公开 API

`Address`、`Hash`、`Data` 保留名称，仅作为原始 wire 字符串，不再是校验后的领域对象。此最小命名保留 constitution 8.1；DTO 内字段使用基础类型。删除 `ParseAddress`/`ParseHash`/`ParseData` 及三个类型的 `MarshalJSON`；不保留兼容层。

```go
type Address string
type Hash string
type Data string
func (v Address) ToString() string
func (v Hash) ToString() string
func (v Data) ToString() string

func Latest() BlockRef
func Pending() BlockRef
func AtNumber(uint64) BlockRef
func (v BlockRef) MarshalJSON() ([]byte, error)

func New(jsonrpc.Caller) (*Client, error)
func (c *Client) ChainID(context.Context) (big.Int, error)
func (c *Client) BlockNumber(context.Context) (uint64, error)
func (c *Client) GetBlock(context.Context, BlockRef) (Block, bool, error)
func (c *Client) GetBalance(context.Context, Address, BlockRef) (big.Int, error)
func (c *Client) GetTransactionCount(context.Context, Address, BlockRef) (uint64, error)
func (c *Client) GetCode(context.Context, Address, BlockRef) (Data, error)
func (c *Client) GetTransaction(context.Context, Hash) (Transaction, bool, error)
func (c *Client) GetTransactionReceipt(context.Context, Hash) (Receipt, bool, error)
func (c *Client) GetLogs(context.Context, LogFilter) ([]Log, error)
func (c *Client) Call(context.Context, CallRequest, BlockRef) (Data, error)
func (c *Client) EstimateGas(context.Context, CallRequest) (uint64, error)
func (c *Client) GasPrice(context.Context) (big.Int, error)
func (c *Client) FeeHistory(context.Context, uint64, BlockRef, []float64) (FeeHistoryResult, error)
func (c *Client) SendRawTransaction(context.Context, Data) (Hash, error)
```

RPC 方法和位置参数保持当前映射。`ethereum`/`bnb` 各自保留 `Config{ExpectedChainID big.Int}`、`New(Config,jsonrpc.Caller)`、内嵌 `*evm.Client` 及 `ValidateNetwork`。ExpectedChainID 必须正数且构造时深拷贝；ValidateNetwork 只调用一次 ChainID，不自动执行。1/56 只是测试样例，不是网络白名单。

#### ER-02：完整 DTO 字段

| DTO | 精确 Go 字段 |
|---|---|
| CallRequest | `From, To, Gas, Value, GasPrice, MaxFeePerGas, MaxPriorityFeePerGas, Data string` |
| LogFilter | `FromBlock, ToBlock BlockRef; Addresses []string; Topics [][]string` |
| Block | `Number, Hash, BaseFeePerGas json.RawMessage; ParentHash, Timestamp, GasLimit, GasUsed string; Transactions []string` |
| Transaction | `Hash, From, Nonce, Gas, Value, Input string` |
| Transaction | `To, BlockHash, BlockNumber, TransactionIndex, GasPrice, MaxFeePerGas, MaxPriorityFeePerGas, ChainID, Type json.RawMessage` |
| Receipt | `TransactionHash, BlockHash, TransactionIndex, BlockNumber, From, CumulativeGasUsed, GasUsed, LogsBloom string; Logs []Log` |
| Receipt | `To, ContractAddress, Status, Root, EffectiveGasPrice, Type json.RawMessage` |
| Log | `Address, Data string; Topics []string` |
| Log | `BlockHash, TransactionHash, BlockNumber, TransactionIndex, LogIndex, Removed json.RawMessage` |
| FeeHistoryResult | `OldestBlock string; BaseFeePerGas []string; GasUsedRatio []float64; Reward json.RawMessage` |

CallRequest 字段全部 `omitempty`：`""` 表示省略，`"0x0"` 表示显式数值零，`Data="0x"` 表示显式空数据；不补默认 data。地址/hash/quantity/data 字符串不规范化大小写或重编码。

LogFilter 的 FromBlock/ToBlock 必须有效；Addresses 空时省略 address，非空编码数组；Topics 顺序不变，空内层编码 null 通配位，非空内层编码 OR 数组，最多四位置。

RawMessage 的 nil 表示键缺失，原始 `null` 表示 JSON null，`"0x0"`/`false` 保持显式值；不合成默认值。Reward 原始数组由调用方按 `[][]string` 解码。不复制 scanner 的 `Removed *bool`：它违反小字段禁用指针，而 bool 又会合并缺失/null/false；使用 RawMessage 无须指针例外。三个 nullable 方法继续 `(零值,false,nil)`、`(值,true,nil)`、`(零值,false,error)`。

#### ER-03/04：转换与单一边界

- `jsonrpc` 负责信封、ID、body、响应限长和 context；EVM 原样传入 context，不新增预取消检查或 typed-nil context 反射。调用方遵守非 nil context 契约，自定义 Caller 负责其 context 行为。构造只校验普通 nil Caller，且只在 New 发生。
- EVM 负责固定方法/参数 wire、结果解码、保留原始信息的错误包装及提交 unknown。删除零值/nil Client 的重复保护、duplicate-key 扫描、全字段白名单、必需键循环、手写 schema validator、DTO 跨字段一致性检查、无证据 fallback/default/recover。不要先 `json.Valid` 再 Unmarshal。
- 请求前仅校验有协议依据的输入：BlockRef 非零；地址 20 字节、hash 32 字节、Data 偶数位且有 `0x`；已提供 CallRequest quantity 为最短非负 hex，gasPrice 与动态费用互斥；Filter 地址/topic 编码和最多四位置；count>0；percentiles 有限、0–100、非递减。nil percentiles 编码 `[]`；不修改输入。
- 结果以 RawMessage 接收后单次标准 `json.Unmarshal` 到局部目标；只有三个 nullable 方法接受顶层 null。其他方法顶层 null、JSON 语法/目标类型错误返回 Protocol、全零结果，并按 §4 保留底层解码错误。不逐字段检查 DTO hex、缺失键、null、组合关系、FeeHistory 维度/比例范围。
- 标量数值 API 保留已有便利转换：Quantity 必须 `0x`、至少一位 hex、无前导零；uint64 溢出报 Protocol，big.Int 不截断。金额单位仍为 wei。公开转换最小集合仅三个 `ToString` 和既有 `AtNumber`；不新增 Parse/ToDTO/Clone/Optional。
- GetCode/Call 返回原始 Data 字符串；提交前要求非空偶数位 Data，原样发送一次；成功返回必须是 32 字节 hash，否则 Protocol+SubmissionUnknown。错误按 §4 保留原始字段和完整依赖错误链；必要的外层包装只补实际方法与提交状态，不重建只剩 code 的副本。
- 标准解码产生独立切片/RawMessage；不额外递归 clone。SDK 不修改输入。Config 与标量 big.Int 返回保持独立存储；DTO 不再含 big.Int。

#### ER-01–05：测试、风险和门禁

| 独立风险 | T025 最小证据 |
|---|---|
| API/接线变化遗漏 | 标量底层类型与 ToString；两链分别 14 方法、精确参数、一次调用；New 零 I/O、普通 nil Caller；身份 1/56 与 mismatch |
| 原始值或存在性丢失 | 每个 DTO 一个完整互异值 fixture；pending/create、status=0/历史 root、缺失/null/`"0x0"`/false、Reward 缺失/空数组；逐字段等值 |
| 请求编码错误 | 字符串空省略/显式零、费用互斥、Filter 通配/OR、非法地址/hash/data/quantity、percentiles NaN/Inf；拒绝时零调用 |
| 不必要验证重新长回 | 未知扩展、DTO 缺字段、非规范但原样保留的 DTO hex、较短 FeeHistory 均可解码；无全字段负例矩阵 |
| 转换/错误/提交 | 标量 >uint64 大整数、uint64 溢出、独立存储；nullable 三态、坏 JSON/顶层类型、RPC Message/Data/Cause、context Is/As；原样载荷、非零 hash、拒绝与 unknown |

T025 只修改原六测试文件，原十路径授权不扩大，本轮不改生产声明。用 JSON fixture 和少量测试侧反射检查变化后的字段/底层类型；新 ToString 用测试本地接口断言。这样旧实现可以编译并在行为/类型断言处 RED，不以未定义符号制造 RED。反射不得进入生产解码。T026 后所有新断言必须实际执行。

保护仍适用的 DI、14 方法、nullable、单次提交和网络身份断言。错误受保护断言改为“不主动附加请求秘密、原始字段/Cause 保留”；撤换旧错误全格式脱敏、Has、固定数组/bytes、全字段 schema/duplicate-key、typed-nil context 与 DTO 大整数所有权断言；对应新断言在 T025 审核后成为受保护证据。T026 不改测试。

验证仅 `go test ./evm ./ethereum ./bnb` 与 `go vet ./evm ./ethereum ./bnb`；T025 须有目标 RED，T026 须 GREEN 并通过 vet。公开 API 是否已有使用者未知；底层类型、字段、Parse/Has 删除均需 Human 接受破坏性变更，无兼容层。本修订总时箱：规划 20 分钟、编码 20 分钟、验证含审核最多 30 分钟，T025/T026 共享。预计最终 7 生产文件约 350–500 行、6 测试文件约 600–900 行，仅作预警。

旧 T025/T026 完成历史保留，相关证据 stale；T023/T024 有效。RAW-ERROR-v1 先依次完成 T017→T022；随后更新并重新执行 T025 RED→独立审核→Human，再 T026 GREEN/Refactor→独立审核→Human。T027 继续暂停。越出字段/路径/依赖、规范冲突、时箱耗尽或缺可信证据即停并报告。

### 3.3.0 BITCOIN-CONTRACT-v1 历史候选（已废弃） / Superseded Historical Candidate

> 本节只保留未获批准候选的历史语义，不得作为实现依据。其 typed-nil context、零值 Client、递归 duplicate-key、全字段 schema、条件必需键与深层所有权设计均被 PLAN-v7-REMAINING-RAW-v1 替代。当前实现契约见后续 §3.3。

**状态与证据 / Status and evidence.** BITCOIN-CONTRACT-v1 暂停且不属于本次批准范围，不得实施或直接复制其旧严格解析模式。2026-11-17 快照 `d3502b8c151de849bfc545656ce2dd81d87d1e86` 无 `bitcoin/`；§3.1只有两个nullable签名，architecture §7.1/7.3/7.6/12.4未冻结其他方法/DTO。此F5缺口只影响PLAN-003/T027–T028；RAW-ERROR-v1 重开 T017–T022 与 T025/T026，T023/T024 保持完成。以下Go字段子集、校验和返回规则是项目候选决策，不声称官方提供相同Go契约。

**已核验官方资料 / Verified sources.** 2026-11-17实际读取Bitcoin Core 29.0：[getblockchaininfo](https://bitcoincore.org/en/doc/29.0.0/rpc/blockchain/getblockchaininfo/)（网络/裁剪）、[getblockcount](https://bitcoincore.org/en/doc/29.0.0/rpc/blockchain/getblockcount/)（高度）、[getblock](https://bitcoincore.org/en/doc/29.0.0/rpc/blockchain/getblock/)（verbosity/确认数/txid）、[gettxout](https://bitcoincore.org/en/doc/29.0.0/rpc/blockchain/gettxout/)（null/mempool/BTC/script）、[getmempoolentry](https://bitcoincore.org/en/doc/29.0.0/rpc/blockchain/getmempoolentry/)（vsize/fees/依赖）、[estimatesmartfee](https://bitcoincore.org/en/doc/29.0.0/rpc/util/estimatesmartfee/)（target/模式/BTC-kvB/无估值）、[testmempoolaccept](https://bitcoincore.org/en/doc/29.0.0/rpc/rawtransactions/testmempoolaccept/)（allowed/条件字段/不广播）、[sendrawtransaction](https://bitcoincore.org/en/doc/29.0.0/rpc/rawtransactions/sendrawtransaction/)（hex/默认费用策略/txid）；Core 28.0 [getrawtransaction](https://bitcoincore.org/en/doc/28.0.0/rpc/rawtransactions/getrawtransaction/)（verbosity=1/coinbase/witness/历史前提）；Bitcoin开发者 [getblockhash](https://developer.bitcoin.org/reference/rpc/getblockhash.html)（按高度查hash）及[历史查询说明](https://developer.bitcoin.org/reference/rpc/getrawtransaction.html)。均为官方一手来源、可信度高；29.0 getrawtransaction页读取超时，未作为证据。固定文档版本不代表最新版；目标是满足此字段形状的节点，不承诺所有历史/未来版本兼容，账号兼容性仍由T036验证。

**选择与可行性 / Choice and feasibility.** 选择标准库精确数字解析、固定verbosity、值DTO和单笔预检。相比float64缩放无精度损失；相比完整Core schema/全部模式/package和公开任意精度金额，首版子集降低复杂度和维护成本，未知字段允许扩展。内部可用标准库math/big，无新依赖或公开抽象。无需另建PoC/benchmark，确定性边界fixture可证明解析可行性。节点缺字段/历史能力必须返回真实错误，不静默降级。

#### BC-01：类型、构造和方法 / Types, construction and methods

```go
// package bitcoin；除 Client 外均值语义。
type Config struct{}
type TxID string
type BlockHash string
type RawTransaction string
type Script string
type Witness []string
type Satoshi int64
type SatoshiPerKVByte int64
type EstimateMode string
const (
    EstimateUnset EstimateMode = "unset"
    EstimateEconomical EstimateMode = "economical"
    EstimateConservative EstimateMode = "conservative"
)
func New(Config, jsonrpc.Caller) (*Client, error)
// 五个string类型各有 ToString() string；两个int64类型各有 ToInt64() int64。
// 转换方法只返回底层值，不解析或校验。
type GetTxOutRequest struct {
    TxID TxID
    Vout uint32
    IncludeMempool, HasIncludeMempool bool
}
type GetRawTransactionRequest struct {
    TxID TxID
    BlockHash BlockHash
    HasBlockHash bool
}
type EstimateSmartFeeRequest struct {
    ConfTarget int64
    Mode EstimateMode
    HasMode bool
}
```

Config明确为空，保留既定构造签名，不新增探测/配置策略；New保存私有Caller，普通 nil 返回Configuration且零调用，不反射探测接口内部 typed-nil。TxID（也用于WTxID字段）/BlockHash是RPC展示顺序的64位hex，不反转字节。RawTransaction是非空偶数位hex；Script可为空；Witness每个string为可为空的偶数位hex。无0x/空白，接受大小写并原样保留。类型转换不绕过方法校验；不增公开Parse/MarshalJSON/UnmarshalJSON入口，不解析交易结构/计算txid/重建载荷。

全部方法接收者为 `*Client`、首参 `ctx context.Context`，表中只省略ctx。普通 nil context为InvalidArgument且零调用，不反射探测 typed-nil context；nil/零值Client为Configuration。预取消context在Caller前返回安全Transport+context哨兵且零调用。合法请求至多调用Caller一次，无辅助RPC。

| 方法 / Method | 参数与返回（省略ctx） / Signature | RPC / 位置参数 |
|---|---|---|
| GetBlockchainInfo | `() (BlockchainInfo,error)` | getblockchaininfo；`[]` |
| GetBlockCount | `() (int64,error)` | getblockcount；`[]`；非负链顶高度 |
| GetBlockHash | `(height int64) (BlockHash,error)` | getblockhash；`[height]`；height>=0 |
| GetBlock | `(hash BlockHash) (Block,error)` | getblock；`[hash,1]`，交易ID列表 |
| GetTxOut | `(req GetTxOutRequest) (TxOut,bool,error)` | gettxout；`[txid,vout]`；HasIncludeMempool=true才附第三个bool，包括false |
| GetRawTransaction | `(req GetRawTransactionRequest) (Transaction,error)` | getrawtransaction；`[txid,1]`；HasBlockHash=true才附第三个blockhash |
| GetMempoolEntry | `(txid TxID) (MempoolEntry,error)` | getmempoolentry；`[txid]` |
| EstimateSmartFee | `(req EstimateSmartFeeRequest) (FeeEstimate,bool,error)` | estimatesmartfee；`[confTarget]`或`[confTarget,mode]`；target 1–1008，显式mode限三个常量 |
| TestMempoolAccept | `(payload RawTransaction) (MempoolAcceptResult,error)` | testmempoolaccept；`[[payload]]`，单笔、不广播 |
| SendRawTransaction | `(payload RawTransaction) (TxID,error)` | sendrawtransaction；`[payload]`，一次提交 |

Has=false忽略对应值且不发null占位；省略的mempool/mode由节点决定，不补默认或钳制target。预检/提交不暴露maxfeerate/maxburnamount，保留节点默认策略；不增加package、其他verbosity或ValidateNetwork。非法hash/hex/target/已提供mode在Caller前InvalidArgument。原始入口继续使用jsonrpc.Client.RawCall，不另增链级入口；禁止钱包、构造、签名、解锁调用。

#### BC-02：金额与费率 / Money and fee rates

所有已列BTC金额为Satoshi，BTC/kvB费率为SatoshiPerKVByte；1 BTC=10^8 sat，1kvB=1000 virtual bytes。0.00001000 BTC/kvB=1000 sat/kvB，不能标为sat/vB。本版不提供sat/vB转换/费用建议，不向上或向下舍入。

直接读取JSON number原始token，以十进制整数运算精确乘10^8，最终须为int64范围非负整数。允许JSON合法指数/多余尾零：1e-8、0.000000010为1sat；1e-9拒绝。-0归零，负非零拒绝。字符串数字、NaN/Inf、坏JSON数字、非整数sat、溢出均Protocol，不经过float32/float64。92233720368.54775807为解析器int64上界，不额外实施共识供应量限制。先按token位数/指数判断可表示性或用有界算法，防止巨大指数无界分配；不新增fuzz/load任务。嵌套vout/fees同样适用。

#### BC-03：最小充分 DTO / Minimal sufficient DTOs

下表为全部首版公开字段，同DTO多行合并。箭头后为精确wire键；除可选者均必需非null。Has不来自wire；可选键缺失时Has=false且值为零，提供null为Protocol。整数token不用小数/指数/字符串，检查溢出；高度/时间/计数非负，size/vsize/weight正数，sequence/vout/locktime为uint32；Block.Confirmations允许-1或非负数。

| DTO | 精确Go字段 / Fields ← wire | 存在性 / Presence |
|---|---|---|
| BlockchainInfo | `Chain string; Blocks, Headers int64; BestBlockHash BlockHash; Pruned bool` ← chain/blocks/headers/bestblockhash/pruned | chain非空，不封闭网络名枚举 |
| BlockchainInfo | `PruneHeight int64; HasPruneHeight bool` ← pruneheight | 可选，不据此承诺txindex/历史可用 |
| Block | `Hash BlockHash; Height, Confirmations, Time int64; TxIDs []TxID` ← hash/height/confirmations/time/tx | tx非null数组，可空，逐元素校验；hash与请求hex等值（大小写等价） |
| Block | `PreviousBlockHash, NextBlockHash BlockHash; HasPreviousBlockHash, HasNextBlockHash bool` ← previousblockhash/nextblockhash | 分别可缺失；genesis/链顶不合成零hash |
| ScriptPubKey | `Hex Script; Type string; Address string; HasAddress bool` ← hex/type/address | hex/type必需，type非空且允许新脚本类型；address可选非空字符串，不推导地址 |
| TxOut | `BestBlock BlockHash; Confirmations int64; Value Satoshi; ScriptPubKey ScriptPubKey; Coinbase bool` ← bestblock/confirmations/value/scriptPubKey/coinbase | 0确认/0金额合法；空对象不是不存在 |
| Transaction | `TxID, WTxID TxID; Hex RawTransaction; Version int32; LockTime uint32; Size, VSize, Weight int64; Vin []TxIn; Vout []TxOutput` ← txid/hash/hex/version/locktime/size/vsize/weight/vin/vout | vin/vout非空非null数组；txid与请求等值，wtxid可不同，不本地重算 |
| Transaction | `BlockHash BlockHash; HasBlockHash bool; Confirmations, BlockTime int64; HasConfirmations, HasBlockTime bool; InActiveChain, HasInActiveChain bool` ← blockhash/confirmations/blocktime/in_active_chain | 四键分别可缺省，0/false保留Has=true；in_active_chain仅在请求带blockhash时允许且要求返回，返回blockhash若有须与请求匹配 |
| TxIn | `Sequence uint32; TxID TxID; HasTxID bool; Vout uint32; HasVout bool; ScriptSig Script; HasScriptSig bool; Coinbase Script; HasCoinbase bool; Witness Witness; HasWitness bool` ← sequence/txid/vout/scriptSig.hex/coinbase/txinwitness | coinbase存在时txid/vout/scriptSig缺失，否则三者必需；scriptSig为object且hex必需，空scriptSig合法。witness可缺省，提供即非null字符串数组，允许空数组/空元素，顺序不变 |
| TxOutput | `N uint32; Value Satoshi; ScriptPubKey ScriptPubKey` ← n/value/scriptPubKey | 不从地址生成脚本，不合并输出 |
| MempoolEntry | `VSize, Weight, Time, Height int64; BaseFee Satoshi; Depends, SpentBy []TxID` ← vsize/weight/time/height/fees.base/depends/spentby | fees为object且base必需；两列表必需非null、可空；不暴露modified/ancestor/descendant费用 |
| FeeEstimate | `FeeRate SatoshiPerKVByte; Blocks int64; Errors []string` ← feerate/blocks/errors | blocks必需整数，无估值允许-1或非负，成功估值非负；Errors 保留节点顺序；条件规则见下 |
| MempoolAcceptResult | `TxID, WTxID TxID; Allowed bool; VSize int64; HasVSize bool; BaseFee Satoshi; HasBaseFee bool; RejectReason, RejectDetails string` ← txid/wtxid/allowed/vsize/fees.base/reject-reason/reject-details | RPC顶层恰一个object数组；allowed必需。true时vsize/fees.base必需且Has=true；false时保留节点拒绝字段，不按英文内容分类 |

GetTxOut顶层null为`(TxOut{},false,nil)`，合法对象为`(值,true,nil)`，错误为`(TxOut{},false,error)`。EstimateSmartFee必须为object：有效feerate存在为`(值,true,nil)`，errors可缺失或空字符串数组；无feerate时须有非空errors字符串数组和有效blocks，返回`(FeeEstimate{Blocks:节点值, Errors:原始数组},false,nil)`，不合成0费率。feerate与非空errors并存、两者均无、坏类型/顶层null均Protocol。该 bool 表示 `hasEstimate`，不是对象不存在；§3.1 的零 DTO 通则明确排除此已收到节点原生估费结果的场景。其余八方法顶层null均Protocol；present不表示确认成功。

未列asm/descriptor/完整区块头/prevout/package费率等字段允许忽略，不宣称完整Core schema。此字段集支持architecture §12.4的txid/wtxid/区块关联/vin/vout/脚本/金额核对；地址非每种脚本必有，script/witness不授权交易构造。

#### BC-04/05：历史、wire、错误和所有权 / History, wire, errors and ownership

- 无blockhash历史查询依赖mempool或就绪txindex；指定blockhash仍须区块可用且包含交易，裁剪节点不保证历史。历史/索引不可用、未找到、不在mempool等错误按 §4 保留原始 code/message/data，不改为null、不按文本猜测不存在，不预探测/补查询/切换节点。
- Caller结果以json.RawMessage接收，严格检查单个JSON value；递归拒绝重复键（含转义同名、未知字段内重复键）、尾随文档、坏JSON。键按精确wire名识别，不用大小写折叠替代必需键；未知字段忽略业务schema但JSON仍须合法。缺键/null/坏嵌套元素/溢出均Protocol、全零结果，不返半解析DTO；fake/自定义Caller同样适用，不能只依赖协议层。
- 输入字符串不修改；Block.TxIDs、Transaction.Vin/Vout、每个TxIn.Witness、MempoolEntry.Depends/SpentBy与注入依赖、其他字段/结果、后续调用不共享可变存储。新分配解码或深拷贝均可；不增Clone/通用Optional，不要求调用方浅拷贝后自动写时复制。
- 非结构化 Caller 错误归 Transport 并将原 error 作为 Cause；结构化错误按 §4 保留原始字段、完整包装链和实际 RPC 方法。查询/预检 `SubmissionUnknown=false`。
- SendRawTransaction原样载荷一次调用；成功txid不等于确认。本地校验失败/Caller前取消unknown=false；进入Caller后Transport/HTTP/Protocol/取消及畸形成功响应unknown=true；可信结构化RPC拒绝false并保留数字code含0及原始字段，其他分类按§4。不凭文本猜测拒绝，不补发。

#### BC-01–05：风险、预算和门禁 / Risks, budget and gates

| 独立风险 / Risk | T027原路径内最小证据 / Evidence |
|---|---|
| 隐藏I/O/接线错 | client_test：New 普通 nil/零调用且不做 typed-nil 反射防御；十方法逐项检查方法/参数/一次调用/实际返回；普通 nil/预取消 context 和非法参数零调用。types_test：七个标量转换非零等值 |
| 精度/单位错 | gettxout集中覆盖1sat/指数/尾零/int64边界/负数/非整数sat/溢出/字符串；vout、mempool/预检fees.base、估费各有互异非零精确正例，不复制五套负例 |
| DTO丢字段/缺失混淆 | 各DTO完整正例逐字段等值；coinbase/普通输入、witness含空项、无地址脚本、确认/未确认关联、孤块-1、UTXO三态；表格删除必需键，不仅测空对象 |
| 历史/Has参数错 | blockhash省略/忽略坏值/显式参数、include_mempool=false、历史RPC code不吞、关联hash/txid错失败、零辅助RPC |
| wire/存储泄漏 | 重复/转义重复/尾随/坏hex/type/null/大小写替代键/未知字段；修改返回切片含witness及fake缓冲后其他字段和后续结果不变 |
| 无估值/预检拒绝被误报成功 | target/mode边界、估值/无估值/畸形null、allowed true/false、数组数量和条件字段；一次调用且无submit |
| 提交拒绝/未知混淆、信息丢失 | 非零txid逐字符等值、大小写payload原样；本地拒绝零调用、RPC code/message/data/Cause、Transport/HTTP/Protocol/取消/畸形txid unknown；不主动附加请求秘密 |

五测试文件局部fixture/helper去重，不增工具包/依赖。必要时client_test可用一个httptest证明链层到既有Bitcoin方言接线，其余fake，不重测协议层。L1–L3必需，L4仅离线契约；L5限原始错误信息保留、不主动追加请求秘密、所有权及有界解析，报告脱敏仅属于 harness/runner 输出边界；不新增race/fuzz/benchmark；L6不适用，真实验收T036。T027仅完整声明和Unsupported/零值分支，编译后目标行为断言RED才有效；T028只在原五路径实现，全部受保护断言实际执行且GREEN。

T027 最多 70 分钟：规划 20、Coder 20、验证及固定 Auditor 30；预计声明 160–220 行、测试 650–900 行，220/900 只作预警而非写满目标。当前候选暂停；若以后恢复，仍须重新核对 constitution 1.3 并另获 Human 批准。七路径和 T028 五路径不变，不按行数拆任务/扩时。T023/T024 证据有效，T017–T022 与 T025/T026 的受影响错误证据 stale。

### 3.3 Bitcoin 原始 DTO 与必要金额转换 / Bitcoin Raw DTO Contract

`Config` 为空值结构体。`New(Config, jsonrpc.Caller) (*Client, error)` 只在构造边界拒绝普通 nil Caller，不反射探测 typed-nil，零 I/O。方法原样传递 `context.Context`；不做零值 Client、防御性预取消或 typed-nil context 检查。

`TxID`、`BlockHash`、`RawTransaction`、`Script`、`EstimateMode` 均为命名 `string` 并提供 `ToString() string`；`Satoshi`、`SatoshiPerKVByte` 为命名 `int64` 并提供 `ToInt64() int64`。估费常量名称和值固定为 `EstimateUnset EstimateMode = "unset"`、`EstimateEconomical = "economical"`、`EstimateConservative = "conservative"`。

```go
type GetTxOutRequest struct {
    TxID TxID
    Vout uint32
    IncludeMempool, HasIncludeMempool bool
}
type GetRawTransactionRequest struct {
    TxID TxID
    BlockHash BlockHash
    HasBlockHash bool
}
type EstimateSmartFeeRequest struct {
    ConfTarget int64
    Mode EstimateMode
    HasMode bool
}
```

| 方法签名（均以 `context.Context` 为首参） | RPC 与位置参数 |
|---|---|
| `GetBlockchainInfo() (BlockchainInfo,error)` | `getblockchaininfo []` |
| `GetBlockCount() (int64,error)` | `getblockcount []` |
| `GetBlockHash(int64) (BlockHash,error)` | `getblockhash [height]` |
| `GetBlock(BlockHash) (Block,error)` | `getblock [hash,1]` |
| `GetTxOut(GetTxOutRequest) (TxOut,bool,error)` | `gettxout [txid,vout]`，仅 HasIncludeMempool 时追加 bool |
| `GetRawTransaction(GetRawTransactionRequest) (Transaction,error)` | `getrawtransaction [txid,1]`，仅 HasBlockHash 时追加 blockhash |
| `GetMempoolEntry(TxID) (MempoolEntry,error)` | `getmempoolentry [txid]` |
| `EstimateSmartFee(EstimateSmartFeeRequest) (FeeEstimate,bool,error)` | `estimatesmartfee [target]`，仅 HasMode 时追加 mode |
| `TestMempoolAccept(RawTransaction) (MempoolAcceptResult,error)` | `testmempoolaccept [[payload]]` |
| `SendRawTransaction(RawTransaction) (TxID,error)` | `sendrawtransaction [payload]` |

| DTO | 精确字段；括号内为 wire key |
|---|---|
| `BlockchainInfo` | `Chain string; Blocks,Headers int64; BestBlockHash string(bestblockhash); Pruned bool; PruneHeight json.RawMessage(pruneheight)` |
| `Block` | `Hash string; Height,Confirmations,Time int64; TxIDs []string(tx); PreviousBlockHash,NextBlockHash json.RawMessage(previousblockhash,nextblockhash)` |
| `ScriptPubKey` | `Hex,Type string; Address json.RawMessage(address)` |
| `TxOut` | `BestBlock string(bestblock); Confirmations int64; Value Satoshi; ScriptPubKey ScriptPubKey(scriptPubKey); Coinbase bool` |
| `Transaction` | `TxID string(txid); WTxID string(hash); Hex string; Version int32; LockTime uint32(locktime); Size,VSize,Weight int64; Vin []TxIn; Vout []TxOutput; BlockHash json.RawMessage(blockhash); Confirmations json.RawMessage(confirmations); BlockTime json.RawMessage(blocktime); InActiveChain json.RawMessage(in_active_chain)` |
| `TxIn` | `Sequence uint32; TxID,Vout,ScriptSig,Coinbase,Witness json.RawMessage(txid,vout,scriptSig,coinbase,txinwitness)` |
| `TxOutput` | `N uint32; Value Satoshi; ScriptPubKey ScriptPubKey(scriptPubKey)` |
| `MempoolEntry` | `VSize,Weight,Time,Height int64; BaseFee Satoshi; Depends,SpentBy []string(depends,spentby)` |
| `FeeEstimate` | `FeeRate SatoshiPerKVByte; Blocks int64; Errors []string` |
| `MempoolAcceptResult` | `TxID,WTxID string(txid,wtxid); Allowed bool; VSize json.RawMessage(vsize); Fees json.RawMessage(fees); RejectReason,RejectDetails string(reject-reason,reject-details)` |

请求只验证：hash/txid 为 64 位无 `0x` hex；height 非负；payload 为非空偶数位无 `0x` hex；target 为 1–1008；显式 mode 属于三个常量。Has=false 时对应值不校验且不发 null。大小写原样保留；不增加其他 verbosity、maxfeerate、maxburnamount、package、`ValidateNetwork` 或链级 Raw 入口。

BTC 金额转换只发生于 `TxOut.Value`、`TxOutput.Value`、`MempoolEntry` 的 `fees.base`、`FeeEstimate.feerate`。内部以 `json.Number`/`RawMessage` 接收，使用标准库十进制整数运算精确乘 10^8，不经 `float64`。结果必须为非负 int64 整数；`1e-8` 与 `0.000000010` 为 1，`1e-9`、负非零、溢出、字符串数字失败；`-0` 为零。转换必须先检查小数位/指数界限，禁止巨大指数导致无界分配。任何必转金额键缺失或为 null 都返回 Protocol，不能合成零。预检 `Fees` 保留原始 JSON，不建立第二套金额模型。

全部响应使用标准 `json.Unmarshal`。未知字段、重复键及大小写匹配服从 `encoding/json`；不递归检查所有 hex、必需键、coinbase schema、字段关联、hash/txid 与请求相等或嵌套未知对象。JSON 语法、顶层类型和目标字段标准类型错误为 Protocol，失败不返回半 DTO。

`GetBlockCount` 结果必须非负；`GetBlockHash` 必须为 64 位 hex；只有 `GetTxOut` 接受顶层 null，并返回 `(TxOut{},false,nil)`。除标量、GetTxOut 与 TestMempoolAccept 外，其余 DTO 方法顶层必须是 object。`TestMempoolAccept` 顶层必须是恰一个 object 的数组且 `allowed` 为 bool。`EstimateSmartFee` 有非 null feerate 时精确转换并返回 `hasEstimate=true`；无 feerate 时保留 Blocks/Errors 并返回 false；显式 `feerate:null` 为 Protocol。标准解码提供独立切片/RawMessage，不增加递归 clone。

历史/txindex/mempool 限制直接传播节点原始错误，不预探测、不补查询、不按英文 message 分类。`SendRawTransaction` 载荷原样调用一次；成功必须得到 64 位 hex txid。进入 Caller 后的 Transport/HTTP/Protocol/取消及畸形成功结果为 `SubmissionUnknown=true`，可信 RPC 拒绝为 false；不补发。

### 3.4 Solana 精确契约 / Solana Contract

`Config` 为 `struct{}`；`New(Config,jsonrpc.Caller) (*Client,error)`。`Pubkey`、`Hash`、`Signature`、`EncodedMessage`、`Commitment`、`Encoding` 为命名 string 并提供 `ToString() string`。常量名称和值固定为 `Processed="processed"`、`Confirmed="confirmed"`、`Finalized="finalized"`，以及 `Base58="base58"`、`Base64="base64"`、`JSON="json"`、`JSONParsed="jsonParsed"`。`EncodedTransaction` 为值结构体 `{Data string; Encoding Encoding}`，不提供编码或签名方法。

| 请求类型 | 精确字段 |
|---|---|
| `ReadOptions` | `Commitment Commitment; MinContextSlot uint64; HasMinContextSlot bool` |
| `BlockhashOptions` | 同 `ReadOptions` |
| `BlockOptions` | `Commitment Commitment; Encoding Encoding; MaxSupportedTransactionVersion uint64; HasMaxSupportedTransactionVersion bool; Rewards bool; HasRewards bool` |
| `TransactionOptions` | `Commitment Commitment; Encoding Encoding; MaxSupportedTransactionVersion uint64; HasMaxSupportedTransactionVersion bool` |
| `AccountOptions` | `Commitment Commitment; Encoding Encoding; MinContextSlot uint64; HasMinContextSlot bool` |
| `StatusOptions` | `SearchTransactionHistory bool; HasSearchTransactionHistory bool` |
| `SimulateOptions` | `Commitment Commitment; SigVerify bool; HasSigVerify bool; ReplaceRecentBlockhash bool; HasReplaceRecentBlockhash bool; MinContextSlot uint64; HasMinContextSlot bool; InnerInstructions bool; HasInnerInstructions bool` |
| `SendOptions` | `SkipPreflight bool; HasSkipPreflight bool; PreflightCommitment Commitment; MaxRetries uint64; HasMaxRetries bool; MinContextSlot uint64; HasMinContextSlot bool` |

空 commitment/encoding 表示省略，其他可选字段由 Has 控制，显式零/false 必须发送。选项使用 RPC camelCase key，固定位置的 options 即使为空也发送 `{}`；不补默认 commitment、encoding、maxRetries 或版本。

| 方法签名（均以 `context.Context` 为首参） | RPC 参数 |
|---|---|
| `GetGenesisHash() (Hash,error)` | `getGenesisHash []` |
| `GetSlot(ReadOptions) (uint64,error)` | `getSlot [opts]` |
| `GetBlock(uint64,BlockOptions) (Block,bool,error)` | `getBlock [slot,opts]`，固定 `transactionDetails:"full"` |
| `GetBalance(Pubkey,ReadOptions) (BalanceResult,error)` | `getBalance [pubkey,opts]` |
| `GetAccountInfo(Pubkey,AccountOptions) (AccountInfoResult,error)` | `getAccountInfo [pubkey,opts]` |
| `GetLatestBlockhash(BlockhashOptions) (BlockhashResult,bool,error)` | `getLatestBlockhash [opts]` |
| `GetSignatureStatuses([]Signature,StatusOptions) (SignatureStatusesResult,error)` | `getSignatureStatuses [signatures,opts]` |
| `GetTransaction(Signature,TransactionOptions) (Transaction,bool,error)` | `getTransaction [signature,opts]` |
| `GetFeeForMessage(EncodedMessage,ReadOptions) (FeeResult,error)` | `getFeeForMessage [message,opts]` |
| `SimulateTransaction(EncodedTransaction,SimulateOptions) (SimulationResult,bool,error)` | `simulateTransaction [Data,opts+encoding]` |
| `SendTransaction(EncodedTransaction,SendOptions) (Signature,error)` | `sendTransaction [Data,opts+encoding]` |

| DTO | 精确字段 |
|---|---|
| `Context` | `Slot uint64; APIVersion json.RawMessage(apiVersion)` |
| `BalanceResult` | `Context Context; Value uint64` |
| `AccountInfoResult` | `Context Context; Value json.RawMessage` |
| `BlockhashValue` | `Blockhash string; LastValidBlockHeight uint64` |
| `BlockhashResult` | `Context Context; Value BlockhashValue` |
| `SignatureStatusesResult` | `Context Context; Value []json.RawMessage` |
| `FeeResult` | `Context Context; Value json.RawMessage` |
| `Transaction` | `Slot uint64; BlockTime,Transaction,Meta,Version json.RawMessage` |
| `Block` | `Blockhash,PreviousBlockhash string; ParentSlot uint64; BlockTime,BlockHeight,Rewards json.RawMessage; Transactions []BlockTransaction` |
| `BlockTransaction` | `Transaction,Meta,Version json.RawMessage` |
| `SimulationResult` | `Context Context; Value SimulationValue` |
| `SimulationValue` | `Err,Logs,Accounts,ReturnData,UnitsConsumed,ReplacementBlockhash,InnerInstructions json.RawMessage` |

RawMessage 保留合法 JSON 原值；不把 `meta.err`、确认状态或模拟 err 归一为 success bool。Account/Fee 的 `value:null` 原样返回并保留 Context；账户 data/lamports/owner 等由 Value 原样承载。Statuses 的 object/null 项保持顺序。

请求验证仅包括：非空 commitment 属于三个常量；GetBlock/GetTransaction 不接受 processed。GetBlock/GetTransaction encoding 可省略或为 json/jsonParsed/base58/base64；AccountInfo 可省略或为 base58/base64/jsonParsed；不提供 dataSlice 或 zstd。交易编码必须显式 base58/base64 且 Data 非空；base64 用标准库验证，base58 只校验字符集。Pubkey/Hash/Signature 非空且为 base58 字符集；statuses 数量 1–256；EncodedMessage 为非空合法 base64；显式版本只支持 0；`SigVerify=true` 与 `ReplaceRecentBlockhash=true` 互斥。不得做曲线、账户存在、签名密码学、message 解析或深层 schema 检查。

GetBlock/GetTransaction 顶层 null 返回零值/false/nil；GetLatestBlockhash/SimulateTransaction 不接受顶层或 value null，成功返回 true。所有 envelope 顶层必须是 object，且 `context`/`value` 键必须存在并具有本段声明的类型；缺失 value 不能解码成零余额。Context 内不做全字段 schema。Balance 为 uint64；Fee.Value 为 uint64 JSON number 或 null；Account Value 为 object 或 null；Statuses Value 必须是与请求等长的数组，每项只能是 object 或 null。`SendTransaction` 返回非空 base58 Signature；不重算签名、不刷新 blockhash、不轮询、不把节点 maxRetries 当 SDK 重试。

### 3.5 Tron 精确契约 / Tron Contract

`Config` 为 `struct{}`；`New(Config,httpjson.Poster) (*Client,error)`。`Address`、`TxID` 为命名 string 并提供 `ToString()`。`TransactionPayload` 为值结构体 `{JSON json.RawMessage}`。

```go
type GetAccountRequest struct { Address Address; Visible bool }
type ContractCallRequest struct {
    OwnerAddress, ContractAddress Address
    FunctionSelector, Parameter string
    CallValue int64
    HasCallValue bool
    Visible bool
}
```

| 方法（均以 `context.Context` 为首参） | 返回与 POST 路径 |
|---|---|
| `GetNodeInfo()` | `(NodeInfo,error)`；`/wallet/getnodeinfo` |
| `GetNowBlock()` | `(Block,bool,error)`；`/wallet/getnowblock` |
| `GetBlockByNum(int64)` | `(Block,bool,error)`；`/wallet/getblockbynum` |
| `GetAccount(GetAccountRequest)` | `(Account,bool,error)`；`/wallet/getaccount` |
| `GetAccountResource(GetAccountRequest)` | `(AccountResource,error)`；`/wallet/getaccountresource` |
| `GetChainParameters()` | `(ChainParameters,error)`；`/wallet/getchainparameters` |
| `GetTransactionByID(TxID)` | `(Transaction,bool,error)`；`/wallet/gettransactionbyid` |
| `GetTransactionInfoByID(TxID)` | `(TransactionInfo,bool,error)`；`/wallet/gettransactioninfobyid` |
| `GetSolidifiedAccount(GetAccountRequest)` | `(Account,bool,error)`；`/walletsolidity/getaccount` |
| `GetSolidifiedNowBlock()` | `(Block,bool,error)`；`/walletsolidity/getnowblock` |
| `GetSolidifiedBlockByNum(int64)` | `(Block,bool,error)`；`/walletsolidity/getblockbynum` |
| `GetSolidifiedTransactionByID(TxID)` | `(Transaction,bool,error)`；`/walletsolidity/gettransactionbyid` |
| `GetSolidifiedTransactionInfoByID(TxID)` | `(TransactionInfo,bool,error)`；`/walletsolidity/gettransactioninfobyid` |
| `TriggerConstantContract(ContractCallRequest)` | `(ConstantResult,error)`；`/wallet/triggerconstantcontract` |
| `EstimateEnergy(ContractCallRequest)` | `(EnergyEstimate,error)`；`/wallet/estimateenergy` |
| `BroadcastTransaction(TransactionPayload)` | `(BroadcastResult,bool,error)`；`/wallet/broadcasttransaction`，直接传 payload.JSON |

查询 body 分别为 `{}`、`{"num":height}`、`{"address":address,"visible":bool}`、`{"value":id}`。solidified 请求形状与 FullNode 对应方法相同。合约计算使用 `owner_address`、`contract_address`、`function_selector`、`parameter`、`visible`，仅 HasCallValue 时加入 `call_value`；不提供 ABI 编码、deployment、token 字段或交易构建。

| DTO | 精确字段 |
|---|---|
| `NodeInfo` | `Block string(block); SolidityBlock string(solidityBlock); CurrentConnectCount int64(currentConnectCount); ConfigNodeInfo json.RawMessage(configNodeInfo); PeerList json.RawMessage(peerList)` |
| `Block` | `BlockID string(blockID); BlockHeader json.RawMessage(block_header); Transactions []Transaction(transactions)` |
| `Account` | `Address string(address); Balance int64(balance); AccountResource json.RawMessage(account_resource); AssetV2 json.RawMessage(assetV2)` |
| `AccountResource` | `FreeNetUsed int64(freeNetUsed); FreeNetLimit int64(freeNetLimit); NetUsed int64(NetUsed); NetLimit int64(NetLimit); EnergyUsed int64(EnergyUsed); EnergyLimit int64(EnergyLimit); TotalNetLimit int64(TotalNetLimit); TotalNetWeight int64(TotalNetWeight); TotalEnergyLimit int64(TotalEnergyLimit); TotalEnergyWeight int64(TotalEnergyWeight)` |
| `ChainParameter`/`ChainParameters` | `Key string(key); Value int64(value)`；`ChainParameter []ChainParameter(chainParameter)` |
| `Transaction` | `TxID string(txID); RawData json.RawMessage(raw_data); RawDataHex string(raw_data_hex); Signature []string(signature); Ret json.RawMessage(ret)` |
| `TransactionInfo` | `ID string(id); Fee int64(fee); BlockNumber int64(blockNumber); BlockTimeStamp int64(blockTimeStamp); ContractResult []string(contractResult); ContractAddress string(contract_address); Receipt json.RawMessage(receipt); Log json.RawMessage(log); InternalTransactions json.RawMessage(internal_transactions); Result json.RawMessage(result); ResMessage json.RawMessage(resMessage)` |
| `NodeResult` | `Result bool(result); Code string(code); Message string(message)` |
| `ConstantResult` | `Result NodeResult(result); ConstantResult []string(constant_result); EnergyUsed int64(energy_used); Transaction json.RawMessage(transaction)` |
| `EnergyEstimate` | `Result NodeResult(result); EnergyRequired int64(energy_required)` |
| `BroadcastResult` | `Result bool(result); TxID string(txid); Code string(code); Message string(message)` |

Tron proto3 默认值省略：balance/resource/parameter value 缺失可表示零，不增加 Has 或 fallback。带 bool 查询的空 object `{}` 表示不存在；非空 Account 必须含 address，Block 必须含 blockID，Transaction 必须含 txID，TransactionInfo 必须含 id。这只是 present 判定，不扩展为全字段 schema；顶层 null/非 object 为 Protocol。

请求验证仅包括：height 非负；TxID 为 64 位无 `0x` hex；visible=false 地址为 42 位、41 前缀 hex；visible=true 地址为 34 字符、T 前缀 base58 字符集；不转换、不计算 checksum。合约 selector 非空，parameter 可空否则为偶数位无 `0x` hex，显式 callValue 非负。payload 必须是非空 JSON object；不检查/重建 raw_data、signature、TAPOS 或 txID，不修改输入语义。传给 Poster 的值是 RawMessage；既有 HTTP JSON 标准 marshal 可以改变无语义的空白或转义表示，但不得改变解码后的数字、字符串或签名载荷语义；“无损”不承诺 HTTP body 与输入原始空白字节完全一致。

HTTP 200 内顶层 `Error` 存在且为 string 时返回 RPC：`Message=Error`、`Data=完整已收到的原始响应 object`。计算 result 必须是 object 且内部 result 为 bool；false 返回 RPC，`NodeCode/Message` 取内部原字段，Data 为完整响应。广播 result 必须为 bool；false 返回可信 RPC 拒绝，`NodeCode/Message/Data` 同样原样保留且 unknown=false；true 返回 DTO/true/nil，txid 若存在必须为 64 位 hex，缺失时不得本地计算或填入。`Error`/result 类型错误或缺判定字段为 Protocol，广播时 unknown=true。Poster/HTTP 错误按 §4 包装，不伪造 HTTPStatus 或 ResponseBody；依赖已提供的 ResponseBody/Cause 完整保留。成功 TransactionInfo.receipt/result 或 Transaction.ret 是链上执行事实，不转为 RPC error。原始字段不解码或按内容分类。balance/fee/call_value 单位为 sun，Energy 与 Bandwidth 不混用；solidified 只是节点视图，不额外计算确认或轮询。

### 3.6 剩余链官方来源登记 / Remaining Chain Source Register

| 来源 | 版本/访问日期 | 支持结论 |
|---|---|---|
| Bitcoin Core 29.0 RPC 文档；getrawtransaction 使用 Core 28.0 固定页 | 2026-11-17 | 十方法 wire、金额、null、历史/txindex 与拒绝语义 |
| [Solana getBlock](https://solana.com/docs/rpc/http/getblock)、[getTransaction](https://solana.com/docs/rpc/http/gettransaction)、[getAccountInfo](https://solana.com/docs/rpc/http/getaccountinfo)、[getFeeForMessage](https://solana.com/docs/rpc/http/getfeeformessage) | 滚动官方文档，2026-11-17 | encoding/version、nullable block/transaction、envelope context 与 value:null |
| [Solana getLatestBlockhash](https://solana.com/docs/rpc/http/getlatestblockhash)、[getSignatureStatuses](https://solana.com/docs/rpc/http/getsignaturestatuses)、[simulateTransaction](https://solana.com/docs/rpc/http/simulatetransaction)、[sendTransaction](https://solana.com/docs/rpc/http/sendtransaction) | 滚动官方文档，2026-11-17 | 有效高度、状态数组、模拟/preflight、接受与确认差异 |
| [Tron GetAccount](https://developers.tron.network/reference/account-getaccount)、[GetAccountResource](https://developers.tron.network/reference/getaccountresource)、[ChainParameters](https://developers.tron.network/reference/wallet-getchainparameters) | 滚动官方文档，2026-11-17 | address/visible、空账户、proto3 零字段省略、资源单位 |
| [Tron transaction](https://developers.tron.network/reference/wallet-gettransactionbyid)、[receipt](https://developers.tron.network/reference/gettransactioninfobyid)、[solidified receipt](https://developers.tron.network/reference/gettransactioninfobyid-1)、[TriggerConstantContract](https://developers.tron.network/reference/triggerconstantcontract)、[EstimateEnergy](https://developers.tron.network/reference/estimateenergy)、[BroadcastTransaction](https://developers.tron.network/reference/broadcasttransaction) | 滚动官方文档，2026-11-17 | 查询/回执/solidified 差异、计算、Energy/sun、接受不证明执行 |
| java-tron `GetNodeInfoServlet`/`GetChainParametersServlet` 与官方历史 HTTP 文档 | develop/master 滚动源码，2026-11-17 | 文档虽展示 GET，servlet/历史文档支持 POST；不扩展 Getter，实际兼容性由 T036 验证 |

外部协议结论可信度高；滚动文档/源码不构成固定发布版本兼容承诺。DTO 字段子集、Go 名称、RawMessage 与选项范围均为本项目候选决策。

## 4. 错误与提交语义

**依据。** 当前实现没有 Message/Data 字段，私有 cause 只保留归一化 context 哨兵；JSON-RPC 已解码 message 但会丢弃 message/data，协议客户端也会丢弃 HTTP 错误正文与普通底层 error。JSON-RPC 2.0 定义 `code`、`message` 和可选任意 JSON `data`；Go `errors.Is/As` 沿 `Unwrap` 遍历错误链。来源：[JSON-RPC 2.0](https://www.jsonrpc.org/specification)、[Go errors](https://pkg.go.dev/errors)，访问日期 2026-11-17。协议字段是外部事实；下述 Go API 与格式是本项目候选决策。

| 方案 | 复杂度/性能/扩展 | 兼容性/风险/维护 | 决定 |
|---|---|---|---|
| 直接返回底层 error，取消 SDK 包装 | 实现少，但节点 JSON 仍须造错误，分类/unknown 无统一承载 | 丢失现有状态与 code 契约 | 拒绝 |
| 扩展现有 `apierror.Error` | 标准库、有限字段、响应继续限长 | 保留分类/unknown；公开字段和格式有明确兼容性变化 | 采用 |
| 多套 Node/Transport/SafeView 错误与脱敏策略 | 类型、配置与分支显著增加 | 把业务输出策略重新带入 SDK | 拒绝 |

```go
type Kind string

const (
    Configuration Kind = "configuration"
    InvalidArgument Kind = "invalid_argument"
    Transport       Kind = "transport"
    HTTP            Kind = "http"
    Protocol        Kind = "protocol"
    RPC             Kind = "rpc"
    Unsupported     Kind = "unsupported"
    NetworkMismatch Kind = "network_mismatch"
)

type Error struct {
    Kind                  Kind
    Method                string
    HTTPStatus            int
    RPCCode               int
    HasRPCCode            bool
    NodeCode              string
    Message               string
    Data                  json.RawMessage
    ResponseBody          []byte
    ResponseBodyTruncated bool
    SubmissionUnknown     bool
    Cause                 error
}

func (e Error) Error() string
func (e Error) Unwrap() error
func (e Error) WithContextCause(err error) Error
```

`HasRPCCode=false` 表示没有节点数字码；为 true 时保留 `RPCCode` 原值，包括零。精确契约如下：

1. **Error()：**以 `Kind.ToString()` 为基础；Message 非空则追加 `": " + Message`；否则 Cause 非 nil 则追加 `": " + Cause.Error()`；否则只返回 Kind。零 Kind 时不产生开头 `": "`，只返回选中的文本。Message 和 Cause 同时存在时字符串选 Message，完整 Cause 仍通过字段和 Unwrap 可达。Data、ResponseBody、Header 和请求载荷不自动拼接。该字符串不承诺脱敏。
2. **Unwrap：**精确返回 `e.Cause`，不替换、归一化、提取最底层或过滤 error。
3. **WithContextCause：**保留已有签名；返回值副本，唯一变化为 `Cause=err`。nil 清空；普通错误、包装错误、`errors.Join` 均原样保存。名称是已有公开 API，不新增另一套 setter 或错误构造抽象。
4. **值语义与兼容性：**SDK 返回 Error 值，`var e apierror.Error; errors.As(err,&e)` 不变。新增 RawMessage/切片后 Error 不再可比较；旧 `==` 使用者须改为逐字段检查或 `errors.Is/As`。Error() 格式和 WithContextCause 行为变化属于明确兼容性变更，不提供兼容层。
5. **节点字段：**JSON-RPC Message 是节点字符串的解码值，Data 是原始 JSON value。Data 不存在为 nil，显式 null 原样保留；不把节点 Message 合成为 SDK 模板文本。NodeCode 用于链原生字符串码，不设内容白名单。
6. **ResponseBody：**协议客户端失败且已经读取响应时，保存实际读到的原始字节，最多 MaxResponseBytes；包括 HTTP 错误、无效协议响应及 RPC 错误。仅检测到额外字节时置 `ResponseBodyTruncated=true`。部分读取失败的 Cause 保留实际 read error，不承诺正文完整。成功返回不额外保存正文。Do 同时返回 response/error 时保存已知 HTTP status、原 error 并按原资源契约关闭 body，不为收集诊断额外读取已失败的响应。
7. **分类优先级：**完整、未超限且匹配方言/ID 的 RPC error → RPC，同时保留 HTTPStatus；不满足有效错误信封的非 2xx → HTTP；2xx 无效信封/结果 → Protocol；实际 I/O 失败 → Transport。读取不完整或截断内容不得作为可信 RPC 拒绝。
8. **Cause 来源：**marshal、URL/请求构建、Do、Read、JSON decode 等实际返回的 error 均保存；纯布尔校验失败无底层 error 时 Cause=nil，不为每个分支制造新错误。解析器内部返回已取得的解码 error；不为了美化诊断重写整套解析器。
9. **context：**协议层直接观察到 `ctx.Err()` 时保存该 error；收到依赖返回的包装 error 时保存原依赖 error。不得因同时观察到 ctx 取消而用 `ctx.Err()` 覆盖无关的实际错误，也不合成取消分类。原链同时匹配 canceled/deadline 时两者都可匹配。链客户端不新增重复 context 检查、typed-nil 反射或自定义 Is/As。
10. **链包装：**收到 error 后保留整个 error 作为外层 Cause；若 `errors.As` 可提取 apierror.Error，复制其原始节点字段、状态及类别，再设置实际 Method 和该方法的 SubmissionUnknown。不能重建只剩 code 的副本，不能丢弃外层 fmt/error wrapper。无结构化错误时归 Transport，Cause 为原 error。不新增 Kind 白名单防御。
11. **提交 unknown：**进入注入依赖后发生 Transport/HTTP/Protocol/取消或畸形成功结果时 `SubmissionUnknown=true`；本地校验失败或可信节点明确拒绝为 false。保留信息不等于判断可重试，SDK 不根据 Message/Data/正文英文文本决定是否送达、接受或补发。
12. **职责隔离：**SDK 不主动收集或追加 URL、Header、请求、交易载荷。原 error 和节点响应中已有内容仍可达。无内置 logger、脱敏开关、回调策略、错误扫描器、recover 或新依赖。

这些规则不要求捕获此前未作为返回错误处理的 body.Close 错误；本轮不改变其现有生命周期与成功/失败优先级。

## 5. 五个功能 Plan

| Plan | 功能边界 | Tasks | 独立验收 |
|---|---|---|---|
| PLAN-001 公共基础 | module、DI、错误、JSON-RPC、HTTP JSON、live harness 基础 | T017–T024 | AC-01–04、09、12、13 |
| PLAN-002 EVM | EVM 原生类型与方法、Ethereum/BNB 具体客户端 | T025–T026 | AC-02、05、12、13；T036 最终验收 |
| PLAN-003 Bitcoin | Bitcoin 查询、预检与提交 | T027–T028 | AC-02、06、12、13；T036 最终验收 |
| PLAN-004 Solana | Solana 查询、模拟与提交 | T029–T030 | AC-02、07、12、13；T036 最终验收 |
| PLAN-005 Tron 与最终集成 | Tron、五链 runner、文档与最终验收 | T031–T036 | AC-01–13 |

每个行为实现 Task 必须依赖更早的失败测试 Task；RED 由测试 Task 留证，GREEN/Refactor 由对应实现 Task 完成。不得把两者写成同一 Task 的内部步骤来替代依赖门禁。各链方法范围仍以 spec §3 和本方案既有契约为准，不按单个 RPC 方法继续拆分。

默认依序处理 PLAN-001→002→003→004→005。依赖满足并不自动授权并行 Agent。测试/实现 Task 之间不设置要求全部测试为绿的独立交付门禁；子 Plan 交付时必须全部转绿并完成规定审核。

当前状态以 PLAN-v7-REMAINING-RAW-v1 为准：T017–T027 已完成对应 RED/GREEN、独立 Auditor、Human 门禁并集成；旧 reopened/stale 描述只保留历史。T028 已单独授权进入实现，T029–T036 仍按 T028→T029→T030→T031→T032→T033→T034→T035→T036 顺序逐项过门禁，不授权并行实施，也不新增 T037。

## 6. 请求流程

```mermaid
sequenceDiagram
    participant H as Host
    participant C as Concrete Chain Client
    participant R as Injected RPC

    H->>C: 查询 / 节点计算 / 交易提交
    C->>R: 单次节点请求
    R-->>C: 链原生结果或错误
    C-->>H: 类型化结果或结构化错误
```

## 7. 测试与真实验收设计

### 7.1 本地测试

- fake Caller/Poster 验证方法、参数、交易载荷字节、调用次数和错误分类。
- `httptest` 验证 HTTP 状态、超限响应、body 关闭、取消与路径边界。
- 表格驱动 fixture 验证金额精度、编码、null、pending、commitment、solidified 等链原生语义。
- 提交测试断言载荷不变且 SDK 只发起一次调用。
- API 审核逐项检查命名常量转换方法和小结构体值语义；指针冲突必须区分 SDK 自有设计、数据库 `NULL` 与第三方 API 契约。
- 依赖与公开 API 审核确认核心不存在私钥、构造、编码、签名或节点解锁能力；可选说明性示例不得进入核心依赖图。

### 7.2 live harness：HARNESS-CONTRACT-v1（已批准）

以下 HC-01–HC-08 是 AC-09 的已批准精确契约；条款 ID 不属于活动 Task。它们仅作用于 `internal/test/live`，不能扩展核心 SDK 的配置读取、调度或链访问职责。

#### HC-01：边界与开关

- `internal/test/live` 是测试装配；核心 SDK 仍不读 env/file。
- `runHarness(..., enabled=false, ...)` 立即返回 `NOT_RUN`，不调用配置 loader、manifest loader、clock、waiter 或 case；请求数为零。
- 实际入口只能读取 live 开关以决定是否启用；关闭后不得读取配置路径变量、Header 引用变量、配置文件、manifest 或交易材料。
- 无默认 endpoint、HTTP client、配置路径、凭据、重试、轮询、后台 worker、文件 watcher 或网络 fallback。
- 本次仅批准离线基础；T033/T034 负责五链装配，T036 才可能在独立授权后执行真实验收。

#### HC-02：内部测试边界

```go
type harnessConfig struct {
    Alias, Endpoint, Network string
    Headers map[string]string
    MaxRequests, MaxConcurrent int
    MinInterval, TotalTimeout time.Duration
    DynamicInputsPath, SubmissionMaterialPath string
}

type harnessDeps struct {
    Load func() (harnessConfig, error)
    LoadManifest func() ([]byte, string, error)
    Now func() time.Time
    Wait func(context.Context, time.Duration) error
}

type harnessRequest func(func(context.Context) error) error

type harnessCase struct {
    ID string
    Run func(context.Context, harnessRequest) (string, error)
}

type harnessReport struct {
    Status, Category, Digest string
    Requests int
    Cases map[string]string
}

func runHarness(context.Context, bool, harnessDeps, []string, []harnessCase) harnessReport
func loadHarnessConfig(path, alias string, readFile func(string) ([]byte, error), lookupEnv func(string) (string, bool)) (harnessConfig, error)
func verifyManifest(raw []byte, expectedDigest string) (string, error)
```

- `LoadManifest` 避免 live 开关判断前读取材料；request closure 不再接收调用方 context，所有请求使用 harness 派生 context。
- 小结构体为值语义，配置 map 必须复制；`harnessRequest` 是函数类型，不是命名常量类型。
- `Now` 与 `Wait` 不允许 nil fallback；真实装配显式注入同一时钟域的 `time.Now` 和 context-aware timer wait。
- deadline 使用标准库 `context.WithDeadline`，不新增 clock framework、第三方依赖或 deadline factory。
- callback 必须同步结束自身启动的 goroutine，不得保存 request closure。harness 自身不启动 worker goroutine，不支持 operation 递归调用同一 limiter。
- 非 nil context、loader、clock、waiter、case callback 的校验失败均在请求前 BLOCKED。

#### HC-03：外部配置

配置格式冻结为 JSON：

```json
{
  "targets": {
    "example": {
      "endpoint_env": "CHAIN_API_GO_EXAMPLE_ENDPOINT",
      "header_env": {"Authorization": "CHAIN_API_GO_EXAMPLE_AUTHORIZATION"},
      "network": "example-network",
      "max_requests": 10,
      "max_concurrent": 1,
      "min_interval": "1s",
      "total_timeout": "30s",
      "dynamic_inputs_path": "",
      "submission_material_path": ""
    }
  }
}
```

- `path` 和 `alias` 必须显式非空；只读取指定路径一次，不搜索 CWD、home、参考仓库或公共节点。
- 具体链 runner 显式选择 alias 并创建具体客户端；`targets` 不建立 SDK dispatcher。
- endpoint 仅通过 `endpoint_env` 获取；Header 值仅通过 `header_env` 获取。环境值按原样使用，不支持插值、递归展开、默认值、shell 或命令替换。
- env 引用名匹配 `[A-Za-z_][A-Za-z0-9_]*`；缺失或空值 BLOCKED，错误不得回显变量值、endpoint、Header 或路径。
- endpoint 必须是有 host 的 `http`/`https` URL 且无 fragment；不改写、不拼接、不猜测其他链 URL。
- Header 名必须为合法 HTTP token，大小写归一后不能重复；值不得含 CR/LF。
- 必须恰有一份 JSON object；拒绝 unknown fields、重复键（包括解码后相同的转义键）、尾随文档和类型错误。
- 四个限额均严格大于零；duration 使用 `time.ParseDuration`，不接受隐含单位、缺省值或溢出。
- 两个 material path 是可选字符串，仅保存、不展开、不打开；对应材料必需性仍由 T033/T034 验证。
- loader 不返回部分可用配置；错误只保留安全类别，不保留原始 `PathError`、URL 或 JSON 错误文本。
- `config.example.json` 只含上述占位引用，不含实际 URL、Header 值、Token 或交易载荷；普通测试只可读取仓库模板。

#### HC-04：Manifest 身份

- `verifyManifest` 对传入 raw bytes 计算 SHA-256，返回 64 位小写十六进制；预期摘要必须独立冻结，不能由同一输入运行时生成。
- 哈希覆盖全部字节，包括空白、换行、键顺序、数组顺序和时间元数据；不删除字段、不 canonicalize。
- 输入还须为一份合法非 null JSON object，拒绝重复键和尾随文档；语义内容与五链覆盖仍由后续 runner 负责。
- 缺失、格式错误或摘要不匹配在 case/网络调用前 BLOCKED；成功报告绑定实际摘要，失败不输出原始 manifest。
- 本条取代 architecture §12.5 的规范化摘要草案；本修订不编辑 architecture，也不形成 constitution 例外。

#### HC-05：请求与资源

- 一个请求单位是一次被 limiter 接纳并进入 operation callback 的调用，包括失败调用；不宣称等于实际网络报文数或供应商 credit。
- 每个 operation 最多调用一次注入的实际请求依赖；不得隐藏多次 RPC、重试或广播。
- 达到 `MaxRequests` 后下一次请求立即拒绝并 BLOCKED；拒绝不增加计数。
- 容量满时等待且响应 suite context 取消；容量本身不产生 FAIL/BLOCKED。
- 活跃 operation 不超过 `MaxConcurrent`；请求计数、槽位、开始时刻和错误账本必须并发安全。
- 相邻被接纳 operation 的实际开始时刻至少间隔 `MinInterval`；第一项无需等待。等待后重新检查 deadline 和预算。
- operation 返回后释放槽位；取消、等待失败和预算拒绝不得泄露槽位或消耗请求数。
- 不承诺 FIFO、公平性、优先级或后台调度。

#### HC-06：总 timeout

- 总执行预算从本地配置、manifest、catalog 校验完成并准备执行首个 case 时开始。
- `deadline = min(parent deadline, start + TotalTimeout)`；所有 case、排队、速率等待和 operation 共用派生 context。
- 总时限覆盖整个 case 执行阶段和在途 I/O；同步本地文件读取不承诺被该 deadline 强行中断。
- admission 前、等待后、operation 返回后检查 context 和时间；`now >= deadline` 不接纳新 operation。
- deadline 到达时 `Done()` 关闭，operation 可观察 `context.DeadlineExceeded`；父取消传播，不得 detached。
- 返回前取消派生 context 并释放 timer；不关闭调用方 HTTP transport。
- 依赖须尊重 context；harness 不以泄漏 goroutine 的方式强杀不合作 callback。

#### HC-07：账本、错误与优先级

- required IDs 独立、非空、无空 ID/重复 ID。cases 可缺项以保留事实；重复/未知/child-only/nil Run 属于 catalog invalid，在请求前 BLOCKED。
- 每个 required ID 初始 `NOT_RUN`，每个 case 至多执行一次；不能用其他 ID 补足。
- 启用后：存在明确失败则总体 `FAIL`；无失败但有 BLOCKED、NOT_RUN、缺项或资源阻塞则总体 `BLOCKED`；全部 required PASS 且无基础错误才总体 `PASS`。关闭开关总体 `NOT_RUN`。
- 非法 callback status 或未分类非 nil callback/operation error 为 FAIL；明确 context cancel/deadline 为 BLOCKED。
- 包内值错误 `harnessFailure{Status, Category string}` 仅接纳 FAIL/BLOCKED 和类别 assertion/environment；非法值按 `FAIL/invalid_result`，`Error()` 只输出类别。
- 预期节点拒绝由 callback 完成断言后返回 nil；limiter/operation 失败为 sticky，callback PASS 不能覆盖。
- case 与总体均按 `FAIL > BLOCKED > PASS`；未运行项保留 NOT_RUN。相同严重度按固定类别字典序选择摘要，避免并发顺序影响报告。
- 报告仅含状态、固定安全类别、合法 required ID、digest 和请求数；不保存任意错误文本或 config。
- SDK 原始错误是 callback 可检查的输入，不是 report 的序列化模型；request closure 既有返回固定 harness 类别的行为不变。不得把 harness 输出脱敏反向应用到 SDK error。
- 不做 panic recover/logging；callback panic 由 Go 测试进程失败。

#### HC-08：确定性离线验证

- 使用 Go 1.25 标准库 `testing/synctest` 与 channel/barrier，不新增依赖；真实启动 `limit+1` 个 goroutine，证明额外请求等待、释放后继续，并覆盖 limit 1/2。
- 使用原子或 mutex 验证 peak、callback 次数、请求账本及全部 goroutine 退出；race 检查是强制门禁。
- synctest 内使用 `time.Now` 和 context-aware timer waiter，统一 fake time 与 context timer；不再只修改局部 fake clock。
- 检查 operation 的 Deadline、更早父 deadline、预取消、排队/速率等待取消、在途 Done、截止边界与 timer 清理。
- raw digest 使用独立 literal；空白/键序变化必须在旧摘要下失败。malformed/duplicate/trailing 使用各自匹配 raw digest，避免只因 hash 不同造成假覆盖。
- config 测试注入 reader/env map，断言精确路径/读取次数；仅使用临时合成文件组合正例，不读取用户实际 env/config。
- 脱敏覆盖 harness 自有 loader、parser、manifest、waiter、operation、case 错误及 report；检查其 JSON 与 `%v/%+v/%#v` 无合成秘密。本断言不要求 SDK 原始错误满足输出脱敏。
- 覆盖混合 FAIL+BLOCKED、预算错误被忽略、request error 被 PASS 覆盖、缺项和 child-only 冒充。

SDK→runner→harness 的信息边界由既有 T033/T034 覆盖：callback 可在内存检查 Message/Data/ResponseBody/Cause，runner 只向 report 输出固定类别；不在 T023/T024 重复 SDK 字段矩阵或伪造新的 RED。

### 7.3 开关

- `CHAIN_API_GO_LIVE_TEST=1`：允许真实只读验收。
- `CHAIN_API_GO_LIVE_BROADCAST=1`：在 live 开关之外单独允许受控提交。
- `CHAIN_API_GO_LIVE_CONFIG`：由用户设置的外部配置路径；不得提交配置内容。

### 7.4 Task Contract / Task Contract

```yaml
planning_baseline:
  version: PLAN-v7-REMAINING-RAW-v1
  approval_status: awaiting_human_approval
  approved_file_set:
    - docs/superpowers/001-chain-api-go/spec.md
    - docs/superpowers/001-chain-api-go/plan.md
    - docs/superpowers/001-chain-api-go/tasks.md
  constitution_version: "1.3 Active；只读，不在本次修改集合"
  manifest: not_applicable_single_requirement

workflow_state:
  current: AWAITING_FULL_REMAINING_BASELINE_APPROVAL
  active_task: none
  first_authorizable_task: T027
  blocked_tasks: [T027, T028, T029, T030, T031, T032, T033, T034, T035, T036]
  preserved_completed_tasks: [T017, T018, T019, T020, T021, T022, T023, T024, T025, T026]
  reopened_tasks: []
  execution_sequence: [T027, T028, T029, T030, T031, T032, T033, T034, T035, T036]
  coder_may_report: [IMPLEMENTED]
  transitions: [IMPLEMENTED, LOCALLY_VERIFIED, READY_FOR_AUDIT, AUDIT_PASSED, DONE]
  done_requires: [valid_snapshot_bound_evidence, auditor_pass_recommendation, human_final_approval]

scope:
  allowed_paths: "每个 Task 中展开列出的路径，仅对该 Task 有效"
  allowed_behavior_changes: "spec §3、§4.3–§4.5、AC-01–13 与本方案 §3.3–§4、§7.8 的批准契约"
  invariants: "spec C-01–10；Human 批准后的 constitution 1.4；实现阶段不得修改 constitution"
  non_goals: "spec §6；无服务、索引、后台机制、新依赖或本期钱包示例"
  protected_evidence: "先行测试断言、批准验收标准、冻结样本及摘要"
  requires_approval:
    - "扩大路径、行为、公开 API、依赖或配置范围"
    - "改变受保护测试、样本或验收标准"
    - "新增后台行为、抽象层或并发机制"
    - "constitution、数据库 NULL 或第三方契约冲突"

audit_test_scope:
  allowed_paths: []
  access: read_only
  supplemental_test_writes_require_separate_approval: true

verification_layers:
  L1: "必需：需求、不变量与双向追踪"
  L2: "必需：聚焦编译、vet、依赖和公开 API"
  L3: "必需：表格驱动单元、边界和异常测试"
  L4: "子 Plan 仅 fake/httptest 契约；真实集成延后 T036"
  L5: "仅批准风险：原始错误信息保留与 harness 输出隔离、资源关闭、请求预算、JSON-RPC ID race、live limiter/账本 race 与 context deadline 生命周期"
  L6: "不适用：无部署交付；真实链验收属于 T036"

evm_raw_dto_delta:
  affected_tasks: [T025, T026]
  authority: "spec §4.2 ER-01–05；plan §3.2"
  status: historical_completed
  protected_test_revision: "仅 T025 按 §3.2 撤换被替代契约；新 RED 审核后冻结"
  historical_evidence: "T025/T026 的 RED/GREEN、审核、Human 门禁与集成已完成；本轮不重开"

raw_error_delta:
  authority: "Human 2026-11-17 原始错误裁决；spec AC-04/RE-01–08；plan §4"
  status: historical_completed
  affected_plans: [PLAN-001, PLAN-002, PLAN-003, PLAN-004, PLAN-005]
  reopened_tasks: []
  preserved_completed_tasks: [T017, T018, T019, T020, T021, T022, T023, T024, T025, T026]
  paused_unapproved_tasks: []
  new_plans: 0
  new_tasks: 0
  new_packages: 0
  new_dependencies: 0
  new_runtime_configuration: 0
  architecture_sync: not_authorized_this_revision
  protected_evidence_revision:
    permitted_in: [T017, T019, T021, T025]
    replace_only: "原始字段禁止、Cause 裁剪、取消优先归一化、SDK 错误全格式脱敏"
    retain: "DI、协议信封、限长、body 关闭、ID 并发、路径边界、DTO、nullable、单次提交、网络身份"
    becomes_protected_after: "对应当前 RED、固定 Auditor 及 Human 门禁"
  stop_conditions:
    - "需要新增路径/API/配置/依赖，且不在本候选精确字段清单内"
    - "无法保留 Cause 而不改变既有提交语义"
    - "出现新的协议或值语义冲突"
    - "共享时箱耗尽或固定角色不可用"

bitcoin_contract_delta:
  affected_tasks: [T027, T028]
  authority: "历史 BITCOIN-CONTRACT-v1；已由 PLAN-v7-REMAINING-RAW-v1 候选替代"
  status: superseded_unapproved_history
  scope: "T027原7路径；T028原5路径；无新Plan/Task/依赖"
  preserved_evidence: "无实现证据；T017–T026 已完成证据不受本候选影响"

remaining_scope_delta:
  authority: "spec BC/SOL/TR/LR；plan §3.3–§3.5、§7.8"
  status: awaiting_human_approval
  new_plans: 0
  new_tasks: 0
  new_dependencies: 0
  core_configuration_sources_added: 0
  historical_architecture_write_authorized: false
  governance_file_write_authorized: false
  live_execution_authorized: false
  git_operations_authorized: false

harness_contract_delta:
  affected_tasks: [T023, T024]
  authority: "spec AC-09/§5.1；plan §7.2 HC-01–HC-08"
  status: completed_preserved_in_raw_error_v1
  protected_test_revision:
    permitted_only_in: T023
    reason: "旧断言包含未经批准的 digest/admission/status 选择，且缺 config/deadline/race 边界"
    becomes_protected_after: "HARNESS-CONTRACT-v1 的 T023 RED 与固定 Auditor 审核通过；RAW-ERROR-v1 不重开"
  production_concurrency:
    permitted: "请求 closure 的可取消等待、同步槽位/计数/速率控制"
    prohibited: "后台 worker、自动重试、轮询、FIFO/优先级框架"
  external_read:
    permitted: "仅 test-only 显式路径与注入 env lookup"
    prohibited: "核心 SDK 读取、默认路径、公共 RPC fallback、关闭开关后读取"
  historical_evidence: "HARNESS-CONTRACT-v1 之前的旧 T023 仅作历史；已完成的 T023/T024 证据在 RAW-ERROR-v1 中继续有效，不重新要求 RED"

evidence_provenance:
  bind_to: [task_id, acceptance_ids, command, exit_code_and_result, code_snapshot, test_snapshot, dependency_state, redacted_configuration_identity, environment]
  invalidate_when: "相关输入变化时只使受影响证据失效"
  rerun_policy: "复用有效证据；失败修复或输入变化后只重跑受影响检查"

failure_routing:
  F1_F4: "同一 Coder 闭环；权限扩大交 Human"
  F5_F6: "有证据才返回 Planner；需求裁决由 Human 作出"
  max_same_signature_diagnostic_cycles: 3
  on_exhaustion: "停止循环，报告日志、已尝试方案与所需输入"

integration_stage:
  test_commands_allowed: false
  permitted_checks: "Git/worktree/diff/身份/冲突结构检查"
  commit_push_pr_merge: "无授权；当前用户未要求"
  on_content_change: "退出集成，返回受影响实现、验证、审核和 Human 门禁"

rollback:
  method: "依据 pre-task snapshot 撤回该 Task 自有增量；不得 reset 全仓或覆盖用户改动"
  trigger: "越界、无法复现、违反不变量或审批撤回"

stop_conditions:
  - "三件套未落盘、元数据不一致、缺字段，或当前待执行 Task 对应基线未获 Human 明确批准"
  - "角色固定模型不可用且没有明确降级授权"
  - "需要扩大范围或出现规范/协议冲突"
  - "无法取得可信 RED/GREEN 或最终真实材料"
```

### 7.5 RAW-ERROR-v1 测试—风险矩阵 / Test-Risk Matrix

| 风险 | 最少证据 | 任务/层级 |
|---|---|---|
| 原错误类型或包装链丢失 | 普通 sentinel、类型化 error、包装 context、双 context join；验证 Cause 身份及 `errors.Is/As` | T017→T018；L3 |
| 字段与格式不一致 | Message 优先、Cause 回退、空 Kind、Data 不自动拼入字符串、值副本不改原值 | T017→T018；L3 |
| JSON-RPC detail 丢失 | code=0、Message 原样、Data 缺失/null/大数字/对象；2xx 与非 2xx 有效信封 | T019→T020；L3/L4 |
| 正文丢失或伪称完整 | 非 JSON HTTP body、刚好限长、超限一字节、部分读失败及原错误 | T019→T022；L4 |
| 用 context 覆盖真正故障 | 包装 context 保留；context 同时取消但实际错误不匹配时不替换 | T019→T022；L3 |
| SDK 主动追加秘密 | 请求合成标记不在响应/Cause 时错误不得新增；节点或底层自带时必须保留 | T019→T022；L4 |
| 链包装再次裁剪 | EVM 查询保留字段和原 wrapper；两具体客户端复用同一接线 | T025→T026；L3 |
| unknown 被文本改变 | 不同英文 Message、相同 code/阶段得到相同分类；拒绝、超时、畸形结果和一次调用 | T025→T032；L3 |
| 原始 error 进入证据 | callback 能读取原始字段，最终 report/fixture 无合成标记；context 分类仍正确 | T033→T034；L4 |
| 过度防御反弹 | 每个新增分支可追溯 RE 或既有协议必要性；无内容扫描、重复 context、fallback/recover | 各受影响 Task；L1/L2 |

不新增 PoC、benchmark、fuzz、全字段 schema 或 E2E。现有标准库字段与包装机制足以建立可行性，真实链验收仍只在 T036。

### 7.6 两级验证与双向追踪 / Two-Level Verification and Traceability

| Plan | per_plan_checks | final_acceptance_contribution |
|---|---|---|
| PLAN-001 | T017–T024 对应命令；协议 fake/httptest、race、安全与 harness 契约 | T036 的开关、预算、摘要、账本与脱敏 |
| PLAN-002 | `go test ./evm ./ethereum ./bnb`；两具体客户端分别覆盖 | T036 Ethereum 与 BNB 各至少 3 样本及分别授权提交 |
| PLAN-003 | `go test ./bitcoin` | T036 Bitcoin 样本、历史能力、精确金额、授权提交 |
| PLAN-004 | `go test ./solana` | T036 Solana 样本、meta/版本/动态输入、授权提交 |
| PLAN-005 | `go test ./tron`；runner 离线契约；`go vet ./...` | T036 Tron 样本及五链最终去重验收 |

| 计划变化 / Planned Change | 批准依据 / Authority |
|---|---|
| transport、apierror、jsonrpc、httpjson | spec §3.1/§4.4，C-02/05/06，AC-02–04；RE-01–06 |
| evm、ethereum、bnb | spec §3.2，AC-05 |
| bitcoin | spec §4.3 BC-01–05；plan §3.3；AC-02/04/06/12/13；T027→T028（候选） |
| solana | spec §4.3 SOL-01–05；plan §3.4；AC-02/04/07/12/13；T029→T030（候选） |
| tron 与五个 solidified 查询 | spec §4.3 TR-01–05；plan §3.5；AC-02/04/08/12/13；T031→T032（候选） |
| live 配置、开关、账本、runner、样本、证据 | spec §5，AC-09–11 |
| HARNESS-CONTRACT-v1 HC-01–HC-08 | spec AC-09、AC-04、C-02/05/06/08/09；T023 RED → T024 GREEN；不改变 T033–T036 依赖与真实验收职责 |
| RAW-ERROR-v1 SDK→runner 输出隔离 | spec RE-08；T033 RED → T034 GREEN；不重开 T023/T024 |
| 常量转换、值语义和核心依赖/API 审查 | constitution 5.1–5.3、8.5/8.6，AC-12/13 |
| README/GoDoc 基线同步；AGENTS/constitution/architecture 只读核对 | AC-01、12、13；不授权修改治理或历史架构文件 |

`final_acceptance_checks` 只有 T036：依赖全部五个 Plan 的本地交付；缺少真实调用、提交授权或材料时状态为 BLOCKED，不能 PASS。

### 7.7 资源预算与冲突门禁 / Budget and Conflict Gate

EVM-RAW-DTO-v1/T025–T026 统一采用 §3.2 的 20/20/30 分钟共享时箱与行数预警，替代此前 EVM 声明骨架和小时预算。T025 原十路径不扩大且本轮仅写六测试文件；T026 原七生产路径不变。

RAW-ERROR-v1 增量共享规划 20 分钟、生产编码 20 分钟、测试执行与独立审核 30 分钟时箱，覆盖 T017–T022 及 T025/T026 中本次错误契约增量，不按 Task 重置。预计增量生产代码约 120–220 行、测试代码约 220–400 行，均在既有路径内；EVM 原始 DTO 工作单独记账，同一工作不得重复计费。时箱耗尽时保留快照并报告，不新增 Task、不弱化断言、不自动延长。

HARNESS-CONTRACT-v1 的 T023、T024 各自最多 70 分钟：规划 20 分钟、编码 20 分钟、验证最多 30 分钟。该时间是实际执行上限，不再套用 40%/5%/55% 的机械小时换算。任务总数仍为 20；生产行、测试行和文件数只作为范围漂移预警，不是目标额度，也不得成为延长工期、增加抽象或重复验证的理由。

上述三段是已完成 T017–T026 的历史预算，不授权剩余工作。PLAN-v7-REMAINING-RAW-v1 的以下硬时箱已随基线获 Human 批准：T027/T028、T029/T030、T031/T032、T033/T034 各功能对共享规划 20、生产编码 20、验证（含测试代码、Controller、Auditor）30 分钟；T035 与 T036 各自最多 70 分钟并沿用 20/20/30 分段，不需要的生产编码额度不得转移。任何共享时箱不按 Task、方法/修复轮次重置；到期保留快照并报告，不能自动恢复旧 7–8h、弱化验收或新增 Task。

40%/5%/55% 仅作为资源估算与偏差诊断，不替代硬时箱：每个功能对预计 50 分钟（20/2.5/27.5）；T035 预计 25 分钟（10/1.25/13.75）；T036 预计 50 分钟（20/2.5/27.5）。剩余整体预计 275 分钟，规划 110、编码 13.75、验证 151.25；硬时箱合计上限 420 分钟。等待 Human、外部材料或链确认不计作自动延时授权。

| Task | 生产/声明行预警 | 测试/文档/证据预警 | 路径数 |
|---|---:|---:|---:|
| T027 | 180–260 声明 | 500–750 测试 | 7 |
| T028 | 350–500 生产 | 0 新测试 | 5 |
| T029 | 180–260 声明 | 550–800 测试 | 7 |
| T030 | 350–500 生产 | 0 新测试 | 5 |
| T031 | 180–260 声明 | 550–850 测试 | 7 |
| T032 | 350–500 生产 | 0 新测试 | 5 |
| T033 | ≤120 声明 | 500–800 测试 | 6 |
| T034 | 450–650 生产 | 入口测试含于 `live_test.go` | 5 |
| T035 | 0 | 350–550 文档 | 13 |
| T036 | 0 | 按实际样本，不以行数作目标 | 8 |

超过约 400 生产行或 8 个主要路径只触发范围/风险复核；不能按分钟或行数拆分既有 Plan，不增加 T037。

`docs/architecture.md` v0.4 是历史设计草案。其 §10.1 原始错误方向与 RAW-ERROR-v1 一致，但具体字段、值接收者、HTTP body、Error() 与 Cause 语义以本方案 §4 为准；草案的 `*Error`/`*int` 不恢复。§7.2–7.4 的指针/nullable、旧 EVM Parse API、全字段校验与 §12.5 摘要均不得恢复。当前候选只修改三件套，不同步 architecture、constitution 或 AGENTS.md，也不把这些路径放入 T035。

### 7.8 五链 runner 完整契约 / Five-chain Runner Contract

> §7.8.1 之前的内容仅为历史摘要，不作为实现或材料校验依据。runner 的入口、类型、JSON 字典、断言、状态与输出完整地由 §7.8.1–§7.8.7 替代。

runner 只存在于 `internal/test/live`，是测试装配而不是核心统一链入口。每链直接接收具体客户端并创建独立 case closures；不得创建通用 `ChainClient`、RPC 方法执行表、注册表或 dispatcher。已完成的 `harness.go`、`config.go`、`manifest.go` 及其受保护测试不修改。

内部入口固定为：

```go
ethereumReadCases(*ethereum.Client, readManifest, dynamicInputs) ([]harnessCase, error)
bnbReadCases(*bnb.Client, readManifest, dynamicInputs) ([]harnessCase, error)
bitcoinReadCases(*bitcoin.Client, readManifest, dynamicInputs) ([]harnessCase, error)
solanaReadCases(*solana.Client, readManifest, dynamicInputs) ([]harnessCase, error)
tronReadCases(*tron.Client, readManifest, dynamicInputs) ([]harnessCase, error)
// 五条链各自另有 <chain>SubmissionCases，接收具体 Client 与 submissionMaterial。
```

共用部分只包含数据容器、材料读取和固定安全结果，不统一链 API。共用声明放 `evm.go`，链特定输入/断言放对应文件，不新增 helper 包或路径。

`readManifest` 的 JSON 结构固定为：`manifest_version=1`、`chain`、`network`、`network_identity`、`samples`。每个 sample 包含非空 `sample_id`、本链批准 `category`、公开 `source_url`、RFC3339 `observed_at`、原生 `transaction_id`、`block_hash`、`height_or_slot`、`addresses`、`expected`、覆盖 expected 全部 key 的 `field_sources`、`rpc_preflight` 与 `requirements`。`expected`/`network_identity` 使用 `map[string]json.RawMessage`，整数不得经 float64。manifest 不存自身摘要；独立摘要来自 dynamic inputs，仍使用 HC-04 原始文件字节 SHA-256。

`dynamicInputs` 顶层字段固定为：`manifest_sha256`、`max_response_bytes>0`、仅 EVM 使用的十进制正数 `expected_chain_id`、`inputs`、覆盖全部动态方法且 status=PASS 的 `preflight`、RFC3339 `observed_at/expires_at`。只读取 `cfg.DynamicInputsPath`，不搜索默认路径、不递归解析 env、不读取参考仓库配置。过期或身份不匹配为 BLOCKED。

| 链 | `inputs` 固定字段 |
|---|---|
| EVM | `address evm.Address; at_block_number uint64; call/estimate evm.CallRequest; fee_count uint64; fee_percentiles []float64`；外部 log_filter 为 from/to block number、addresses、topics，runner 用 `evm.AtNumber` 构造 |
| Bitcoin | `txout bitcoin.GetTxOutRequest; mempool_txid bitcoin.TxID; estimate bitcoin.EstimateSmartFeeRequest; preflight_payload bitcoin.RawTransaction` |
| Solana | `account solana.Pubkey; read/blockhash/account_options/transaction_options/block_options/status_options`；`fee_message`、`simulate_transaction`、`simulate_options` |
| Tron | `account tron.GetAccountRequest; contract_call tron.ContractCallRequest` |

固定 required catalog 不能从实际 cases 或 manifest 反推：Ethereum/BNB 各 13 个非提交方法；Bitcoin 9 个；Solana 10 个；Tron 15 个（含五个 solidified）。ID 为 `<chain>.<公开方法名>`，另固定 `<chain>.sample.1` 至 `.sample.3`。每链 samples 至少三笔不同交易，按声明顺序前三笔满足 required sample；额外样本不补齐缺项。

历史样本断言固定为：EVM 核对 transaction/receipt/block 的 hash、区块、from/to/value、status、logs；Bitcoin 以显式 blockhash 的 raw transaction + block 核对 txid/wtxid、vin/vout、script/witness、精确金额；Solana transaction + block 核对 signature、slot、message.accountKeys、meta.err/fee/balances、version；Tron transaction/info/block 及对应 solidified transaction/info 核对 txID、区块、raw_data、receipt、fee、resource、log。可空字段用 JSON null，缺失字段用 `{"absent":true}`；不得实现通用 JSONPath/断言语言。

动态余额、nonce、latest、费用、确认数不与采集时值永久等同；固定历史能力使用固定 block/slot。计算成功或原生拒绝必须由独立 preflight 明确记录。每次实际 SDK 调用放在一个 `harnessRequest` operation 中，最多调用依赖一次。字段差异为 FAIL/assertion；已识别的材料、权限、历史能力前提为 BLOCKED/environment；未知 SDK error 为 FAIL，不按英文 message 分类。原始 SDK error 只在 callback 内检查，report 只含 harness 已批准字段。

`submissionMaterial` 固定字段：`version=1`、`chain`、`network`、`endpoint_alias`、`authorization_id`、`authorized=true`、`expires_at`、`network_identity`、`payload_path`、原始字节 `payload_sha256`、调用方预先给出的 `transaction_id`、仅 Bitcoin 可用的 `block_hash`、`send_options`、`expected_execution`。除 Solana 外 send_options 必须为空；Solana 使用冻结的 `SendOptions` 与 `EncodedTransaction.Encoding`。Tron payload 文件是 JSON 原交易，其余链为编码字符串原始文件字节，不 trim；payload 不嵌入 manifest/evidence。

`TestLiveRead` 首先且只读取 `CHAIN_API_GO_LIVE_TEST`；不是 `1` 时 Skip 且零其他读取。启用后读取 `CHAIN_API_GO_LIVE_CONFIG`，使用固定 alias `ethereum_read`、`bnb_read`、`bitcoin_read`、`solana_read`、`tron_read`，并固定读取 `testdata/<chain>/samples.json`。test-only 宿主创建禁止 redirect/重试的 HTTP Client 并显式注入；核心 SDK 不创建客户端。五链逐一 `runHarness`，只有启用入口可写 `evidence/read.json`。

`TestLiveBroadcast` 先依次检查 `CHAIN_API_GO_LIVE_TEST=1` 与 `CHAIN_API_GO_LIVE_BROADCAST=1`，缺任一项即 Skip 且零配置/材料读取。使用固定 broadcast alias 和 `cfg.SubmissionMaterialPath`。逐链核对授权、网络、alias、有效期、payload digest、transaction ID 与材料使用账本；提交前通过 EVM ChainID、Bitcoin chain+genesis、Solana GenesisHash、Tron genesis BlockID 核对身份。

每链 broadcast required IDs 为 identity、submit、lookup。每份材料只提交一次；失败或 unknown 后不得重发。submit 成功时 EVM/BTC/Solana 返回 ID 必须匹配材料，Tron 有 txid 时必须匹配。lookup 始终使用原材料 transaction ID：EVM receipt、Bitcoin raw transaction、Solana statuses、Tron solidified info；只查询一次，未达到 expected_execution 时 BLOCKED，不轮询、不重发。后续只读复查需新授权且不得重跑 submit。报告只写 `evidence/broadcast.json`，不写 payload、完整错误或秘密。

独立 preflight 只在 T036 由直接 HTTP 路径完成，不调用被测 SDK，不由 runner 重跑，也不由 SDK 生成 expected；请求计入 T036 人工账本。真实提交再次运行必须另获授权并使用新的专用材料。

#### 7.8.1 装配入口与读取边界 / Entry Points

以下均为包内接口，不导出 SDK API。共用数据声明、入口和安全报告放 `evm.go`；各链 case 放对应链文件；`live_test.go` 只装配真实依赖并把结果映射为 testing 的 Skip/Fail。不新增文件。

```go
type runnerDeps struct {
    LookupEnv func(string) (string, bool)
    ReadFile func(string) ([]byte, error)
    NewDoer func(harnessConfig) (transport.HTTPDoer, func(), error)
    Now func() time.Time
    Wait func(context.Context, time.Duration) error
    WriteReadReport func([]byte) error
    ReadBroadcastJournal func() ([]byte, error)
    AppendBroadcastRecord func([]byte) error
}

func runLiveRead(context.Context, runnerDeps) suiteReport
func runLiveBroadcast(context.Context, runnerDeps) suiteReport

func ethereumReadCases(*ethereum.Client, readManifest, dynamicInputs) ([]harnessCase, error)
func bnbReadCases(*bnb.Client, readManifest, dynamicInputs) ([]harnessCase, error)
func bitcoinReadCases(*bitcoin.Client, readManifest, dynamicInputs) ([]harnessCase, error)
func solanaReadCases(*solana.Client, readManifest, dynamicInputs) ([]harnessCase, error)
func tronReadCases(*tron.Client, readManifest, dynamicInputs) ([]harnessCase, error)

type claimMaterial func() error

func ethereumSubmissionCases(*ethereum.Client, submissionMaterial, []byte, claimMaterial) ([]harnessCase, error)
func bnbSubmissionCases(*bnb.Client, submissionMaterial, []byte, claimMaterial) ([]harnessCase, error)
func bitcoinSubmissionCases(*bitcoin.Client, submissionMaterial, []byte, claimMaterial) ([]harnessCase, error)
func solanaSubmissionCases(*solana.Client, submissionMaterial, []byte, claimMaterial) ([]harnessCase, error)
func tronSubmissionCases(*tron.Client, submissionMaterial, []byte, claimMaterial) ([]harnessCase, error)
```

`runLiveRead` 第一个且唯一允许先执行的依赖调用是 `LookupEnv("CHAIN_API_GO_LIVE_TEST")`；值不是 `"1"` 时返回 `Status=NOT_RUN`，其他依赖全部零调用。`runLiveBroadcast` 先检查同一开关，仅当为 `"1"` 时再读取 `CHAIN_API_GO_LIVE_BROADCAST`；任一不为 `"1"` 即 `NOT_RUN`，禁止读取配置路径、文件、时钟、账本或创建客户端。入口关闭时不验证其他依赖，不以 nil 检查掩盖此零调用契约。

启用后从 `LookupEnv("CHAIN_API_GO_LIVE_CONFIG")` 获取非空显式路径。read alias 精确为 `ethereum_read`、`bnb_read`、`bitcoin_read`、`solana_read`、`tron_read`；broadcast alias 精确为 `ethereum_broadcast`、`bnb_broadcast`、`bitcoin_broadcast`、`solana_broadcast`、`tron_broadcast`。按此顺序串行执行，不用链注册表或可配置 dispatcher。固定 required 目录独立于实际 cases。

`NewDoer` 由 test-only 宿主显式实现，创建 `http.Client{Timeout:cfg.TotalTimeout}` 和其自有 `http.Transport`，`CheckRedirect` 返回 `http.ErrUseLastResponse`，无重试包装。返回 cleanup 只关闭该宿主拥有的 idle connections，runner 在该链结束时调用一次。协议 Config 的 Endpoint/Headers 来自 harnessConfig，MaxResponseBytes 来自下述 dynamicInputs；EVM/Solana 显式 `jsonrpc.V2`，Bitcoin 显式 `jsonrpc.Bitcoin`；Tron 显式 `httpjson.Config`。不修改核心构造器或 harness。

每链在开关启用后读取配置、dynamic 文件和 manifest/material 各一次，在任何网络调用前验证全部本地材料。read manifest 固定路径为 `testdata/<chain>/samples.json`；broadcast material 来自 `cfg.SubmissionMaterialPath`；两种模式均必须有 `cfg.DynamicInputsPath`。不存在配置或材料即 `BLOCKED`，不搜索替代路径。

read 时 `runHarness.LoadManifest` 返回已读的 samples.json 原始字节及 `dynamic.ManifestSHA256`；broadcast 时返回已读的 submission material 原始字节及同一字段。Load 返回已读配置的值副本。已有 harness 负责摘要检查、限额和 deadline；不能让广播使用自身运行时生成的摘要。所有实际节点请求均在 harnessRequest 内，每个 operation 只调用一个 SDK 方法。

#### 7.8.2 数据结构与 JSON 字典 / Data Contracts

```go
type networkIdentity struct {
    ChainID string `json:"chain_id"`
    BitcoinChain string `json:"bitcoin_chain"`
    GenesisHash string `json:"genesis_hash"`
    TronGenesisBlockID string `json:"tron_genesis_block_id"`
}

type rpcExpectation struct {
    Outcome string `json:"outcome"`
    RPCCode int `json:"rpc_code"`
    HasRPCCode bool `json:"has_rpc_code"`
    NodeCode string `json:"node_code"`
    Expected json.RawMessage `json:"expected"`
}

type preflightRecord struct {
    Status string `json:"status"`
    ObservedAt string `json:"observed_at"`
    EndpointAlias string `json:"endpoint_alias"`
    NetworkIdentity networkIdentity `json:"network_identity"`
    Methods map[string]rpcExpectation `json:"methods"`
}

type readSample struct {
    SampleID string `json:"sample_id"`
    Category string `json:"category"`
    SourceURL string `json:"source_url"`
    ObservedAt string `json:"observed_at"`
    TransactionID string `json:"transaction_id"`
    BlockHash string `json:"block_hash"`
    HeightOrSlot uint64 `json:"height_or_slot"`
    Addresses []string `json:"addresses"`
    Expected json.RawMessage `json:"expected"`
    FieldSources map[string]string `json:"field_sources"`
    RPCPreflight preflightRecord `json:"rpc_preflight"`
    Requirements []string `json:"requirements"`
}

type readManifest struct {
    ManifestVersion int `json:"manifest_version"`
    Chain string `json:"chain"`
    Network string `json:"network"`
    NetworkIdentity networkIdentity `json:"network_identity"`
    Samples []readSample `json:"samples"`
}

type dynamicInputs struct {
    ManifestSHA256 string `json:"manifest_sha256"`
    MaxResponseBytes int64 `json:"max_response_bytes"`
    CodeSnapshot string `json:"code_snapshot"`
    Inputs json.RawMessage `json:"inputs"`
    Preflight preflightRecord `json:"preflight"`
    ObservedAt string `json:"observed_at"`
    ExpiresAt string `json:"expires_at"`
}

type evmLogInput struct {
    FromBlockNumber uint64 `json:"from_block_number"`
    ToBlockNumber uint64 `json:"to_block_number"`
    Addresses []string `json:"addresses"`
    Topics [][]string `json:"topics"`
}

type evmReadInputs struct {
    Address evm.Address `json:"address"`
    AtBlockNumber uint64 `json:"at_block_number"`
    Call evm.CallRequest `json:"call"`
    Estimate evm.CallRequest `json:"estimate"`
    LogFilter evmLogInput `json:"log_filter"`
    FeeCount uint64 `json:"fee_count"`
    FeePercentiles []float64 `json:"fee_percentiles"`
}

type bitcoinReadInputs struct {
    TxOut bitcoin.GetTxOutRequest `json:"txout"`
    MempoolTxID bitcoin.TxID `json:"mempool_txid"`
    Estimate bitcoin.EstimateSmartFeeRequest `json:"estimate"`
    PreflightPayload bitcoin.RawTransaction `json:"preflight_payload"`
}

type solanaReadInputs struct {
    Account solana.Pubkey `json:"account"`
    Read solana.ReadOptions `json:"read"`
    Blockhash solana.BlockhashOptions `json:"blockhash"`
    AccountOptions solana.AccountOptions `json:"account_options"`
    TransactionOptions solana.TransactionOptions `json:"transaction_options"`
    BlockOptions solana.BlockOptions `json:"block_options"`
    StatusOptions solana.StatusOptions `json:"status_options"`
    FeeMessage solana.EncodedMessage `json:"fee_message"`
    SimulateTransaction solana.EncodedTransaction `json:"simulate_transaction"`
    SimulateOptions solana.SimulateOptions `json:"simulate_options"`
}

type tronReadInputs struct {
    Account tron.GetAccountRequest `json:"account"`
    ContractCall tron.ContractCallRequest `json:"contract_call"`
}

type evmSampleExpected struct {
    Transaction evm.Transaction `json:"transaction"`
    Receipt evm.Receipt `json:"receipt"`
    BlockHash string `json:"block_hash"`
    BlockNumber string `json:"block_number"`
    ParentHash string `json:"parent_hash"`
}

type bitcoinSampleExpected struct {
    Transaction bitcoin.Transaction `json:"transaction"`
    BlockHash string `json:"block_hash"`
    BlockHeight int64 `json:"block_height"`
    TxIDs []string `json:"txids"`
}

type solanaSampleExpected struct {
    Transaction solana.Transaction `json:"transaction"`
    Blockhash string `json:"blockhash"`
    PreviousBlockhash string `json:"previous_blockhash"`
    ParentSlot uint64 `json:"parent_slot"`
}

type tronSampleExpected struct {
    Transaction tron.Transaction `json:"transaction"`
    TransactionInfo tron.TransactionInfo `json:"transaction_info"`
    BlockID string `json:"block_id"`
}

type executionExpectation struct {
    EVMStatus string `json:"evm_status"`
    BitcoinMinConfirmations int64 `json:"bitcoin_min_confirmations"`
    SolanaCommitment solana.Commitment `json:"solana_commitment"`
    SolanaErr json.RawMessage `json:"solana_err"`
    TronReceiptResult json.RawMessage `json:"tron_receipt_result"`
}

type solanaSubmissionOptions struct {
    Encoding solana.Encoding `json:"encoding"`
    Options solana.SendOptions `json:"options"`
}

type submissionMaterial struct {
    Version int `json:"version"`
    Chain string `json:"chain"`
    Network string `json:"network"`
    EndpointAlias string `json:"endpoint_alias"`
    AuthorizationID string `json:"authorization_id"`
    Authorized bool `json:"authorized"`
    ExpiresAt string `json:"expires_at"`
    NetworkIdentity networkIdentity `json:"network_identity"`
    PayloadPath string `json:"payload_path"`
    PayloadSHA256 string `json:"payload_sha256"`
    TransactionID string `json:"transaction_id"`
    BlockHash string `json:"block_hash"`
    SendOptions json.RawMessage `json:"send_options"`
    ExpectedExecution executionExpectation `json:"expected_execution"`
}
```

上述 SDK 请求值结构体嵌入外部 inputs 时，JSON 字段名固定使用其 Go 字段原名，包括 Has 字段；这只是 test-only 材料格式，不是节点 wire 格式。`evm.CallRequest`/Log 的字段按其已有 JSON tags。EncodedTransaction 的材料键精确为 `Data`、`Encoding`。SDK DTO 位于 expected 时采用 §3.2–§3.5 的 JSON tags；Bitcoin expected 中已转换字段直接存 sat/sat-kvB 整数，不能再存 BTC 小数。独立 preflight 工具按公开字段表投影和转换，禁止调用被测链 SDK 生成预期。

材料 JSON 只接纳一份 object，使用已有 strictObject 检查重复键/尾随；再标准解码到上述值结构体。该严格性仅属于已批准的验收材料边界，不复制到链 DTO。固定字典外的材料顶层字段为 `BLOCKED`；对 raw transaction/meta 等节点原始 JSON 不应用材料白名单。各链 Inputs 和 Sample.Expected 按上述四个具体类型解码，不实现通用链执行器。

network_identity 字典：EVM 仅 chain_id 非空，为十进制正整数；Bitcoin 仅 bitcoin_chain 与 genesis_hash 非空；Solana 仅 genesis_hash 非空；Tron 仅 tron_genesis_block_id 非空。其他字段必须为空，不能将不同链身份混合。身份字符串按对应公开 API 原生格式核对；EVM chain_id 按整数比较，其他值逐字符串比较。manifest/material 的 Chain/Network、preflight alias/identity 必须与本次固定链、harnessConfig.Network/Alias、顶层身份一致。

manifest_version/version 必须为 1。samples 至少三笔且前三笔 transaction_id 互异；sample_id、authorization_id 匹配 `[A-Za-z0-9._-]{1,64}`。network 使用同样字符集；source_url 为不含 userinfo 的 http/https 公开浏览器 URL。日期均 RFC3339，observed_at<=当前时间<expires_at；sample observed_at 不要求未过期，dynamic/material 必须未过期。SHA256 为 64 位小写 hex。CodeSnapshot 为 40 或 64 位 hex，由 Controller 对实际代码树验证，runner 只验证格式并绑定报告，不伪称能独立证明当前 Git 树。

preflight.status 必须 `PASS`。Methods 的 key 为本链公开方法名，值 Outcome 只能 `success/rpc_error`。success 时 HasRPCCode=false、NodeCode 为空；rpc_error 时 JSON-RPC 链须 HasRPCCode=true，Tron 须有非空 NodeCode 或原生顶层 Error 拒绝（以 NodeCode="" 明确记录）；此状态只说明独立预检查成功记录了预期拒绝，不表示被测 RPC 成功。不得按 message 文本匹配拒绝。Expected 的类型和用途由下方方法表冻结；无需静态预期的动态方法固定为 null。每个 required 方法都必须有记录；不能用 map 实际条目生成 required catalog。

sample 的 RPCPreflight 必须覆盖其 sample case 实际调用的全部方法且全部 Outcome=success；FieldSources 精确覆盖该链 Sample.Expected 的顶层 key，值为 `browser:<公开URL>` 或 `independent_rpc:<公开方法名>`。requirements 为人工可读的非秘密节点前提，不成为可执行配置。

样本类别固定：Ethereum 为 native/token/failed；BNB 为 native/token/contract；Bitcoin 为 multi_output/segwit/structure；Solana 为 native/token/failed；Tron 为 native/token/failed。前三个样本按此顺序各对应一个类别；一笔交易不重复占两个 required sample。

expected_execution：EVM 只填写 evm_status，取 `"0x0"` 或 `"0x1"`；Bitcoin 只填写 bitcoin_min_confirmations，必须>=1；Solana 只填写 solana_commitment=confirmed/finalized 和非缺失 solana_err（成功为 JSON null，失败为原始错误 object/string）；Tron 只填写非缺失 tron_receipt_result（原始结果字符串或 `{"absent":true}`）。其他链字段必须为零值/缺失。该字典不接受跨链 success bool。

#### 7.8.3 历史样本比较 / Historical Sample Comparison

每个 sample case 每条调用各占一个 harnessRequest。所有返回必须 present=true；否则 FAIL/assertion。Sample.Expected 的必需顶层 key 不得缺失，DTO 内字段按原始 DTO 契约，不额外强制完整官方 schema。

| 链 | 每个 sample case 的调用与断言 |
|---|---|
| Ethereum/BNB | GetTransaction(id)、GetTransactionReceipt(id)、GetBlock(AtNumber(height))。Transaction/Receipt 全部公开字段与 expected 对应值比较；Block.Hash/Number 与 expected 的 JSON 字符串值比较，ParentHash 逐字符串比较；样本 ID 和区块 ID 同时与返回匹配。 |
| Bitcoin | GetRawTransaction({TxID:id,BlockHash:sample.block_hash,HasBlockHash:true})、GetBlock(block_hash)。Transaction 除 Confirmations 外全部公开字段比较；Confirmations 单独要求可解码为非负整数，不能永久等于采样时值。Block.Hash/Height/TxIDs 与 expected 对应字段比较。 |
| Solana | GetTransaction(id,inputs.TransactionOptions)、GetBlock(slot,inputs.BlockOptions)。Transaction 全部公开字段比较；Block 的 Blockhash/PreviousBlockhash/ParentSlot 与 expected 比较；返回 Slot 必须等于 sample.height_or_slot。 |
| Tron | GetTransactionByID(id)、GetTransactionInfoByID(id)、GetBlockByNum(height)、GetSolidifiedTransactionByID(id)、GetSolidifiedTransactionInfoByID(id)。两种交易与同一 expected.Transaction 比较，两种 info 与同一 expected.TransactionInfo 比较；BlockID 匹配 expected。 |

RawMessage 比较保留缺失与 null 差异：nil 只等于 nil，显式 null 只等于 null。两个非 nil JSON 值用标准 Decoder.UseNumber 解码后比较对象键值和数组顺序；json.Number 按原 token 字符串比较，不经 float64。此辅助函数只比较已选字段，不提供 JSONPath、字段选择语言、RPC 路由或通用 schema。普通字段/切片按值比较。sample expected 中的 RawMessage nil 由字段缺失表达，JSON null 由显式 null 表达；不在 SDK DTO 内插入 absent 标记。

Tron expected_execution 的 `{"absent":true}` 仅表示 receipt.result 键缺失；它不是节点值，也不用于一般 DTO 比较。

#### 7.8.4 固定方法 case 表 / Method Cases

S1 表示 manifest.Samples[0]；I 表示解码后的本链 inputs。每个方法 case 恰好调用表内一次方法，sample cases 的多方法历史核对独立计数。所有 nullable 方法在成功 case 中要求 present=true，唯一例外为表内显式允许 null 的动态结果。所有历史方法只允许 preflight Outcome=success；动态方法可按表接纳预先记录的 rpc_error。

| Ethereum/BNB 方法 | 输入 | success 断言；preflight.Expected |
|---|---|---|
| ChainID | 无；固定第一个 case | 等于 manifest 身份；Expected=null |
| BlockNumber | 无 | >=S1.height_or_slot；Expected=null |
| GetBlock | AtNumber(S1.height_or_slot) | 使用 S1 block 字段断言；Expected=null |
| GetBalance | I.Address,AtNumber(I.AtBlockNumber) | 等于 Expected 十进制非负整数字符串，big.Int 比较 |
| GetTransactionCount | 同上 | 等于 Expected uint64 |
| GetCode | 同上 | 等于 Expected 原始字符串 |
| GetTransaction | S1.transaction_id | S1 完整 Transaction；Expected=null |
| GetTransactionReceipt | S1.transaction_id | S1 完整 Receipt；Expected=null |
| GetLogs | I.LogFilter 转换为 evm.LogFilter | 等于 Expected []evm.Log；固定 number 范围 |
| Call | I.Call,AtNumber(I.AtBlockNumber) | 等于 Expected 字符串；允许预期 rpc_error |
| EstimateGas | I.Estimate | >0；Expected=null；允许预期 rpc_error |
| GasPrice | 无 | >=0；Expected=null；允许预期 rpc_error |
| FeeHistory | I.FeeCount,AtNumber(I.AtBlockNumber),I.FeePercentiles | 等于 Expected evm.FeeHistoryResult；允许预期 rpc_error |

EVM 身份失败后，所有其余 case 闭包返回 BLOCKED/environment 且不调用 SDK；以局部闭包 bool 保存身份结果，不引入指针 DTO 或全局状态。

| Bitcoin 方法 | 输入 | success 断言；preflight.Expected |
|---|---|---|
| GetBlockchainInfo | 无；第一个 case | Chain 等于身份 bitcoin_chain；Expected=null |
| GetBlockHash | 0；第二个 case | 等于身份 genesis_hash；Expected=null |
| GetBlockCount | 无 | >=S1.height_or_slot；Expected=null |
| GetBlock | S1.block_hash | S1 BlockHash/BlockHeight/TxIDs；Expected=null |
| GetRawTransaction | S1 id 及显式 blockhash | S1 交易比较规则；Expected=null |
| GetTxOut | I.TxOut | Expected 为 bool，比较 present；present 时 Value>=0 且 BestBlock 非空；允许预期 rpc_error |
| GetMempoolEntry | I.MempoolTxID | BaseFee>=0、VSize>0；Expected=null；允许预期 rpc_error |
| EstimateSmartFee | I.Estimate | Expected 为 bool，比较 hasEstimate；true 时 FeeRate>=0，false 时原 Errors 可读取；允许预期 rpc_error |
| TestMempoolAccept | I.PreflightPayload | Expected 为 bool，比较 Allowed；不调用提交；允许预期 rpc_error |

Bitcoin 前两个身份 case 均通过后才执行其余请求；任一个失败，其余 case BLOCKED 且零调用。

| Solana 方法 | 输入 | success 断言；preflight.Expected |
|---|---|---|
| GetGenesisHash | 无；第一个 case | 等于身份 genesis_hash；Expected=null |
| GetSlot | I.Read | >=S1.height_or_slot；Expected=null |
| GetBlock | S1.height_or_slot,I.BlockOptions | S1 block 字段；Expected=null |
| GetBalance | I.Account,I.Read | Context 存在且 Value 为 uint64；Expected=null |
| GetAccountInfo | I.Account,I.AccountOptions | Expected 为 bool，比较 Value 是否非 null；Context 保留 |
| GetLatestBlockhash | I.Blockhash | present=true、Value.Blockhash 非空；LastValidBlockHeight 原值可读，不与 slot 比较；Expected=null |
| GetSignatureStatuses | []Signature{S1.id},I.StatusOptions | 一项非 null，slot 等于 S1.slot；Expected=null |
| GetTransaction | S1.id,I.TransactionOptions | S1 完整 Transaction；Expected=null |
| GetFeeForMessage | I.FeeMessage,I.Read | Expected 为 bool，比较 Value 是否非 null；非 null 须 uint64；允许预期 rpc_error |
| SimulateTransaction | I.SimulateTransaction,I.SimulateOptions | present=true；Value.Err 与 Expected 原始 JSON 比较；允许预期 rpc_error |

Solana 历史材料的 TransactionOptions/BlockOptions 必须显式 Encoding=json、HasMaxSupportedTransactionVersion=true 且版本 0，Commitment 为 confirmed/finalized；StatusOptions 必须显式 SearchTransactionHistory=true。此限制仅用于可比较的 live 材料，不缩小公开 SDK API。GenesisHash 失败后其余 case 零调用并 BLOCKED。

| Tron 方法 | 输入 | success 断言；preflight.Expected |
|---|---|---|
| GetBlockByNum | 0；第一个 case | present=true、BlockID 等于身份 tron_genesis_block_id；Expected=null |
| GetNodeInfo | 无 | Block 与 SolidityBlock 非空；Expected=null |
| GetNowBlock | 无 | present=true、BlockID 非空；Expected=null |
| GetAccount | I.Account | Expected 为 bool，比较 present；存在时 Address 非空、Balance>=0 |
| GetAccountResource | I.Account | 已公开资源字段均>=0；Expected=null；允许预期 rpc_error |
| GetChainParameters | 无 | ChainParameter 非空且各 Key 非空；不按数组位置解释；Expected=null |
| GetTransactionByID | S1.id | S1.Transaction；Expected=null |
| GetTransactionInfoByID | S1.id | S1.TransactionInfo；Expected=null |
| GetSolidifiedAccount | I.Account | Expected 为 bool，比较 present；存在时 Address 非空、Balance>=0 |
| GetSolidifiedNowBlock | 无 | present=true、BlockID 非空；Expected=null |
| GetSolidifiedBlockByNum | S1.height_or_slot 转换为 int64 | present=true、BlockID 等于 S1 expected.BlockID；Expected=null |
| GetSolidifiedTransactionByID | S1.id | S1.Transaction；Expected=null |
| GetSolidifiedTransactionInfoByID | S1.id | S1.TransactionInfo；Expected=null |
| TriggerConstantContract | I.ContractCall | Result.Result=true，ConstantResult 与 Expected []string 比较；允许预期 rpc_error |
| EstimateEnergy | I.ContractCall | Result.Result=true 且 EnergyRequired>=0；Expected=null；允许预期 rpc_error |

Tron genesis 身份失败后其余 case 零调用并 BLOCKED。uint64 高度转 int64 前检查可表示性；材料超限在请求前 BLOCKED。

read required 目录精确为上述每链全部公开方法名，ID=`<chain>.<方法名>`，另加 `<chain>.sample.1`、`.sample.2`、`.sample.3`。未允许 rpc_error 的方法如果 preflight 记录 rpc_error 则材料 BLOCKED，不将历史查询失败视为 PASS。

预期 rpc_error 在 operation 内用 errors.As 取 apierror.Error 并核对 Kind=RPC 及 RPCCode/HasRPCCode 或 NodeCode；匹配后 operation 返回 nil，因此不会触发 harness sticky 失败。不比较 Message 英文内容；原始 Message/Data/Cause 可供离线合成断言。非预期 SDK 错误为 FAIL/assertion；context 取消/期限继续由 harness 处理。缺材料、过期、身份前提未满足为 BLOCKED/environment，不按错误文案推测权限问题。

#### 7.8.5 广播状态与材料单次使用 / Broadcast State

broadcast dynamic.Inputs 必须是空 object；其 Preflight.Methods 覆盖身份查询且 Outcome=success；ManifestSHA256 冻结 submissionMaterial 文件，MaxResponseBytes 仍显式提供。材料 NetworkIdentity 与 Preflight 匹配。payload 只读取 PayloadPath 一次，按全部原始字节核对 PayloadSHA256，不 trim、不解析重建交易。非 Solana SendOptions 必须为空 object；Solana 按 solanaSubmissionOptions 解码。Bitcoin 之外 BlockHash 必须为空。

每个具体 SubmissionCases 工厂拥有局部闭包状态：identityPassed、claimSucceeded、sendAttempted、accepted、unknown，初始均 false；不共享到其他链，不保存为 SDK 生命周期对象。固定返回三个 case，顺序与 ID 为 `<chain>.identity`、`<chain>.submit`、`<chain>.lookup`。

identity 只调用：EVM ChainID 一次；Bitcoin GetBlockchainInfo 与 GetBlockHash(0) 各一次；Solana GetGenesisHash 一次；Tron GetBlockByNum(0) 一次。每次调用各占一个 request，全部等于材料身份才置 identityPassed=true。失败不能留下部分通过状态。

submit 进入规则：

1. identityPassed=false 时 BLOCKED 且零请求。
2. 调用 claimMaterial 一次；失败时 BLOCKED 且零提交，不能吞掉账本错误。
3. claim 成功后只进行一次 harnessRequest，其中调用该链提交 API 一次，调用前置 sendAttempted=true。
4. EVM/BTC/Solana 返回 ID 须等于材料 TransactionID；Tron 响应 txid 存在时须匹配。成功才 accepted=true。
5. error 可提取 SubmissionUnknown=true 时 unknown=true，submit 返回 BLOCKED/environment；可信拒绝、ID 不匹配或其他非预期错误为 FAIL/assertion。无自动重发。

lookup 进入规则：未 sendAttempted，或提交明确拒绝/ID 不匹配时 BLOCKED 且零请求；accepted 或 unknown 时仅按材料原 TransactionID 查询一次。unknown 即使查询得到结果也不抹去 submit 的 BLOCKED 账本；最终报告保留两个事实，须由另行授权的人工验收处理，不自动重跑 submit。

lookup 断言：

- EVM：GetTransactionReceipt 原 ID；不存在为 BLOCKED；存在时 TransactionHash 匹配且 Status 等于 ExpectedExecution.EVMStatus。
- Bitcoin：GetRawTransaction 原 ID，BlockHash 非空才显式附加；Confirmations 须可解码为整数且达到 BitcoinMinConfirmations。未收录/尚未满足为 BLOCKED；其他错误不按文本推断为未确认。
- Solana：GetSignatureStatuses 单个原 ID，显式 SearchTransactionHistory=true；null 为 BLOCKED；confirmationStatus 达到材料要求（finalized 高于 confirmed），err 与 SolanaErr 原 JSON 匹配；未达到为 BLOCKED。
- Tron：GetSolidifiedTransactionInfoByID 原 ID；不存在为 BLOCKED；存在时 ID 匹配，从 Receipt 提取原始 result 并按 TronReceiptResult 比较，`{"absent":true}` 要求键确实缺失。

执行结果与明确期望不符为 FAIL；只缺确认状态为 BLOCKED。没有轮询、等待重试或自动后续入口。后续只读复查由 Human 另行授权、用已有具体 SDK 查询执行并由 Controller 追加脱敏结果；不得重跑整个提交套件。

材料使用账本复用 T036 已有允许路径 `evidence/broadcast.json`，不新增文件。该文件格式明确为追加式 JSON Lines：每行一个下面定义的 broadcastRecord。扩展名保留既有路径，不能将它误读成单个 JSON 文档。每个 claim 必须在任何提交调用前追加并 Sync 成功；claim 永久消费该 `(chain,authorization_id,payload_sha256)`，包括拒绝、未知结果或进程中断。读到既有 claim 即禁止再次提交。

```go
type materialClaim struct {
    Chain string `json:"chain"`
    AuthorizationID string `json:"authorization_id"`
    PayloadSHA256 string `json:"payload_sha256"`
    MaterialSHA256 string `json:"material_sha256"`
}

type broadcastRecord struct {
    Version int `json:"version"`
    Type string `json:"type"`
    RecordedAt string `json:"recorded_at"`
    Claim materialClaim `json:"claim"`
    Report suiteReport `json:"report"`
}

func readMaterialClaims([]byte) ([]materialClaim, error)
func claimSubmission(materialClaim, string, []materialClaim, func([]byte) error) error
```

readMaterialClaims 逐行标准解码：空文件为无 claim，非空文件任何非法/截断行都 BLOCKED；Type 只能 claim/report。claim 行 Version=1、Claim 完整、Report 零值；report 行 Version=1、Claim 零值、Report 完整。claimSubmission 的第二参数是 RFC3339 记录时间，第四参数接收含末尾换行的一条完整 claim JSON 字节；若已有相同 chain+authorization_id 或相同 chain+payload_sha256，拒绝消费。不得以不同 authorization_id 重用同 payload。

runner 启用后读取历史账本一次，并在每次成功 claim 后更新本次内存 claim 列表。AppendBroadcastRecord 真实装配使用 os.OpenFile 的 O_CREATE|O_WRONLY|O_APPEND、Write、Sync、Close，任何失败停止该次提交；禁止截断或覆盖旧账本。ReadBroadcastJournal 仅把 os.ErrNotExist 映射为空字节成功，其余错误 BLOCKED。单写者是 T036 Controller 启动前门禁：同一工作区只允许运行一个广播验收进程；不引入锁文件、数据库或跨进程协调机制。无法保证独占时不得启动。

#### 7.8.6 报告与输出 / Reports

```go
type chainReport struct {
    Chain string `json:"chain"`
    Alias string `json:"alias"`
    CodeSnapshot string `json:"code_snapshot"`
    MaterialDigest string `json:"material_digest"`
    Result harnessReport `json:"result"`
}

type suiteReport struct {
    Version int `json:"version"`
    Mode string `json:"mode"`
    Status string `json:"status"`
    StartedAt string `json:"started_at"`
    FinishedAt string `json:"finished_at"`
    Chains []chainReport `json:"chains"`
}
```

harnessReport 沿用已有字段和其标准 JSON 键 `Status/Category/Digest/Requests/Cases`，不另造错误序列化模型。Mode 仅 read/broadcast，Status 仅 NOT_RUN/PASS/FAIL/BLOCKED；启用时 Version=1，时间 RFC3339。链列表固定五条且次序与 alias 表一致；本地装配失败仍保留对应链 BLOCKED 结果，不能省略链。整体 FAIL 优先，其次 BLOCKED，只有五链全部 PASS 才 PASS。关闭入口返回 Mode 及 Status=NOT_RUN，时间、Chains 保持零值，不调用 Now 或写报告。

read 将完整 suiteReport 标准 JSON 编码，通过 WriteReadReport 写到已有 `evidence/read.json`；写入失败时，已有 FAIL 保持 FAIL，否则返回 BLOCKED；测试必须失败，不伪造已保存证据。broadcast 先按上述规则追加 claim 行，运行结束再追加 Type=report 行；追加报告失败时，已有 FAIL 保持 FAIL，否则返回 BLOCKED，且不得删除已持久化 claim。report 只含固定链/alias、经格式校验的 snapshot/digest、harness 固定类别、计数及 required ID，不存原始错误、配置、任意 expected 差异、Header、endpoint 或 payload。实际差异在 callback 内确定，只输出 FAIL/assertion。

最终 Controller 将代码树/依赖/外部配置身份与报告绑定，写入既有 final.md；CodeSnapshot 材料值不替代实际快照核对。FA-02 独立 HTTP 请求另有人工预算账本，不混报为 harness 请求数。

#### 7.8.7 T033/T034 离线证据 / Offline Evidence

T033 在原两个测试文件中调用 runLiveRead/runLiveBroadcast 及具体 case 工厂。关闭开关测试注入 LookupEnv 记录精确键序列，所有其他依赖均为一旦调用即 t.Fatal 的函数；证明零配置/材料/时钟/客户端/账本/输出调用。live 开关关闭时只允许读取一个开关；broadcast 开关关闭时只允许读取两个开关。

开启路径只用合成 ReadFile、fake Doer/Caller/Poster、固定 Now/Wait 和内存报告/账本函数。覆盖：固定 catalog 漏项；每种材料身份/摘要不匹配；preflight 缺方法；期望拒绝在 operation 内消化；identity 失败后 submit 零调用；claim 失败后 submit 零调用；claim 先于 send；已有 claim 禁止重发；unknown 只查原 ID 一次；报告不含合成秘密；截断账本 BLOCKED；读取/写入失败不能 PASS；断言 FAIL 与报告写入失败并存时仍为 FAIL。

T034 的 TestLiveRead/TestLiveBroadcast 仅创建 runnerDeps 并调用入口；NOT_RUN 时 t.Skip，非 PASS 时 t.Fail，不打印原始 error。验证命令显式强制两个开关为 0，禁止继承宿主已有的开启值。T034 不运行任何开启状态的真实装配；全部开启场景由 T033 合成依赖证明。

#### Final acceptance checks

| ID | 依赖/输入 | 通过标准 |
|---|---|---|
| FA-01 样本来源 | T035；每链三笔、公开浏览器、网络 | 来源/字段出处可追踪、类别覆盖；浏览器交易不得直接作为广播材料 |
| FA-02 独立 preflight | FA-01；用户账号、授权、限额 | 非 SDK 直接 HTTP 核对网络、历史能力与 expected；五链逐项 PASS |
| FA-03 冻结 | FA-02 | manifest 原始字节 SHA-256 独立冻结；动态输入有效；替换样本有记录 |
| FA-04 五链只读 | FA-03；单开关 | `TestLiveRead` 的固定方法与样本目录全部 PASS，无漏项 |
| FA-05 五链受控提交 | 逐链授权、材料和身份 | `TestLiveBroadcast` 一次提交、原标识查询；无材料或未确认为 BLOCKED |
| FA-06 最终审计 | FA-04/05 与有效离线证据 | 最终只读 Auditor、Human 批准；快照、依赖、配置身份与证据一致 |

## 8. 验证命令

以下命令在对应 Task 实现后建立：

```sh
go test ./apierror ./jsonrpc ./httpjson
go test -race ./jsonrpc
go test ./evm ./ethereum ./bnb
go test ./bitcoin ./solana ./tron
go test ./internal/test/live -run '^TestHarness' -count=1
go test -race ./internal/test/live -run '^TestHarness' -count=1
go test ./internal/test/live -run '^TestHarness|TestReadContract|TestSubmissionContract'
go vet ./...
```

真实只读和提交只在审批后的最终窗口运行：

```sh
CHAIN_API_GO_LIVE_TEST=1 go test ./internal/test/live -run '^TestLiveRead$' -count=1
CHAIN_API_GO_LIVE_TEST=1 CHAIN_API_GO_LIVE_BROADCAST=1 go test ./internal/test/live -run '^TestLiveBroadcast$' -count=1
```

## 9. 角色与门禁

- 所有角色开始工作前读取根目录 `AGENTS.md` 与 `constitution.md`；违反 constitution 的设计或实现不得进入审核。
- Planner：`gpt-6-astra / high`，只读规划。
- Coder：`gpt-6-astra / low`，只能修改获批 Task 的路径。
- Auditor：固定 `gpt-5.6-sol / medium`，必须是全新只读上下文；当前运行时未提供该模型时不得静默替换。
- Human：PLAN-v7-REMAINING-RAW-v1 已批准；T017–T027 的 RED/GREEN、Auditor、Human 门禁与集成已完成。T027 Auditor 的 `gpt-6.1-sol / medium` 替代路由仅针对该次审计。T028 已单独授权进入实现；不自动授权 T029 或后续 Task。没有真实 RPC、广播、推送或 PR 授权。

当前活动任务为 T028。仍为 5 Plan / 20 Task；T017–T027 已完成，T029–T036 未启动。后续固定 Auditor `gpt-5.6-sol / medium` 若当前运行时不可用，仍必须逐次取得 Human 降级裁决，不得沿用 T027 的专项替代授权。
