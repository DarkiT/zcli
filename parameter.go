package zcli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Parameter 参数定义
type Parameter struct {
	Name        string             `json:"name"`
	Default     string             `json:"default"`
	Description string             `json:"description"`
	Short       string             `json:"short"`
	Long        string             `json:"long"`
	EnumValues  []string           `json:"enumValues,omitempty"`
	Required    bool               `json:"required"` // 是否必需
	Hidden      bool               `json:"hidden"`   // 是否在帮助中隐藏
	Type        string             `json:"type"`     // 参数类型(string/int/bool)
	Validate    func(string) error `json:"-"`
	flags       uint8              // 使用位域存储状态标志
	value       atomic.Value       // 使用atomic.Value存储值
}

const (
	flagRequired = 1 << iota
	flagHidden
	flagTypeString
	flagTypeInt
	flagTypeBool
)

// manager 参数管理器
type manager struct {
	mu         sync.RWMutex          // 保护 maps 的互斥锁
	params     map[string]*Parameter // 存储参数定义
	commands   map[string]*command   // 存储自定义命令
	values     map[string]string     // 存储参数值
	paramOrder []string              // 存储参数顺序
	parsed     bool                  // 解析状态标志
}

// command 定义命令结构
type command struct {
	Name        string // 命令名称
	Description string // 命令描述
	Hidden      bool   // 是否在帮助中隐藏
	Run         func() // 子命令启动回调函数。
}

// NewParamManager 创建参数管理器
func NewParamManager() *manager {
	return &manager{
		params:     make(map[string]*Parameter),
		commands:   make(map[string]*command),
		values:     make(map[string]string),
		paramOrder: make([]string, 0),
	}
}

// AddCommand 添加自定义命令
func (pm *manager) AddCommand(name, description string, run func(), hidden bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.commands[name] = &command{
		Name:        name,
		Description: description,
		Hidden:      hidden,
		Run:         run,
	}
}

// AddParam 添加参数
func (pm *manager) AddParam(p *Parameter) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if p.Name == "" {
		return fmt.Errorf("parameter name is required")
	}

	// 检查参数是否已存在
	if _, exists := pm.params[p.Name]; exists {
		return fmt.Errorf("parameter %s already exists", p.Name)
	}

	// 预处理标志位
	p.flags = 0
	if p.Required {
		p.flags |= flagRequired
	}
	if p.Hidden {
		p.flags |= flagHidden
	}

	// 根据类型设置标志位
	switch strings.ToLower(p.Type) {
	case "string":
		p.flags |= flagTypeString
	case "int":
		p.flags |= flagTypeInt
	case "bool":
		p.flags |= flagTypeBool
	default:
		p.flags |= flagTypeString
	}

	// 处理短格式和长格式
	p.Short = strings.TrimPrefix(p.Short, "-")
	p.Long = strings.TrimPrefix(p.Long, "--")

	// 注册flag
	p.registerFlags()

	// 存储参数
	pm.params[p.Name] = p
	pm.paramOrder = append(pm.paramOrder, p.Name)

	// 设置默认值
	if p.Default != "" {
		pm.values[p.Name] = p.Default
	}

	return nil
}

// registerFlags 注册命令行参数
func (p *Parameter) registerFlags() {
	registerFlag := func(name string) {
		switch {
		case p.flags&flagTypeString != 0:
			var value string
			flag.StringVar(&value, name, p.Default, p.Description)
			p.value.Store(value)
		case p.flags&flagTypeInt != 0:
			var value int
			flag.IntVar(&value, name, 0, p.Description)
			p.value.Store(strconv.Itoa(value))
		case p.flags&flagTypeBool != 0:
			var value bool
			flag.BoolVar(&value, name, false, p.Description)
			p.value.Store(strconv.FormatBool(value))
		}
	}

	if p.Short != "" {
		registerFlag(p.Short)
	}
	if p.Long != "" {
		registerFlag(p.Long)
	}
}

// Parse 解析命令行参数
func (pm *manager) Parse() error {
	var errors []error
	for _, name := range pm.paramOrder {
		param := pm.params[name]
		if val, ok := pm.values[name]; ok {
			if err := pm.validateValue(param, val); err != nil {
				errors = append(errors, fmt.Errorf("parameter '%s' validation failed: %w", name, err))
			}
		} else if param.Required {
			errors = append(errors, fmt.Errorf("required parameter '%s' is missing", name))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("parameter validation failed: %v", errors)
	}

	return nil
}

// validateValue 验证参数值
func (pm *manager) validateValue(p *Parameter, value string) error {
	// 必需参数检查
	if p.flags&flagRequired != 0 && value == "" {
		return fmt.Errorf("parameter %s is required", p.Name)
	}

	// 枚举值检查
	if len(p.EnumValues) > 0 && value != "" {
		valid := false
		for _, enum := range p.EnumValues {
			if value == enum {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value for parameter %s: must be one of %v", p.Name, p.EnumValues)
		}
	}

	// 自定义验证
	if p.Validate != nil {
		if err := p.Validate(value); err != nil {
			return fmt.Errorf("validation failed for parameter %s: %v", p.Name, err)
		}
	}

	return nil
}

// GetString 获取字符串参数值
func (pm *manager) GetString(name string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.values[name]
}

// GetInt 获取整型参数值
func (pm *manager) GetInt(name string) int {
	if v, ok := pm.values[name]; ok {
		num, _ := strconv.Atoi(v)
		return num
	}
	return 0
}

// GetBool 获取布尔参数值
func (pm *manager) GetBool(name string) bool {
	if v, ok := pm.values[name]; ok {
		return v == "true" || v == "1"
	}
	return false
}

// SetValue 设置参数值
func (pm *manager) SetValue(name, value string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if p, ok := pm.params[name]; ok {
		if err := pm.validateValue(p, value); err != nil {
			return err
		}
		pm.values[name] = value
		return nil
	}
	return fmt.Errorf("parameter %s not found", name)
}

// GetParam 获取参数定义
func (pm *manager) GetParam(name string) *Parameter {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.params[name]
}

// GetAllParams 获取所有参数
func (pm *manager) GetAllParams() []*Parameter {
	params := make([]*Parameter, len(pm.paramOrder))

	for i, name := range pm.paramOrder {
		params[i] = pm.params[name]
	}

	return params
}

// HasParam 检查参数是否存在
func (pm *manager) HasParam(name string) bool {
	_, exists := pm.params[name]
	return exists
}

// ResetValues 重置所有参数值为默认值
func (pm *manager) ResetValues() {
	pm.values = make(map[string]string)
	pm.parsed = false

	for _, name := range pm.paramOrder {
		if p, ok := pm.params[name]; ok {
			if p.Default != "" {
				pm.values[name] = p.Default
			}
		}
	}
}
