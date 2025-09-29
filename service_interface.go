package zcli

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// =============================================================================
// 优雅的服务接口设计
// =============================================================================

// ServiceRunner 定义服务运行接口，提供类型安全的服务管理
type ServiceRunner interface {
	// Run 运行服务主逻辑，接收上下文用于优雅关闭
	Run(ctx context.Context) error

	// Stop 停止服务，执行清理工作
	Stop() error

	// Name 返回服务名称
	Name() string
}

// ServiceConfig 服务配置结构，提供完整的类型安全
type ServiceConfig struct {
	Name         string            `validate:"required,min=3,max=50"`
	DisplayName  string            `validate:"required,max=100"`
	Description  string            `validate:"max=500"`
	Version      string            `validate:"semver"`
	WorkDir      string            `validate:"dir_path"`
	Username     string            `validate:"username"`
	Dependencies []string          `validate:"dive,required"`
	EnvVars      map[string]string `validate:"dive,keys,required,endkeys,required"`
	Arguments    []string
	Executable   string
	ChRoot       string
	Options      map[string]interface{}
}

// Validate 验证配置的有效性
func (sc *ServiceConfig) Validate() error {
	var errs []error

	// 验证必需字段
	if sc.Name == "" {
		errs = append(errs, errors.New("服务名称不能为空"))
	}
	if len(sc.Name) < 3 || len(sc.Name) > 50 {
		errs = append(errs, errors.New("服务名称长度必须在3-50个字符之间"))
	}

	if sc.DisplayName == "" {
		errs = append(errs, errors.New("显示名称不能为空"))
	}
	if len(sc.DisplayName) > 100 {
		errs = append(errs, errors.New("显示名称不能超过100个字符"))
	}

	// 验证可选字段
	if len(sc.Description) > 500 {
		errs = append(errs, errors.New("描述不能超过500个字符"))
	}

	// 验证依赖项
	for i, dep := range sc.Dependencies {
		if dep == "" {
			errs = append(errs, fmt.Errorf("依赖项[%d]不能为空", i))
		}
	}

	// 验证环境变量
	for key, value := range sc.EnvVars {
		if key == "" {
			errs = append(errs, errors.New("环境变量键不能为空"))
		}
		if value == "" {
			errs = append(errs, fmt.Errorf("环境变量[%s]的值不能为空", key))
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// ValidationError 配置验证错误
type ValidationError struct {
	Errors []error
}

func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 1 {
		return fmt.Sprintf("配置验证失败: %v", ve.Errors[0])
	}
	return fmt.Sprintf("配置验证失败，共%d个错误: %v", len(ve.Errors), ve.Errors[0])
}

func (ve *ValidationError) Unwrap() []error {
	return ve.Errors
}

// =============================================================================
// 便捷的服务实现基类
// =============================================================================

// BaseService 提供默认的服务实现，用户可以嵌入此结构体
type BaseService struct {
	config   ServiceConfig
	running  bool
	stopChan chan struct{}
	onStop   []func() error
}

// NewBaseService 创建基础服务实例
func NewBaseService(config ServiceConfig) (*BaseService, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("创建服务失败: %w", err)
	}

	return &BaseService{
		config:   config,
		stopChan: make(chan struct{}),
	}, nil
}

// Name 返回服务名称
func (bs *BaseService) Name() string {
	return bs.config.Name
}

// Run 运行服务的默认实现，子类应该重写此方法
func (bs *BaseService) Run(ctx context.Context) error {
	if bs.running {
		return errors.New("服务已在运行")
	}

	bs.setRunning(true)
	defer bs.setRunning(false)

	// 默认实现：等待上下文取消或停止信号
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-bs.stopChan:
		return nil
	}
}

// Stop 停止服务
func (bs *BaseService) Stop() error {
	if !bs.running {
		return nil
	}

	// 执行停止回调
	var errs []error
	for _, stopFunc := range bs.onStop {
		if err := stopFunc(); err != nil {
			errs = append(errs, err)
		}
	}

	// 关闭停止通道
	select {
	case <-bs.stopChan:
		// 已经关闭
	default:
		close(bs.stopChan)
	}
	bs.running = false

	if len(errs) > 0 {
		return fmt.Errorf("停止服务时发生错误: %v", errs)
	}
	return nil
}

// AddStopHandler 添加停止处理函数
func (bs *BaseService) AddStopHandler(handler func() error) {
	bs.onStop = append(bs.onStop, handler)
}

// IsRunning 检查服务是否正在运行
func (bs *BaseService) IsRunning() bool {
	return bs.running
}

// WaitForStop 等待停止信号
func (bs *BaseService) WaitForStop() <-chan struct{} {
	return bs.stopChan
}

// setRunning 设置运行状态（内部使用）
func (bs *BaseService) setRunning(running bool) {
	bs.running = running
}

// =============================================================================
// 函数式服务实现
// =============================================================================

// FuncService 基于函数的服务实现，用于简单场景
type FuncService struct {
	*BaseService
	runFunc  func(context.Context) error
	stopFunc func() error
}

// NewFuncService 创建函数式服务
func NewFuncService(config ServiceConfig, runFunc func(context.Context) error, stopFunc func() error) (*FuncService, error) {
	base, err := NewBaseService(config)
	if err != nil {
		return nil, err
	}

	if runFunc == nil {
		return nil, errors.New("运行函数不能为空")
	}

	fs := &FuncService{
		BaseService: base,
		runFunc:     runFunc,
		stopFunc:    stopFunc,
	}

	// 添加停止函数
	if stopFunc != nil {
		fs.AddStopHandler(stopFunc)
	}

	return fs, nil
}

// Run 运行服务
func (fs *FuncService) Run(ctx context.Context) error {
	if fs.running {
		return errors.New("服务已在运行")
	}

	fs.setRunning(true)
	defer fs.setRunning(false)

	// 创建合并的上下文
	mergedCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 监听停止信号
	go func() {
		select {
		case <-fs.WaitForStop():
			cancel()
		case <-mergedCtx.Done():
			// 上下文已取消
		}
	}()

	// 运行用户函数
	return fs.runFunc(mergedCtx)
}

// =============================================================================
// 服务生命周期管理器
// =============================================================================

// ServiceLifecycle 服务生命周期管理接口
type ServiceLifecycle interface {
	// BeforeStart 服务启动前调用
	BeforeStart() error

	// AfterStart 服务启动后调用
	AfterStart() error

	// BeforeStop 服务停止前调用
	BeforeStop() error

	// AfterStop 服务停止后调用
	AfterStop() error
}

// ManagedService 带生命周期管理的服务
type ManagedService struct {
	ServiceRunner
	lifecycle ServiceLifecycle
}

// NewManagedService 创建带生命周期管理的服务
func NewManagedService(runner ServiceRunner, lifecycle ServiceLifecycle) *ManagedService {
	return &ManagedService{
		ServiceRunner: runner,
		lifecycle:     lifecycle,
	}
}

// Run 运行带生命周期管理的服务
func (ms *ManagedService) Run(ctx context.Context) error {
	// 启动前处理
	if ms.lifecycle != nil {
		if err := ms.lifecycle.BeforeStart(); err != nil {
			return fmt.Errorf("启动前处理失败: %w", err)
		}
	}

	// 启动服务
	errChan := make(chan error, 1)
	go func() {
		errChan <- ms.ServiceRunner.Run(ctx)
	}()

	// 启动后处理
	if ms.lifecycle != nil {
		if err := ms.lifecycle.AfterStart(); err != nil {
			return fmt.Errorf("启动后处理失败: %w", err)
		}
	}

	// 等待服务结束或上下文取消
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		// 停止前处理
		if ms.lifecycle != nil {
			if err := ms.lifecycle.BeforeStop(); err != nil {
				// 记录错误但继续停止
				fmt.Printf("停止前处理失败: %v\n", err)
			}
		}

		// 停止服务
		if err := ms.ServiceRunner.Stop(); err != nil {
			return fmt.Errorf("停止服务失败: %w", err)
		}

		// 停止后处理
		if ms.lifecycle != nil {
			if err := ms.lifecycle.AfterStop(); err != nil {
				// 记录错误但继续
				fmt.Printf("停止后处理失败: %v\n", err)
			}
		}

		return ctx.Err()
	}
}

// =============================================================================
// 服务工厂
// =============================================================================

// ServiceFactory 服务工厂接口
type ServiceFactory interface {
	CreateService(config ServiceConfig) (ServiceRunner, error)
}

// DefaultServiceFactory 默认服务工厂
type DefaultServiceFactory struct{}

// CreateService 创建默认服务
func (dsf *DefaultServiceFactory) CreateService(config ServiceConfig) (ServiceRunner, error) {
	return NewBaseService(config)
}

// FuncServiceFactory 函数式服务工厂
type FuncServiceFactory struct {
	runFunc  func(context.Context) error
	stopFunc func() error
}

// NewFuncServiceFactory 创建函数式服务工厂
func NewFuncServiceFactory(runFunc func(context.Context) error, stopFunc func() error) *FuncServiceFactory {
	return &FuncServiceFactory{
		runFunc:  runFunc,
		stopFunc: stopFunc,
	}
}

// CreateService 创建函数式服务
func (fsf *FuncServiceFactory) CreateService(config ServiceConfig) (ServiceRunner, error) {
	return NewFuncService(config, fsf.runFunc, fsf.stopFunc)
}

// =============================================================================
// 超时和重试机制
// =============================================================================

// TimeoutService 带超时的服务包装器
type TimeoutService struct {
	ServiceRunner
	startTimeout time.Duration
	stopTimeout  time.Duration
}

// NewTimeoutService 创建带超时的服务
func NewTimeoutService(service ServiceRunner, startTimeout, stopTimeout time.Duration) *TimeoutService {
	return &TimeoutService{
		ServiceRunner: service,
		startTimeout:  startTimeout,
		stopTimeout:   stopTimeout,
	}
}

// Run 带超时的运行
func (ts *TimeoutService) Run(ctx context.Context) error {
	if ts.startTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ts.startTimeout)
		defer cancel()
	}

	return ts.ServiceRunner.Run(ctx)
}

// Stop 带超时的停止
func (ts *TimeoutService) Stop() error {
	if ts.stopTimeout <= 0 {
		return ts.ServiceRunner.Stop()
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- ts.ServiceRunner.Stop()
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(ts.stopTimeout):
		return fmt.Errorf("停止服务超时（%v）", ts.stopTimeout)
	}
}

// ===================================================================
// 便利函数
// ===================================================================

// NewSimpleService 创建简单服务实现
func NewSimpleService(name string, runFunc func(context.Context) error, stopFunc func() error) ServiceRunner {
	config := ServiceConfig{
		Name:        name,
		DisplayName: name,
		Description: fmt.Sprintf("Simple service: %s", name),
		EnvVars:     make(map[string]string),
	}

	service, err := NewFuncService(config, runFunc, stopFunc)
	if err != nil {
		// 对于简单服务，我们使用panic而不是返回错误
		panic(fmt.Sprintf("创建简单服务失败: %v", err))
	}

	return service
}
