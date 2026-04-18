package zcli

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// ShutdownReason 描述服务运行上下文被取消的原因类别。
type ShutdownReason string

const (
	ShutdownReasonSignal         ShutdownReason = "signal"
	ShutdownReasonServiceStop    ShutdownReason = "service_stop"
	ShutdownReasonExternalCancel ShutdownReason = "external_cancel"
)

// ShutdownCause 表示传递给 Run(ctx) 的统一关闭原因。
// 用户可通过 context.Cause(ctx) 或 GetShutdownCause(ctx) 获取。
type ShutdownCause struct {
	Reason ShutdownReason
	Signal os.Signal
	Cause  error
}

func (c *ShutdownCause) Error() string {
	if c == nil {
		return context.Canceled.Error()
	}

	switch c.Reason {
	case ShutdownReasonSignal:
		if c.Signal != nil {
			return fmt.Sprintf("service shutdown requested by signal %s", c.Signal)
		}
		return "service shutdown requested by signal"
	case ShutdownReasonServiceStop:
		return "service stop requested"
	case ShutdownReasonExternalCancel:
		if c.Cause != nil && !errors.Is(c.Cause, context.Canceled) {
			return fmt.Sprintf("service canceled by parent context: %v", c.Cause)
		}
		return "service canceled by parent context"
	default:
		if c.Cause != nil {
			return c.Cause.Error()
		}
		return context.Canceled.Error()
	}
}

func (c *ShutdownCause) Unwrap() error {
	if c == nil {
		return nil
	}
	return c.Cause
}

// GetShutdownCause 返回运行上下文上的结构化关闭原因。
func GetShutdownCause(ctx context.Context) (*ShutdownCause, bool) {
	if ctx == nil {
		return nil, false
	}

	var shutdownCause *ShutdownCause
	if cause := context.Cause(ctx); errors.As(cause, &shutdownCause) {
		return shutdownCause, true
	}
	return nil, false
}

func newShutdownCause(reason ShutdownReason, signal os.Signal, cause error) *ShutdownCause {
	return &ShutdownCause{
		Reason: reason,
		Signal: signal,
		Cause:  cause,
	}
}

func shutdownCauseFromContext(ctx context.Context, fallback *ShutdownCause) error {
	if ctx == nil {
		if fallback != nil {
			return fallback
		}
		return context.Canceled
	}

	cause := context.Cause(ctx)
	if cause == nil {
		cause = ctx.Err()
	}
	if cause == nil {
		if fallback != nil {
			return fallback
		}
		return context.Canceled
	}

	var shutdownCause *ShutdownCause
	if errors.As(cause, &shutdownCause) {
		return shutdownCause
	}

	if fallback == nil {
		return cause
	}

	if !errors.Is(cause, context.Canceled) && !errors.Is(cause, context.DeadlineExceeded) {
		return cause
	}

	clone := *fallback
	clone.Cause = cause
	return &clone
}
