# PowerShell 打包脚本 - 创建发布包

param(
    [string]$Version = "dev"
)

$ErrorActionPreference = "Stop"

$PackageName = "bg3sync-windows-amd64-$Version"
$PackageDir = "dist\$PackageName"
$BinDir = "bin"

Write-Host "=========================================" -ForegroundColor Green
Write-Host "  创建发布包: $PackageName" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green

# 清理旧的打包目录
if (Test-Path $PackageDir) {
    Remove-Item -Recurse -Force $PackageDir
}
New-Item -ItemType Directory -Path $PackageDir -Force | Out-Null

# 复制可执行文件
Write-Host "复制可执行文件..." -ForegroundColor Cyan
Copy-Item "$BinDir\bg3-sync.exe" "$PackageDir\"

# 复制文档
Write-Host "复制文档..." -ForegroundColor Cyan
Copy-Item "README_DIST.md" "$PackageDir\README.md"

# 创建快捷启动说明
$UsageText = @"
BG3 存档同步客户端 - 使用说明
========================================

1. 双击运行 bg3-sync.exe

2. 程序会在系统托盘显示图标（可能需要在防火墙中允许网络访问）

3. 右键点击托盘图标，选择"设置"配置服务器地址

4. 配置完成后，勾选"自动同步"即可开始使用

详细说明请查看 README.md 文件

日志文件位置：%APPDATA%\BG3SyncClient\logs\
配置文件位置：%APPDATA%\BG3SyncClient\config.json
"@

$UsageText | Out-File -FilePath "$PackageDir\使用说明.txt" -Encoding UTF8

# 创建 zip 压缩包
Write-Host "创建压缩包..." -ForegroundColor Cyan
$ZipPath = "dist\$PackageName.zip"
if (Test-Path $ZipPath) {
    Remove-Item -Force $ZipPath
}

Compress-Archive -Path $PackageDir -DestinationPath $ZipPath -CompressionLevel Optimal

Write-Host ""
Write-Host "✓ 打包完成: $ZipPath" -ForegroundColor Green
$ZipFile = Get-Item $ZipPath
$SizeMB = [math]::Round($ZipFile.Length / 1MB, 2)
Write-Host "文件大小: $SizeMB MB" -ForegroundColor Yellow
