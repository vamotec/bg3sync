package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type NebulaAPI struct {
	baseURL  string
	deviceID string
	client   *http.Client
}

func NewNebulaAPI(baseURL, deviceID string) *NebulaAPI {
	return &NebulaAPI{
		baseURL:  baseURL,
		deviceID: deviceID,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadSave 上传存档到云端
func (api *NebulaAPI) UploadSave(ctx context.Context, fileName string, data []byte) (*SaveGame, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, fmt.Errorf("创建表单文件失败: %w", err)
	}

	if _, err := part.Write(data); err != nil {
		return nil, fmt.Errorf("写入文件数据失败: %w", err)
	}

	// 添加元数据（可选）
	writer.WriteField("device_id", api.deviceID)
	writer.WriteField("timestamp", time.Now().Format(time.RFC3339))

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭writer失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/games/upload", api.baseURL),
		&buf,
	)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Device-ID", api.deviceID)

	// 发送请求
	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("服务器错误: %s", errResp.Error)
		}
		return nil, fmt.Errorf("上传失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var uploadResp UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return uploadResp.Save, nil
}

// ListSaves 获取存档列表
func (api *NebulaAPI) ListSaves(ctx context.Context, limit int) ([]*SaveGame, error) {
	url := fmt.Sprintf("%s/games/list?limit=%d", api.baseURL, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-Device-ID", api.deviceID)

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("获取列表失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	var listResp SaveGameListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return listResp.Saves, nil
}

// DownloadSave 下载存档
func (api *NebulaAPI) DownloadSave(ctx context.Context, saveID string) ([]byte, error) {
	url := fmt.Sprintf("%s/games/%s/download", api.baseURL, saveID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-Device-ID", api.deviceID)

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("下载失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return data, nil
}

// DeleteSave 删除存档
func (api *NebulaAPI) DeleteSave(ctx context.Context, saveID string) error {
	url := fmt.Sprintf("%s/games/%s", api.baseURL, saveID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("X-Device-ID", api.deviceID)

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetLatestSave 获取最新的存档
func (api *NebulaAPI) GetLatestSave(ctx context.Context) (*SaveGame, error) {
	saves, err := api.ListSaves(ctx, 1)
	if err != nil {
		return nil, err
	}

	if len(saves) == 0 {
		return nil, fmt.Errorf("没有找到存档")
	}

	return saves[0], nil
}

// CheckHealth 检查服务器健康状态
func (api *NebulaAPI) CheckHealth(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", api.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器不健康 (状态码: %d)", resp.StatusCode)
	}

	return nil
}
