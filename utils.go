package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Debouncer 防抖器
type Debouncer struct {
	mu    sync.Mutex
	timer *time.Timer
	delay time.Duration
}

func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{delay: delay}
}

func (d *Debouncer) Do(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.delay, fn)
}

// 获取应用数据目录
func getAppDataDir() string {
	var dir string
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		dir = filepath.Join(appData, "BG3SyncClient")
	} else {
		// macOS/Linux 开发环境
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".bg3sync")
	}
	os.MkdirAll(dir, 0755)
	return dir
}

// 配置管理
func getConfigPath() string {
	return filepath.Join(getAppDataDir(), "config.json")
}

func loadConfig() *Config {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return &Config{
			AutoSync:   true,
			AutoUpload: true,
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &Config{
			AutoSync:   true,
			AutoUpload: true,
		}
	}
	return &config
}

func saveConfig(config *Config) error {
	path := getConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// 格式化文件大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// 生成设备ID
func generateDeviceID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
}

// 初始化日志系统
func initLogger() (*os.File, error) {
	// 创建日志目录
	logsDir := filepath.Join(getAppDataDir(), "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 日志文件名使用日期时间
	logFileName := fmt.Sprintf("bg3sync_%s.log", time.Now().Format("2006-01-02_15-04-05"))
	logFilePath := filepath.Join(logsDir, logFileName)

	// 打开日志文件
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %w", err)
	}

	// 设置日志输出到文件和控制台
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Printf("========================================\n")
	log.Printf("日志文件: %s\n", logFilePath)
	log.Printf("========================================\n")

	// 清理旧日志文件（保留最近7天）
	go cleanOldLogs(logsDir, 7)

	return logFile, nil
}

// 清理旧日志文件
func cleanOldLogs(logsDir string, keepDays int) {
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -keepDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 删除超过保留期限的日志文件
		if info.ModTime().Before(cutoff) {
			logPath := filepath.Join(logsDir, entry.Name())
			os.Remove(logPath)
			log.Printf("已删除旧日志文件: %s\n", entry.Name())
		}
	}
}

// 将文件夹打包为 zip
func zipFolder(folderPath string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	defer zipWriter.Close()

	// 遍历文件夹
	err := filepath.Walk(folderPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过文件夹本身
		if info.IsDir() {
			return nil
		}

		// 创建 zip 文件头
		relPath, err := filepath.Rel(folderPath, filePath)
		if err != nil {
			return err
		}

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// 读取并写入文件内容
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}

		_, err = zipFile.Write(data)
		return err
	})

	if err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// 解压 zip 到指定文件夹
func unzipToFolder(zipData []byte, destPath string) error {
	// 创建 zip reader
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return err
	}

	// 确保目标文件夹存在
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return err
	}

	// 解压每个文件
	for _, file := range reader.File {
		filePath := filepath.Join(destPath, file.Name)

		// 创建子文件夹（如果需要）
		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, 0755)
			continue
		}

		// 确保父文件夹存在
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		// 打开 zip 中的文件
		rc, err := file.Open()
		if err != nil {
			return err
		}

		// 读取内容
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		// 写入文件
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return err
		}
	}

	return nil
}
