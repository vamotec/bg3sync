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

# Windows Debug 版本（带控制台输出）
build-windows-debug:
	@echo "========================================="
	@echo "  编译 Windows Debug 版本（带控制台）"
	@echo "========================================="
	@echo "版本: $(VERSION)"
	@echo "时间: $(BUILD_TIME)"
	@echo ""
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(OUTPUT_WIN) .
	@echo ""
	@echo "✓ 编译完成: $(OUTPUT_WIN)"
	@ls -lh $(OUTPUT_WIN)

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

# 打包发布版本 (跨平台)
package: build-windows-debug
	@echo "打包发布版本..."
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File scripts/package.ps1 -Version $(VERSION)
else
	@bash scripts/package.sh $(VERSION)
endif

# 清理
clean:
	@echo "清理编译文件..."
	@rm -rf $(BIN_DIR) dist
	@echo "✓ 清理完成"

# 帮助
help:
	@echo "BG3 存档同步客户端 - 编译命令"
	@echo ""
	@echo "使用方法:"
	@echo "  make                   - 编译 Windows 版本"
	@echo "  make build-windows     - 编译 Windows 版本（需在 Windows 上运行）"
	@echo "  make run               - 运行（开发模式）"
	@echo "  make dev               - 运行（详细日志）"
	@echo "  make test              - 运行测试"
	@echo "  make clean             - 清理编译文件"
	@echo "  make help              - 显示帮助"