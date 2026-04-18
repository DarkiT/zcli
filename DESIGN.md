# zcli DESIGN

> 设计说明与关键决策的正式记录。

## 模块定位

`zcli` 是一个基于 Cobra 的 Go CLI 封装库，重点解决三类问题：

1. **Builder 组装**：把基础信息、命令树、service runtime、hooks、error handlers 组合为一致的 `Cli`
2. **Service 生命周期**：统一前台运行、系统服务安装/启动/停止、`RunWait` 优雅退出与超时处理
3. **I18n / UI 输出**：提供层次化语言包、service 本地化输出、help/version 等 UI 文案支撑

## 当前结构

### Builder / Config

- `builder.go`：Builder 生命周期、延迟命令收集、构建阶段校验
- `options.go`：配置结构与 runtime/basic/service 拷贝逻辑
- `command.go`：`Cli` 执行包装、Cobra 全局隔离、执行入口

### Service

- `service.go`：`sManager` 核心状态、上下文联动、运行/停止、配置生成、等待退出
- `service_commands.go`：service CLI 装配、`run/install/start/stop/restart/status` 命令构建
- `service_support.go`：强制退出、权限检查、service error handler 链
- `service_interface.go`：`ServiceRunner` 等对外接口与错误类型

### I18n

- `i18n.go`：语言领域模型（Service/UI/Error/Format）
- `i18n_manager.go`：`LanguageManager`、回退策略、语言包校验
- `i18n_localizer.go`：`ServiceLocalizer` 与 service 输出封装
- `i18n_builtin.go`：内置 `zh/en` 语言包工厂与全局语言访问入口

### Repo 文档

- `README.md`：外部使用入口与 API 示例
- `examples/README.md`：官方示例入口与目录说明

## 核心设计决策

### 1. Builder 使用 error-first，而非 panic-first

- 推荐调用链：`NewBuilder(...).BuildWithError()`
- `Build()` / `MustBuild()` 仍保留，为兼容已有调用方，但视为“确定不会失败”的快捷入口
- `WithServiceRunner(nil)` 不再在配置期立即 panic，而是在构建阶段返回错误，保证 Builder API 一致性

**原因**：Builder 需要承载 runtime、hook、service、validator 多类可失败配置，error-first 更适合库级调用与测试。

### 2. 命令注册延迟到最终构建阶段

- `WithCommand()` 不再隐式触发 `Build()`
- pending commands 与 init hooks 在最终构建阶段统一落入 `Cli`

**原因**：避免“先注册命令、后补名称/配置”时 Builder 被提前冻结，破坏链式配置语义。

### 3. Service 错误必须统一回流

- `executeRunCommand()` 中的 `Run()` 错误不能只打日志，必须向上返回
- service 子命令统一通过 `wrapRunE()` 接入 `handleError()`
- `waitForServiceCompletion()` 必须能消费并返回运行期错误

**原因**：CLI 库的最重要职责之一，是把运行失败显式暴露给调用方，而不是静默吞错。

### 4. Service 生命周期必须支持重复运行

- 每轮 `Run()` 前重置 `stopOnce`、`stopFuncOnce`、`stopExecuted`
- 停止、上下文取消、强制退出之间的状态切换以 `sManager` 为唯一事实源

**原因**：测试、守护进程适配与某些嵌入式 CLI 场景会复用同一 manager，多轮 run/stop 必须可靠。

### 5. 全局副作用要执行期隔离

- `WithMousetrapDisabled(true)` 不直接永久修改 Cobra 全局变量
- 在 `ExecuteContext()` / `ExecuteContextC()` 周期内临时应用并恢复

**原因**：同进程多 CLI 实例不应互相污染全局展示行为。

### 6. 大文件按职责拆分，而不是按“工具函数”碎片化

当前拆分遵循“低风险 + 低心智负担”：

- `service.go` 保留 manager 核心状态与生命周期主链
- `service_commands.go` 独立承接命令装配与命令工厂
- `service_support.go` 承接 support 级逻辑（force exit / permission / handlers）
- `i18n.go` 保留领域模型；`LanguageManager` 拆至 `i18n_manager.go`；`ServiceLocalizer` 拆至 `i18n_localizer.go`；内置语言字典与全局语言访问入口移至 `i18n_builtin.go`

**原因**：优先压低文件体积与复杂度，同时避免过度抽象导致阅读跳转过多。

## 兼容性与实现约束

### Builder / Config

- `WithCommand()` 延迟收集命令，避免 Builder 在链式配置完成前被冻结
- `BuildWithError()` 负责统一附加 init hooks，并以错误返回承载构建期失败
- `WithConfig()` 深拷贝 runtime 时保留 `ErrorHandlers`，避免覆盖已有错误处理链

### Service

- `executeRunCommand()` 必须把 `service.Run()` 错误回传给调用方
- `waitForServiceCompletion()` 必须消费并返回运行期错误
- 同一 `sManager` 重复运行时，`stopOnce` / `stopFuncOnce` 等停止状态必须在新一轮运行前复位

### API / 全局状态

- `WithServiceRunner(nil)` 在构建阶段返回错误，而不是配置期 panic
- `WithMousetrapDisabled(true)` 仅在执行期临时调整 Cobra 全局展示行为，并在执行结束后恢复

## 仍保留的边界约束

1. `Build()` / `MustBuild()` 仍是 panic 语义，适合应用层，不适合不确定配置路径
2. Cobra 仍有少量全局模型；当前已做执行期隔离，但不承诺高并发多 CLI 实例完全无共享副作用
3. `ServiceLocalizer` 与 `sManager` 仍紧耦合，这是有意为之：service 子系统优先追求一致日志/错误体验，而非过早抽象

## 测试与验证策略

### 回归测试重点

- Builder 延迟构建与 init hook 挂载
- `WithServiceRunner(nil)` 构建期报错
- `WithMousetrapDisabled` 不污染全局
- service 多轮 `Run → Stop → Run → Stop`
- `waitForServiceCompletion()` 返回已处理错误

### 验证命令

```bash
gofmt -w builder.go command.go i18n.go i18n_manager.go i18n_localizer.go i18n_builtin.go options.go service.go service_commands.go service_support.go enhanced_test.go service_concurrent_test.go
go test ./...
node /opt/work/data/codex/skills/tools/verify-change/scripts/change_analyzer.js .
node /opt/work/data/codex/skills/tools/verify-quality/scripts/quality_checker.js .
```
