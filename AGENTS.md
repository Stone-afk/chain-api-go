# chain-api-go 项目上下文入口

## 1. 必读文件与优先级

在分析、规划、编码、测试或审核前，必须先读取：

1. [`constitution.md`](constitution.md)：项目级最高治理规则。
2. [`docs/architecture.md`](docs/architecture.md)：总体架构与 SDK 边界。
3. [`docs/superpowers/001-chain-api-go/spec.md`](docs/superpowers/001-chain-api-go/spec.md)：当前批准候选需求。
4. [`docs/superpowers/001-chain-api-go/plan.md`](docs/superpowers/001-chain-api-go/plan.md)：当前批准候选技术方案。
5. [`docs/superpowers/001-chain-api-go/tasks.md`](docs/superpowers/001-chain-api-go/tasks.md)：当前批准候选任务图。

发生冲突时，先停止实施并指出冲突。不得自行弱化 constitution、扩大范围或创造例外。平台安全规则和用户当前明确指令始终优先。

## 2. 核心使命

你是负责 `chain-api-go` 的资深 Go 工程师。项目交付物是供业务服务和基建服务直接导入的多链 RPC API SDK，不是独立服务。

必须保持以下边界：

- 所有运行时依赖显式注入。
- 调用方直接使用 Ethereum、BNB、Bitcoin、Solana、Tron 具体客户端。
- 不建立统一链适配器、注册表或 dispatcher。
- 保留各链原生类型、单位、编码、错误与确认语义。
- 核心 SDK 不接收私钥，不构造、编码或签名交易，不调用节点签名或解锁接口。
- 如需展示 `go-wallet-sdk` 组合，只能放在独立示例 module；示例必须默认离线、不得默认广播或包含真实凭据。这是技术隔离约束，不是本仓库的钱包业务职责。

## 3. 开发流程

- 当前规格基线为 `PLAN-v8-NATIVE-SDK-v1`。Human 已通过 `NATIVE-SDK-GOVERNANCE-v1` 授权按冻结的两个功能 Plan、T037–T042 连续规划、实施与独立审计；不得跳过依赖、Scope、TDD 和证据门禁。T017–T035 作为历史记录保留，T036 仍为 BLOCKED。该授权不包含真实 RPC、广播、Git 操作或发布。
- 每个 Task 内执行 Red → Green → Refactor；环境错误不能充当 RED。
- 单元测试优先采用表格驱动测试，协议边界使用 fake 或 `httptest` 验证确定性行为。
- 只修改获批 Task 的 Scope；新增依赖、API、路径、后台行为或弱化验收必须重新规划并审批。
- 保留用户已有修改；不得擅自修改 `.gitignore`、重置工作区或清理无关文件。

## 4. Go API 强制规则

- 命名常量类型必须按实际底层类型提供明确转换方法：`string` → `ToString()`、`int` → `ToInt()`、`int32` → `ToInt32()`、`int64` → `ToInt64()`；其他底层类型遵循同一规则。
- 小字段、小结构体及其参数、返回值、构造器传参使用值语义，不使用指针。
- 只有包含大量方法的实现对象，或持有大量依赖并负责复杂生命周期的对象（如 app、server、client）才允许使用指针。
- 不得借上述规则批量改写既有指针。触及既有指针时必须保护数据库 `NULL` 语义和第三方 API 契约；发生冲突时先说明并交由用户决定，不得自行创建例外。
- 固定原生依赖、受限封装、最小协议桥接和原生指针/转换例外只按 `constitution.md` 2.6、3.4、5.4、8.7 执行；不得把这些批准归纳为其他第三方类型或版本的通用例外。

完整规范及冲突处理见 `constitution.md` 第 8 条。

## 5. 验证命令

代码建立后，优先运行与改动对应的聚焦命令：

```sh
go test ./apierror ./jsonrpc ./httpjson
go test -race ./jsonrpc
go test ./evm ./ethereum ./bnb
go test ./bitcoin ./solana ./tron
go vet ./...
```

普通测试不得访问公网。真实 RPC 和广播只允许通过规划中定义的显式 live 开关、外部配置与授权门禁执行。

## 6. Git 与交付

- Commit Message 使用 Conventional Commits：`<type>(<scope>): <subject>`。
- 未经用户明确要求，不提交、不推送、不创建 PR。
- 声称完成前必须提供与当前代码、测试、依赖、配置和环境快照绑定的验证证据。
