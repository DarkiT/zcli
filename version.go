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
	Version      string      `json:"version"`
	GoVersion    string      `json:"goVersion"`
	GitCommit    string      `json:"gitCommit"`
	GitBranch    string      `json:"gitBranch"`
	GitTag       string      `json:"gitTag"`
	Platform     string      `json:"platform"`
	Architecture string      `json:"architecture"`
	Compiler     string      `json:"compiler"`
	Debug        atomic.Bool `json:"debug"`
	BuildTime    time.Time   `json:"buildTime"`
}

// SetDebug 是否开启调试模式
func (bi *VersionInfo) SetDebug(debug bool) *VersionInfo {
	bi.Debug.Store(debug)
	return bi
}

// SetVersion 设置构建版本号
func (bi *VersionInfo) SetVersion(version string) *VersionInfo {
	bi.Version = version
	return bi
}

// SetBuildTime 设置构建时间
func (bi *VersionInfo) SetBuildTime(t time.Time) *VersionInfo {
	bi.BuildTime = t
	return bi
}

// String 返回格式化的构建信息
func (bi *VersionInfo) String() string {
	// 预定义字段映射
	fields := []struct {
		name  string
		value interface{}
		cond  bool
	}{
		{"Version", bi.Version, true},
		{"Go Version", bi.GoVersion, true},
		{"Compiler", bi.Compiler, true},
		{"Platform", fmt.Sprintf("%s/%s", bi.Platform, bi.Architecture), true},
		{"Git Branch", bi.GitBranch, bi.GitBranch != ""},
		{"Git Tag", bi.GitTag, bi.GitTag != ""},
		{"Git Commit", bi.GitCommit, bi.GitCommit != ""},
		{"Build Mode", map[bool]string{true: "Debug", false: "Release"}[bi.Debug.Load()], true},
		{"Build Time", bi.BuildTime.Format(time.DateTime), !bi.BuildTime.IsZero()},
	}

	// 预分配 builder
	var b strings.Builder
	b.Grow(256)

	// 统一格式化输出
	for _, f := range fields {
		if f.cond {
			_, _ = fmt.Fprintf(&b, "%-15s %v%s", f.name+":", f.value, separator)
		}
	}

	return b.String()
}

// NewVersionInfo 创建构建信息
func NewVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:      "1.0.0",
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		Compiler:     runtime.Compiler,
	}
}
