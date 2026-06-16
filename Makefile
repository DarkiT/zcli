.PHONY: help test cover lint bench clean install deps

# 默认目标
.DEFAULT_GOAL := help

# 项目信息
PROJECT_NAME := zcli
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')

# 目录
BIN_DIR := bin
COVERAGE_DIR := coverage

# Go 命令
GO := go
GOTEST := $(GO) test
GOCOVER := $(GO) tool cover
GOLINT := golangci-lint

# 测试标志
TEST_FLAGS := -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic
BENCH_FLAGS := -bench=. -benchmem -benchtime=3s

# 颜色输出
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

## help: 显示帮助信息
help:
	@echo '$(GREEN)使用方法:$(NC)'
	@echo '  make <target>'
	@echo ''
	@echo '$(GREEN)可用目标:$(NC)'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## test: 运行所有测试
test:
	@echo "$(GREEN)运行测试...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) $(TEST_FLAGS) ./...
	@echo "$(GREEN)✓ 测试通过$(NC)"

## test-short: 运行短测试（跳过长时间测试）
test-short:
	@echo "$(GREEN)运行短测试...$(NC)"
	@$(GOTEST) -v -race -short ./...
	@echo "$(GREEN)✓ 短测试通过$(NC)"

## test-integration: 运行集成测试
test-integration:
	@echo "$(GREEN)运行集成测试...$(NC)"
	@$(GOTEST) -v -race -tags=integration ./tests/integration/...
	@echo "$(GREEN)✓ 集成测试通过$(NC)"

## cover: 生成覆盖率报告
cover: test
	@echo "$(GREEN)生成覆盖率报告...$(NC)"
	@$(GOCOVER) -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@$(GOCOVER) -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@echo "$(GREEN)✓ 覆盖率报告已生成: $(COVERAGE_DIR)/coverage.html$(NC)"

## cover-check: 检查覆盖率是否达标（≥90%）
cover-check: test
	@echo "$(GREEN)检查覆盖率...$(NC)"
	@$(GOCOVER) -func=$(COVERAGE_DIR)/coverage.out | tail -1 | \
		awk '{print $$3}' | sed 's/%//' | \
		awk '{if($$1<90){print "$(RED)✗ 覆盖率不达标: " $$1 "% (要求 ≥90%)$(NC)"; exit 1} else {print "$(GREEN)✓ 覆盖率达标: " $$1 "%$(NC)"}}'

## lint: 运行代码检查
lint:
	@echo "$(GREEN)运行代码检查...$(NC)"
	@$(GOLINT) run --timeout=5m
	@echo "$(GREEN)✓ 代码检查通过$(NC)"

## lint-fix: 自动修复代码问题
lint-fix:
	@echo "$(GREEN)自动修复代码问题...$(NC)"
	@$(GOLINT) run --fix --timeout=5m
	@echo "$(GREEN)✓ 代码问题已修复$(NC)"

## fmt: 格式化代码
fmt:
	@echo "$(GREEN)格式化代码...$(NC)"
	@$(GO) fmt ./...
	@goimports -w -local github.com/darkit/zcli .
	@echo "$(GREEN)✓ 代码已格式化$(NC)"

## vet: 运行 go vet
vet:
	@echo "$(GREEN)运行 go vet...$(NC)"
	@$(GO) vet ./...
	@echo "$(GREEN)✓ go vet 检查通过$(NC)"

## bench: 运行基准测试
bench:
	@echo "$(GREEN)运行基准测试...$(NC)"
	@$(GOTEST) $(BENCH_FLAGS) ./... | tee $(COVERAGE_DIR)/bench.txt
	@echo "$(GREEN)✓ 基准测试完成，结果保存至 $(COVERAGE_DIR)/bench.txt$(NC)"

## clean: 清理构建产物和缓存
clean:
	@echo "$(YELLOW)清理构建产物...$(NC)"
	@rm -rf $(BIN_DIR) $(COVERAGE_DIR)
	@$(GO) clean -cache -testcache -modcache
	@echo "$(GREEN)✓ 清理完成$(NC)"

## deps: 安装/更新依赖
deps:
	@echo "$(GREEN)安装依赖...$(NC)"
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "$(GREEN)✓ 依赖已更新$(NC)"

## install: 安装开发工具
install:
	@echo "$(GREEN)安装开发工具...$(NC)"
	@which golangci-lint > /dev/null || (echo "安装 golangci-lint..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin)
	@which goimports > /dev/null || (echo "安装 goimports..." && \
		$(GO) install golang.org/x/tools/cmd/goimports@latest)
	@echo "$(GREEN)✓ 开发工具已安装$(NC)"

## ci: 运行 CI 流程（lint + test + cover-check）
ci: deps lint test cover-check
	@echo "$(GREEN)✓ CI 流程完成$(NC)"

## version: 显示项目版本信息
version:
	@echo "$(GREEN)项目信息:$(NC)"
	@echo "  名称:      $(PROJECT_NAME)"
	@echo "  版本:      $(VERSION)"
	@echo "  构建时间:  $(BUILD_TIME)"
	@echo "  Go 版本:   $(GO_VERSION)"
