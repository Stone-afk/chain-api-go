# Native migration offline status; T036 final acceptance: BLOCKED

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
