//go:build !windows

package zcli

// isWindowsColorSupported 在非 Windows 平台上始终返回 true
func isWindowsColorSupported() bool {
	return true
}
