package zcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ============================================================================
// 文本处理工具函数
// ============================================================================

// wordWrap 将文本按指定宽度换行
func wordWrap(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	line := words[0]
	for _, word := range words[1:] {
		if len(line)+1+len(word) <= width {
			line += " " + word
		} else {
			lines = append(lines, line)
			line = word
		}
	}
	lines = append(lines, line)
	return strings.Join(lines, "\n")
}

// ============================================================================
// 命令路径工具函数
// ============================================================================

// getCommandPath 获取完整的命令路径
func getCommandPath(cc *cobra.Command) string {
	base := filepath.Base(os.Args[0])
	if cc.Name() == "" {
		return base
	}
	// 如果是子命令且有父命令
	if cc.Parent() != nil {
		// 返回 "可执行文件名 子命令名"
		return fmt.Sprintf("%s %s", base, cc.Name())
	}

	return cc.CommandPath()
}

// ============================================================================
// 平台兼容性检测工具函数
// ============================================================================

// isColorSupported 检查终端是否支持彩色输出
func isColorSupported() bool {
	// 1. 首先检查是否明确禁用了颜色
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}

	// 2. 检查是否在CI环境中
	if os.Getenv("CI") != "" {
		// 检查常见的CI环境
		ciEnvs := []string{
			"GITHUB_ACTIONS",
			"GITLAB_CI",
			"TRAVIS",
			"CIRCLECI",
			"JENKINS_URL",
			"TEAMCITY_VERSION",
		}
		for _, env := range ciEnvs {
			if os.Getenv(env) != "" {
				return true
			}
		}
	}

	// 3. 检查 COLORTERM 环境变量
	if os.Getenv("COLORTERM") != "" {
		return true
	}

	// 4. 检查终端类型
	term := os.Getenv("TERM")
	if term != "" {
		colorTerms := []string{
			"xterm",
			"vt100",
			"color",
			"ansi",
			"cygwin",
			"linux",
		}
		for _, cterm := range colorTerms {
			if strings.Contains(term, cterm) {
				return true
			}
		}
	}

	// 5. 平台特定检查
	if !isWindowsColorSupported() {
		return false
	}

	// 6. 检查是否是标准终端
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		return true
	}

	return false
}

// ============================================================================
// 字符串处理工具函数
// ============================================================================

// trimVersion 清理版本字符串，移除前导的'v'
func trimVersion(version string) string {
	return strings.TrimLeft(version, "v")
}

// ensurePrefix 确保字符串有指定前缀
func ensurePrefix(s, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		return s
	}
	return prefix + s
}

// ensureSuffix 确保字符串有指定后缀
func ensureSuffix(s, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		return s
	}
	return s + suffix
}

// ============================================================================
// 环境检测工具函数
// ============================================================================

// isCI 检查是否在CI环境中运行
func isCI() bool {
	ciEnvs := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"TRAVIS",
		"CIRCLECI",
		"JENKINS_URL",
		"TEAMCITY_VERSION",
		"APPVEYOR",
		"BUILDKITE",
	}

	for _, env := range ciEnvs {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

// isDevelopment 检查是否在开发环境中运行
func isDevelopment() bool {
	return os.Getenv("GO_ENV") == "development" ||
		os.Getenv("NODE_ENV") == "development" ||
		os.Getenv("ENVIRONMENT") == "development"
}

// isProduction 检查是否在生产环境中运行
func isProduction() bool {
	return os.Getenv("GO_ENV") == "production" ||
		os.Getenv("NODE_ENV") == "production" ||
		os.Getenv("ENVIRONMENT") == "production"
}
