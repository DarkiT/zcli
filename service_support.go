package zcli

import (
	"fmt"
	"math"
	"os"
	"time"

	service "github.com/darkit/daemon"
)

// ExitWithTimeout 在指定时间后强制退出程序
func (sm *sManager) ExitWithTimeout(timeout time.Duration, debugMsg string, exitCode int) {
	go func() {
		time.Sleep(timeout)
		if debugMsg != "" {
			_, _ = fmt.Fprintln(os.Stderr, debugMsg)
		}
		exitFunc(exitCode)
	}()
}

func (sm *sManager) scheduleForceExit(timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	if service.Interactive() {
		return
	}
	if sm.forceExitOnce.Swap(true) {
		return
	}
	seconds := int(math.Ceil(timeout.Seconds()))
	msg := sm.localizer.FormatError("timeout", seconds)

	sm.mu.Lock()
	if sm.forceExitTimer == nil {
		sm.forceExitTimer = time.AfterFunc(timeout, func() {
			if msg != "" {
				_, _ = fmt.Fprintln(os.Stderr, msg)
			}
			exitFunc(1)
		})
	}
	sm.mu.Unlock()
}

func (sm *sManager) cancelForceExit() {
	sm.mu.Lock()
	if sm.forceExitTimer != nil {
		sm.forceExitTimer.Stop()
		sm.forceExitTimer = nil
	}
	sm.forceExitOnce.Store(false)
	sm.mu.Unlock()
}

// checkPermissions 检查文件或目录的权限
func checkPermissions(path string, requiredPerm os.FileMode, localizer *ServiceLocalizer) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s", localizer.FormatError("pathNotExist", path))
		}
		return fmt.Errorf("%s", localizer.FormatError("getPathInfo", err))
	}

	currentPerm := fileInfo.Mode() & os.ModePerm

	// 路径类型校验：仅在显式要求目录时校验
	if requiredPerm&os.ModeDir != 0 && !fileInfo.IsDir() {
		return fmt.Errorf("%s", localizer.FormatError("needDir", path))
	}
	if requiredPerm&os.ModeDir == 0 && fileInfo.IsDir() {
		return fmt.Errorf("%s", localizer.FormatError("needFile", path))
	}

	// 分别检查读/写/执行权限，更清晰地提示缺失项
	type permCheck struct {
		bit  os.FileMode
		name string
	}
	checks := []permCheck{
		{0o400, "read"},
		{0o200, "write"},
		{0o100, "exec"},
	}
	for _, p := range checks {
		if requiredPerm&p.bit != 0 && currentPerm&p.bit == 0 {
			return fmt.Errorf("%s", localizer.FormatError("insufficientPerm",
				p.name,
				fmt.Sprintf("%o", currentPerm)))
		}
	}

	return nil
}

// AddErrorHandler 添加错误处理器
func (sm *sManager) AddErrorHandler(handler ErrorHandler) {
	if handler != nil {
		sm.errorHandlers = append(sm.errorHandlers, handler)
	}
}

// handleError 通过错误处理器链处理错误
func (sm *sManager) handleError(err error) error {
	if err == nil {
		return nil
	}
	for _, handler := range sm.errorHandlers {
		err = handler.HandleError(err)
	}
	return err
}

func (sm *sManager) wrapServiceError(err error, code ErrorCode, operation string) *ServiceError {
	return WrapServiceOperationError(err, code, operation, sm.Name())
}
