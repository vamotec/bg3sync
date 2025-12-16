package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// 配置管理
func getConfigPath() string {
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
	return filepath.Join(dir, "config.json")
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
