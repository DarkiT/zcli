package zcli

import (
	"fmt"
	"runtime"
	"runtime/debug"
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
	vi := &VersionInfo{
		Version:      "1.0.0",
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Compiler:     runtime.Compiler,
		BuildTime:    time.Time{},
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.GoVersion != "" {
			vi.GoVersion = info.GoVersion
		}
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			vi.Version = info.Main.Version
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if s.Value != "" {
					vi.GitCommit = s.Value
				}
			case "vcs.time":
				if t, err := time.Parse(time.RFC3339, s.Value); err == nil {
					vi.BuildTime = t
				}
			}
		}
	}

	return vi
}

// String 返回格式化的构建信息
func (vi *VersionInfo) String() string {
	version := vi.Version
	if strings.Contains(vi.Version, "v") {
		version = strings.TrimLeft(vi.Version, "v")
	}
	fields := []struct {
		name  string
		value interface{}
		cond  bool
	}{
		// 核心标识（用户最关心）
		{"Version", fmt.Sprintf("%s", version), true},
		{"Build Time", vi.BuildTime.Format(time.RFC3339), !vi.BuildTime.IsZero()},
		{"Build Mode", map[bool]string{true: "Debug", false: "Release"}[vi.Debug.Load()], true},

		// Git 信息（追溯性）
		{"Git Tag", vi.GitTag, vi.GitTag != ""},
		{"Git Branch", vi.GitBranch, vi.GitBranch != ""},
		{"Git Commit", vi.GitCommit, vi.GitCommit != ""},

		// 运行环境（技术细节）
		{"Go Version", vi.GoVersion, true},
		{"Platform", fmt.Sprintf("%s/%s", vi.Platform, vi.Architecture), true},
		{"Compiler", vi.Compiler, true},
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
