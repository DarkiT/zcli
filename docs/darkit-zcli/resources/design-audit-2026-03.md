# ZCli 设计审计与边界说明（2026-03）

本审计文档记录本轮对 `github.com/darkit/zcli` 的设计级问题分析、已修复项与仍需知晓的边界约束。兼容方向的正式契约见 `cobra-fusion-contract.md`。

## 本轮已确认并修复的问题

### 1. Builder 生命周期边界不稳

- `WithCommand()` 旧行为会提前触发 `Build()`
- 现已改为延迟收集命令，并在最终构建阶段统一落盘
- `BuildWithError()` 现在也会统一附加 init hooks

### 2. runtime ErrorHandlers 深拷贝断链

- `WithErrorHandler()` 配好的处理器在 `WithConfig()` 深拷贝后会丢失
- 现已在 `cloneRuntime()` 中复制 `ErrorHandlers`

### 3. service run 路径吞错

- `executeRunCommand()` 旧行为只记录 `Run()` 错误日志，不向调用方返回
- 现已通过错误通道回传，并统一走 handler 链

### 4. service 重复运行边界缺失

- `stopOnce` / `stopFuncOnce` 不在新一轮 `Run()` 前复位
- 现已在每次运行前重置 stop 状态

### 5. API error-first 语义不一致

- `WithServiceRunner(nil)` 旧行为会立即 panic
- 现已改为积累 build error，并由 `BuildWithError()` 返回

### 6. Mousetrap 全局副作用

- `WithMousetrapDisabled(true)` 旧行为会直接修改全局 `MousetrapHelpText`
- 现已改为执行期局部生效，并在执行后恢复

### 7. 文档与版本叙事漂移

- README 与资源文档中曾同时存在“`Go 1.23+` 最低要求”与源码 `go.mod` `go 1.25.0` 的不一致表述
- 部分 API 文档和示例仍把 `cobra` / `pflag` 当作主路径依赖，弱化了 `zcli` 的常规使用叙事
- 已在兼容契约中明确：文档叙事必须服从源码真相，`develop` 分支实验性重构不得写成主线既成事实

## 当前仍需知晓的边界

### 1. `Build()` / `MustBuild()` 仍是 panic-first API

生产代码优先使用：

- `BuildWithError()`

### 2. Cobra 仍是进程级全局模型

虽然本轮已经把 `Mousetrap` 等行为收束到执行期，但 Cobra 仍存在全局开关模型。

建议：

- 同进程内多个 CLI 实例尽量串行执行

### 3. service 代码仍属于高状态复杂度区域

`service.go` 同时承载前后台运行、超时、service commands、signal handling、error chain。

后续若继续演进，建议考虑拆分运行生命周期、命令适配与错误策略。

### 4. 公开词汇仍有少量底层泄漏

当前主线虽然采用 alias-first 兼容策略，但仍存在少量公开签名直接暴露 `*cobra.Command` / `cobra.CompletionFunc` 的情况。

建议：

- 主路径文档优先讲 `zcli` 词汇
- 对底层返回值保留明确的 raw escape hatch 定位
- 通过增量 API 收口，而不是改走全量 wrapper 重写

## 对使用者的推荐约束

1. 优先使用 `BuildWithError()`
2. 在 `RunFunc` / `ServiceRunner.Run` 中始终监听 `ctx.Done()`
3. 若需命令扩展，可在 build 前使用 `WithCommand()`，也可在 build 后直接 `app.AddCommand()`
4. 不要把 `WithMousetrapDisabled(true)` 理解为“永久修改全局默认值”

## 回归状态

本轮代码与新增测试已经通过：

```bash
go test ./...
```
