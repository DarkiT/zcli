//go:build ignore
// +build ignore

// 这个文件用于演示编译时类型安全检查
// 使用 go build 时会报错，证明类型系统正常工作

package main

import (
	"context"

	"github.com/darkit/zcli"
)

func wrongSignatureFunc(a, b int) error {
	// 错误的函数签名 - 参数不匹配
	return nil
}

func wrongReturnFunc(ctx context.Context) {
	// 错误的函数签名 - 缺少错误返回
}

func correctRunFunc(ctx context.Context) error {
	// 正确的运行函数签名
	return nil
}

func correctStopFunc() error {
	// 正确的停止函数签名
	return nil
}

func wrongStopFunc() {
	// 错误的停止函数签名 - 缺少错误返回
}

func main() {
	// ✅ 正确用法 - 完整配置
	app1 := zcli.NewBuilder().
		WithService(correctRunFunc, correctStopFunc).
		Build()

	// ✅ 正确用法 - 仅配置运行函数（停止函数可选）
	app2 := zcli.NewBuilder().
		WithService(correctRunFunc).
		Build()

	// ❌ 错误用法 - 应该编译失败
	// app4 := zcli.NewBuilder().
	//     WithService(wrongSignatureFunc, correctStopFunc).  // 类型不匹配
	//     Build()

	// ❌ 错误用法 - 应该编译失败
	// app5 := zcli.NewBuilder().
	//     WithService(wrongReturnFunc, correctStopFunc).  // 类型不匹配
	//     Build()

	// ❌ 错误用法 - 应该编译失败
	// app6 := zcli.NewBuilder().
	//     WithService(correctRunFunc, wrongStopFunc).  // 类型不匹配
	//     Build()

	_ = app1
	_ = app2
	_ = app2
}
