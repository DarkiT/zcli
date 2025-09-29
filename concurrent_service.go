package zcli

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// 优化的并发安全服务管理器
// =============================================================================

// ServiceState 服务状态枚举
type ServiceState int32

const (
	StateStopped ServiceState = iota
	StateStarting
	StateRunning
	StateStopping
	StateError
)

// String 返回状态的字符串表示
func (s ServiceState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// ConcurrentServiceManager 并发安全的服务管理器
type ConcurrentServiceManager struct {
	// 原子操作字段
	state      atomic.Int32 // ServiceState
	startCount atomic.Int64 // 启动次数
	stopCount  atomic.Int64 // 停止次数

	// 保护字段的互斥锁
	mu sync.RWMutex

	// 服务相关
	runner    ServiceRunner
	config    ServiceConfig
	lifecycle ServiceLifecycle

	// 上下文管理
	ctx        context.Context
	cancel     context.CancelFunc
	cancelOnce sync.Once

	// 通道管理
	stopChan     chan struct{}
	stopChanOnce sync.Once

	// 错误处理
	lastError   error
	errorLogger func(error)

	// 超时设置
	startTimeout time.Duration
	stopTimeout  time.Duration

	// 状态监听器
	stateListeners []func(ServiceState, ServiceState)
	listenerMu     sync.RWMutex
}

// NewConcurrentServiceManager 创建并发安全的服务管理器
func NewConcurrentServiceManager(runner ServiceRunner, config ServiceConfig) *ConcurrentServiceManager {
	ctx, cancel := context.WithCancel(context.Background())

	csm := &ConcurrentServiceManager{
		runner:       runner,
		config:       config,
		ctx:          ctx,
		cancel:       cancel,
		stopChan:     make(chan struct{}),
		startTimeout: 30 * time.Second,
		stopTimeout:  10 * time.Second,
	}

	// 初始状态为停止
	csm.state.Store(int32(StateStopped))

	return csm
}

// =============================================================================
// 状态管理方法
// =============================================================================

// GetState 获取当前状态
func (csm *ConcurrentServiceManager) GetState() ServiceState {
	return ServiceState(csm.state.Load())
}

// setState 安全地设置状态并通知监听器
func (csm *ConcurrentServiceManager) setState(newState ServiceState) {
	oldState := ServiceState(csm.state.Swap(int32(newState)))

	// 通知状态监听器
	if oldState != newState {
		csm.notifyStateChange(oldState, newState)
	}
}

// compareAndSwapState 原子性地比较并交换状态
func (csm *ConcurrentServiceManager) compareAndSwapState(oldState, newState ServiceState) bool {
	return csm.state.CompareAndSwap(int32(oldState), int32(newState))
}

// IsRunning 检查服务是否正在运行
func (csm *ConcurrentServiceManager) IsRunning() bool {
	state := csm.GetState()
	return state == StateRunning || state == StateStarting
}

// IsStopped 检查服务是否已停止
func (csm *ConcurrentServiceManager) IsStopped() bool {
	return csm.GetState() == StateStopped
}

// =============================================================================
// 服务生命周期管理
// =============================================================================

// Start 启动服务
func (csm *ConcurrentServiceManager) Start() error {
	// 尝试将状态从停止改为启动中
	if !csm.compareAndSwapState(StateStopped, StateStarting) {
		currentState := csm.GetState()
		switch currentState {
		case StateRunning:
			return ErrServiceAlreadyRunning(csm.config.Name)
		case StateStarting:
			return NewError(ErrServiceStart).
				Service(csm.config.Name).
				Operation("start").
				Message("服务正在启动中").
				Build()
		case StateStopping:
			return NewError(ErrServiceStart).
				Service(csm.config.Name).
				Operation("start").
				Message("服务正在停止中，请等待停止完成").
				Build()
		default:
			return NewError(ErrServiceStart).
				Service(csm.config.Name).
				Operation("start").
				Messagef("无法从状态 %s 启动服务", currentState).
				Build()
		}
	}

	// 增加启动计数
	csm.startCount.Add(1)

	// 使用超时上下文
	startCtx := csm.ctx
	if csm.startTimeout > 0 {
		var cancel context.CancelFunc
		startCtx, cancel = context.WithTimeout(csm.ctx, csm.startTimeout)
		defer cancel()
	}

	// 执行启动前生命周期
	if csm.lifecycle != nil {
		if err := csm.lifecycle.BeforeStart(); err != nil {
			csm.setState(StateError)
			csm.setLastError(err)
			return NewError(ErrServiceStart).
				Service(csm.config.Name).
				Operation("beforeStart").
				Message("启动前处理失败").
				Cause(err).
				Build()
		}
	}

	// 启动服务
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- NewError(ErrRuntime).
					Service(csm.config.Name).
					Operation("run").
					Messagef("服务运行时发生panic: %v", r).
					Build()
			}
		}()

		// 标记为运行状态
		csm.setState(StateRunning)

		// 执行启动后生命周期
		if csm.lifecycle != nil {
			if err := csm.lifecycle.AfterStart(); err != nil {
				errChan <- err
				return
			}
		}

		// 运行服务
		if err := csm.runner.Run(startCtx); err != nil {
			errChan <- err
		} else {
			// 服务正常结束，设置状态为停止
			csm.setState(StateStopped)
			errChan <- nil
		}
	}()

	// 等待启动完成或超时
	select {
	case err := <-errChan:
		if err != nil {
			csm.setState(StateError)
			csm.setLastError(err)
			return NewError(ErrServiceStart).
				Service(csm.config.Name).
				Operation("run").
				Message("服务运行失败").
				Cause(err).
				Build()
		}
		return nil

	case <-startCtx.Done():
		// 启动超时
		csm.setState(StateError)
		timeoutErr := ErrServiceStartTimeout(csm.config.Name, csm.startTimeout)
		csm.setLastError(timeoutErr)
		return timeoutErr
	}
}

// Stop 停止服务
func (csm *ConcurrentServiceManager) Stop() error {
	currentState := csm.GetState()

	// 检查当前状态
	switch currentState {
	case StateStopped:
		return ErrServiceAlreadyStopped(csm.config.Name)
	case StateStopping:
		return NewError(ErrServiceStop).
			Service(csm.config.Name).
			Operation("stop").
			Message("服务正在停止中").
			Build()
	case StateError:
		// 允许从错误状态停止
	case StateStarting, StateRunning:
		// 正常停止流程
	default:
		return NewError(ErrServiceStop).
			Service(csm.config.Name).
			Operation("stop").
			Messagef("无法从状态 %s 停止服务", currentState).
			Build()
	}

	// 尝试设置为停止中状态
	if !csm.compareAndSwapState(currentState, StateStopping) {
		// 状态已经改变，重新尝试
		return csm.Stop()
	}

	// 增加停止计数
	csm.stopCount.Add(1)

	// 执行停止前生命周期
	if csm.lifecycle != nil {
		if err := csm.lifecycle.BeforeStop(); err != nil {
			// 记录错误但继续停止
			csm.logError(NewError(ErrServiceStop).
				Service(csm.config.Name).
				Operation("beforeStop").
				Message("停止前处理失败").
				Cause(err).
				Build())
		}
	}

	// 使用超时控制停止过程
	stopCtx := context.Background()
	if csm.stopTimeout > 0 {
		var cancel context.CancelFunc
		stopCtx, cancel = context.WithTimeout(stopCtx, csm.stopTimeout)
		defer cancel()
	}

	// 停止服务
	errChan := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- NewError(ErrRuntime).
					Service(csm.config.Name).
					Operation("stop").
					Messagef("停止服务时发生panic: %v", r).
					Build()
			}
		}()

		// 取消上下文
		csm.cancelOnce.Do(func() {
			csm.cancel()
		})

		// 关闭停止通道
		csm.stopChanOnce.Do(func() {
			close(csm.stopChan)
		})

		// 调用服务的停止方法
		if err := csm.runner.Stop(); err != nil {
			errChan <- err
		} else {
			errChan <- nil
		}
	}()

	// 等待停止完成或超时
	select {
	case err := <-errChan:
		// 执行停止后生命周期
		if csm.lifecycle != nil {
			if lifecycleErr := csm.lifecycle.AfterStop(); lifecycleErr != nil {
				// 记录错误但不影响停止结果
				csm.logError(NewError(ErrServiceStop).
					Service(csm.config.Name).
					Operation("afterStop").
					Message("停止后处理失败").
					Cause(lifecycleErr).
					Build())
			}
		}

		if err != nil {
			csm.setState(StateError)
			csm.setLastError(err)
			return NewError(ErrServiceStop).
				Service(csm.config.Name).
				Operation("stop").
				Message("停止服务失败").
				Cause(err).
				Build()
		}

		csm.setState(StateStopped)
		return nil

	case <-stopCtx.Done():
		// 停止超时
		csm.setState(StateError)
		timeoutErr := ErrServiceStopTimeout(csm.config.Name, csm.stopTimeout)
		csm.setLastError(timeoutErr)
		return timeoutErr
	}
}

// Restart 重启服务
func (csm *ConcurrentServiceManager) Restart() error {
	// 先停止服务
	if !csm.IsStopped() {
		if err := csm.Stop(); err != nil {
			return NewError(ErrServiceRestart).
				Service(csm.config.Name).
				Operation("restart").
				Message("重启时停止服务失败").
				Cause(err).
				Build()
		}
	}

	// 等待一小段时间确保完全停止
	time.Sleep(100 * time.Millisecond)

	// 重新启动
	if err := csm.Start(); err != nil {
		return NewError(ErrServiceRestart).
			Service(csm.config.Name).
			Operation("restart").
			Message("重启时启动服务失败").
			Cause(err).
			Build()
	}

	return nil
}

// =============================================================================
// 配置和设置方法
// =============================================================================

// SetLifecycle 设置生命周期处理器
func (csm *ConcurrentServiceManager) SetLifecycle(lifecycle ServiceLifecycle) {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	csm.lifecycle = lifecycle
}

// SetErrorLogger 设置错误日志记录器
func (csm *ConcurrentServiceManager) SetErrorLogger(logger func(error)) {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	csm.errorLogger = logger
}

// SetStartTimeout 设置启动超时
func (csm *ConcurrentServiceManager) SetStartTimeout(timeout time.Duration) {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	csm.startTimeout = timeout
}

// SetStopTimeout 设置停止超时
func (csm *ConcurrentServiceManager) SetStopTimeout(timeout time.Duration) {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	csm.stopTimeout = timeout
}

// AddStateListener 添加状态变化监听器
func (csm *ConcurrentServiceManager) AddStateListener(listener func(ServiceState, ServiceState)) {
	csm.listenerMu.Lock()
	defer csm.listenerMu.Unlock()
	csm.stateListeners = append(csm.stateListeners, listener)
}

// =============================================================================
// 状态查询方法
// =============================================================================

// GetStats 获取服务统计信息
func (csm *ConcurrentServiceManager) GetStats() ServiceStats {
	return ServiceStats{
		Name:       csm.config.Name,
		State:      csm.GetState(),
		StartCount: csm.startCount.Load(),
		StopCount:  csm.stopCount.Load(),
		LastError:  csm.getLastError(),
	}
}

// ServiceStats 服务统计信息
type ServiceStats struct {
	Name       string       `json:"name"`
	State      ServiceState `json:"state"`
	StartCount int64        `json:"start_count"`
	StopCount  int64        `json:"stop_count"`
	LastError  error        `json:"last_error,omitempty"`
}

// GetName 获取服务名称
func (csm *ConcurrentServiceManager) GetName() string {
	return csm.config.Name
}

// GetLastError 获取最后一次错误
func (csm *ConcurrentServiceManager) GetLastError() error {
	return csm.getLastError()
}

// =============================================================================
// 私有辅助方法
// =============================================================================

// setLastError 设置最后一次错误
func (csm *ConcurrentServiceManager) setLastError(err error) {
	csm.mu.Lock()
	defer csm.mu.Unlock()
	csm.lastError = err
}

// getLastError 获取最后一次错误
func (csm *ConcurrentServiceManager) getLastError() error {
	csm.mu.RLock()
	defer csm.mu.RUnlock()
	return csm.lastError
}

// logError 记录错误
func (csm *ConcurrentServiceManager) logError(err error) {
	csm.mu.RLock()
	logger := csm.errorLogger
	csm.mu.RUnlock()

	if logger != nil {
		logger(err)
	}

	// 同时设置为最后一次错误
	csm.setLastError(err)
}

// notifyStateChange 通知状态变化
func (csm *ConcurrentServiceManager) notifyStateChange(oldState, newState ServiceState) {
	csm.listenerMu.RLock()
	listeners := make([]func(ServiceState, ServiceState), len(csm.stateListeners))
	copy(listeners, csm.stateListeners)
	csm.listenerMu.RUnlock()

	// 异步通知监听器
	for _, listener := range listeners {
		go func(l func(ServiceState, ServiceState)) {
			defer func() {
				if r := recover(); r != nil {
					csm.logError(NewError(ErrRuntime).
						Operation("stateListener").
						Messagef("状态监听器发生panic: %v", r).
						Build())
				}
			}()
			l(oldState, newState)
		}(listener)
	}
}

// =============================================================================
// 服务管理器接口实现
// =============================================================================

// ServiceManagerInterface 服务管理器接口
type ServiceManagerInterface interface {
	Start() error
	Stop() error
	Restart() error
	GetState() ServiceState
	IsRunning() bool
	IsStopped() bool
	GetStats() ServiceStats
	GetName() string
	GetLastError() error
}

// 确保ConcurrentServiceManager实现了ServiceManagerInterface接口
var _ ServiceManagerInterface = (*ConcurrentServiceManager)(nil)
