package zcli

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

// VersionInfo 构建信息
type VersionInfo struct {
	Version      string      `json:"version"`      // 版本号
	GoVersion    string      `json:"goVersion"`    // Go版本
	GitCommit    string      `json:"gitCommit"`    // Git提交哈希
	GitBranch    string      `json:"gitBranch"`    // Git分支
	GitTag       string      `json:"gitTag"`       // Git标签
	Platform     string      `json:"platform"`     // 运行平台
	Architecture string      `json:"architecture"` // 系统架构
	Compiler     string      `json:"compiler"`     // 编译器
	Debug        atomic.Bool `json:"debug"`        // 调试模式
	BuildTime    time.Time   `json:"buildTime"`    // 构建时间
}

// NewVersion 创建版本信息
func NewVersion() *VersionInfo {
	return &VersionInfo{
		Version:      "1.0.0",
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Compiler:     runtime.Compiler,
		BuildTime:    time.Now(),
	}
}

// String 返回格式化的构建信息
func (vi *VersionInfo) String() string {
	fields := []struct {
		name  string
		value interface{}
		cond  bool
	}{
		{"Version", vi.Version, true},
		{"Go Version", vi.GoVersion, true},
		{"Compiler", vi.Compiler, true},
		{"Platform", fmt.Sprintf("%s/%s", vi.Platform, vi.Architecture), true},
		{"Git Branch", vi.GitBranch, vi.GitBranch != ""},
		{"Git Tag", vi.GitTag, vi.GitTag != ""},
		{"Git Commit", vi.GitCommit, vi.GitCommit != ""},
		{"Build Mode", map[bool]string{true: "Debug", false: "Release"}[vi.Debug.Load()], true},
		{"Build Time", vi.BuildTime.Format(time.DateTime), !vi.BuildTime.IsZero()},
	}

	var b strings.Builder
	b.Grow(256)

	for _, f := range fields {
		if f.cond {
			_, _ = fmt.Fprintf(&b, "%-15s %v%s", f.name+":", f.value, separator)
		}
	}

	return b.String()
}
