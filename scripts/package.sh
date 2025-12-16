#!/bin/bash
# 打包脚本 - 创建发布包

set -e

VERSION=${1:-"dev"}
PACKAGE_NAME="bg3sync-windows-amd64-${VERSION}"
PACKAGE_DIR="dist/${PACKAGE_NAME}"
BIN_DIR="bin"

echo "========================================="
echo "  创建发布包: ${PACKAGE_NAME}"
echo "========================================="

# 清理旧的打包目录
rm -rf "${PACKAGE_DIR}"
mkdir -p "${PACKAGE_DIR}"

# 复制可执行文件
echo "复制可执行文件..."
cp "${BIN_DIR}/bg3-sync.exe" "${PACKAGE_DIR}/"

# 复制文档
echo "复制文档..."
cp README_DIST.md "${PACKAGE_DIR}/README.md"

# 创建快捷启动说明
cat > "${PACKAGE_DIR}/使用说明.txt" << 'EOF'
BG3 存档同步客户端 - 使用说明
========================================

1. 双击运行 bg3-sync.exe

2. 程序会在系统托盘显示图标（可能需要在防火墙中允许网络访问）

3. 右键点击托盘图标，选择"设置"配置服务器地址

4. 配置完成后，勾选"自动同步"即可开始使用

详细说明请查看 README.md 文件

日志文件位置：%APPDATA%\BG3SyncClient\logs\
配置文件位置：%APPDATA%\BG3SyncClient\config.json
EOF

# 创建 zip 压缩包
echo "创建压缩包..."
cd dist
zip -r "${PACKAGE_NAME}.zip" "${PACKAGE_NAME}"
cd ..

echo ""
echo "✓ 打包完成: dist/${PACKAGE_NAME}.zip"
ls -lh "dist/${PACKAGE_NAME}.zip"
