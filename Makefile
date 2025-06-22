# ES Tool Makefile
# 支持编译成全平台的可执行文件

# 项目信息
BINARY_NAME=es-tool
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 编译参数
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}"

# 支持的平台和架构
PLATFORMS=linux/amd64 linux/arm64 linux/386 linux/arm darwin/amd64 darwin/arm64 windows/amd64 windows/386

# 默认目标
.PHONY: all
all: clean build

# 构建当前平台的可执行文件
.PHONY: build
build:
	@echo "Building for current platform..."
	go build ${LDFLAGS} -o ${BINARY_NAME} main.go
	@echo "Build completed: ${BINARY_NAME}"

# 构建所有平台的可执行文件
.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@for platform in ${PLATFORMS}; do \
		IFS='/' read -r GOOS GOARCH <<< "$$platform"; \
		BINARY_NAME_EXT=$${BINARY_NAME}; \
		if [ "$$GOOS" = "windows" ]; then \
			BINARY_NAME_EXT=$${BINARY_NAME}.exe; \
		fi; \
		echo "Building for $$GOOS/$$GOARCH..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build ${LDFLAGS} -o dist/$${BINARY_NAME}-$$GOOS-$$GOARCH$${BINARY_NAME_EXT:+.exe} main.go; \
	done
	@echo "All builds completed!"

# 构建特定平台
.PHONY: build-linux-amd64
build-linux-amd64:
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 main.go

.PHONY: build-linux-arm64
build-linux-arm64:
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-arm64 main.go

.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 main.go

.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 main.go

.PHONY: build-windows-amd64
build-windows-amd64:
	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe main.go

# 创建发布包
.PHONY: release
release: build-all
	@echo "Creating release packages..."
	@mkdir -p release
	@for platform in ${PLATFORMS}; do \
		IFS='/' read -r GOOS GOARCH <<< "$$platform"; \
		BINARY_NAME_EXT=$${BINARY_NAME}; \
		if [ "$$GOOS" = "windows" ]; then \
			BINARY_NAME_EXT=$${BINARY_NAME}.exe; \
		fi; \
		RELEASE_NAME=${BINARY_NAME}-${VERSION}-$$GOOS-$$GOARCH; \
		mkdir -p release/$$RELEASE_NAME; \
		cp dist/$${BINARY_NAME}-$$GOOS-$$GOARCH$${BINARY_NAME_EXT:+.exe} release/$$RELEASE_NAME/$${BINARY_NAME_EXT}; \
		cp README.md release/$$RELEASE_NAME/; \
		cp LICENSE release/$$RELEASE_NAME/ 2>/dev/null || true; \
		if [ "$$GOOS" = "windows" ]; then \
			cd release && zip -r $$RELEASE_NAME.zip $$RELEASE_NAME && rm -rf $$RELEASE_NAME; \
		else \
			cd release && tar -czf $$RELEASE_NAME.tar.gz $$RELEASE_NAME && rm -rf $$RELEASE_NAME; \
		fi; \
	done
	@echo "Release packages created in release/ directory"

# 安装依赖
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# 清理构建文件
.PHONY: clean
clean:
	@echo "Cleaning build files..."
	@rm -rf dist/
	@rm -rf release/
	@rm -f ${BINARY_NAME}
	@rm -f ${BINARY_NAME}.exe
	@echo "Clean completed"

# 显示帮助信息
.PHONY: help
help:
	@echo "ES Tool Makefile"
	@echo "================"
	@echo ""
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-all      - Build for all supported platforms"
	@echo "  build-linux-amd64    - Build for Linux AMD64"
	@echo "  build-linux-arm64    - Build for Linux ARM64"
	@echo "  build-darwin-amd64   - Build for macOS AMD64"
	@echo "  build-darwin-arm64   - Build for macOS ARM64"
	@echo "  build-windows-amd64  - Build for Windows AMD64"
	@echo "  release        - Create release packages for all platforms"
	@echo "  deps           - Install dependencies"
	@echo "  test           - Run tests"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"
	@echo "  clean          - Clean build files"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Supported platforms:"
	@echo "  ${PLATFORMS}"
	@echo ""
	@echo "Version: ${VERSION}"
	@echo "Build Time: ${BUILD_TIME}"
	@echo "Commit Hash: ${COMMIT_HASH}"

# 创建必要的目录
.PHONY: prepare
prepare:
	@mkdir -p dist
	@mkdir -p release

# 默认目标
.DEFAULT_GOAL := help 