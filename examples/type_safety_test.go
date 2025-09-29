//go:build ignore
// +build ignore

// 这个文件用于演示编译时类型安全检查
// 使用 go build 时会报错，证明类型系统正常工作

package main

import (
	"context"

	"github.com/darkit/zcli"
)

func wrongSignatureFunc(a, b int) {
	// 错误的函数签名
}

func correctModernFunc(ctx context.Context) {
	// 正确的现代API函数签名
}

func correctLegacyFunc() {
	// 正确的传统API函数签名
}

func main() {
	// 正确用法示例
	app1 := zcli.NewBuilder().
		WithSystemService(correctModernFunc).
		Build()

	app2 := zcli.NewBuilder().
		WithSystemServiceLegacy(correctLegacyFunc).
		Build()

	// ❌ 错误用法 - 应该编译失败
	// app3 := zcli.NewBuilder().
	//     WithSystemService(wrongSignatureFunc).  // 类型不匹配
	//     Build()

	// ❌ 错误用法 - 应该编译失败
	// app4 := zcli.NewBuilder().
	//     WithSystemServiceLegacy(correctModernFunc).  // 类型不匹配
	//     Build()

	_ = app1
	_ = app2
}
