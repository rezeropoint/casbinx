.PHONY: help build test test-unit test-integration test-coverage clean fmt vet lint deps test-env-up test-env-down examples

# 默认目标
help: ## 显示帮助信息
	@echo "CasbinX 项目构建和测试命令"
	@echo "=========================="
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# 构建相关
build: ## 构建项目
	@echo "构建 CasbinX 项目..."
	go build ./...

clean: ## 清理构建产物
	@echo "清理构建产物..."
	go clean ./...
	rm -rf coverage.out coverage.html

# 代码质量
fmt: ## 格式化代码
	@echo "格式化 Go 代码..."
	go fmt ./...

vet: ## 运行 go vet 静态分析
	@echo "运行 go vet 静态分析..."
	go vet ./...

lint: ## 运行代码检查 (需要安装 golangci-lint)
	@echo "运行代码检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint 未安装，跳过代码检查"; \
		echo "安装命令: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 依赖管理
deps: ## 下载和整理依赖
	@echo "下载和整理依赖..."
	go mod download
	go mod tidy

deps-upgrade: ## 升级依赖到最新版本
	@echo "升级依赖..."
	go get -u ./...
	go mod tidy

# 测试相关
test: test-integration ## 运行集成测试
	@echo "所有测试完成"

test-integration: test-env-check ## 运行集成测试
	@echo "运行集成测试..."
	go test -v -race ./tests/integration/...

test-all: test-integration ## 运行所有测试（等同于集成测试）
	@echo "运行所有测试..."

test-coverage: test-env-check ## 生成测试覆盖率报告
	@echo "生成测试覆盖率报告..."
	go test -coverprofile=coverage.out ./tests/integration/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

test-coverage-report: test-env-check ## 生成详细的覆盖率分析报告
	@echo "生成详细覆盖率分析报告..."
	go test -coverprofile=coverage.out ./tests/integration/...
	go run -c 'package main; import ("fmt"; "os"; "casbinx/tests/helpers"); func main() { analyzer := helpers.NewCoverageAnalyzer("coverage.out", true); report, err := analyzer.AnalyzeCoverage(); if err != nil { fmt.Printf("分析失败: %v\n", err); os.Exit(1) }; analyzer.PrintReport(report); analyzer.GenerateHTMLReport(report, "coverage_report.html") }'
	@echo "详细报告已生成: coverage_report.html"

test-coverage-func: test-env-check ## 显示函数级别的测试覆盖率
	@echo "函数级别测试覆盖率..."
	go test -coverprofile=coverage.out ./tests/integration/...
	go tool cover -func=coverage.out

# 测试环境管理
test-env-up: ## 启动测试环境 (PostgreSQL + Redis)
	@echo "启动测试环境..."
	docker-compose -f tests/docker-compose.test.yml up -d
	@echo "等待服务启动..."
	@sleep 5
	@echo "检查服务状态..."
	docker-compose -f tests/docker-compose.test.yml ps

test-env-down: ## 停止测试环境
	@echo "停止测试环境..."
	docker-compose -f tests/docker-compose.test.yml down -v

test-env-logs: ## 查看测试环境日志
	docker-compose -f tests/docker-compose.test.yml logs

test-env-clean: ## 清理测试环境数据
	@echo "清理测试环境数据..."
	docker-compose -f tests/docker-compose.test.yml down -v --remove-orphans
	docker volume prune -f

test-env-check: ## 检查测试环境是否就绪
	@echo "检查测试环境..."
	@if ! docker-compose -f tests/docker-compose.test.yml ps | grep -q "Up"; then \
		echo "测试环境未启动，正在启动..."; \
		$(MAKE) test-env-up; \
	else \
		echo "测试环境已就绪"; \
	fi

# 示例运行
examples: ## 运行基础使用示例
	@echo "运行基础使用示例..."
	cd examples/basic_usage && go run main.go

examples-multi-tenant: ## 运行多租户示例
	@echo "运行多租户示例..."
	cd examples/multi_tenant && go run main.go

# 开发工作流
dev-setup: deps test-env-up ## 设置开发环境
	@echo "开发环境设置完成"

dev-test: fmt vet test ## 开发时快速测试
	@echo "开发测试完成"

dev-test-full: fmt vet test-all ## 开发时完整测试
	@echo "完整测试完成"

ci: deps fmt vet lint test-all ## CI 流程
	@echo "CI 流程完成"

# 数据库相关
db-reset: test-env-down test-env-up ## 重置测试数据库
	@echo "测试数据库已重置"

# 性能测试
benchmark: test-env-check ## 运行性能测试
	@echo "运行性能测试..."
	go test -bench=. -benchmem ./tests/integration/...

benchmark-cpu: test-env-check ## 运行 CPU 性能分析
	@echo "运行 CPU 性能分析..."
	go test -bench=. -cpuprofile=cpu.prof ./tests/integration/...
	@echo "使用 'go tool pprof cpu.prof' 查看结果"

benchmark-mem: test-env-check ## 运行内存性能分析
	@echo "运行内存性能分析..."
	go test -bench=. -memprofile=mem.prof ./tests/integration/...
	@echo "使用 'go tool pprof mem.prof' 查看结果"

# 文档生成
docs: ## 生成 Go 文档
	@echo "生成 Go 文档..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "在浏览器中打开 http://localhost:6060/pkg/casbinx/"; \
		godoc -http=:6060; \
	else \
		echo "godoc 未安装，请运行: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# 版本管理
version: ## 显示版本信息
	@echo "Go 版本: $(shell go version)"
	@echo "Git 版本: $(shell git describe --tags --always --dirty 2>/dev/null || echo 'unknown')"
	@echo "构建时间: $(shell date)"

# 检查工具安装
check-tools: ## 检查开发工具是否安装
	@echo "检查开发工具..."
	@command -v go >/dev/null 2>&1 || (echo "Go 未安装" && exit 1)
	@command -v docker >/dev/null 2>&1 || (echo "Docker 未安装" && exit 1)
	@command -v docker-compose >/dev/null 2>&1 || (echo "Docker Compose 未安装" && exit 1)
	@echo "✅ 所有必需工具已安装"

# 安装开发工具
install-tools: ## 安装开发工具
	@echo "安装开发工具..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/godoc@latest
	@echo "开发工具安装完成"