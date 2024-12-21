package zcli

import (
	"fmt"
	"sync"
	"time"
)

// Config 服务配置
type Config struct {
	Version      string            `toml:"version"`
	LastModified int64             `toml:"last_modified"`
	Args         map[string]string `toml:"args"`
	Language     string            `toml:"language"`
	Debug        bool              `toml:"debug"`
	Runtime      sync.Map          `toml:"-"`
}

var configPool = sync.Pool{
	New: func() interface{} {
		return &Config{
			Version:      "1.0.0",
			LastModified: time.Now().Unix(),
			Args:         make(map[string]string),
		}
	},
}

// LoadConfig 加载配置
func (s *Service) LoadConfig() error {
	// 从对象池获取默认配置
	config := configPool.Get().(*Config)

	// 设置基本配置
	config.Version = "1.0.0"
	config.LastModified = time.Now().Unix()
	config.Args = make(map[string]string)

	// 更新服务配置
	s.config = config

	return nil
}

// SaveConfig 保存配置
func (s *Service) SaveConfig() error {
	// 更新配置信息
	s.config.LastModified = time.Now().Unix()
	s.config.Language = s.GetCurrentLanguage()
	s.config.Debug = s.IsDebug()

	// 收集当前参数值
	values := make(map[string]string)
	s.paramMgr.values.Range(func(key, value interface{}) bool {
		values[key.(string)] = value.(string)
		return true
	})
	s.config.Args = values

	return nil
}

// GetConfigValue 获取配置值
func (s *Service) GetConfigValue(key string) (interface{}, bool) {
	return s.config.Runtime.Load(key)
}

// SetConfigValue 设置配置值
func (s *Service) SetConfigValue(key string, value interface{}) {
	s.config.Runtime.Store(key, value)
}

// DeleteConfigValue 删除配置值
func (s *Service) DeleteConfigValue(key string) {
	s.config.Runtime.Delete(key)
}

// HasConfigValue 检查配置值是否存在
func (s *Service) HasConfigValue(key string) bool {
	_, exists := s.config.Runtime.Load(key)
	return exists
}

// GetConfigKeys 获取所有配置键
func (s *Service) GetConfigKeys() []string {
	var keys []string
	s.config.Runtime.Range(func(key, _ interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys
}

// ClearConfig 清除配置
func (s *Service) ClearConfig() error {
	// 重置配置
	s.config = configPool.Get().(*Config)
	s.paramMgr.ResetValues()

	return nil
}

// ValidateConfig 验证配置
func (s *Service) ValidateConfig() error {
	var errors []error

	// 并发验证配置的不同部分
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	wg.Add(3)
	go func() {
		defer wg.Done()
		if err := s.validateConfigBasic(); err != nil {
			errChan <- fmt.Errorf("basic config validation failed: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.validateArgs(); err != nil {
			errChan <- fmt.Errorf("args validation failed: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.validateLanguage(); err != nil {
			errChan <- fmt.Errorf("language validation failed: %w", err)
		}
	}()

	wg.Wait()
	close(errChan)

	// 收集所有错误
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("config validation failed: %v", errors)
	}

	return nil
}

// validateConfigBasic 验证基本配置
func (s *Service) validateConfigBasic() error {
	if s.config.Version == "" {
		return fmt.Errorf("config version is required")
	}
	return nil
}

// validateArgs 验证参数
func (s *Service) validateArgs() error {
	var errors []error
	var wg sync.WaitGroup
	errChan := make(chan error, len(s.config.Args))

	// 并发验证所有参数
	for name, value := range s.config.Args {
		wg.Add(1)
		go func(name, value string) {
			defer wg.Done()
			if p := s.paramMgr.GetParam(name); p != nil {
				if err := s.paramMgr.SetValue(name, value); err != nil {
					errChan <- fmt.Errorf("invalid parameter '%s': %w", name, err)
				}
			}
		}(name, value)
	}

	wg.Wait()
	close(errChan)

	// 收集验证错误
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("parameter validation failed: %v", errors)
	}

	return nil
}

// validateLanguage 验证语言设置
func (s *Service) validateLanguage() error {
	if s.config.Language != "" && !s.SetLanguage(s.config.Language) {
		return fmt.Errorf("unsupported language: %s", s.config.Language)
	}
	return nil
}
