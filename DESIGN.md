# zcli DESIGN

> 设计说明与关键决策的正式记录。更细的审计细节见 `docs/darkit-zcli/resources/design-audit-2026-03.md`，兼容契约见 `docs/darkit-zcli/resources/cobra-fusion-contract.md`。

## 模块定位

`zcli` 是一个基于 Cobra 的 Go CLI 封装库，重点解决三类问题：

1. **Builder 组装**：把基础信息、命令树、service runtime、hooks、error handlers 组合为一致的 `Cli`
2. **Service 生命周期**：统一前台运行、系统服务安装/启动/停止、`RunWait` 优雅退出与超时处理
3. **I18n / UI 输出**：提供层次化语言包、service 本地化输出、help/version 等 UI 文案支撑

## Cobra 融合契约

当前主线采用 **alias-first compatibility core + additive ergonomic layer**：

1. **兼容内核**：`Command` / `FlagSet` / `Flag` 继续直接对齐 Cobra / pflag 真相源，避免维护一套难以追平的全量 wrapper。
2. **增强装配**：`Builder`、help/version/UI、service commands、init hooks、error handlers 通过 `Cli` / root command 的装配流程植入。
3. **优雅糖层**：允许继续增加 `App` 语义、`NewCommand()` 等 DX 能力，但只能做加法，不能替代兼容内核。
4. **原生逃生口**：高级调用方仍然需要能够访问底层 Cobra 能力，不能为了“纯 zcli 表面”把 escape hatch 封死。

明确拒绝的方向：

- 直接把 `develop` 分支的全量 wrapper 重写当作主线方案
- 为了文档表面整洁而牺牲 Cobra 兼容面
- 在没有验证的前提下承诺低于源码真实要求的 Go 版本支持

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
- `docs/darkit-zcli/resources/design-audit-2026-03.md`：设计审计与边界问题详细记录

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

本轮拆分遵循“低风险 + 低心智负担”：

- `service.go` 保留 manager 核心状态与生命周期主链
- `service_commands.go` 独立承接命令装配与命令工厂
- `service_support.go` 承接 support 级逻辑（force exit / permission / handlers）
- `i18n.go` 保留领域模型；`LanguageManager` 拆至 `i18n_manager.go`；`ServiceLocalizer` 拆至 `i18n_localizer.go`；内置语言字典与全局语言访问入口移至 `i18n_builtin.go`

**原因**：优先压低文件体积与复杂度，同时避免过度抽象导致阅读跳转过多。

### 7. 文档叙事必须服从源码真相

- 对外主路径可以优先讲 `zcli` 词汇，但不能伪造当前尚未实现的公开签名
- README / examples / resources 对 Go 版本的承诺，必须以 `go.mod` 与验证结果为准
- `develop` 分支中的实验性重构，只能作为参考资产，不得被文档误写成既成事实

**原因**：公共 CLI 框架最怕“代码能跑、文档误导、升级踩坑”；文档漂移本身就是兼容性风险。

## 已确认并修复的边界问题

### Builder / Config

- `WithCommand()` 提前触发构建，导致 Builder 冻结过早
- `BuildWithError()` 曾漏挂 init hooks
- `WithConfig()` 深拷贝 runtime 时曾丢失 `ErrorHandlers`

### Service

- `executeRunCommand()` 曾吞掉 `service.Run()` 错误
- `waitForServiceCompletion()` 曾无法把运行错误回传给调用方
- 同一 `sManager` 重复运行时，`stopOnce` / `stopFuncOnce` 曾不复位

### API / 全局状态

- `WithServiceRunner(nil)` 曾立即 panic
- `WithMousetrapDisabled(true)` 曾直接污染全局 `MousetrapHelpText`

## 仍保留的边界约束

1. `Build()` / `MustBuild()` 仍是 panic 语义，适合应用层，不适合不确定配置路径
2. Cobra 仍有少量全局模型；当前已做执行期隔离，但不承诺高并发多 CLI 实例完全无共享副作用
3. `ServiceLocalizer` 与 `sManager` 仍紧耦合，这是有意为之：service 子系统优先追求一致日志/错误体验，而非过早抽象
4. `go.mod` 当前声明 `go 1.25.0`；在重新验证前，不应继续把 README 中的 `Go 1.23+` 当作正式兼容承诺

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

## 变更历史

### 2026-03-07 - 生命周期与大文件拆分收口

**变更内容**

- 修复 Builder 生命周期、service error propagation、重复运行边界、Mousetrap 全局污染
- 将 service 拆为核心/commands/support 三个文件
- 将 i18n 按模型/manager/localizer/builtin 四层拆分，内置语言包与全局语言访问入口移至 `i18n_builtin.go`
- 新增正式 `DESIGN.md`，沉淀设计决策与边界约束

**变更理由**

- 收敛设计级缺陷
- 降低超长文件带来的维护成本
- 让 README、实现、审计文档三者口径一致

**影响范围**

- Builder 生命周期
- service runtime / command 装配
- i18n 内置语言组织方式
- repo 文档与设计记录
