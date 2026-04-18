# DarkiT-ZCli 排障指南

本页只记录高频真故障与最短排障链，不重复展开完整 API 说明。

## 1. `BuildWithError()` 返回错误

优先检查：

1. 是否调用了 `WithName(...)`
2. 自定义 validator 是否返回了错误
3. 是否本应使用 `BuildWithError()` 却误用了 `Build()`

建议联查：

- `builder.go`
- `config-options.md`
- `enhanced_test.go`

## 2. Ctrl+C 后服务不退出

优先检查：

1. `RunFunc` 或 `ServiceRunner.Run` 是否监听 `ctx.Done()`
2. ticker / worker / goroutine 是否跟随 context 收敛
3. `WithShutdownTimeouts` 与 `WithServiceTimeouts` 是否设置合理

建议联查：

- `context-lifecycle.md`
- `service-management.md`
- `service.go`
- `service_signal_test.go`

## 3. `stopService()` 没执行或执行异常

优先检查：

1. 当前是否真的走到了 ctx 驱动退出路径
2. 停止函数本身是否阻塞
3. 是否已进入 timeout / force-exit 路径

建议联查：

- `service.go`
- `service_concurrent_test.go`

## 4. help / completion / version 输出异常

优先检查：

1. 根命令是否经过 `addRootCommand`
2. 后续代码是否覆盖了 help command / help func
3. 版本信息是否正确写入 `BuildInfo` 或 `Basic.Version`

建议联查：

- `builder.go`
- `command.go`
- `zcli.go`
- `final_integration_test.go`

## 5. 文案出现 `[Missing: ...]`

通常是语言包路径没命中，而不是 renderer 崩溃。

优先检查：

1. primary 语言是否注册成功
2. fallback 语言是否存在
3. path 是否与结构体字段名匹配

建议联查：

- `i18n.go`

## 6. service commands 不出现

例如：`install`、`start`、`stop`、`status`。

优先检查：

1. 是否同时设置了 `Name` 与 `Run`
2. 当前 app 是否真的是 service 场景
3. service 初始化是否被绕过

建议联查：

- `command.go`
- `service.go`
- `zcli.go`

## 7. `build.sh` / `build.bat` 跑不通

当前仓库更接近“库 + examples + tests”，不要盲信旧脚本。

优先改用：

- `examples/README.md`
- `go test -v`
- `README.md`

## 建议排障顺序

1. 先定位问题在 build / service / command / i18n 哪一层
2. 只打开本页对应小节
3. 再跳到对应 repo 文档或源码
4. 最后用 `*_test.go` 反证真实行为边界
