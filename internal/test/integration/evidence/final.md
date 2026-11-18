# Native migration offline status; T036 live runs PASS, pending audit and Human gate

> 2026-11-18 更新：在当前代码快照上，T036 的真实只读与广播各有一次 PASS 运行，五链全部 PASS（只读此前另有一次修复前的运行，已失效），详见文末“T036 真实验收运行”。以下各节为历史记录；其中“T036 BLOCKED”的表述反映当时状态。独立审计有条件通过（P0/P1=0）；Human 已于 2026-11-18 批准 T036 通过。

Date: 2026-11-18  
Baseline: `PLAN-v8-NATIVE-SDK-v1`, constitution 1.5  
T042 Coder recommendation: `IMPLEMENTED`; independent final Auditor and Human final acceptance pending.  
T036 / AC-10/11 / FA-01–06: `BLOCKED`, not run.

T037–T041 的原生迁移已有阶段证据：T039 与 T041 独立收据均 AUDIT_PASSED（P0/P1/P2=0）；T042 本轮文档与离线检查已执行，尚待独立最终审计。历史 T017–T035 不替代新基线证据。完整命令、退出码、缓存修复、快照与限制见 [offline.md](offline.md) 和仓库内 `.superpowers/sdd/plan/task-42-report.md`。本文不表示五链真实验收通过，也不表示生产就绪。

## T042 实际离线结果

工具链 go1.26.0 darwin/amd64，宿主 Darwin arm64。go.mod 精确 go 1.26.0，直接依赖固定 go-ethereum v1.17.10 与 solana-go/v2 v2.3.0，无 replace/fork/额外直接依赖。

所有离线 Go 命令使用 GOPROXY=off、GOTOOLCHAIN=auto、GOCACHE=/private/tmp/chain-api-go-gocache、GOMODCACHE=/private/tmp/chain-api-go-gomodcache，CHAIN_API_GO_LIVE_TEST=0、CHAIN_API_GO_LIVE_BROADCAST=0。

- `go test ./...`：exit 0，十个测试包 PASS，根包/transport 无测试；部分为 Go cache。
- `go vet ./...`：exit 0，无诊断。
- `go list -deps ./...`：exit 0，373 package 路径。
- `go list -m all`：首次离线 cache 缺失 exit 1；经 Controller 授权补齐依赖缓存、精确恢复保护元数据后，最终 exit 0，236 module 行。
- 全部 78 个 Go 文件 `gofmt -l`：exit 0，空输出；覆盖 T037–T042。
- README 两个合成示例 `go run`：各 exit 0；没有 RPC、提交或签名。
- API/调用路径核查：受限原生 Client、准确方法目录、encoded submit once、serialized-byte simulation、原始错误与双关闭零读取符合当前离线范围。test-only unsafe 私有字段检查存在，生产文件无 unsafe import；依赖自身可含未调用签名能力。

配置模板未改，SHA-256 `e7d099804ebaf4421f89925ca7b36aeac307b42d09eb159a9f9bc0988f792c21`。70 个受保护源码/测试/module/模板文件聚合 SHA-256 `2e5cff053e75fed0271445fb8822bc7f8094b4080d54e052de193d6ab8243af0`；33 个合成测试 fixture 聚合 `59033127f8c99b912c9dd6044217d01af2454aac7b882cd01b4ee6c9c3faed39`。算法及逐文件清单见 task-42-report.md。go.mod/go.sum 分别为 `e24bad16723d84ef5e3fd1ba63c2468e13a7bf7512c868e404c579a81ca38dfc` / `e70baf26204f57a987c7b58491584d02b8252a5abb10dfda48f9d9fd35c58d03`。

## 未获授权的操作

- 访问用户或公共区块链 RPC。
- 从区块链浏览器采集真实地址、交易哈希或区块样本。
- 读取真实 live 配置、环境变量、凭据或交易材料。
- 运行 `CHAIN_API_GO_LIVE_TEST=1` 的真实只读验收。
- 运行 `CHAIN_API_GO_LIVE_BROADCAST=1` 的任何广播或后续链上查询。

因此，本轮没有真实链 RPC、广播或浏览器采样，也没有生成真实请求计数、链上结果或材料摘要。依赖缓存补齐使用过获准的 module 下载网络访问；不把它混称为真实链请求或普通测试联网。

## 缺失材料与门禁

- 五链浏览器样本文件均缺失：Ethereum、BNB、Bitcoin、Solana、Tron 的 `samples.json`。
- 缺少调用方提供的外部 live 配置和五链独立账号 RPC preflight 记录。
- `evidence/read.json` 不存在；FA-01–FA-04 未执行，不能判定 PASS。
- `evidence/broadcast.json` 不存在；FA-05 未执行，不能判定 PASS。
- 缺少逐链明确广播授权、专用且未消费的签名材料、网络身份确认和单写者确认。
- FA-06 最终证据绑定与 Human 门禁尚未发生。

## 恢复条件

只有 Human 另行明确授权真实 RPC、浏览器采集以及需要执行的具体广播链，并提供全部外部配置、冻结样本、独立 preflight 证据和专用材料后，T036 才可恢复。广播必须逐链授权；缺少任一链必需项仍为 `BLOCKED`。不得以离线 fixture、公共 RPC、随机样本或自生成预期代替上述材料。

## PLAN-v9 只读场景一次运行（2026-11-18）

命令：`CHAIN_API_GO_LIVE_SCENARIO=1`，广播与探针开关为 0，`go test ./internal/test/integration -run '^TestLiveScenario$' -count=1`。退出码 1。请求数 47。耗时约 30 秒。本地 YAML 被 git 忽略，日志未保存 endpoint 或 Header。

| 链 | 结果 |
|---|---|
| Ethereum | 14 个非提交方法和 RawCall 均为 PASS |
| BNB | 14 个非提交方法均为 PASS |
| Bitcoin | 9 个非提交方法均为 BLOCKED。身份方法也是 BLOCKED，因此不是链名不符，也不是协议 FAIL |
| Solana | 10 个非提交方法均为 BLOCKED，同样从身份方法开始 |
| Tron | 多数 PASS。`EstimateEnergy`、`GetTransactionByID`、`GetSolidifiedTransactionByID` 为 FAIL |

提交方法保持 `NOT_AUTHORIZED`。拒绝探针没有运行。T036、AC-10、AC-11 仍是 BLOCKED。这次结果不是整体通过。

## PLAN-v9 只读场景诊断与重跑（2026-11-18）

诊断（只输出状态码与错误类别，不含 endpoint、Header 或凭据）：

- Solana：原样重跑后 10 个非提交方法全部 PASS。上次 BLOCKED 是 Infura 端的临时故障，代码未改。
- Tron 交易查询：区块内交易与按 ID 查询结果的 `raw_data_hex` 相同，`raw_data` 语义也相同，只是 JSON 字段顺序不同。经 Human 批准，场景比对改为 `txID` 加 `raw_data_hex`。
- Tron `EstimateEnergy`：TronGrid 返回的 `result` 对象只有 `code`（`CONTRACT_VALIDATE_ERROR`）和表明节点不支持能量估算的 `message`，没有 `result` 键。proto3 JSON 省略了 `result:false`，SDK 原先把它归为 Protocol。经 Human 批准（spec §9.6 AM-03）修改 `tron/compute.go`：`result` 缺失且 `code` 为非空字符串时归为 RPC 拒绝，并保留 `NodeCode`。经 Human 批准（AM-06），本地 YAML 加入 `known_unsupported: tron/EstimateEnergy`。
- Bitcoin：本机经透明代理（fake-IP `198.18.x.x`）访问 UTXO 节点时，HTTPS 握手被断开，HTTP 返回 502；同一代理访问其他站点正常。等待 Human 核查节点状态或代理路由。

TDD：`tron/compute_test.go` 新增 1 个 proto3 拒绝用例（修复前失败，修复后通过）和 5 个仍为 Protocol 的边界守护用例（修复前后都通过，用于防止修复范围过宽）。`scenario_tron_test.go` 新增字段顺序用例（修复前 FAIL，修复后 PASS）和 proto3 拒绝已知不支持用例；tamper 用例同步改变 `raw_data_hex`。

验证期间 Human 正在把 `bitcoin.New`/`solana.New` 改为单参数，集成包调用点尚未同步，暂时无法编译。因此场景测试和真实重跑在暂存区快照叠加本次 4 个文件后执行，快照随后删除。之后经 Human 批准，集成包 9 处调用改为单参数；因构造器不再检查 nil 依赖，删除了 bitcoin、ethereum、bnb、evm、solana、tron 测试中的 nil 依赖断言（ethereum/bnb 的 typed-nil 用例保留）。`go vet ./...`、`go test ./...`、`go test -race ./jsonrpc` 全部通过，四个 live 开关均为 0。

重跑：`CHAIN_API_GO_LIVE_SCENARIO=1`，其余开关为 0。请求数 56，耗时约 35 秒，总体 `BLOCKED/chain_result`，无 FAIL。

| 链 | 结果 |
|---|---|
| Ethereum、BNB | 全部 PASS |
| Solana | 10 个非提交方法全部 PASS |
| Tron | 15 个 PASS；`EstimateEnergy` 为 BLOCKED（节点不支持） |
| Bitcoin | 9 个 BLOCKED（节点不可达） |

提交方法保持 `NOT_AUTHORIZED`，拒绝探针未运行。T036、AC-10、AC-11 仍是 BLOCKED。

## PLAN-v9 Bitcoin 端点替换与 JSON-RPC 2.0（2026-11-18）

Human 批准（spec §9.6 AM-01、AM-05）：Bitcoin 改用 Tatum 公共网关（不带 key，URL 只写在被忽略的本地 YAML），并新增 `jsonrpc_version` 参数，1.0 和 2.0 各跑一遍。本地 infra 中已有的 Bitcoin 端点在本机都不可达。

- 代码：`scenario_bitcoin.go` 读取可选参数 `jsonrpc_version`，取值 `1.0`（默认）或 `2.0`，其他值为 FAIL configuration 且不发请求。SDK 未改动。
- TDD：`scenario_bitcoin_test.go` 的假节点按请求的版本返回 Bitcoin Core 对应格式（2.0 只带 `result` 或 `error` 之一）。新增显式 1.0、2.0 和非法版本 3 个用例；2.0 与非法版本用例在修复前失败，修复后通过。模板增加 `jsonrpc_version: 1.0`。
- 限流：Tatum 不带 key 约每分钟 5 个请求，超出后返回 429 和 `retry-after: 60`。TronGrid 在频繁重跑后也返回 429。本地 `min_interval` 从 200ms 调到 13s。

| 运行 | 范围 | 结果 |
|---|---|---|
| 1.0 | 五链 | 请求 50，约 606 秒。Ethereum、BNB、Solana 全部 PASS；Bitcoin 9 个非提交方法全部 PASS；Tron 第一个请求起整链 BLOCKED |
| 2.0 | Bitcoin、Tron | 请求 11，约 123 秒。Bitcoin 9 个非提交方法全部 PASS；Tron 第一个请求起整链 BLOCKED |

Tron 单独诊断时，9 个请求中 8 个返回 200、1 个返回 429；同日较早一次运行中 Tron 为 15 PASS、`EstimateEnergy` BLOCKED（节点不支持）。场景按设计不重试，因此 TronGrid 限流期间整链记为 BLOCKED。

等待 5 分钟后两次只跑 Tron，都是第一个请求失败。经 SDK 取得的响应为 HTTP 429，正文表明该 API key 当日 10 万次额度已用尽、超额后限每秒 1 次。根因是本地 TronGrid API key 当日 10 万次额度已用尽，超额后被降为每秒 1 次。该 key 与 infra 服务共用，额度不是被本场景耗尽的（本场景当日请求只有几百次）。

Human 批准（AM-02）Tron 改为不带 key 访问 TronGrid（公共匿名额度）。本地 YAML 删除 Tron 请求头后只跑 Tron：请求 16，约 201 秒。15 个非提交方法 PASS，`EstimateEnergy` 为 BLOCKED（节点不支持，已在 `known_unsupported`）。

当日汇总（各链取最近一次运行）：Ethereum、BNB、Solana 全部 PASS；Bitcoin 在 1.0 和 2.0 下 9 个非提交方法均 PASS；Tron 15 PASS、1 BLOCKED。五链没有 FAIL。场景整体仍为 BLOCKED，原因是 `EstimateEnergy`；且五链不是在同一次运行中全部通过。本地 YAML 已恢复为五链。

## PLAN-v9 拒绝探针（T060，2026-11-18）

Human 授权后，本地 YAML 临时写入 `submit_probe_authorization`（审批标记，非凭据）。命令：`CHAIN_API_GO_LIVE_SCENARIO=1 CHAIN_API_GO_LIVE_SCENARIO_SUBMIT_PROBE=1 go test ./internal/test/integration -run '^TestLiveScenarioSubmitProbe$' -count=1 -v`，只运行一次，退出码 0，约 86 秒。

| 链 | 固定畸形载荷 | 结果 |
|---|---|---|
| Ethereum | `0x00` | PASS（节点拒绝） |
| BNB | `0x00` | PASS（节点拒绝） |
| Bitcoin（Tatum） | `00` | PASS（节点拒绝） |
| Solana | `AA==` | PASS（节点拒绝） |
| Tron（TronGrid，不带 key） | `{"visible":false}` | PASS（节点拒绝） |

没有任何载荷被接受，无资金效果，也没有后续查询。运行结束后已从本地 YAML 删除授权标记，防止再次执行。T036、AC-10、AC-11 仍是 BLOCKED；探针通过不等于 T036 通过。

## PLAN-v9.1 修订与快照绑定（2026-11-18）

T057 首次独立审计结论为“不通过”：偏离只写在证据中，没有带版本号的基线修订。Human 随后书面批准 PLAN-v9.1-LIVE-SCENARIO-AMEND-v1：

- spec §9.6 逐条记录 AM-01–AM-09；plan FA-S7、§11.5 和 tasks T057 增加对应例外。
- constitution 升至 1.7：6.4 增加限定例外，只读场景与拒绝探针经逐链书面批准可用公共端点（本次为 AM-01 Bitcoin、AM-02 Tron），T036/AC-10/AC-11 不适用。
- AC-02 修订为构造器不再承诺拒绝 nil 依赖；README、`bitcoin/doc.go`、`solana/doc.go`、`tron/doc.go` 同步为单参数构造器，README 补充 Tron proto3 拒绝规则。
- 证据中删除 endpoint URL 和节点原始报错正文，修正 TDD 失败用例数量的表述，删除与 T060 矛盾的旧句。

快照可复现性：

- 首次运行、诊断后重跑（含暂存区快照上的那次）对应的代码已变化，且未记录摘要，不可复现。
- 自“1.0 五链”运行起至 T060，Go 逻辑代码未再改动；之后只改了三个 `doc.go` 的注释。
- 当前快照：HEAD `16c3f44`（另有未提交工作区改动），go1.26.0 darwin/amd64。94 个文件（全部 `*.go`、`go.mod`、`go.sum`、`config.example.yaml`，按 `LC_ALL=C sort` 排序后逐个 `shasum -a 256`，再对清单取 SHA-256）的聚合摘要为 `623c5894f09cc9b5934fcb6351423f99f5b5b75b4a511831706c0ddb99464d5a`。完整命令（在仓库根目录执行，路径为不带 `./` 的相对路径）：`git ls-files -co --exclude-standard -- '*.go' go.mod go.sum internal/test/integration/config.example.yaml | LC_ALL=C sort | xargs shasum -a 256 | shasum -a 256`。

T057 复审（同一 Auditor，全新只读）：有条件通过；三项 minor 条件（README 与 `bitcoin/doc.go` 的 JSON-RPC 版本表述、摘要命令、`code` 为 SUCCESS 的边界说明）已在本节写入后处理。

## T036 真实验收运行（2026-11-18）

Human 授权：使用用户账号节点（Infura；Bitcoin 用带 key 的 Tatum），自选浏览器样本，本地生成测试网地址并签名一笔测试网交易广播；修复 runner 的 FeeHistory 比较缺陷。所有 endpoint、key、私钥与外部材料只存放在仓库外 0600 文件中，本文与报告不含这些内容。T036 没有使用公共 RPC。

### Runner 修复（TDD）

`evm.go` 的 FeeHistory 期望比较原为 `reflect.DeepEqual`。`big.Int` 零值有两种内部表示，导致数值相同也判 FAIL。改为与其他方法一致的 `valueEqual`。`read_contract_test.go` 新增两行表格用例（零值相等 PASS、数值不同 FAIL）：修复前前者失败，修复后两者都符合预期。SDK 未改动。

### 只读（AC-10）

命令：`CHAIN_API_GO_LIVE_TEST=1 CHAIN_API_GO_LIVE_BROADCAST=0 go test ./internal/test/integration -run '^TestLiveRead$' -count=1`，只读一次。时间 14:33:12–14:34:56（+08:00），总体 PASS。

| 链 | 网络 | 请求数 | 用例 | 结果 |
|---|---|---|---|---|
| Ethereum | mainnet | 22 | 16 | PASS |
| BNB | mainnet | 22 | 16 | PASS |
| Bitcoin | mainnet | 15 | 12 | PASS |
| Solana | mainnet | 16 | 13 | PASS |
| Tron | mainnet | 30 | 18 | PASS |

样本来自公开浏览器，按类别每链 3 个；期望值由独立 Python 脚本直接请求同一账号节点采集，不经过 SDK。`evidence/read.json` SHA-256 `ce8ad987e26bef672298f34bc4d7372010ad39c635f09ad43694311502c209f6`。修复前的首次运行 74/75 PASS（唯一失败为上述 BNB FeeHistory），其报告已被本次覆盖。

### 广播（AC-11）

命令：`CHAIN_API_GO_LIVE_TEST=1 CHAIN_API_GO_LIVE_BROADCAST=1 go test ./internal/test/integration -run '^TestLiveBroadcast$' -count=1`，只运行一次（加锁防重入）。时间 18:07:47–19:13:50（+08:00），退出码 0，总体 PASS。journal 共 6 行：5 条 claim 与 1 条最终报告。`evidence/broadcast.json` SHA-256 `394f5f321a543c1d0e91a707f126281e1fb357754bf386f605a126b91161da4b`。

每链一笔测试网交易，签名在仓库外的独立签名程序中完成。SDK 只负责提交已编码载荷与查询结果。

| 链 | 网络 | 交易 | 期望 | 请求数 | identity / submit / lookup |
|---|---|---|---|---|---|
| Ethereum | Sepolia（11155111） | 0 值自转账 `0xaea3a1af…89eed5` | receipt status `0x1` | 3 | PASS / PASS / PASS |
| BNB | BSC testnet（97） | 0 值自转账 `0x7a01363b…8001c0` | receipt status `0x1` | 3 | PASS / PASS / PASS |
| Bitcoin | testnet4 | 自转账 `bd5f0ac9…df351b`，手续费 2000 sat | 确认数 ≥ 1 | 4 | PASS / PASS / PASS |
| Solana | devnet | durable nonce 加 1 lamport 自转账 `kYDYzvoT…jxRz` | `confirmed`，err 为 null | 3 | PASS / PASS / PASS |
| Tron | Nile | 1 sun 转给水龙头发送方 `66594ec1…c33a88d` | 原生转账无合约回执 | 3 | PASS / PASS / PASS |

Bitcoin 需注意：runner 的 lookup 依据 Tatum 节点的返回（确认数 1，所在区块 `00000000002d8283…903879`，高度 162051）。运行结束后复核时，mempool.space 在同一高度 162051 的区块为 `00000000f8dc34a2…b59709`，不认识 Tatum 的区块，并显示该交易未确认。testnet4 当时在该高度存在两个竞争区块。PASS 反映的是被测节点当时的回答，符合 runner 设计；若 Tatum 一侧的分支被淘汰，交易会回到交易池等待重新打包。20:35（+08:00）只读复核：分叉已收敛，Tatum 一侧的区块胜出。mempool.space 显示该交易已确认，位于区块 `00000000002d8283…903879`（高度 162051），链高 162055；Tatum 返回确认数 5，区块哈希相同。未重新广播。

测试网资金来源：Solana devnet airdrop、Nile 水龙头、coinfaucet.eu（用户完成 hCaptcha）、BNB Chain 官方 Telegram 支持机器人、pk910 Sepolia PoW 水龙头。pk910 拒绝本机代理出口 IP；经 Human 授权，临时在本机代理覆写脚本中让该域名直连，领取后已按备份逐字节恢复。代理的“启动后自动连接”开关需由用户手动关回。

### 快照绑定与离线复核

只读报告与广播报告中五链的 `code_snapshot` 都是 `7cb1195021841462217c3de40f32cfd498579d1c186f86a2d2c1351891901b9c`，与当前工作区一致。HEAD `16c3f44`（另有未提交工作区改动），94 个文件，go1.26.0 darwin/amd64，计算命令同上节。四个 live 开关均为 0 时，`go test ./...` 全部通过，`go vet ./...` 无诊断。

结论：AC-10 与 AC-11 的真实运行均为 PASS。独立审计有条件通过（P0/P1=0），Human 于 2026-11-18 批准 T036 通过。

