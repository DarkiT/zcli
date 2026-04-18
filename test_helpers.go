package zcli

import (
	"os"
	"strconv"
	"time"
)

// testTimeoutMultiplier 返回测试超时倍数
// 可通过环境变量 ZCLI_TEST_TIMEOUT_MULTIPLIER 调整
// 用于在高负载 CI 环境中增加超时容忍度
func testTimeoutMultiplier() float64 {
	if v := os.Getenv("ZCLI_TEST_TIMEOUT_MULTIPLIER"); v != "" {
		if m, err := strconv.ParseFloat(v, 64); err == nil && m > 0 {
			return m
		}
	}
	return 1.0
}

// testDuration 根据环境变量倍数调整测试超时时间
// 示例: testDuration(100 * time.Millisecond)在 ZCLI_TEST_TIMEOUT_MULTIPLIER=2 时返回 200ms
func testDuration(base time.Duration) time.Duration {
	return time.Duration(float64(base) * testTimeoutMultiplier())
}
