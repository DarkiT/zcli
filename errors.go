// Package zcli 提供命令行工具构建功能，包括增强的错误处理机制。
// 本包实现了结构化错误处理、错误聚合和详细的错误追踪功能。
package zcli

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// =============================================================================
// 增强的错误处理机制
// =============================================================================

// ErrorCode 错误代码类型
type ErrorCode string

const (
	// 配置相关错误
	ErrConfigValidation ErrorCode = "CONFIG_VALIDATION"
	ErrConfigMissing    ErrorCode = "CONFIG_MISSING"
	ErrConfigInvalid    ErrorCode = "CONFIG_INVALID"

	// 服务相关错误
	ErrServiceCreate   ErrorCode = "SERVICE_CREATE"
	ErrServiceStart    ErrorCode = "SERVICE_START"
	ErrServiceStop     ErrorCode = "SERVICE_STOP"
	ErrServiceRestart  ErrorCode = "SERVICE_RESTART"
	ErrServiceNotFound ErrorCode = "SERVICE_NOT_FOUND"
	ErrServiceRunning  ErrorCode = "SERVICE_ALREADY_RUNNING"
	ErrServiceStopped  ErrorCode = "SERVICE_ALREADY_STOPPED"
	ErrServiceTimeout  ErrorCode = "SERVICE_TIMEOUT"

	// 系统相关错误
	ErrPermission        ErrorCode = "PERMISSION_DENIED"
	ErrPathNotFound      ErrorCode = "PATH_NOT_FOUND"
	ErrPathInvalid       ErrorCode = "PATH_INVALID"
	ErrExecutableInvalid ErrorCode = "EXECUTABLE_INVALID"

	// 运行时错误
	ErrRuntime          ErrorCode = "RUNTIME_ERROR"
	ErrContextCancelled ErrorCode = "CONTEXT_CANCELLED"
	ErrTimeout          ErrorCode = "TIMEOUT"

	// 网络和通信错误
	ErrNetwork    ErrorCode = "NETWORK_ERROR"
	ErrConnection ErrorCode = "CONNECTION_ERROR"
)

// ServiceError 增强的服务错误类型
type ServiceError struct {
	Code      ErrorCode      `json:"code"`
	Operation string         `json:"operation"`
	Service   string         `json:"service"`
	Message   string         `json:"message"`
	Cause     error          `json:"cause,omitempty"`
	Context   map[string]any `json:"context,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Stack     []string       `json:"stack,omitempty"`
}

// NewServiceError 创建新的服务错误
func NewServiceError(code ErrorCode, operation, service, message string) *ServiceError {
	return &ServiceError{
		Code:      code,
		Operation: operation,
		Service:   service,
		Message:   message,
		Context:   make(map[string]any),
		Timestamp: time.Now(),
	}
}

// Error 实现error接口
func (se *ServiceError) Error() string {
	if se.Service != "" {
		return fmt.Sprintf("[%s] 服务 %s %s 失败: %s", se.Code, se.Service, se.Operation, se.Message)
	}
	return fmt.Sprintf("[%s] %s 失败: %s", se.Code, se.Operation, se.Message)
}

// WithCause 添加原因错误
func (se *ServiceError) WithCause(cause error) *ServiceError {
	se.Cause = cause
	return se
}

// WithContext 添加上下文信息
func (se *ServiceError) WithContext(key string, value any) *ServiceError {
	if se.Context == nil {
		se.Context = make(map[string]any)
	}
	se.Context[key] = value
	return se
}

// WithStack 添加堆栈信息
func (se *ServiceError) WithStack(stack []string) *ServiceError {
	se.Stack = stack
	return se
}

// Unwrap 展开错误链
func (se *ServiceError) Unwrap() error {
	return se.Cause
}

// Is 检查错误类型
func (se *ServiceError) Is(target error) bool {
	if t, ok := target.(*ServiceError); ok {
		return se.Code == t.Code
	}
	return false
}

// GetCode 获取错误代码
func (se *ServiceError) GetCode() ErrorCode {
	return se.Code
}

// GetOperation 获取操作名称
func (se *ServiceError) GetOperation() string {
	return se.Operation
}

// GetService 获取服务名称
func (se *ServiceError) GetService() string {
	return se.Service
}

// GetContext 获取上下文信息
func (se *ServiceError) GetContext() map[string]any {
	if se.Context == nil {
		return make(map[string]any)
	}
	return se.Context
}

// ToJSON 转换为JSON格式
func (se *ServiceError) ToJSON() map[string]any {
	result := map[string]any{
		"code":      se.Code,
		"operation": se.Operation,
		"service":   se.Service,
		"message":   se.Message,
		"timestamp": se.Timestamp.Format(time.RFC3339),
	}

	if se.Cause != nil {
		result["cause"] = se.Cause.Error()
	}

	if len(se.Context) > 0 {
		result["context"] = se.Context
	}

	if len(se.Stack) > 0 {
		result["stack"] = se.Stack
	}

	return result
}

// =============================================================================
// 错误构建器
// =============================================================================

// ErrorBuilder 错误构建器
type ErrorBuilder struct {
	err *ServiceError
}

// NewError 创建新的错误构建器
func NewError(code ErrorCode) *ErrorBuilder {
	return &ErrorBuilder{
		err: &ServiceError{
			Code:      code,
			Context:   make(map[string]any),
			Timestamp: time.Now(),
		},
	}
}

// Operation 设置操作名称
func (eb *ErrorBuilder) Operation(operation string) *ErrorBuilder {
	eb.err.Operation = operation
	return eb
}

// Service 设置服务名称
func (eb *ErrorBuilder) Service(service string) *ErrorBuilder {
	eb.err.Service = service
	return eb
}

// Message 设置错误消息
func (eb *ErrorBuilder) Message(message string) *ErrorBuilder {
	eb.err.Message = message
	return eb
}

// Messagef 设置格式化错误消息
func (eb *ErrorBuilder) Messagef(format string, args ...any) *ErrorBuilder {
	eb.err.Message = fmt.Sprintf(format, args...)
	return eb
}

// Cause 设置原因错误
func (eb *ErrorBuilder) Cause(cause error) *ErrorBuilder {
	eb.err.Cause = cause
	return eb
}

// Context 添加上下文信息
func (eb *ErrorBuilder) Context(key string, value any) *ErrorBuilder {
	eb.err.Context[key] = value
	return eb
}

// Stack 添加堆栈信息
func (eb *ErrorBuilder) Stack(stack []string) *ErrorBuilder {
	eb.err.Stack = stack
	return eb
}

// Build 构建错误
func (eb *ErrorBuilder) Build() *ServiceError {
	return eb.err
}

// =============================================================================
// 预定义错误函数
// =============================================================================

// ErrServiceAlreadyRunning 服务已在运行错误
func ErrServiceAlreadyRunning(service string) *ServiceError {
	return NewError(ErrServiceRunning).
		Service(service).
		Operation("start").
		Message("服务已在运行中").
		Build()
}

// ErrServiceAlreadyStopped 服务已停止错误
func ErrServiceAlreadyStopped(service string) *ServiceError {
	return NewError(ErrServiceStopped).
		Service(service).
		Operation("stop").
		Message("服务已停止").
		Build()
}

// ErrServiceNotInstalled 服务未安装错误
func ErrServiceNotInstalled(service string) *ServiceError {
	return NewError(ErrServiceNotFound).
		Service(service).
		Operation("status").
		Message("服务未安装").
		Build()
}

// ErrServiceStartTimeout 服务启动超时错误
func ErrServiceStartTimeout(service string, timeout time.Duration) *ServiceError {
	return NewError(ErrServiceTimeout).
		Service(service).
		Operation("start").
		Messagef("服务启动超时（%v）", timeout).
		Context("timeout", timeout.String()).
		Build()
}

// ErrServiceStopTimeout 服务停止超时错误
func ErrServiceStopTimeout(service string, timeout time.Duration) *ServiceError {
	return NewError(ErrServiceTimeout).
		Service(service).
		Operation("stop").
		Messagef("服务停止超时（%v）", timeout).
		Context("timeout", timeout.String()).
		Build()
}

// ErrConfigValidationFailed 配置验证失败错误
func ErrConfigValidationFailed(details []error) *ServiceError {
	var messages []string
	for _, err := range details {
		messages = append(messages, err.Error())
	}

	return NewError(ErrConfigValidation).
		Operation("validate").
		Message("配置验证失败").
		Context("details", messages).
		Context("count", len(details)).
		Build()
}

// ErrPermissionDenied 权限被拒绝错误
func ErrPermissionDenied(path string, required, current string) *ServiceError {
	return NewError(ErrPermission).
		Operation("permission_check").
		Messagef("权限不足，路径: %s", path).
		Context("path", path).
		Context("required", required).
		Context("current", current).
		Build()
}

// ErrPathNotExists 路径不存在错误
func ErrPathNotExists(path string) *ServiceError {
	return NewError(ErrPathNotFound).
		Operation("path_check").
		Messagef("路径不存在: %s", path).
		Context("path", path).
		Build()
}

// =============================================================================
// 错误处理中间件
// =============================================================================

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(err error) error
}

// LoggingErrorHandler 日志记录错误处理器
type LoggingErrorHandler struct {
	logger Logger
}

// Logger 日志接口
type Logger interface {
	Error(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Info(msg string, fields ...any)
}

// NewLoggingErrorHandler 创建日志错误处理器
func NewLoggingErrorHandler(logger Logger) *LoggingErrorHandler {
	return &LoggingErrorHandler{logger: logger}
}

// HandleError 处理错误
func (leh *LoggingErrorHandler) HandleError(err error) error {
	if serviceErr, ok := err.(*ServiceError); ok {
		leh.logger.Error("服务错误",
			"code", serviceErr.Code,
			"service", serviceErr.Service,
			"operation", serviceErr.Operation,
			"message", serviceErr.Message,
			"context", serviceErr.Context,
		)
	} else {
		leh.logger.Error("未知错误", "error", err.Error())
	}
	return err
}

// RecoveryErrorHandler 恢复错误处理器
type RecoveryErrorHandler struct {
	retryCount int
	retryDelay time.Duration
}

// NewRecoveryErrorHandler 创建恢复错误处理器
func NewRecoveryErrorHandler(retryCount int, retryDelay time.Duration) *RecoveryErrorHandler {
	return &RecoveryErrorHandler{
		retryCount: retryCount,
		retryDelay: retryDelay,
	}
}

// HandleError 处理错误并尝试恢复
func (reh *RecoveryErrorHandler) HandleError(err error) error {
	if serviceErr, ok := err.(*ServiceError); ok {
		// 根据错误类型决定是否可以恢复
		switch serviceErr.Code {
		case ErrServiceTimeout, ErrNetwork, ErrConnection:
			// 可以尝试恢复的错误
			return reh.retryOperation(serviceErr)
		default:
			// 不可恢复的错误
			return err
		}
	}
	return err
}

// retryOperation 重试操作
func (reh *RecoveryErrorHandler) retryOperation(err *ServiceError) error {
	for i := 0; i < reh.retryCount; i++ {
		time.Sleep(reh.retryDelay)
		// 这里应该重新执行失败的操作
		// 由于这是示例，我们只是简单地返回原错误
	}

	// 重试失败后，添加重试信息到错误中
	return err.WithContext("retry_count", reh.retryCount).
		WithContext("retry_delay", reh.retryDelay.String())
}

// =============================================================================
// 错误聚合器
// =============================================================================

// ErrorAggregator 错误聚合器
type ErrorAggregator struct {
	errors []error
}

// NewErrorAggregator 创建错误聚合器
func NewErrorAggregator() *ErrorAggregator {
	return &ErrorAggregator{}
}

// Add 添加错误
func (ea *ErrorAggregator) Add(err error) {
	if err != nil {
		ea.errors = append(ea.errors, err)
	}
}

// HasErrors 检查是否有错误
func (ea *ErrorAggregator) HasErrors() bool {
	return len(ea.errors) > 0
}

// Count 返回错误数量
func (ea *ErrorAggregator) Count() int {
	return len(ea.errors)
}

// Errors 返回所有错误
func (ea *ErrorAggregator) Errors() []error {
	return ea.errors
}

// Error 返回聚合错误信息
func (ea *ErrorAggregator) Error() string {
	if len(ea.errors) == 0 {
		return ""
	}

	if len(ea.errors) == 1 {
		return ea.errors[0].Error()
	}

	var messages []string
	for i, err := range ea.errors {
		messages = append(messages, fmt.Sprintf("%d. %s", i+1, err.Error()))
	}

	return fmt.Sprintf("发生%d个错误:\n%s", len(ea.errors), strings.Join(messages, "\n"))
}

// Clear 清除所有错误
func (ea *ErrorAggregator) Clear() {
	ea.errors = nil
}

// =============================================================================
// 错误工具函数
// =============================================================================

// IsServiceError 检查是否为服务错误
func IsServiceError(err error) bool {
	_, ok := err.(*ServiceError)
	return ok
}

// GetServiceError 获取服务错误
func GetServiceError(err error) (*ServiceError, bool) {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr, true
	}
	return nil, false
}

// IsErrorCode 检查错误代码
func IsErrorCode(err error, code ErrorCode) bool {
	if serviceErr, ok := GetServiceError(err); ok {
		return serviceErr.Code == code
	}
	return false
}

// WrapError 包装普通错误为服务错误
func WrapError(err error, code ErrorCode, operation string) *ServiceError {
	return NewError(code).
		Operation(operation).
		Message(err.Error()).
		Cause(err).
		Build()
}

// CombineErrors 合并多个错误
func CombineErrors(errs ...error) error {
	aggregator := NewErrorAggregator()
	for _, err := range errs {
		aggregator.Add(err)
	}

	if !aggregator.HasErrors() {
		return nil
	}

	if aggregator.Count() == 1 {
		return aggregator.Errors()[0]
	}

	return errors.New(aggregator.Error())
}
