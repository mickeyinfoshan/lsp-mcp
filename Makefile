# MCP-LSP Bridge Service Makefile

# 项目信息
PROJECT_NAME := lsp-mcp
VERSION := 1.1.0
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关配置
GO := go
GOFLAGS := -ldflags "-X main.version=$(VERSION) -X 'main.buildTime=$(BUILD_TIME)' -X main.gitCommit=$(GIT_COMMIT)"
BINARY_NAME := $(PROJECT_NAME)
BUILD_DIR := ./bin
CMD_DIR := ./cmd/server

# 默认目标
.PHONY: all
all: clean build

# 构建
.PHONY: build
build:
	@echo "构建 $(PROJECT_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "构建完成: $(BUILD_DIR)/$(BINARY_NAME)"

# 构建所有平台
.PHONY: build-all
build-all: clean
	@echo "构建所有平台版本..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "所有平台构建完成"

# 运行
.PHONY: run
run: build
	@echo "启动 $(PROJECT_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# 运行开发模式
.PHONY: dev
dev:
	@echo "开发模式启动 $(PROJECT_NAME)..."
	$(GO) run $(CMD_DIR) -config ./config/config.yaml

# 测试
.PHONY: test
test:
	@echo "运行测试..."
	$(GO) test -v ./...

# 测试覆盖率
.PHONY: test-coverage
test-coverage:
	@echo "运行测试并生成覆盖率报告..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 代码格式化
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "代码检查..."
	$(GO) vet ./...

# 依赖管理
.PHONY: mod-tidy
mod-tidy:
	@echo "整理依赖..."
	$(GO) mod tidy

.PHONY: mod-download
mod-download:
	@echo "下载依赖..."
	$(GO) mod download

# 清理
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# 安装
.PHONY: install
install: build
	@echo "安装 $(PROJECT_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "安装完成: /usr/local/bin/$(BINARY_NAME)"

# 卸载
.PHONY: uninstall
uninstall:
	@echo "卸载 $(PROJECT_NAME)..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "卸载完成"

# 创建配置目录和示例配置
.PHONY: setup-config
setup-config:
	@echo "设置配置文件..."
	@mkdir -p ./config
	@if [ ! -f ./config/config.yaml ]; then \
		echo "配置文件已存在，跳过创建"; \
	else \
		echo "配置文件不存在，请手动创建"; \
	fi

# 创建日志目录
.PHONY: setup-logs
setup-logs:
	@echo "创建日志目录..."
	@mkdir -p ./logs

# 完整设置
.PHONY: setup
setup: setup-config setup-logs mod-download
	@echo "项目设置完成"

# 代码质量检查
.PHONY: lint
lint: fmt vet
	@echo "代码质量检查完成"

# 完整的CI流程
.PHONY: ci
ci: mod-tidy lint test build
	@echo "CI流程完成"

# 显示帮助
.PHONY: help
help:
	@echo "MCP-LSP Bridge Service Makefile"
	@echo ""
	@echo "可用目标:"
	@echo "  build          - 构建项目"
	@echo "  build-all      - 构建所有平台版本"
	@echo "  run            - 构建并运行项目"
	@echo "  dev            - 开发模式运行"
	@echo "  test           - 运行测试"
	@echo "  test-coverage  - 运行测试并生成覆盖率报告"
	@echo "  fmt            - 格式化代码"
	@echo "  vet            - 代码检查"
	@echo "  lint           - 代码质量检查 (fmt + vet)"
	@echo "  mod-tidy       - 整理依赖"
	@echo "  mod-download   - 下载依赖"
	@echo "  clean          - 清理构建文件"
	@echo "  install        - 安装到系统"
	@echo "  uninstall      - 从系统卸载"
	@echo "  setup          - 项目初始设置"
	@echo "  setup-config   - 设置配置文件"
	@echo "  setup-logs     - 创建日志目录"
	@echo "  ci             - 完整CI流程"
	@echo "  help           - 显示此帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make build     # 构建项目"
	@echo "  make dev       # 开发模式运行"
	@echo "  make test      # 运行测试"
	@echo "  make ci        # 运行完整CI流程"

# 版本信息
.PHONY: version
version:
	@echo "项目: $(PROJECT_NAME)"
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "Git提交: $(GIT_COMMIT)"

# 发布release包
.PHONY: release
release: build
	@echo "准备release目录..."
	@rm -rf release/lsp-mcp-release
	@mkdir -p release/lsp-mcp-release
	@cp ./bin/$(BINARY_NAME) release/lsp-mcp-release/lsp-mcp
	@cp release/config.yaml release/lsp-mcp-release/config.yaml
	@cd release && zip -y -r lsp-mcp-release-$(VERSION).zip lsp-mcp-release
	@echo "release包已生成: release/lsp-mcp-release-$(VERSION).zip"