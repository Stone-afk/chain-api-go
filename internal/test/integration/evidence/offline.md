# T042 原生 SDK 最终离线证据（2026-11-18）

基线：constitution 1.5 / PLAN-v8-NATIVE-SDK-v1。EVM 与 Solana 已实现固定原生 API，本轮文档与离线检查已执行，Coder 建议 IMPLEMENTED；独立 T042 Auditor 和 Human 最终验收尚未完成。**T036、AC-10/11、FA-01–06 仍 BLOCKED / 未执行**。没有真实浏览器采样、账号 RPC 或广播。下方 T039/T035 仅为阶段历史，其中旧 Solana API、Go/依赖、版本 1 及 internal/test/live 路径不代表当前基线。

## 当前快照、工具与实际命令

仓库 /Users/stone/the_web3/chain-api-go。go.mod 精确 `go 1.26.0`；唯一两个直接依赖为 `github.com/ethereum/go-ethereum v1.17.10`、`github.com/solana-foundation/solana-go/v2 v2.3.0`，无 replace/fork。实测工具链 `go version go1.26.0 darwin/amd64`，宿主 `Darwin arm64`。架构差异如实记录。

以下 E 为每个离线 Go 命令显式设置的完整环境：

```text
GOPROXY=off GOTOOLCHAIN=auto
GOCACHE=/private/tmp/chain-api-go-gocache
GOMODCACHE=/private/tmp/chain-api-go-gomodcache
CHAIN_API_GO_LIVE_TEST=0 CHAIN_API_GO_LIVE_BROADCAST=0
```

| 本轮实际命令 | 退出码与结果 |
|---|---|
| E `go version`；`uname -sm` | 各 0；上述工具链/宿主 |
| E `go test ./...` | 0；十个测试包 PASS，根包/transport 无测试；BNB/Ethereum/EVM/Solana 为 cached，其余实际执行 |
| E `go vet ./...` | 0，无诊断 |
| E `go list -deps ./...` | 0，373 行 package 路径，含标准库、本 module、固定原生依赖与必要间接 imports |
| E `go list -m all`（首次） | 1，233 行相同的 `go: module lookup disabled by GOPROXY=off`；不是 RED 或通过证据 |
| E `go list -m all`（缓存补齐及元数据精确恢复后） | 0，236 行（主 module + 235 dependency modules），两个直接版本准确 |
| E `gofmt -w solana/doc.go doc.go` | 0 |
| E `gofmt -l` 加全部 78 个仓库 Go 文件 | 0，空输出；覆盖 T037–T042 全部现存 Go 文件 |
| E `go run /private/tmp/t042-verification/example0.go`、`example1.go` | 各 0，无输出；分别是 README EVM 与 Solana 合成示例 |

缓存补齐过程由 Controller 明确确认属于既有固定依赖及必要间接依赖授权。使用同一缓存、工具链和双关闭开关，仅将 GOPROXY 改为 https://proxy.golang.org 执行 `go mod download all`：沙箱内首次 exit 1，233 行 DNS 失败；提升权限后 exit 0，无输出。这是依赖下载，不是 RPC 或网络测试。下载临时追加 go.sum（SHA-256 ff532570d534773f8f33a17d437d618ba46134b456f4c867299aaa801800cdc7）。依 Controller 指示运行 E `go mod tidy` 后，go.mod/go.sum 精确恢复本任务前 SHA-256；恢复后模块图命令再次 exit 0，文件仍一致。没有留下依赖变更。

一次辅助 `shasum -a 256 go.mod go.sum` 因宿主 Perl C.UTF-8 locale panic 返回 9；随后 Python hashlib 比对成功。一次临时 Python direct-require 正则误跨行导致辅助脚本 AssertionError，改为逐行解析后检查通过；不是产品失败或行为 RED。完整测试没有失败。

## 证据身份与范围

测试只使用仓库内 33 个受保护 *_test.go 的合成 JSON、fake Caller/Poster/HTTPDoer、本地 httptest、固定时钟和内存账本；没有真实配置/凭据/交易材料。fixture 聚合 SHA-256 为 `59033127f8c99b912c9dd6044217d01af2454aac7b882cd01b4ee6c9c3faed39`。70 个受保护实现/测试/module/示例配置文件的聚合为 `2e5cff053e75fed0271445fb8822bc7f8094b4080d54e052de193d6ab8243af0`。算法均为按相对路径排序，逐项拼接 path + NUL + 文件 SHA-256 + 换行，再 SHA-256；完整清单见 task-42-report.md。

go.mod：`e24bad16723d84ef5e3fd1ba63c2468e13a7bf7512c868e404c579a81ca38dfc`；go.sum：`e70baf26204f57a987c7b58491584d02b8252a5abb10dfda48f9d9fd35c58d03`。`config.example.json` 符合现有 schema，不需迁移，字节保持不变，SHA-256 `e7d099804ebaf4421f89925ca7b36aeac307b42d09eb159a9f9bc0988f792c21`。模板仅有 synthetic env 引用，没有 endpoint/凭据或 live 开关；两个开关默认关闭，验收又显式设 0。

独立收据：task-39-audit.md（PLAN-002）和 task-41-audit.md（Solana GREEN）均 AUDIT_PASSED、P0/P1/P2=0。task-41-report.md 的实现/测试/module 哈希与当前相符。它们不是 T042 最终审计结论；本轮未提前填 Auditor PASS。当前任务只留下五个获准文档/证据文件改动，config 模板不变；报告是流程例外。不改生产行为、测试或治理，无 Git 操作。

## API、调用路径与默认关闭核查

- EVM 13 个原生查询/计算 + 1 个字符串提交；Ethereum/BNB 嵌入项目受限 Client 并显式 ValidateNetwork。Solana 精确 11 个方法，README 名称自动比对 11/11，所有参数/返回签名按源码逐项核对。旧 types.go、DTO 别名和转发层不存在。
- EVM 私有 ethclient/rpc 使用调用方显式注入的 RawURL 和内存 `callerRoundTripper`，Solana 私有 NewWithCustomRPCClient/`callerRPCClient`；均无完整上游 Client 暴露、默认网络 transport、后台或重试。geth 构造内部 crypto/rand 熵读取仍披露。
- 项目公开 API 和实际调用路径未见私钥输入、交易对象构造/MarshalBinary/签名/节点 unlock。EVM 仅 eth_sendRawTransaction 字符串；Solana 仅 SendEncodedTransactionWithOpts 字符串及 SimulateRawTransactionWithOpts 已序列化字节的 base64 传输包装。上游依赖图包含 crypto、bip39、unsafe 等包；这不等于项目调用它们的签名能力，审查遵循 constitution 5.4。
- 项目生产文件不导入 unsafe。solana/caller_rpc_client_test.go 使用已批准的 reflect/unsafe 读取私有原生 client 以测试 callback/batch；这是固定版本耦合，只在测试。公共 jsonrpc/httpjson 的 reflect 用于输出指针校验和原子解码，不用于 typed-nil 依赖探测。
- 原始错误字段、正文与完整 Cause 保留；Caller 错误不重新编码为有损错误信封。非法提交零请求、明确 RPC 拒绝已知、其他提交不确定错误 unknown，非零标识不表示执行/确认。
- TestReadContractDisabled/TestSubmissionContractDisabled 通过完整 integration 测试执行；其 forbidden-dependency 回调断言关闭后零配置/材料/时钟/客户端/账本/输出调用。真实入口仅读取开关后 NOT_RUN/Skip；没有仅以 Skip 替代零读取证明。

## 当前材料版本与 Solana 字典

当前目录为 internal/test/integration。Ethereum/BNB/Solana read manifest 的 manifest_version 和 submission material 的 version 为 2；Bitcoin/Tron 为 1；旧原生链 v1 请求前 BLOCKED，report/journal schema 未改。每次 EVM Caller 调用单独计预算，完整块与两个叔块计三次，预算二时不发第三次。

Solana read catalog 是 README 十个非提交方法的准确名称。inputs 为 account（原生 PublicKey）、read（GetSlotOpts）、blockhash（GetLatestBlockhashOpts）、account_options、transaction_options、block_options、search_transaction_history（bool）、fee_message（string）、simulate_transaction（[]byte 的 JSON base64）、simulate_options。历史 block/transaction 材料显式 json、confirmed/finalized、MaxSupportedTransactionVersion 指向 0，search_transaction_history=true。send_options 直接使用 srpc.TransactionOpts 原生字段，无旧 encoding/options 包装；payload 文件仍为已编码字符串。Account expected 为原生 Account，fee expected 为 nullable uint64，simulation expected 为原生执行 Err；无旧 present bool 字典。独立 expected 来源要求不变，本轮合成 fixture 不能用于 T036。

## 限制

本轮未运行新的 race、真实浏览器/账号 RPC/广播、发布或 Git 命令。不宣称生产就绪。read.json、broadcast.json 和五链 samples.json 仍不存在；AC-10/11 与 FA-01–06 未执行。需要另行授权及完整真实材料才能恢复 T036。全新 T042 Auditor 与 Human 最终门禁由 Controller 后续处理。

---

# T039 原生 EVM 离线证据（2026-11-17）

当前范围：constitution 1.5、PLAN-v8-NATIVE-SDK-v1，T037–T039 的 EVM/Ethereum/BNB。**T036 仍为 BLOCKED**；没有真实浏览器采样、账号 preflight、真实 RPC 或广播结果。T042 最终全仓/依赖验收和 Solana 原生迁移尚未完成。下方 T035 保留为历史记录，其中 Go 1.25、标准库唯一依赖、旧 EVM DTO/API 和版本 1 材料不再证明当前 EVM 基线。

## 当前依赖、环境与 fixture

go.mod 最低 Go 1.26.0；当前唯一直接依赖为 `github.com/ethereum/go-ethereum v1.17.10`，间接依赖见 go.mod/go.sum，无 replace/fork。本阶段尚未加入后续 T040 的 Solana 原生依赖。T039 实测 `go version go1.26.0 darwin/amd64`，host `Darwin arm64`；架构差异如实披露。

以下 E 表示三个环境赋值（不是可执行程序）：`GOPROXY=off GOCACHE=/private/tmp/chain-api-go-gocache GOMODCACHE=/private/tmp/chain-api-go-gomodcache`。T038/T039 使用已有工具链和 module 缓存，未下载。T037 先前获授权下载工具链/依赖，初次 DNS 失败不算 RED。T037 审计中添加 GOSUMDB=off 的尝试在工具链校验前失败，也不算 RED；此处验证沿用正常缓存校验。

测试只使用受保护测试文件内的合成 JSON、fake Caller/HTTPDoer 和固定时钟；完整交易由合成 JSON 解码，未构造、签名或编码交易对象。未读取真实凭据或外部验收材料，未写真实广播 journal。Fixture 身份绑定于下述 42 文件摘要清单中的测试文件。

## T037 RED 与 T038 GREEN（复用前序实际运行记录）

完整命令、输出与 SHA-256 清单见仓库内 `.superpowers/sdd/plan/task-37-report.md`、`task-38-report.md`；独立收据见 `task-37-audit.md` 的 Fix round 1 和 `task-38-audit.md`。T037 最终为 RED 测试审计 AUDIT_PASSED，T038 为 SPEC PASS / QUALITY PASS / AUDIT_PASSED，均 P0/P1/P2=0；不等于 T039 审计、最终 Human 验收或真实链验收。

| 阶段与实际命令 | 观察结果 |
|---|---|
| T037 `GOCACHE=/private/tmp/chain-api-go-gocache GOMODCACHE=/private/tmp/chain-api-go-gomodcache go test ./evm ./ethereum ./bnb` | exit 1；编译成功，旧客户端缺少原生接口/仍暴露旧方法的行为 RED |
| T037 同缓存 `go test ./internal/test/integration -run '^TestNativeEVM' -count=1` | exit 1；原生目录、v2 材料与预算契约 RED，非环境错误 |
| T037 Fix round 1：E `go test ./evm -run '^TestNativeEVMWrappedErrorAndRequestIsolation$' -count=1` | exit 1；15 个 query/compute/submit × RPC/HTTP/typed/cancel/deadline 用例执行至原生接口断言 |
| T038 E `go test ./evm ./ethereum ./bnb` | 改动前 exit 1；实现后 exit 0（0.971s / 1.269s / 0.664s） |
| T038 E `go test ./internal/test/integration -run '^TestNativeEVM' -count=1` | 初次 budget-2 被归类为 FAIL，修复后通过；最终 exit 0，0.309s |
| T038 `CHAIN_API_GO_LIVE_TEST=0 CHAIN_API_GO_LIVE_BROADCAST=0` 加 E `go test ./internal/test/integration -count=1` | 最终 exit 0，2.317s；包含保留链与关闭入口回归 |
| T038 E `go vet ./evm ./ethereum ./bnb ./internal/test/integration` | 初次指出原生 Transaction noCopy 值复制；改用批准的原生指针后最终 exit 0，无输出 |
| T038 Auditor E `go test ./evm ./ethereum ./bnb -count=1` | exit 0，1.029s / 0.722s / 1.332s；独立执行保护断言 |
| T038 Auditor 关闭两个 live 开关，加 E `go test ./internal/test/integration -count=1` | exit 0，3.899s |
| T038 Auditor E `go test ./internal/test/integration -run '^TestNativeEVM' -count=1` | exit 0，0.454s |
| T038 Auditor E `go vet ./evm ./ethereum ./bnb ./internal/test/integration` | exit 0，无输出 |

T039 编辑前已用 Python SHA-256 比对 T038 audit 清单：42/42 一致，包括全部 32 个测试、go.mod/go.sum 和八个现存相关实现文件。T039 只改文档，因此上述完整区块/叔块、预算、原始错误与提交证据可复用；不把历史运行描述成本次新运行。

## 当前 EVM 材料和请求预算

Ethereum/BNB read manifest 的 `manifest_version=2`，submission material 的 `version=2`；版本 1 在 Caller 请求前 BLOCKED。当前 Bitcoin/Solana/Tron 仍为版本 1；Solana 的版本迁移属于后续任务。journal/report 格式未改，不能把 journal 的版本字段一并升级。

EVM 动态输入保留 address/at_block_number/call/estimate/log_filter/fee_count/fee_percentiles 键，但内容采用 common.Address、geth.CallMsg、geth.FilterQuery；不再使用旧 CallRequest/Has/BlockRef 字典。CallMsg、FilterQuery 使用原生 JSON 字段形式（如 FromBlock/ToBlock/Addresses/Topics），字节结果的 JSON expected 是 base64；block_number 为 uint64 数字，FeeHistory 使用原生字段名。历史 transaction/receipt 采用 core/types 原生 JSON，缺失遵循原生 null/NotFound/pending 契约。冻结字典权威为 plan §10 与当前 runner，不应从下方历史示例生成 EVM 材料。

`BlockByNumber` 读取完整交易和叔块。每次底层 Caller 调用单独扣一次预算，不把一个 SDK 方法固定计作一个请求。确定性两叔块用例证明：预算 3 发生三次有序调用；预算 2 仅两次、第三次未发送、结果 BLOCKED。`callerRoundTripper` 首错停止，无并发 fanout 或重试。私有 integration context gate 将每次 Caller 调用交给既有 harness；普通 SDK context 不增加配置来源或后台行为。

## T039 验证边界

T039 实际运行 E `go test ./evm ./ethereum ./bnb`：exit 0，三个包均为 cached PASS；`gofmt -l evm/doc.go ethereum/doc.go bnb/doc.go`：exit 0，无输出。Python 比对 T038 audit 清单：编辑前后均 42/42 一致；五个文档的尾随空白、终止换行和冲突标记检查通过。README 方法名清单自动比对 14/14，参数、返回值及两个构造/ValidateNetwork 示例已按源码逐项检查；Markdown 示例没有另行编译执行。

最终文档摘要见 `.superpowers/sdd/plan/task-39-report.md`。本任务禁止 Git，因此未运行 brief 中的 `git diff --check`，使用上述非 Git 空白检查，明确不宣称两者等价。未运行 full repository test/vet、race、最终依赖图审计或真实入口。T036 必须继续等待独立真实材料、请求预算及明确授权；离线合成成功不能替代该门禁。

---

# T035 历史离线交付核验（旧 EVM 语义已由上文替代）

基线：PLAN-v7-REMAINING-RAW-v1+PARALLEL-v1；constitution 1.4。本文不是 T036 真实验收报告。真实浏览器采样、独立账号 RPC preflight、真实只读 RPC 和广播均未在 T035 执行或获授权；AC-10/AC-11、FA-01–06 不据此通过，不宣称生产就绪。

## 已有证据与本次范围

T034 独立复审 task-34-audit-fix1.md 为 AUDIT_PASSED：read/submission 离线契约、两个开关强制关闭的入口验证、聚焦 vet 均通过。开启场景使用内存合成 reader/Doer/Caller/Poster、合成配置和固定时钟；没有读取真实凭据。关闭开关零依赖调用由 forbidden-dependency 断言证明，不能仅以入口 Skip 为证。

T035 只改 README、根及十个子包 doc.go、本文；生产实现、测试、依赖和 live JSON 不变。复用未改实现/测试的既有证据，不重跑全套行为测试；新增核验为 go vet、go list -deps 和源码 API/路径检查。准确结果与快照见末尾及任务报告。

## test-only 配置和固定 alias

规则仅属于 internal/test/live，不是核心 SDK 配置或统一链入口。精确字典见批准 plan §7.2、§7.8.1–§7.8.7，不能使用 §7.8.1 之前的历史摘要。

| 固定链顺序 | read alias | broadcast alias | 非提交方法数 |
|---|---|---|---:|
| Ethereum | ethereum_read | ethereum_broadcast | 13 |
| BNB | bnb_read | bnb_broadcast | 13 |
| Bitcoin | bitcoin_read | bitcoin_broadcast | 9 |
| Solana | solana_read | solana_broadcast | 10 |
| Tron | tron_read | tron_broadcast | 15 |

Read 首先只读 CHAIN_API_GO_LIVE_TEST；不是 1 则 NOT_RUN，配置、材料、时钟、客户端、账本及输出均零调用。Broadcast 先检查该开关，再检查 CHAIN_API_GO_LIVE_BROADCAST；任一不是 1 则 NOT_RUN。开关不是操作授权的替代品。本任务未开启真实入口。

启用后才读取显式 CHAIN_API_GO_LIVE_CONFIG 路径。配置 targets 项允许 endpoint_env、header_env、network、max_requests、max_concurrent、min_interval、total_timeout、dynamic_inputs_path、submission_material_path。endpoint/Header 只通过指定 env 引用获取；不搜索、展开或猜节点。四个限额为正，duration 带单位。test-only 宿主创建自己的 HTTP client/transport，拒绝 redirect、无重试并显式注入；核心 SDK 不创建默认客户端。

required IDs 固定为 `<chain>.<方法名>` 及 `.sample.1`–`.sample.3`，不从 cases/材料反推。缺链、漏方法或样本、缺 preflight 均不能 PASS。每 operation 只调用一个 SDK 方法，请求计数不等同于供应商 credit；并发等待可取消，间隔和总 context deadline 由 harness 控制，无后台 worker。FAIL 优先于 BLOCKED，未执行保留 NOT_RUN。

## 严格材料 schema

材料只接纳单份 JSON object，拒绝重复键、未知字典字段、尾随文档与类型错误；不把材料 schema 复制到原始 transaction/meta 或链 DTO。

- read manifest 固定为 testdata/各链/samples.json；顶层 manifest_version=1、chain、network、network_identity、samples。sample 含 sample_id/category/source_url/observed_at/transaction_id/block_hash/height_or_slot/addresses/expected/field_sources/rpc_preflight/requirements。
- dynamic 文件来自 DynamicInputsPath；顶层 manifest_sha256/max_response_bytes/code_snapshot/inputs/preflight/observed_at/expires_at。没有历史摘要中的 expected_chain_id 字段。SHA-256 冻结 read manifest 或 broadcast material 全部原始字节，包括空白、键顺序、时间，不规范化，也不能由同次被测输入生成预期摘要。
- preflight 含 status/observed_at/endpoint_alias/network_identity/methods；status 必须 PASS，方法 outcome 为 success/rpc_error，含 rpc_code/has_rpc_code/node_code/expected。PASS 可能表示独立检查记录了预期拒绝，不等于 RPC 成功；覆盖和允许拒绝范围由固定目录规定，不匹配英文 message。
- network_identity：EVM 仅十进制正数 chain_id；Bitcoin 仅 bitcoin_chain/genesis_hash；Solana 仅 genesis_hash；Tron 仅 tron_genesis_block_id。其他字段为空，禁止混合身份。
- 每链至少三笔互异交易；前三类按序为 Ethereum native/token/failed、BNB native/token/contract、Bitcoin multi_output/segwit/structure、Solana native/token/failed、Tron native/token/failed。公开 URL 不含 userinfo；field_sources 精确覆盖 expected 顶层键，值为 browser:公开URL 或 independent_rpc:公开方法名。
- sample_id/authorization_id/network 使用批准的安全字符集；日期 RFC3339，dynamic/material 必须未过期。code_snapshot 为 40/64 位 hex；runner 仅检格式、绑定报告，Controller 另核实实际代码树。

独立 preflight 须使用获授权账号的非 SDK HTTP 路径，不能用被测 SDK 生成预期。历史与动态状态分开；EVM 余额/nonce 使用固定 block reference，确认数不永久等于采样值。JSON 比较使用 UseNumber，数字 token 不经 float64；缺失、null 保持不同。

inputs 固定为四种具体结构：EVM address/at_block_number/call/estimate/log_filter/fee_count/fee_percentiles；Bitcoin txout/mempool_txid/estimate/preflight_payload；Solana account/read/blockhash/account_options/transaction_options/block_options/status_options/fee_message/simulate_transaction/simulate_options；Tron account/contract_call。嵌入 SDK 请求的材料键用 Go 字段名（含 Has），evm.CallRequest 用其 tags；EncodedTransaction 用 Data/Encoding。这是材料格式，不是节点 wire。

expected 使用 DTO tags；Bitcoin 已转换 Value/BaseFee/FeeRate 存 sat/sat-kvB 整数，不存 BTC 小数。RawMessage 缺失通过省略键表达，null 显式保存；不在 DTO 插入 absent 标记。Solana 历史材料显式 json、版本 0、confirmed/finalized 和 SearchTransactionHistory=true；不缩小 SDK 公共选项范围。

## 单次材料、claim 与查询

submission material 顶层为 version=1/chain/network/endpoint_alias/authorization_id/authorized=true/expires_at/network_identity/payload_path/payload_sha256/transaction_id/block_hash/send_options/expected_execution。payload 独立读取一次，摘要覆盖原始字节，不 trim；Tron 为原交易 JSON，其余为已编码字符串。报告不保存完整交易、秘密或凭据。

broadcast dynamic.inputs 必须为空对象，preflight 覆盖身份方法；非 Solana send_options 必须为空对象，Solana 用 encoding/options 与既定 SendOptions；只有 Bitcoin 可填 block_hash。expected_execution 分链：EVM evm_status=0x0/0x1；Bitcoin bitcoin_min_confirmations>=1；Solana solana_commitment=confirmed/finalized 和非缺失 solana_err（null/object/string）；Tron tron_receipt_result 为原字符串或 `{"absent":true}`。此标记仅表示 receipt.result 键缺失。

每链 identity→submit→lookup。身份由 EVM ChainID、Bitcoin chain+genesis、Solana genesis、Tron genesis BlockID 核对，失败禁止提交。claim 先追加并 Sync 成功，才允许一次 send。已有相同 chain+authorization_id 或 chain+payload_sha256 禁止再次提交；换授权号不能重复 payload。拒绝、unknown 或 claim 后进程中断均永久消费材料。

evidence/broadcast.json 为追加式 JSON Lines，不是单 JSON 文档。记录含 version/type/recorded_at/claim/report；type 仅 claim/report；claim 含 chain/authorization_id/payload_sha256/material_sha256。空文件允许，非法/截断行 BLOCKED，持久化失败阻止发送。单写者由 T036 Controller 启动前保证，没有跨进程锁；不能保证独占就不得启动。

接受或 unknown 后只按原材料 transaction_id 查询一次：EVM receipt、Bitcoin raw transaction、Solana statuses、Tron solidified info。执行不符 FAIL，缺确认 BLOCKED；unknown 即使查到对象也不抹去 submit BLOCKED。无重发、轮询或等待确认；后续只读复查须另获授权，不得重跑提交套件。

read.json 保存固定 suiteReport；broadcast 先追加 claim 再追加 report。报告只含安全状态、类别、ID、alias、摘要、快照、时间、计数。SDK 原始 Message/Data/ResponseBody/Cause 仅在 callback 内检查，不格式化整个 error、配置、payload 或 expected 差异进入报告；Error() 不承诺脱敏。

## T036 保持未完成

仍须另获授权并取得五链浏览器样本、独立账号 preflight、冻结材料与请求预算；广播还需逐链授权、未消费专用材料、正确网络和单写者保证。任一缺失均 BLOCKED，不用公共节点、随机样本或离线 fixture 代替。本任务未改 samples.json、read.json、broadcast.json 或 final.md。

## T035 verification results (2026-11-17)

Environment: go1.25.14 darwin/amd64; go.mod specifies Go 1.25.0 and no third-party require. Both commands explicitly used GOCACHE=/private/tmp/chain-api-go-t035 GOPROXY=off GOSUMDB=off.

- go vet ./...: exit 0, no diagnostics.
- go list -deps ./...: exit 0; only this module and Go standard library, including toolchain vendor packages.
- gofmt: only the eleven doc.go files; gofmt -l returns no paths.
- Current source method inventory: EVM 14 (inherited by each concrete client), two ValidateNetwork methods, Bitcoin 10, Solana 11, Tron 16. README covers all names and signatures.
- Before/after byte comparison: exactly the 13 authorized documentation paths changed; no implementation, test or dependency changes. No trailing whitespace found. Git commands were prohibited; Controller will perform the read-only diff check instead of Coder running git diff --check.

Unchanged implementation/test/go.mod snapshot excluding doc.go: 66 files, SHA-256 aggregate `53fd5aa2db0b490d58d844f4dd115c2f10eef1433f5f8e24c1cd9023573cff82`. Aggregate input is sorted relative path + NUL + file SHA-256 + newline. This preserves the inputs of prior behavioral evidence; complete documentation hashes are in task-35-report.md. No new behavioral RED/GREEN or real-chain PASS is claimed.

## PLAN-v9 场景离线证据（2026-11-18）

四个 live 开关为 0 时，`go test ./internal/test/integration -run '^TestScenario' -count=1` 退出码 0，`go vet ./internal/test/integration` 退出码 0。覆盖场景基座、EVM、Bitcoin、Solana、Tron 和拒绝探针的本地 fake。没有公网请求。真实只读运行和探针不在本节声称通过。T036 仍 BLOCKED。
