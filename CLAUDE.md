# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **bg3sync** (BG3 存档同步客户端) - a Baldur's Gate 3 cloud save synchronization client written in Go with a Fyne GUI. The application monitors local save files, uploads them to a Nebula cloud server, and allows users to restore previous saves.

## Build Commands

```bash
# Build Windows executable (main target platform)
make build-windows

# Build macOS version (for development/testing)
make build-mac

# Build Linux version
make build-linux

# Build all platforms
make build-all

# Run in development mode
make run

# Run with race detection and verbose logging
make dev

# Run tests
make test

# Clean build artifacts
make clean
```

The Windows build includes `-H windowsgui` to suppress console window. Version and build time are injected at compile time via ldflags.

## Architecture

### Core Components

- **main.go**: Entry point, application initialization, config loading, version management
- **client.go**: Client struct containing core UI and sync logic
  - GUI window management (main window, settings dialog)
  - System tray integration
  - File watcher setup and event handling
  - Game process monitoring (checks for bg3.exe/bg3_dx11.exe every 5 seconds)
  - Save upload/download orchestration
- **nebula_api.go**: HTTP client for Nebula cloud API
  - Upload/download save files
  - List saves, get latest save
  - Delete saves
  - Health check endpoint
- **types.go**: Data structures (SaveGame, responses, etc.)
- **utils.go**: Shared utilities (Debouncer, config management, file size formatting, device ID generation)
- **utils_windows.go**: Windows-specific process detection using Windows API
- **utils_darwin.go**: macOS process detection using `pgrep` (for development)
- **utils_linux.go**: Linux process detection
- **icon.go**: Embedded icon resource

### Key Architectural Patterns

1. **Platform-specific builds**: Uses Go build tags (`//go:build windows`) to provide different implementations for process monitoring across platforms
2. **Fyne GUI framework**: Uses Fyne v2 for cross-platform GUI with system tray support
3. **File watching**: Uses fsnotify to watch save directory for .lsv file changes
4. **Debouncing**: 2-second debouncer prevents duplicate uploads when files change rapidly
5. **Context-based timeouts**: All API calls use context.Context for timeout management
6. **Binding for UI updates**: Uses Fyne's data binding for reactive status bar updates

### Data Flow

1. **Upload flow**: File watcher detects .lsv file write → Debouncer delays → Client.handleSaveFile reads file → NebulaAPI.UploadSave sends multipart form → Server responds with SaveGame metadata
2. **Download flow**: User selects save from list → Client.restoreSave shows confirmation → Client.performRestore calls NebulaAPI.DownloadSave → Writes data to local save path
3. **Game monitoring**: Ticker checks process list every 5 seconds → Updates UI when game state changes → On game exit, optionally checks for newer cloud saves (if AutoSync enabled)

### Configuration

Config stored in:
- Windows: `%APPDATA%/BG3SyncClient/config.json`
- macOS/Linux: `~/.bg3sync/config.json`

Config fields:
- `nebula_url`: Nebula server base URL
- `device_id`: Auto-generated unique device identifier
- `save_path`: Path to BG3 save directory (defaults to Windows LocalAppData path)
- `auto_sync`: Enable automatic sync when game exits
- `auto_upload`: Enable automatic upload during gameplay

### Nebula API Endpoints

- `POST /games/upload` - Upload save file (multipart/form-data)
- `GET /games/list?limit=N` - List saves
- `GET /games/{id}/download` - Download specific save
- `DELETE /games/{id}` - Delete save
- `GET /health` - Server health check

All requests include `X-Device-ID` header for device identification.

## Development Notes

- The main target platform is Windows, but the app is designed to run on macOS for development/testing
- Default save path is hardcoded for Windows in `main.go:getDefaultSavePath()`
- Game process names are Windows-specific: "bg3.exe" or "bg3_dx11.exe" (see client.go:302)
- Window close minimizes to system tray rather than exiting
- File watcher only triggers upload when game is running AND AutoSync is enabled
- Version and BuildTime variables are set via ldflags during compilation
