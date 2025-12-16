package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

var (
	Version   = "dev"     // 会被编译时注入
	BuildTime = "unknown" // 会被编译时注入
)

type Config struct {
	NebulaURL   string `json:"nebula_url"`
	DeviceID    string `json:"device_id"`
	SavePath    string `json:"save_path"`
	AutoSync    bool   `json:"auto_sync"`
	AutoUpload  bool   `json:"auto_upload"`
	AutoRestore bool   `json:"auto_restore"` // 游戏退出后自动恢复云端最新存档
}

func main() {
	// 可以在日志或关于对话框中显示版本
	log.Printf("BG3 存档同步客户端 v%s (构建时间: %s)\n", Version, BuildTime)
	// 创建 Fyne 应用
	a := app.NewWithID("com.mosia.bg3sync")
	a.SetIcon(resourceIconPng) // 你的图标

	// 加载配置
	config := loadConfig()
	if config.SavePath == "" {
		config.SavePath = getDefaultSavePath()
	}
	if config.DeviceID == "" {
		config.DeviceID = generateDeviceID()
		saveConfig(config)
	}

	// 创建客户端
	client := NewClient(config, a)

	// 如果支持系统托盘
	if desk, ok := a.(desktop.App); ok {
		client.setupSystemTray(desk)
	}

	// 启动文件监听
	go func() {
		if err := client.StartWatching(); err != nil {
			log.Printf("启动文件监听失败: %v\n", err)
		}
	}()

	// 显示主窗口
	client.showMainWindow()

	a.Run()
}

func getDefaultSavePath() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("LOCALAPPDATA")
		return filepath.Join(
			appData,
			"Larian Studios",
			"Baldur's Gate 3",
			"PlayerProfiles",
			"Public",
			"Savegames",
			"Story",
		)
	} else if runtime.GOOS == "darwin" {
		// macOS 路径
		home, _ := os.UserHomeDir()
		return filepath.Join(
			home,
			"Library",
			"Application Support",
			"Larian Studios",
			"Baldur's Gate 3",
			"PlayerProfiles",
			"Public",
			"Savegames",
			"Story",
		)
	}
	// Linux 或其他系统
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "Larian Studios", "Baldur's Gate 3", "PlayerProfiles", "Public", "Savegames", "Story")
}
