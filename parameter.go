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

// ParamManager 参数管理器
type ParamManager struct {
	params     sync.Map     // 存储参数定义
	values     sync.Map     // 存储参数值
	paramOrder atomic.Value // 存储参数顺序
	parsed     atomic.Bool  // 解析状态标志
	commands   sync.Map     // 存储自定义命令
}

// command 定义命令结构
type command struct {
	Name        string // 命令名称
	Description string // 命令描述
	Hidden      bool   // 是否在帮助中隐藏
	Run         func() // 子命令启动回调函数。
}

// NewParamManager 创建参数管理器
func NewParamManager() *ParamManager {
	pm := &ParamManager{}
	pm.paramOrder.Store(make([]string, 0, 10))
	return pm
}

// AddCommand 添加自定义命令
func (pm *ParamManager) AddCommand(name, description string, run func(), hidden bool) {
	pm.commands.Store(name, &command{
		Name:        name,
		Description: description,
		Hidden:      hidden,
		Run:         run,
	})
}

// AddParam 添加参数
func (pm *ParamManager) AddParam(p *Parameter) error {
	if p.Name == "" {
		return fmt.Errorf("parameter name is required")
	}

	// 检查参数是否已存在
	if _, exists := pm.params.Load(p.Name); exists {
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
	pm.params.Store(p.Name, p)

	// 更新参数顺序
	order := pm.paramOrder.Load().([]string)
	newOrder := append(order, p.Name)
	pm.paramOrder.Store(newOrder)

	// 设置默认值
	if p.Default != "" {
		pm.values.Store(p.Name, p.Default)
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
func (pm *ParamManager) Parse() error {
	var errors []error
	pm.params.Range(func(key, value interface{}) bool {
		param := value.(*Parameter)
		if val, ok := pm.values.Load(param.Name); ok {
			if err := pm.validateValue(param, val.(string)); err != nil {
				errors = append(errors, fmt.Errorf("parameter '%s' validation failed: %w", param.Name, err))
			}
		} else if param.Required {
			errors = append(errors, fmt.Errorf("required parameter '%s' is missing", param.Name))
		}
		return true
	})

	if len(errors) > 0 {
		return fmt.Errorf("parameter validation failed: %v", errors)
	}

	return nil
}

// parseParam 解析单个参数
func (pm *ParamManager) parseParam(name string) error {
	p, ok := pm.params.Load(name)
	if !ok {
		return nil
	}
	param := p.(*Parameter)

	// 获取当前值
	value := param.value.Load()
	if value == nil {
		if v, ok := pm.values.Load(name); ok {
			value = v
		}
	}
	// 调试输出
	fmt.Printf("Parsing parameter: %s, Current value: %v\n", name, value)

	if value != nil {
		strValue := fmt.Sprint(value)
		if err := pm.validateValue(param, strValue); err != nil {
			return err
		}
		pm.values.Store(name, strValue)
	}

	return nil
}

// validateValue 验证参数值
func (pm *ParamManager) validateValue(p *Parameter, value string) error {
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
func (pm *ParamManager) GetString(name string) string {
	if v, ok := pm.values.Load(name); ok {
		return v.(string)
	}
	return ""
}

// GetInt 获取整型参数值
func (pm *ParamManager) GetInt(name string) int {
	if v, ok := pm.values.Load(name); ok {
		num, _ := strconv.Atoi(v.(string))
		return num
	}
	return 0
}

// GetBool 获取布尔参数值
func (pm *ParamManager) GetBool(name string) bool {
	if v, ok := pm.values.Load(name); ok {
		return v.(string) == "true" || v.(string) == "1"
	}
	return false
}

// SetValue 设置参数值
func (pm *ParamManager) SetValue(name, value string) error {
	if p, ok := pm.params.Load(name); ok {
		param := p.(*Parameter)
		if err := pm.validateValue(param, value); err != nil {
			return err
		}
		pm.values.Store(name, value)
		return nil
	}
	return nil //fmt.Errorf("parameter %s not found", name)
}

// GetParam 获取参数定义
func (pm *ParamManager) GetParam(name string) *Parameter {
	if p, ok := pm.params.Load(name); ok {
		return p.(*Parameter)
	}
	return nil
}

// GetAllParams 获取所有参数
func (pm *ParamManager) GetAllParams() []*Parameter {
	order := pm.paramOrder.Load().([]string)
	params := make([]*Parameter, len(order))

	for i, name := range order {
		if p, ok := pm.params.Load(name); ok {
			params[i] = p.(*Parameter)
		}
	}

	return params
}

// HasParam 检查参数是否存在
func (pm *ParamManager) HasParam(name string) bool {
	_, exists := pm.params.Load(name)
	return exists
}

// ResetValues 重置所有参数值为默认值
func (pm *ParamManager) ResetValues() {
	pm.values = sync.Map{}
	pm.parsed.Store(false)

	order := pm.paramOrder.Load().([]string)
	for _, name := range order {
		if p, ok := pm.params.Load(name); ok {
			param := p.(*Parameter)
			if param.Default != "" {
				pm.values.Store(name, param.Default)
			}
		}
	}
}
