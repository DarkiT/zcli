package zcli

import (
	"fmt"
	"reflect"
	"strings"
)

// =============================================================================
// 智能语言包管理器
// =============================================================================

// LanguageManager 智能语言包管理器
type LanguageManager struct {
	primary  *Language
	fallback *Language
	registry map[string]*Language
}

// NewLanguageManager 创建语言包管理器
func NewLanguageManager(primaryLang string) *LanguageManager {
	manager := &LanguageManager{
		registry: make(map[string]*Language),
	}

	manager.registry["zh"] = newChineseLanguage()
	manager.registry["en"] = newEnglishLanguage()

	manager.fallback = manager.registry["en"]
	if lang, exists := manager.registry[primaryLang]; exists {
		manager.primary = lang
	} else {
		manager.primary = manager.registry["en"]
	}

	return manager
}

// NewScopedLanguageManager 创建一个独立的语言管理器快照。
// 它始终保留英文回退，并优先使用传入的语言对象，避免被全局语言状态串改。
func NewScopedLanguageManager(primary *Language) *LanguageManager {
	manager := NewLanguageManager("en")
	if primary == nil {
		return manager
	}

	if primary.Code != "" {
		manager.registry[primary.Code] = primary
	}
	manager.primary = primary
	return manager
}

// RegisterLanguage 注册新的语言包
func (lm *LanguageManager) RegisterLanguage(lang *Language) error {
	if err := lm.validateLanguage(lang); err != nil {
		return err
	}
	lm.registry[lang.Code] = lang
	return nil
}

// SetPrimary 设置主要语言
func (lm *LanguageManager) SetPrimary(langCode string) error {
	lang, exists := lm.registry[langCode]
	if !exists {
		return fmt.Errorf("language not found: %s", langCode)
	}
	lm.primary = lang
	return nil
}

// GetPrimary 获取当前主要语言包
func (lm *LanguageManager) GetPrimary() *Language {
	return lm.primary
}

// GetText 智能获取文本，支持回退机制
func (lm *LanguageManager) GetText(path string) string {
	if text := lm.getTextFromLanguage(lm.primary, path); text != "" {
		return text
	}

	if text := lm.getTextFromLanguage(lm.fallback, path); text != "" {
		return text
	}

	return fmt.Sprintf("[Missing: %s]", path)
}

// getTextFromLanguage 从指定语言包获取文本
func (lm *LanguageManager) getTextFromLanguage(lang *Language, path string) string {
	if lang == nil {
		return ""
	}

	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return ""
	}

	value := reflect.ValueOf(lang).Elem()
	for _, part := range parts {
		fieldName := toPascal(part)
		value = value.FieldByName(fieldName)
		if !value.IsValid() {
			return ""
		}
	}

	if value.Kind() == reflect.String {
		return value.String()
	}

	return ""
}

// toPascal 将字符串首字母大写，其余部分保持原样，用于匹配 Go 导出的结构字段名
func toPascal(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// validateLanguage 验证语言包完整性
func (lm *LanguageManager) validateLanguage(lang *Language) error {
	if lang.Code == "" {
		return fmt.Errorf("language code is required")
	}
	if lang.Name == "" {
		return fmt.Errorf("language name is required")
	}

	criticalPaths := []string{
		"service.operations.install",
		"service.operations.start",
		"service.operations.stop",
		"service.status.running",
		"service.status.stopped",
		"ui.commands.usage",
		"error.prefix",
	}

	for _, path := range criticalPaths {
		if text := lm.getTextFromLanguage(lang, path); text == "" {
			return fmt.Errorf("critical field missing: %s", path)
		}
	}

	return nil
}
