# ZCli × Cobra 融合兼容契约

## 一句话判定

ZCli 主线采用 **alias-first compatibility core + additive ergonomic layer**：  
**保留 Cobra / pflag 作为底层真相源，在其外层做装配和 DX 增强，而不是用一套全量 wrapper 去替代它。**

## 1. 为什么不是全量重写

`develop` 分支已经验证过一条更激进的路线：把 `Cli -> App`、`Command alias -> wrapper`、`FlagSet -> wrapper`、`NewCommand()` DSL、迁移文档、benchmark 一次性全部改掉。

这条路线的优点是表面 API 更纯净，但它不适合作为当前主线方案，原因有三：

1. **兼容性代价高**：会直接改变公开类型与 callback 语义。
2. **维护成本高**：需要长期追平 Cobra / pflag 的完整 surface。
3. **与当前目标冲突**：本轮目标是“保留 Cobra 原有全部能力并无缝植入扩展”，不是“重写成另一套 CLI 框架”。

## 2. 正式契约

### 2.1 兼容内核必须保留

以下能力视为主线不可破坏的 contract：

- `zcli.Command` 与 Cobra 命令对象的直接兼容关系
- `zcli.FlagSet` / `zcli.Flag` 与 pflag 对象的直接兼容关系
- `Builder -> Cli -> root command` 的现有装配主链
- `BuildWithError()` 的 error-first 构建语义
- `WithCommand()` 延迟收集命令的生命周期保证
- `ServiceRunner`、service commands、shutdown cause、timeout / force-exit 运行时语义

### 2.2 优雅能力只能做加法

允许增量引入以下“糖层”：

- `App` 语义别名
- `NewApp()` 入口
- `NewCommand()` 构造辅助
- 更统一的 `zcli` 类型化文档与示例
- 更顺手的 completion / flag export / hook helper

但这些能力必须满足两条硬约束：

1. **不替代兼容内核**
2. **不阻断 raw escape hatch**

### 2.3 原生逃生口必须存在

高级调用方需要能够继续访问底层 Cobra / pflag 能力，用于：

- 对接第三方 flag / completion / hook 库
- 使用 `Cli` 尚未显式封装的底层 API
- 排查底层行为差异

因此：

- 可以把它标注为 **高级互操作接口**
- 但不能为了表面“纯 zcli”而移除或弱化这条路径

## 3. 允许与不允许的演进方式

### 允许

- 在不改变底层 identity 的前提下统一公开词汇
- 在 README / examples 中优先展示 `zcli` 主路径
- 拆分 service 内部职责边界，降低运行时复杂度
- 通过测试与 benchmark 守住回归边界

### 不允许

- 直接把 `develop` 分支的 wrapper-first 重构当作主线既定方案
- 为了文档整洁，牺牲 Cobra 原生能力
- 未验证就继续公开承诺 `Go 1.23+`
- 引入大规模运行时对象转换，只为换取看起来更“统一”的 API

## 4. 当前已识别的风险

### 4.1 公开词汇泄漏

当前仍有少量公开签名直接暴露 `*cobra.Command` 或 `cobra.CompletionFunc`。  
这不是 bug，但会削弱“常规路径只需导入 zcli”的体验，应通过增量 API 收口。

### 4.2 文档漂移

当前文档体系中至少存在三类漂移风险：

- API 文档示例仍使用 `*cobra.Command`
- 完整示例仍把 `cobra` / `pflag` 作为主路径依赖
- README 的 Go 版本承诺与 `go.mod` 不一致

### 4.3 Service 高状态复杂度

`service` 相关代码功能成熟，但状态链长、边界密。  
后续任何改动，都必须以现有测试矩阵为护栏，而不是凭直觉重排。

## 5. 对后续任务的约束

Wave 1 ~ 3 的所有实现任务都必须遵守以下规则：

1. **先保护兼容内核，再谈 API 优雅**
2. **文档叙事必须服从源码真相**
3. **新增 sugar 必须是 additive，而不是 forced migration**
4. **service 改动必须先看测试再动代码**
5. **benchmark 和 release gate 是完成定义的一部分，不是可选项**

## 6. 当前版本说明

- 当前源码基线：以仓库 `go.mod` 为准
- 当前文档若出现与 `go.mod` 不一致的 Go 版本承诺，应视为待修正漂移，而非正式支持声明
- `develop` 分支中的实验性重构资产，当前仅作为参考，不代表主线承诺
