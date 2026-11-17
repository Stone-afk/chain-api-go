# chain-api-go

供业务服务和基建服务直接导入的 Go 多链 RPC SDK，module 为 `github.com/Stone-afk/chain-api-go`，最低 Go 版本为 1.26.0。固定直接依赖为 `github.com/ethereum/go-ethereum v1.17.10` 和 `github.com/solana-foundation/solana-go/v2 v2.3.0`，其余为 module 必需的间接依赖；无 replace/fork。

当前文档对应 `PLAN-v8-NATIVE-SDK-v1` 的 EVM 与 Solana 原生实现。T036 为 **BLOCKED**：五链真实浏览器采样、账号 RPC 与广播验收未执行，本轮授权不包含这些操作；不宣称生产就绪。离线证据与限制见 [offline.md](internal/test/integration/evidence/offline.md)。

## 使用边界与装配

直接导入 `ethereum`、`bnb`、`bitcoin`、`solana` 或 `tron`。根包只提供文档；没有统一链 Client、适配器、注册表、dispatcher、服务端或部署单元。

宿主显式注入 endpoint、Headers、正数 MaxResponseBytes 和 HTTPDoer；HTTP 超时、context deadline、连接池、重定向与传输重试策略均由宿主控制。SDK 不读取环境变量或配置文件，不创建默认网络 HTTP 客户端、后台任务、缓存、节点切换或轮询。EVM 私有持有固定上游 client，通过只绑定内存桥接的 `http.Client` 和 synthetic URL 接入 Caller；没有默认网络 transport fallback，也不暴露完整上游 client。固定上游构造内部读取 crypto/rand 熵，因此零网络不等于绝对零 OS I/O。构造器只做本地验证，零网络 I/O；共享 transport 的生命周期归宿主。注入依赖必须是有效实现，构造器只拒绝接口本身 nil，不探测 typed-nil。

| 构造器 | 配置与依赖 |
|---|---|
| `jsonrpc.New(Config, transport.HTTPDoer) (*Client, error)` | Config 含 Endpoint、Headers、MaxResponseBytes、Version；Ethereum/BNB/Solana 选 `jsonrpc.V2`，Bitcoin 选 `jsonrpc.Bitcoin` |
| `httpjson.New(Config, transport.HTTPDoer) (*Client, error)` | Config 含 Endpoint、Headers、MaxResponseBytes；供 Tron 使用 |
| `ethereum.New(ethereum.Config, jsonrpc.Caller) (*ethereum.Client, error)` | `ExpectedChainID big.Int` 必须正数，构造时深拷贝 |
| `bnb.New(bnb.Config, jsonrpc.Caller) (*bnb.Client, error)` | 同上，各自调用 `ValidateNetwork(ctx)` 显式核对网络 |
| `bitcoin.New(bitcoin.Config, jsonrpc.Caller) (*bitcoin.Client, error)` | 空值配置 `bitcoin.Config{}` |
| `solana.New(solana.Config, jsonrpc.Caller) (*solana.Client, error)` | 空值配置 `solana.Config{}` |
| `tron.New(tron.Config, httpjson.Poster) (*tron.Client, error)` | 空值配置 `tron.Config{}` |
| `evm.New(jsonrpc.Caller) (*evm.Client, error)` | 两个具体 EVM 客户端共享的构造 seam，不做跨链分发 |

`Caller.Call(context.Context, string, any, any) error` 与 `Poster.Post(context.Context, string, any, any) error` 是协议注入边界。`HTTPDoer.Do(*http.Request) (*http.Response, error)` 与标准库兼容。调用方须传有效 context；注入实现须遵守 context 和并发契约。

离线使用可直接注入 fake Caller/Poster；不需要节点、凭据或默认客户端。例如下面完整程序只返回合成 chain ID，不做网络请求：

```go
package main

import (
    "context"
    "encoding/json"
    "math/big"

    "github.com/Stone-afk/chain-api-go/ethereum"
    "github.com/Stone-afk/chain-api-go/bnb"
)

type offlineCaller struct{ chainID string }
func (c offlineCaller) Call(_ context.Context, _ string, _ any, out any) error {
    return json.Unmarshal([]byte(c.chainID), out)
}

func main() {
    var expected big.Int
    expected.SetInt64(1)
    client, err := ethereum.New(ethereum.Config{ExpectedChainID: expected}, offlineCaller{chainID: `"0x1"`})
    if err != nil { panic("offline construction failed") }
    if err := client.ValidateNetwork(context.Background()); err != nil {
        panic("offline identity check failed")
    }
    var bnbExpected big.Int
    bnbExpected.SetInt64(56)
    bnbClient, err := bnb.New(bnb.Config{ExpectedChainID: bnbExpected}, offlineCaller{chainID: `"0x38"`})
    if err != nil { panic("offline BNB construction failed") }
    if err := bnbClient.ValidateNetwork(context.Background()); err != nil {
        panic("offline BNB identity check failed")
    }
}
```

此 fake 仅展示注入与显式身份检查，不模拟其他方法。`ValidateNetwork` 发起一次 ChainID 请求；不会自动执行，也不保证供应商后续永远使用同一网络。

## 当前公开方法

以下签名均省略首参 `ctx context.Context`；EVM/Solana 使用下文注明的上游原生类型，其余类型属于对应链包。bool 的具体含义见下文，不能作为跨链成功标志。

### Ethereum 与 BNB

两个具体 Client 均内嵌项目受限的 `*evm.Client`，提供十三个查询/计算方法及一个字符串提交方法，另有 `ValidateNetwork(ctx) error`。下面 `geth` 指 `github.com/ethereum/go-ethereum`（其包名为 ethereum），`common` 和 `types` 分别指该 module 的 `common` 与 `core/types`；不要与本项目的 ethereum 客户端包混淆。

| 方法（省略 ctx） | 返回 | 含义 |
|---|---|---|
| `ChainID()` | `(*big.Int, error)` | 原生 chain ID |
| `BlockNumber()` | `(uint64, error)` | 当前区块高度 |
| `BlockByNumber(*big.Int)` | `(*types.Block, error)` | 完整区块、交易及叔块 |
| `BalanceAt(common.Address, *big.Int)` | `(*big.Int, error)` | wei |
| `NonceAt(common.Address, *big.Int)` | `(uint64, error)` | nonce |
| `CodeAt(common.Address, *big.Int)` | `([]byte, error)` | 合约字节码 |
| `TransactionByHash(common.Hash)` | `(*types.Transaction, bool, error)` | bool 为 isPending |
| `TransactionReceipt(common.Hash)` | `(*types.Receipt, error)` | 原生回执 |
| `FilterLogs(geth.FilterQuery)` | `([]types.Log, error)` | 地址、topic、区块条件 |
| `CallContract(geth.CallMsg, *big.Int)` | `([]byte, error)` | 调用结果字节 |
| `EstimateGas(geth.CallMsg)` | `(uint64, error)` | gas 单位 |
| `SuggestGasPrice()` | `(*big.Int, error)` | wei/gas |
| `FeeHistory(uint64, *big.Int, []float64)` | `(*geth.FeeHistory, error)` | 原生费用历史；float64 仅为百分位/比例 |
| `SendRawTransaction(string)` | `(common.Hash, error)` | 外部已编码 hex 载荷原样提交一次 |

这是破坏性迁移：旧 EVM 自定义 DTO、Address/Hash/Data、BlockRef、Latest/Pending/AtNumber、Has/present 和旧 Get* 方法已删除，无别名、转发或兼容层。直接使用原生 common、types、geth 类型及指针契约；不再承诺旧 RawMessage 的缺失/null 区分或旧参数校验。nil 区块参数表示 latest，负数区块标签按固定上游规则编码（如 -1 为 pending）。CallMsg、FilterQuery 的可选字段、省略和默认编码均遵循固定上游，不补旧默认或验证层。BNB 节点不支持某方法时返回实际错误，不替换费用模型。

null 区块、交易、回执返回 `geth.NotFound`，可用 `errors.Is(err, geth.NotFound)` 判断；`TransactionByHash` 的 bool 表示 pending，不是 present，也不是确认。完整区块可能触发一次块请求及 N 次按顺序的叔块请求，首错停止。EVM/BNB integration read manifest 的 `manifest_version` 和 submission material 的 `version` 均为 2；版本 1 在请求前 BLOCKED。预算按每次底层 Caller 调用计数：两个叔块需要 3 次预算，预算 2 时不发第三次请求并报告 BLOCKED。

`SendRawTransaction` 只接受非空、带小写 `0x` 前缀的偶数位合法 hex 字符串，不 trim、不重新编码；合法载荷原样调用一次 `eth_sendRawTransaction`。交易在 SDK 外构造、签名及编码，不接受待提交的 `*types.Transaction`，也不提供私钥、交易对象构造/编码/签名 API。返回非零 hash 仅表示节点接受；非法输入零请求，明确 RPC 拒绝为已知失败，提交后的其他 Caller 错误及无效/零 hash 为结果未知。没有重试、后台任务、轮询或默认网络 transport。

### Bitcoin

| 方法（省略 ctx） | 返回 |
|---|---|
| `GetBlockchainInfo()` | `(BlockchainInfo, error)` |
| `GetBlockCount()` | `(int64, error)` |
| `GetBlockHash(int64)` | `(BlockHash, error)` |
| `GetBlock(BlockHash)` | `(Block, error)`；固定 verbosity=1 |
| `GetTxOut(GetTxOutRequest)` | `(TxOut, bool, error)` |
| `GetRawTransaction(GetRawTransactionRequest)` | `(Transaction, error)`；固定 verbosity=1 |
| `GetMempoolEntry(TxID)` | `(MempoolEntry, error)` |
| `EstimateSmartFee(EstimateSmartFeeRequest)` | `(FeeEstimate, bool, error)` |
| `TestMempoolAccept(RawTransaction)` | `(MempoolAcceptResult, error)`；单笔预检，不广播 |
| `SendRawTransaction(RawTransaction)` | `(TxID, error)` |

TxID/BlockHash 为无 `0x` 的 64 位 hex，按 RPC 展示顺序，不反转字节。RawTransaction 为非空偶数位 hex，大小写原样保留。可选请求字段由 HasIncludeMempool/HasBlockHash/HasMode 控制，false 时省略，true 时保留显式 false/零。估费 target 为 1–1008，模式为 EstimateUnset/EstimateEconomical/EstimateConservative。

`TxOut.Value`、`TxOutput.Value`、`MempoolEntry.BaseFee` 为 Satoshi；`FeeEstimate.FeeRate` 为 SatoshiPerKVByte，二者均有 `ToInt64()`。BTC 数字精确乘 10^8，不经过浮点、不舍入；1e-8 BTC=1 sat，0.00001000 BTC/kvB=1000 sat/kvB（不是 sat/vB）。负非零、非整数 sat、溢出、必转金额缺失/null 为 Protocol。预检 Fees 保留原 JSON，仍是节点原生单位。

历史 GetRawTransaction 依赖 mempool、txindex 或显式可用 blockhash；裁剪、权限或历史不可用错误直接返回，不探测、不 fallback。不存在任意地址 UTXO 或交易历史索引。

### Solana

下表 `ns` 指 `github.com/solana-foundation/solana-go/v2`，`srpc` 指其 `rpc` 子包。项目 Client 私有持有通过 `NewWithCustomRPCClient` 创建的原生客户端；私有 bridge 仅把同一 context、method、params、out 转交注入 Caller 一次。callback/batch 返回 Unsupported 且零调用，不暴露完整上游 client、生命周期或默认网络 transport。

| 方法（省略 ctx） | 返回 |
|---|---|
| `GetGenesisHash()` | `(ns.Hash, error)` |
| `GetSlotWithOpts(*srpc.GetSlotOpts)` | `(uint64, error)` |
| `GetBlockWithOpts(uint64, *srpc.GetBlockOpts)` | `(*srpc.GetBlockResult, error)` |
| `GetBalanceWithOpts(ns.PublicKey, *srpc.GetBalanceOpts)` | `(*srpc.GetBalanceResult, error)` |
| `GetAccountInfoWithOpts(ns.PublicKey, *srpc.GetAccountInfoOpts)` | `(*srpc.GetAccountInfoResult, error)` |
| `GetLatestBlockhashWithOpts(*srpc.GetLatestBlockhashOpts)` | `(*srpc.GetLatestBlockhashResult, error)` |
| `GetSignatureStatuses(bool, ...ns.Signature)` | `(*srpc.GetSignatureStatusesResult, error)` |
| `GetTransaction(ns.Signature, *srpc.GetTransactionOpts)` | `(*srpc.GetTransactionResult, error)` |
| `GetFeeForMessageWithOpts(string, *srpc.GetFeeForMessageOpts)` | `(*srpc.GetFeeForMessageResult, error)` |
| `SimulateRawTransactionWithOpts([]byte, *srpc.SimulateTransactionOpts)` | `(*srpc.SimulateTransactionResponse, error)` |
| `SendEncodedTransactionWithOpts(string, srpc.TransactionOpts)` | `(ns.Signature, error)` |

这是破坏性迁移：旧自研 DTO、Pubkey/EncodedTransaction、Has/present 及旧方法删除，无别名、转发或兼容层。使用原生 PublicKey/Hash/Signature、options 指针和 nullable 返回，不恢复旧参数校验或缺失/null 模型。余额与费用为 lamports（1 SOL=10^9 lamports），slot 与 LastValidBlockHeight 不可互换；context、legacy/v0、meta、模拟执行错误及确认状态保持原生语义。

nil options 的 wire 遵循固定上游：block/account 默认 base64；slot/latest blockhash 可省略 options；transaction 不强加项目默认编码。可选指针区分省略和显式零/false，如 MaxSupportedTransactionVersion、Rewards、MinContextSlot。GetSignatureStatuses 的 bool 是 searchTransactionHistory，后跟可变数量签名，false 时上游省略该选项。具体 null/sentinel 见下表；不使用通用 present/success bool。

GetFeeForMessageWithOpts 接收已为 base64 的 message 字符串。SimulateRawTransactionWithOpts 接收调用方已序列化的交易字节，只由上游添加 RPC base64 传输封装；不接收待编码交易对象。提交接收已编码 string，空 Encoding 默认 base64，也支持显式 base64/base58；非空及编码语法校验后原样提交一次，不 trim、不重新编码。SkipPreflight、PreflightCommitment、MaxRetries、MinContextSlot 保留原生选项；节点 MaxRetries 不是 SDK 重试。非法输入零请求，无效/零 signature 及非明确 RPC 拒绝的提交错误为结果未知。非零 signature 只表示接受，不保证执行、confirmed 或 finalized。

下面完整例子使用合成响应与原生显式零选项，零网络、零广播：

```go
package main

import (
    "context"
    "encoding/json"

    clientsol "github.com/Stone-afk/chain-api-go/solana"
    ns "github.com/solana-foundation/solana-go/v2"
    srpc "github.com/solana-foundation/solana-go/v2/rpc"
)

type syntheticCaller struct{}
func (syntheticCaller) Call(_ context.Context, _ string, _ any, out any) error {
    return json.Unmarshal([]byte(`{"context":{"slot":7},"value":0}`), out)
}

func main() {
    client, err := clientsol.New(clientsol.Config{}, syntheticCaller{})
    if err != nil { panic("offline construction failed") }
    zero := uint64(0)
    result, err := client.GetBalanceWithOpts(context.Background(), ns.PublicKey{},
        &srpc.GetBalanceOpts{Commitment: srpc.CommitmentConfirmed, MinContextSlot: &zero})
    if err != nil { panic("offline balance failed") }
    _ = result.Value // lamports; synthetic zero balance
}
```

Ethereum/BNB/Solana 的 read manifest `manifest_version` 和 submission material `version` 均为 2，旧 v1 在请求前 BLOCKED；Bitcoin/Tron 保持 v1，journal/report 格式未改。Solana 动态输入采用原生 options，`search_transaction_history` 为 bool，`simulate_transaction` 为序列化字节的 JSON base64 字符串；`send_options` 直接解码为 srpc.TransactionOpts，无旧 encoding/options 包装。详情见离线证据。

### Tron

| 方法（省略 ctx） | 返回 | POST 路径 |
|---|---|---|
| `GetNodeInfo()` | `(NodeInfo, error)` | `/wallet/getnodeinfo` |
| `GetNowBlock()` | `(Block, bool, error)` | `/wallet/getnowblock` |
| `GetBlockByNum(int64)` | `(Block, bool, error)` | `/wallet/getblockbynum` |
| `GetAccount(GetAccountRequest)` | `(Account, bool, error)` | `/wallet/getaccount` |
| `GetAccountResource(GetAccountRequest)` | `(AccountResource, error)` | `/wallet/getaccountresource` |
| `GetChainParameters()` | `(ChainParameters, error)` | `/wallet/getchainparameters` |
| `GetTransactionByID(TxID)` | `(Transaction, bool, error)` | `/wallet/gettransactionbyid` |
| `GetTransactionInfoByID(TxID)` | `(TransactionInfo, bool, error)` | `/wallet/gettransactioninfobyid` |
| `GetSolidifiedAccount(GetAccountRequest)` | `(Account, bool, error)` | `/walletsolidity/getaccount` |
| `GetSolidifiedNowBlock()` | `(Block, bool, error)` | `/walletsolidity/getnowblock` |
| `GetSolidifiedBlockByNum(int64)` | `(Block, bool, error)` | `/walletsolidity/getblockbynum` |
| `GetSolidifiedTransactionByID(TxID)` | `(Transaction, bool, error)` | `/walletsolidity/gettransactionbyid` |
| `GetSolidifiedTransactionInfoByID(TxID)` | `(TransactionInfo, bool, error)` | `/walletsolidity/gettransactioninfobyid` |
| `TriggerConstantContract(ContractCallRequest)` | `(ConstantResult, error)` | `/wallet/triggerconstantcontract` |
| `EstimateEnergy(ContractCallRequest)` | `(EnergyEstimate, error)` | `/wallet/estimateenergy` |
| `BroadcastTransaction(TransactionPayload)` | `(BroadcastResult, bool, error)` | `/wallet/broadcasttransaction` |

visible=false 使用 41 前缀、42 位 hex 地址；visible=true 使用 T 前缀、34 字符 base58 地址。SDK 验证形状，不转换地址或计算 checksum。TxID 是 64 位无 `0x` hex。balance、fee、call_value 单位为 sun（1 TRX=10^6 sun）；Energy 与 Bandwidth 各保留资源单位，ChainParameters 的单位按其 key 解释。

proto3 可省略零值：有 address 的账户即使没有 balance 也是存在的零余额账户；资源、参数 value 缺失按零值处理。不增加 Has 来伪造“缺失错误”。HTTP 200 内顶层 Error 或计算/广播 result=false 仍可构成 RPC 错误，保留节点原字段。查询中的 Transaction.ret、TransactionInfo.receipt/result 是执行事实，不转成通信错误。

BroadcastTransaction 传入 `TransactionPayload{JSON: json.RawMessage(...)}` 的原生对象；SDK 不重建 raw_data、签名、TAPOS 或 txID。HTTP JSON marshal 可改变无语义空白/转义，不承诺请求 body 字节完全相同；解码后的交易内容不得改变。接受时 txid 可以缺失，SDK 不自行计算。

## 缺失、null 与错误

Bitcoin/Tron 的下述自有 DTO 使用标准 encoding/json 解码，不代表完整官方 schema 已校验；EVM/Solana 使用上述原生契约。普通 string/整数的缺失/null 遵循标准解码；有原生存在性差异的字段用 `json.RawMessage`：nil 表示键缺失，字节 `null` 表示显式 JSON null，零/false/空数组仍为各自原值。不要把它们统一为空或成功。

| 方法 | 合法空结果 |
|---|---|
| EVM BlockByNumber/TransactionByHash/TransactionReceipt | 顶层 null → 原生 `geth.NotFound`；TransactionByHash 的 bool 为 isPending |
| Bitcoin GetTxOut | 顶层 null → TxOut{}、false、nil |
| Bitcoin EstimateSmartFee | bool 是 hasEstimate；无 feerate 时保留 Blocks/Errors、false、nil；显式 feerate:null 为错误 |
| Solana GetBlockWithOpts | 顶层 null → `srpc.ErrNotConfirmed` |
| Solana GetTransaction/GetAccountInfoWithOpts | null transaction / account Value=null → `srpc.ErrNotFound` |
| Solana GetFeeForMessageWithOpts | Value 为 `*uint64`；null → nil，显式零 → 指向零；Context 保留 |
| Solana GetSignatureStatuses | 顶层/Value null → `srpc.ErrNotFound`；数组中的 nil 项保留顺序 |
| Solana GetLatestBlockhashWithOpts/SimulateRawTransactionWithOpts | 保留上游指针解码结果；没有旧成功 bool 或项目补充 null 校验 |
| Tron 带 bool 的查询 | 空对象 {} → 零 DTO、false、nil；非空对象须含对应 address/blockID/txID/id；顶层 null 为错误 |
| Tron BroadcastTransaction | 成功返回 true；拒绝/无效结果返回 error，bool 不表示确认 |

`present` 只表示当前节点返回了对象；false 不能证明交易从未提交、已失败或永不存在。

项目结构化错误使用 `apierror.Error` 值；EVM/Solana 查询/计算也可直接返回原生错误、解码错误及 native/HTTP wrapper。桥接不把 Caller 错误重新编码为 JSON-RPC error envelope，原始节点字段、正文和底层 Cause 保持可达。用 `var detail apierror.Error; errors.As(err, &detail)` 提取；`errors.Is/As` 可沿 Cause 解包原始依赖错误。Kind 包括 Configuration、InvalidArgument、Transport、HTTP、Protocol、RPC、Unsupported、NetworkMismatch；不按英文 message 猜测未找到、拒绝或可重试。

Message/Data/NodeCode 保留原始节点信息；HasRPCCode 区分无数字码与 code=0。ResponseBody 保留已读取的失败响应原始字节，最多 MaxResponseBytes，超限有 ResponseBodyTruncated；部分读失败不保证正文完整，Cause 保留实际错误。Data 缺失与 null 不合并。

**Error() 不是安全日志视图。** 它优先显示原始 Message，否则显示 Cause 文本；Data/正文不自动拼入字符串，但整个 error 结构、节点文本或底层 error 仍可能含敏感信息。SDK 不主动追加 endpoint/Header/请求/载荷，也不删除节点或 Cause 已有内容。业务映射、对外响应、日志脱敏和重试策略属于调用层。WithContextCause 返回仅替换 Cause 的值副本，nil 清空；不再仅保留 context 哨兵。Error 含切片，不能用 == 比较。

## 提交与 Raw 入口

每个类型化提交方法在本地校验通过后至多调用注入依赖一次。可信 RPC 拒绝或本地校验失败的 SubmissionUnknown=false；进入依赖后的 Transport/HTTP/Protocol、取消或畸形成功响应通常为 true，表示结果未知，不是已失败、已接受或可安全重发。宿主注入的传输不得隐藏重放，SDK 不补发、不切换节点、不重建载荷。

节点接受 ≠ 链上执行 ≠ 最终确认。EVM 查询 receipt/status 及区块确认；Bitcoin 查询收录与确认数，预检 Allowed 不等于接受；Solana 查询 signature status/err/commitment，模拟结果不是提交；Tron 查询执行回执及 solidified 视图。SDK 不自动执行这些后续查询或等待。

`jsonrpc.Client.RawCall(ctx, method, params, result)` 与 `Call` 使用同一信封、ID、context、限长和错误规则；`httpjson.Client.RawPost(ctx, path, request, result)` 与 `Post` 使用同一同源路径和响应规则。result 必须是可写非 nil 指针，可使用 RawMessage。Raw 入口不判断链语义或提交 unknown，不增加链级 Raw 方法，不把未封装方法列为已验收能力；调用方负责合法方法与副作用边界，不通过它调用节点签名/解锁。

核心不接收私钥，不构造、编码或签名交易；nonce、UTXO、费用、blockhash、TAPOS 与签名生命周期属于宿主。本仓库没有钱包集成示例；未来 go-wallet-sdk 示例须另获批准并使用独立 module、默认离线。

## 离线验证与真实验收

普通验证不需要公网或真实材料。显式关闭 live 开关，避免继承宿主环境：

```sh
export GOPROXY=off GOTOOLCHAIN=auto
export GOCACHE=/private/tmp/chain-api-go-gocache GOMODCACHE=/private/tmp/chain-api-go-gomodcache
export CHAIN_API_GO_LIVE_TEST=0 CHAIN_API_GO_LIVE_BROADCAST=0
go test ./...
go vet ./...
go list -m all
go list -deps ./...
```

工具链与固定依赖必须已在本地缓存，缺失时报告环境阻塞，不自动改版本或转向公网。T042 实际命令、退出码、快照及验收限制见 [离线证据](internal/test/integration/evidence/offline.md) 和 [最终状态](internal/test/integration/evidence/final.md)。test-only 配置、材料、alias 和单次 claim 不进入核心 SDK API；离线通过不替代独立审计、Human 最终验收或 T036。

治理依据为 [constitution](constitution.md) 与批准的 [spec](docs/superpowers/001-chain-api-go/spec.md)、[plan](docs/superpowers/001-chain-api-go/plan.md)、[tasks](docs/superpowers/001-chain-api-go/tasks.md)。architecture.md §0 为当前迁移架构，其余历史内容中的旧指针/Parse API、全字段校验与规范化摘要设计不作为当前用法。
