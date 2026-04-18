//go:build windows

package zcli

import (
	"os"
	"syscall"
)

// isWindowsColorSupported 检查 Windows 是否支持彩色输出
func isWindowsColorSupported() bool {
	// Windows 终端检查
	windowsTerms := map[string]string{
		"WT_SESSION":     "",       // Windows Terminal
		"TERM_PROGRAM":   "vscode", // VS Code terminal
		"ConEmuANSI":     "ON",     // ConEmu
		"ANSICON":        "",       // ANSICON
		"ALACRITTY_LOG":  "",       // Alacritty
		"MINTTY_VERSION": "",       // MinTTY
	}

	for env, value := range windowsTerms {
		if envValue := os.Getenv(env); envValue != "" {
			if value == "" || envValue == value {
				return true
			}
		}
	}

	// 检查 Windows 10 build 14931 及更高版本
	h, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return false // 无法加载 DLL，假设不支持
	}
	defer h.Release()

	proc, err := h.FindProc("GetVersion")
	if err != nil {
		return false // 无法找到函数，假设不支持
	}

	v, _, _ := proc.Call()
	version := uint32(v)

	major := version & 0xFF
	build := (version >> 16) & 0xFFFF
	return major >= 10 && build >= 14931
}
