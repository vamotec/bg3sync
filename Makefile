.PHONY: build build-windows build-mac clean help run dev

# 版本信息
VERSION := 1.0.0
BUILD_TIME := $(shell date '+%Y-%m-%d %H:%M:%S')
LDFLAGS := -s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'
LDFLAGS_WIN := $(LDFLAGS) -H windowsgui

# 输出目录
BIN_DIR := bin
OUTPUT_WIN := $(BIN_DIR)/bg3-sync.exe
OUTPUT_MAC := $(BIN_DIR)/bg3-sync-darwin
OUTPUT_LINUX := $(BIN_DIR)/bg3-sync-linux

# 默认目标
build: build-windows

# Windows 版本（主要目标）
build-windows:
	@echo "========================================="
	@echo "  编译 Windows 64位版本"
	@echo "========================================="
	@echo "版本: $(VERSION)"
	@echo "时间: $(BUILD_TIME)"
	@echo ""
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS_WIN)" -o $(OUTPUT_WIN) .
	@echo ""
	@echo "✓ 编译完成: $(OUTPUT_WIN)"
	@ls -lh $(OUTPUT_WIN)

# Windows 版本（使用 fyne-cross 交叉编译）
build-windows-cross:
	@echo "========================================="
	@echo "  使用 fyne-cross 编译 Windows 版本"
	@echo "========================================="
	@echo "版本: $(VERSION)"
	@echo "时间: $(BUILD_TIME)"
	@echo ""
	@which fyne-cross > /dev/null || (echo "错误: fyne-cross 未安装。运行: go install github.com/fyne-io/fyne-cross@latest" && exit 1)
	@which docker > /dev/null || (echo "错误: Docker 未安装或未运行" && exit 1)
	fyne-cross windows -arch=amd64 \
		-ldflags="$(LDFLAGS_WIN)" \
		-output bg3-sync.exe
	@echo ""
	@echo "✓ 编译完成: fyne-cross/dist/windows-amd64/bg3-sync.exe"
	@mkdir -p $(BIN_DIR)
	@cp fyne-cross/dist/windows-amd64/bg3-sync.exe $(OUTPUT_WIN)
	@ls -lh $(OUTPUT_WIN)

# macOS 版本（本地测试）
build-mac:
	@echo "编译 macOS 版本..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_MAC)-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_MAC)-arm64 .
	@echo "✓ 编译完成"

# Linux 版本
build-linux:
	@echo "编译 Linux 版本..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_LINUX) .
	@echo "✓ 编译完成: $(OUTPUT_LINUX)"

# 编译所有平台
build-all: build-windows build-mac build-linux

# 运行（开发模式 - macOS）
run:
	@echo "开发模式运行..."
	go run .

# 开发模式（带日志）
dev:
	@echo "开发模式（详细日志）..."
	go run -race .

# 测试
test:
	go test -v ./...

# 清理
clean:
	@echo "清理编译文件..."
	@rm -rf $(BIN_DIR)
	@echo "✓ 清理完成"

# 帮助
help:
	@echo "BG3 存档同步客户端 - 编译命令"
	@echo ""
	@echo "使用方法:"
	@echo "  make                   - 编译 Windows 版本"
	@echo "  make build-windows     - 编译 Windows 版本（需在 Windows 上运行）"
	@echo "  make build-windows-cross - 使用 fyne-cross 交叉编译 Windows 版本（需 Docker）"
	@echo "  make build-mac         - 编译 macOS 版本"
	@echo "  make build-linux       - 编译 Linux 版本"
	@echo "  make build-all         - 编译所有平台"
	@echo "  make run               - 运行（开发模式）"
	@echo "  make dev               - 运行（详细日志）"
	@echo "  make test              - 运行测试"
	@echo "  make clean             - 清理编译文件"
	@echo "  make help              - 显示帮助"