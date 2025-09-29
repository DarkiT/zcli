# ZCli 功能演示目录

本目录包含 ZCli 各种功能的演示程序。

## 📁 文件说明

### flag_export_demo.go
标志导出功能演示，展示如何：
- 获取和过滤命令行标志
- 导出标志给外部包使用（如 Viper）
- 智能排除系统标志
- 自定义过滤业务标志

**运行方式：**
```bash
# 基本演示
cd examples/demos
go run flag_export_demo.go

# 测试命令功能
go run flag_export_demo.go test --output yaml --verbose --config /tmp/config

# 查看帮助
go run flag_export_demo.go --help
go run flag_export_demo.go test --help
```

## 🎯 演示内容

### 系统标志检查
展示 23 个自动排除的系统标志，包括：
- 帮助系统：help, h
- 版本系统：version, v
- 补全系统：completion, completion-*, gen-completion
- 调试标志：debug-completion, trace-completion
- 配置系统：config-help, print-config

### 外部包集成
演示如何将过滤后的标志传递给外部包：
```go
// 基本用法
flags := app.ExportFlagsForViper()
WithBindPFlags(flags...)(config)

// 自定义过滤
flags := app.GetBindableFlagSets("debug", "internal-flag")
WithBindPFlags(flags...)(config)
```

### 实际应用场景
- Viper 配置绑定
- 第三方库集成
- 动态配置管理
- 多级配置系统

## 🔧 技术亮点

1. **智能过滤**：自动排除系统标志
2. **类型安全**：编译时错误检查
3. **可扩展性**：支持自定义排除列表
4. **调试友好**：提供完整的调试工具 