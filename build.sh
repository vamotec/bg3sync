#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================"
echo "  BG3 存档同步客户端 - 交叉编译脚本"
echo "========================================"
echo ""

# 版本信息
VERSION="1.0.0"
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')

echo "版本: $VERSION"
echo "构建时间: $BUILD_TIME"
echo ""

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo -e "${RED}[错误] 未找到 Go 编译器！${NC}"
    echo "请先安装 Go: https://golang.org/dl/"
    exit 1
fi

# 显示 Go 版本
echo "Go 版本:"
go version
echo ""

# 创建输出目录
mkdir -p bin

# LDFLAGS
LDFLAGS="-s -w -H windowsgui -X 'main.Version=$VERSION' -X 'main.BuildTime=$BUILD_TIME'"

echo "正在编译 Windows 64位版本..."
echo ""

# 交叉编译到 Windows
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o bin/bg3-sync.exe .

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}========================================"
    echo "  编译成功！"
    echo "========================================${NC}"
    echo ""
    echo "输出文件: bin/bg3-sync.exe"

    # 显示文件大小
    if [[ "$OSTYPE" == "darwin"* ]]; then
        SIZE=$(ls -lh bin/bg3-sync.exe | awk '{print $5}')
    else
        SIZE=$(ls -lh bin/bg3-sync.exe | awk '{print $5}')
    fi
    echo "文件大小: $SIZE"

    # 可选：使用 UPX 压缩
    if command -v upx &> /dev/null; then
        echo ""
        echo "检测到 UPX，是否压缩？(y/n)"
        read -r response
        if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            echo "正在压缩..."
            upx --best --lzma bin/bg3-sync.exe

            if [[ "$OSTYPE" == "darwin"* ]]; then
                SIZE=$(ls -lh bin/bg3-sync.exe | awk '{print $5}')
            else
                SIZE=$(ls -lh bin/bg3-sync.exe | awk '{print $5}')
            fi
            echo "压缩后大小: $SIZE"
        fi
    fi

    echo ""
    echo -e "${GREEN}✓ 可以将 bin/bg3-sync.exe 复制到 Windows 系统运行${NC}"
    echo ""
else
    echo ""
    echo -e "${RED}========================================"
    echo "  编译失败！"
    echo "========================================${NC}"
    echo ""
    echo "请检查错误信息"
    exit 1
fi