# ZCli API 兼容性演示目录

本目录包含 ZCli API 兼容性和演进历程的演示程序。

## 📁 文件说明

### api_compatibility_demo.go
API 兼容性演示，展示：
- 优雅的可变参数设计
- 向下兼容的传统用法
- 现代 context 最佳实践
- 单一API多种调用方式

**运行方式：**
```bash
cd examples/legacy

# 现代风格（推荐）
go run api_compatibility_demo.go

# 传统风格（兼容）
go run api_compatibility_demo.go legacy

# 查看帮助
go run api_compatibility_demo.go --help
```

## 🎯 设计理念

### 优雅的可变参数设计
```go
// 统一API
WithSystemService(func(...context.Context), ...func())

// 支持三种调用方式：
// 1. 传统：func() - 忽略参数
// 2. 现代：func(ctx) - 使用第一个context  
// 3. 高级：func(ctx1, ctx2) - 多context扩展
```

### 向下兼容策略
```go
// 传统风格 - 无需修改现有代码
func legacyService(ctxs ...context.Context) {
    // 忽略ctxs，使用传统逻辑
    for condition {
        // 业务逻辑
    }
}

// 现代风格 - 利用context优雅停止
func modernService(ctxs ...context.Context) {
    ctx := ctxs[0]
    select {
    case <-ctx.Done():
        return  // 优雅停止
    case <-ticker.C:
        // 业务逻辑
    }
}
```

## 🚀 技术演进

### 第一阶段：双API设计
- `WithSystemService()` - 现代API
- `WithSystemServiceLegacy()` - 传统API
- 问题：API分裂，维护复杂

### 第二阶段：智能分派
- 运行时类型检查
- 自动选择合适的调用方式
- 问题：运行时开销，类型不安全

### 第三阶段：优雅可变参数（最终方案）
- 单一API：`func(...context.Context)`
- 编译时类型安全
- 完全向下兼容
- 支持未来扩展

## 💡 核心洞察

> "func(...context.Context) 的本质是 []context.Context"

这个深刻洞察启发了最优雅的解决方案：
- 0个参数 → 传统模式
- 1个参数 → 现代模式  
- 多个参数 → 高级扩展

## 🎉 最终成果

1. **API统一**：单一方法，多种用法
2. **类型安全**：编译时检查
3. **完全兼容**：现有代码零修改
4. **易于扩展**：支持未来高级用法
5. **优雅简洁**：符合Go设计哲学

这是API设计的最高境界：**简洁、优雅、强大**。 