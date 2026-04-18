package zcli

import (
	"context"
	"fmt"
	"time"

	service "github.com/darkit/daemon"
	"github.com/spf13/cobra"
)

func (sm *sManager) startManagedRun() error {
	sm.mu.Lock()
	if sm.runnerDone != nil {
		select {
		case <-sm.runnerDone:
			sm.runnerDone = nil
			sm.runnerErr = nil
		default:
			sm.mu.Unlock()
			return nil
		}
	}

	done := make(chan struct{})
	errCh := make(chan error, 1)
	sm.runnerDone = done
	sm.runnerErr = errCh
	sm.mu.Unlock()

	go func() {
		defer close(done)
		// daemon StartFunc receives a startup-scoped context that is cancelled
		// as soon as Start returns, so the long-lived service runtime must not
		// inherit it.
		if err := sm.Run(nil); err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	return nil
}

func (sm *sManager) stopManagedRun(ctx context.Context) error {
	stopErr := sm.stopWithCause(
		shutdownCauseFromContext(ctx, newShutdownCause(ShutdownReasonServiceStop, nil, nil)),
		true,
	)
	runErr := sm.waitManagedRun(ctx)
	return CombineErrors(stopErr, runErr)
}

func (sm *sManager) waitManagedRun(ctx context.Context) error {
	sm.mu.RLock()
	done := sm.runnerDone
	errCh := sm.runnerErr
	sm.mu.RUnlock()

	if done == nil {
		return nil
	}

	if ctx == nil {
		<-done
	} else {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	var runErr error
	if errCh != nil {
		select {
		case runErr = <-errCh:
		default:
		}
	}
	if runErr != nil {
		runErr = sm.handleError(runErr)
	}

	sm.mu.Lock()
	if sm.runnerDone == done {
		sm.runnerDone = nil
		sm.runnerErr = nil
	}
	sm.mu.Unlock()

	return runErr
}

// Run 实现 ServiceRunner 接口。
// 它始终保留稳定的命令级上下文，并为用户服务逻辑派生单独的运行上下文。
// 传入的 externalCtx 仅用于显式的运行期取消，不会替代命令级上下文。
func (sm *sManager) Run(externalCtx context.Context) error {
	sm.stopMu.Lock()
	sm.mu.Lock()
	session := sm.ensureCommandSessionLocked()
	runCtx, runCancel := context.WithCancelCause(session.commandCtx)
	session.serviceCtx = runCtx
	session.serviceCancel = runCancel
	// 退出通道每次运行前重置
	select {
	case <-sm.exitChan:
		sm.exitChan = make(chan struct{})
	default:
	}
	sm.stopExecuted.Store(false)
	sm.stopFuncOnce.Store(false)
	sm.running.Store(true)
	sm.mu.Unlock()
	sm.stopMu.Unlock()
	defer runCancel(nil)
	defer sm.clearServiceContext(session)

	configCtx := sm.commands.config.Context()
	if configCtx != nil && configCtx != session.commandCtx {
		go func() {
			select {
			case <-configCtx.Done():
				runCancel(shutdownCauseFromContext(
					configCtx,
					newShutdownCause(ShutdownReasonExternalCancel, nil, nil),
				))
			case <-runCtx.Done():
			}
		}()
	}

	if externalCtx != nil {
		go func() {
			select {
			case <-externalCtx.Done():
				runCancel(shutdownCauseFromContext(
					externalCtx,
					newShutdownCause(ShutdownReasonExternalCancel, nil, nil),
				))
			case <-runCtx.Done():
			}
		}()
	}

	defer sm.running.Store(false)
	defer sm.cancelForceExit()

	if sm.commands.config.runtime.Run != nil {
		if err := sm.commands.config.runtime.Run(runCtx); err != nil {
			if session.commandCancel != nil {
				session.commandCancel(err)
			}
			return err
		}
	}

	// 等待退出信号或上下文取消
	select {
	case <-sm.exitChan:
	case <-runCtx.Done():
	}

	// 交互模式下补偿停止
	if service.Interactive() && !sm.stopExecuted.Load() {
		return sm.stopWithCause(
			shutdownCauseFromContext(runCtx, newShutdownCause(ShutdownReasonServiceStop, nil, nil)),
			true,
		)
	}
	return nil
}

// Stop 实现 ServiceRunner 接口
func (sm *sManager) Stop() error {
	return sm.stopWithCause(newShutdownCause(ShutdownReasonServiceStop, nil, nil), true)
}

func (sm *sManager) stopWithCause(cause error, callUserStop bool) error {
	sm.stopMu.Lock()
	if sm.stopExecuted.Load() {
		sm.stopMu.Unlock()
		return nil
	}
	sm.stopExecuted.Store(true)

	var serviceCancel context.CancelCauseFunc
	var commandCancel context.CancelCauseFunc
	if sm.session != nil {
		serviceCancel = sm.session.serviceCancel
		commandCancel = sm.session.commandCancel
	}
	sm.stopMu.Unlock()

	sm.scheduleForceExit(sm.commands.config.runtime.StopTimeout)
	if serviceCancel != nil {
		serviceCancel(cause)
	}
	if commandCancel != nil {
		commandCancel(cause)
	}

	sm.mu.Lock()
	select {
	case <-sm.exitChan:
	default:
		close(sm.exitChan)
	}
	sm.mu.Unlock()

	var stopErr error
	if callUserStop && !sm.stopFuncOnce.Swap(true) {
		if sm.commands.config.runtime.Stop != nil {
			if err := sm.commands.config.runtime.Stop(); err != nil {
				stopErr = err
			}
		}
	}

	sm.running.Store(false)
	return stopErr
}

// executeRunCommand 执行运行命令，支持前台和服务模式
func (sm *sManager) executeRunCommand(_ *cobra.Command, args []string) error {
	// 如果服务正在运行，显示警告并退出
	if sm.running.Load() {
		sm.localizer.LogError("alreadyRunning", nil)
		return nil
	}

	// 处理运行参数
	serviceArgs := args
	if len(serviceArgs) == 0 && len(sm.commands.config.service.Arguments) > 0 {
		serviceArgs = sm.commands.config.service.Arguments
	}

	sm.mu.Lock()
	session := sm.newCommandSessionLocked()

	// 如果有参数变化，重新创建服务实例
	if len(serviceArgs) > 0 {
		sm.config.Arguments = serviceArgs
		svc, err := service.New(sm.buildRunner(), sm.config)
		if err != nil {
			// 避免泄漏新创建的上下文
			if session.commandCancel != nil {
				session.commandCancel(err)
			}
			sm.mu.Unlock()
			sm.localizer.LogError("createService", err)
			return WrapError(err, ErrServiceCreate, "run")
		}
		sm.service = svc
	}
	sm.mu.Unlock()

	// 重置状态
	sm.stopExecuted.Store(false)

	// 创建监控通道
	runDone := make(chan struct{})
	runErrCh := make(chan error, 1)

	// 在goroutine中运行服务
	go func() {
		defer close(runDone)
		defer close(runErrCh)
		defer func() {
			if r := recover(); r != nil {
				runErrCh <- fmt.Errorf("panic: %v", r)
			}
		}()

		// 启动服务 - 这会调用用户的Run函数
		if err := sm.service.Run(); err != nil {
			runErrCh <- err
		}
	}()

	// 等待服务完成，支持前台/服务模式区分
	return sm.waitForServiceCompletion(session.commandCtx, runDone, runErrCh)
}

// waitForServiceCompletion 等待服务完成，支持交互式和服务模式
func (sm *sManager) waitForServiceCompletion(commandCtx context.Context, runDone chan struct{}, runErrChs ...<-chan error) error {
	var runErrCh <-chan error
	if len(runErrChs) > 0 {
		runErrCh = runErrChs[0]
	}
	popRunErr := func() error {
		if runErrCh == nil {
			return nil
		}
		select {
		case err := <-runErrCh:
			return sm.handleError(err)
		default:
			return nil
		}
	}

	if commandCtx == nil {
		commandCtx = context.Background()
	}

	select {
	case <-runDone:
		// 服务正常退出
		return popRunErr()

	case <-commandCtx.Done():
		// 收到取消信号，尝试优雅停止

		// 如果是交互式模式，安全地关闭退出通道
		sm.mu.Lock()
		select {
		case <-sm.exitChan:
			// 通道已关闭，不需要操作
		default:
			// 安全地关闭退出通道，通知服务停止
			close(sm.exitChan)
		}
		sm.mu.Unlock()

		// 使用超时机制等待服务退出
		initialWait := sm.commands.config.runtime.ShutdownInitial
		if initialWait <= 0 {
			initialWait = 3 * time.Second
		}
		graceWait := sm.commands.config.runtime.ShutdownGrace
		if graceWait <= 0 {
			graceWait = 2 * time.Second
		}

		select {
		case <-runDone:
			// 服务响应信号成功退出
			return popRunErr()

		case <-time.After(initialWait):
			// 超时后优先调用 Stop，并在宽限期内等待退出
			sm.localizer.LogWarning("%s", sm.localizer.GetError("timeoutWarning"))
			if !sm.stopExecuted.Load() {
				_ = sm.stopWithCause(
					shutdownCauseFromContext(
						commandCtx,
						newShutdownCause(ShutdownReasonExternalCancel, nil, nil),
					),
					true,
				)
			}

			// 再等待用户停止逻辑和额外宽限时间
			select {
			case <-runDone:
				// 在额外调用stop后成功退出
				return popRunErr()

			case <-time.After(graceWait):
				// 若用户 Stop 耗时超出宽限，仅记录警告，并按需触发强制退出
				sm.localizer.LogWarning("%s", sm.localizer.GetError("forceTerminate"))
				sm.scheduleForceExit(sm.commands.config.runtime.StopTimeout)
				return nil
			}
		}
	}
}

// callStopFunctions 调用停止函数
func (sm *sManager) callStopFunctions() {
	if err := sm.stopWithCause(newShutdownCause(ShutdownReasonServiceStop, nil, nil), true); err != nil {
		sm.localizer.LogError("stopFailed", err)
	}
}
