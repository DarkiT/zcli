package zcli

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// =============================================================================
// 便利性 API - ServiceLocalizer
// =============================================================================

// ServiceLocalizer 服务本地化器，提供便利的 API
type ServiceLocalizer struct {
	manager       *LanguageManager
	colors        *colors
	out           io.Writer
	err           io.Writer
	silenceErrors bool
	silenceUsage  bool
	colorOut      io.Writer
	colorErr      io.Writer
}

// NewServiceLocalizer 创建服务本地化器
func NewServiceLocalizer(manager *LanguageManager, colors *colors) *ServiceLocalizer {
	return &ServiceLocalizer{
		manager:  manager,
		colors:   colors,
		out:      os.Stdout,
		err:      os.Stderr,
		colorOut: color.Output,
		colorErr: color.Error,
	}
}

func (sl *ServiceLocalizer) ConfigureOutput(out, err io.Writer, silenceErrors, silenceUsage bool) {
	if out != nil {
		sl.out = out
		sl.colorOut = out
	}
	if err != nil {
		sl.err = err
		sl.colorErr = err
	}
	sl.silenceErrors = silenceErrors
	sl.silenceUsage = silenceUsage
}

// GetOperation 获取操作文本
func (sl *ServiceLocalizer) GetOperation(operation string) string {
	path := fmt.Sprintf("service.operations.%s", operation)
	return sl.manager.GetText(path)
}

// GetStatus 获取状态文本
func (sl *ServiceLocalizer) GetStatus(status string) string {
	path := fmt.Sprintf("service.status.%s", status)
	return sl.manager.GetText(path)
}

// GetError 获取错误文本
func (sl *ServiceLocalizer) GetError(errorType string) string {
	path := fmt.Sprintf("error.service.%s", errorType)
	if text := sl.manager.GetText(path); text != "" {
		return text
	}
	path = fmt.Sprintf("error.system.%s", errorType)
	return sl.manager.GetText(path)
}

// GetFormat 获取格式化模板
func (sl *ServiceLocalizer) GetFormat(formatType string) string {
	path := fmt.Sprintf("format.%s", formatType)
	return sl.manager.GetText(path)
}

// LogError 记录错误日志
func (sl *ServiceLocalizer) LogError(errorType string, err error) {
	if sl.silenceErrors {
		return
	}
	message := sl.GetError(errorType)
	if sl.colors != nil {
		out := color.Output
		color.Output = sl.colorErr
		_, _ = sl.colors.Error.Printf("%s: %v\n", message, err)
		color.Output = out
	} else {
		_, _ = fmt.Fprintf(sl.err, "Error: %s: %v\n", message, err)
	}
}

// LogWarning 记录警告日志
func (sl *ServiceLocalizer) LogWarning(message string, args ...interface{}) {
	text := fmt.Sprintf(message, args...)
	if sl.colors != nil {
		out := color.Output
		color.Output = sl.colorErr
		_, _ = sl.colors.Warning.Println(text)
		color.Output = out
	} else {
		_, _ = fmt.Fprintf(sl.err, "Warning: %s\n", text)
	}
}

// LogSuccess 记录成功日志
func (sl *ServiceLocalizer) LogSuccess(serviceName, operation string) {
	format := sl.GetFormat("serviceStatus")
	successText := sl.GetStatus("success")
	if sl.colors != nil {
		out := color.Output
		color.Output = sl.colorOut
		_, _ = sl.colors.Success.Printf(format+"\n", serviceName, successText)
		color.Output = out
	} else {
		_, _ = fmt.Fprintf(sl.out, format+"\n", serviceName, successText)
	}
}

// LogInfo 记录信息日志
func (sl *ServiceLocalizer) LogInfo(serviceName, status string) {
	format := sl.GetFormat("serviceStatus")
	statusText := sl.GetStatus(status)
	if sl.colors != nil {
		out := color.Output
		color.Output = sl.colorOut
		_, _ = sl.colors.Info.Printf(format+"\n", serviceName, statusText)
		color.Output = out
	} else {
		_, _ = fmt.Fprintf(sl.out, format+"\n", serviceName, statusText)
	}
}

// FormatError 格式化错误消息
func (sl *ServiceLocalizer) FormatError(errorType string, args ...interface{}) string {
	template := sl.GetError(errorType)
	if len(args) > 0 {
		return fmt.Sprintf(template, args...)
	}
	return template
}

// FormatServiceStatus 格式化服务状态
func (sl *ServiceLocalizer) FormatServiceStatus(serviceName, status string) string {
	format := sl.GetFormat("serviceStatus")
	statusText := sl.GetStatus(status)
	return fmt.Sprintf(format, serviceName, statusText)
}
