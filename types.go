package main

import (
	"time"
)

// SaveGame 存档信息
type SaveGame struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	FileHash    string    `json:"file_hash"`
	StoragePath string    `json:"storage_path"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	DeviceID    string    `json:"device_id"`
	GameTime    int       `json:"game_time,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

// SaveGameListResponse 存档列表响应
type SaveGameListResponse struct {
	Saves []*SaveGame `json:"saves"`
	Total int         `json:"total"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	Save    *SaveGame `json:"save"`
	Message string    `json:"message"`
}
