package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/fsnotify/fsnotify"
)

type Client struct {
	config    *Config
	api       *NebulaAPI
	watcher   *fsnotify.Watcher
	app       fyne.App
	mainWin   fyne.Window
	statusBar binding.String

	// 进程监控
	gameRunning      bool
	lastUploadedSave *SaveGame // 最后一次上传的存档
	lastUploadTime   time.Time // 最后一次上传的时间
}

func NewClient(config *Config, app fyne.App) *Client {
	statusBar := binding.NewString()
	statusBar.Set("就绪")

	return &Client{
		config:    config,
		api:       NewNebulaAPI(config.NebulaURL, config.DeviceID),
		app:       app,
		statusBar: statusBar,
	}
}

func (c *Client) setupSystemTray(desk desktop.App) {
	menu := fyne.NewMenu("BG3 存档同步",
		fyne.NewMenuItem("打开主界面", func() {
			c.mainWin.Show()
		}),
		fyne.NewMenuItem("立即同步", func() {
			c.manualSync()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("设置", func() {
			c.showSettings()
		}),
		fyne.NewMenuItem("退出", func() {
			c.app.Quit()
		}),
	)

	desk.SetSystemTrayMenu(menu)
}

func (c *Client) showMainWindow() {
	if c.mainWin != nil {
		c.mainWin.Show()
		return
	}

	c.mainWin = c.app.NewWindow("博德之门3 云存档同步")
	c.mainWin.Resize(fyne.NewSize(800, 600))

	// 创建 UI
	content := c.makeMainUI()
	c.mainWin.SetContent(content)

	// 窗口关闭时最小化到托盘
	c.mainWin.SetCloseIntercept(func() {
		c.mainWin.Hide()
	})

	c.mainWin.Show()
}

func (c *Client) makeMainUI() fyne.CanvasObject {
	// 状态栏
	status := widget.NewLabelWithData(c.statusBar)

	// 存档列表
	savesList := widget.NewList(
		func() int { return 0 }, // 动态加载
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(""),
				widget.NewButton("恢复", nil),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {},
	)

	// 刷新按钮
	refreshBtn := widget.NewButton("刷新存档列表", func() {
		c.refreshSavesList(savesList)
	})

	// 设置按钮
	settingsBtn := widget.NewButton("设置", func() {
		c.showSettings()
	})

	// 手动上传按钮
	uploadBtn := widget.NewButton("立即上传", func() {
		c.manualSync()
	})

	// 自动同步开关
	autoSyncCheck := widget.NewCheck("自动同步", func(checked bool) {
		c.config.AutoSync = checked
		saveConfig(c.config)
	})
	autoSyncCheck.SetChecked(c.config.AutoSync)

	// 游戏状态
	gameStatus := widget.NewLabel("游戏状态: 未运行")
	go c.monitorGameProcess(gameStatus)

	// 布局
	toolbar := container.NewBorder(
		nil, nil,
		autoSyncCheck,
		container.NewHBox(uploadBtn, settingsBtn, refreshBtn),
	)

	return container.NewBorder(
		toolbar,
		container.NewVBox(gameStatus, status),
		nil, nil,
		savesList,
	)
}

func (c *Client) refreshSavesList(list *widget.List) {
	c.statusBar.Set("正在加载存档列表...")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		saves, err := c.api.ListSaves(ctx, 100)
		if err != nil {
			// UI 操作需要在主线程
			fyne.Do(func() {
				c.statusBar.Set(fmt.Sprintf("加载失败: %v", err))
				dialog.ShowError(err, c.mainWin)
			})
			return
		}

		// 在主线程更新 UI
		fyne.Do(func() {
			// 更新列表数据
			list.Length = func() int { return len(saves) }
			list.UpdateItem = func(id widget.ListItemID, item fyne.CanvasObject) {
				if id >= len(saves) {
					return
				}

				save := saves[id]
				box := item.(*fyne.Container)

				label := box.Objects[0].(*widget.Label)
				label.SetText(fmt.Sprintf("%s - %s (%s)",
					save.Timestamp.Format("2006-01-02 15:04:05"),
					save.FileName,
					formatSize(save.FileSize),
				))

				btn := box.Objects[1].(*widget.Button)
				btn.OnTapped = func() {
					c.restoreSave(save)
				}
			}

			list.Refresh()
			c.statusBar.Set(fmt.Sprintf("已加载 %d 个存档", len(saves)))
		})
	}()
}

func (c *Client) restoreSave(save *SaveGame) {
	// 确认对话框
	dialog.ShowConfirm(
		"确认恢复",
		fmt.Sprintf("确定要恢复存档?\n\n时间: %s\n文件: %s\n\n当前本地存档将被覆盖!",
			save.Timestamp.Format("2006-01-02 15:04:05"),
			save.FileName,
		),
		func(ok bool) {
			if !ok {
				return
			}

			c.performRestore(save)
		},
		c.mainWin,
	)
}

func (c *Client) performRestore(save *SaveGame) {
	c.statusBar.Set("正在下载存档...")

	go func() {
		ctx := context.Background()

		// 下载文件
		data, err := c.api.DownloadSave(ctx, save.ID)
		if err != nil {
			c.statusBar.Set(fmt.Sprintf("下载失败: %v", err))
			dialog.ShowError(err, c.mainWin)
			return
		}

		// 解压到本地（去掉 .zip 后缀作为文件夹名）
		folderName := strings.TrimSuffix(save.FileName, ".zip")
		saveFolderPath := filepath.Join(c.config.SavePath, folderName)

		// 先删除旧文件夹（如果存在）
		os.RemoveAll(saveFolderPath)

		// 解压 zip 到文件夹
		if err := unzipToFolder(data, saveFolderPath); err != nil {
			c.statusBar.Set(fmt.Sprintf("解压失败: %v", err))
			dialog.ShowError(err, c.mainWin)
			return
		}

		c.statusBar.Set("恢复成功!")
		dialog.ShowInformation("成功", "存档已恢复到本地", c.mainWin)
	}()
}

func (c *Client) StartWatching() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	c.watcher = watcher

	// 监听事件处理
	go func() {
		debouncer := NewDebouncer(2 * time.Second)

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					log.Printf("📁 文件事件: %s (Op: %v)\n", event.Name, event.Op)
					// 获取存档文件夹路径（UUID 文件夹）
					saveFolderPath := filepath.Dir(event.Name)
					log.Printf("📂 存档文件夹: %s\n", saveFolderPath)
					// 确保是存档目录的直接子文件夹（UUID 文件夹）
					parentDir := filepath.Dir(saveFolderPath)
					log.Printf("📌 父目录: %s (期望: %s)\n", parentDir, c.config.SavePath)

					if filepath.Clean(parentDir) != filepath.Clean(c.config.SavePath) {
						log.Printf("⏭️  跳过: 不是直接子文件夹\n")
						continue
					}

					// 只处理 __HonourMode 后缀的文件夹
					folderName := filepath.Base(saveFolderPath)
					log.Printf("📝 文件夹名: %s\n", folderName)
					if !strings.HasSuffix(folderName, "__HonourMode") {
						log.Printf("⏭️  跳过: 不是荣誉模式存档\n")
						continue
					}

					// 如果检测到新创建的文件夹，添加到监听列表
					if event.Has(fsnotify.Create) {
						if info, err := os.Stat(saveFolderPath); err == nil && info.IsDir() {
							log.Printf("检测到新存档文件夹，添加监听: %s\n", saveFolderPath)
							watcher.Add(saveFolderPath)
						}
					}

					// 只在开启自动同步且游戏运行时上传
					log.Printf("🔧 AutoSync: %v\n", c.config.AutoSync)
					//if !c.config.AutoSync || !c.gameRunning {
					//	continue
					//}
					//只在开启自动同步时上传 (调试)
					if !c.config.AutoSync {
						log.Printf("⏭️  跳过: 自动同步未开启\n")
						continue
					}

					log.Printf("⏱️  准备上传 (debounce 2s): %s\n", saveFolderPath)
					debouncer.Do(func() {
						log.Printf("🚀 开始上传: %s\n", saveFolderPath)
						c.handleSaveFolder(saveFolderPath)
						log.Printf("✅ 上传完成: %s\n", saveFolderPath)
					})

					//debouncer.Do(func() {
					//	c.handleSaveFolder(saveFolderPath)
					//})
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v\n", err)
			}
		}
	}()

	// 添加主目录监听
	if err := watcher.Add(c.config.SavePath); err != nil {
		return err
	}

	// 递归添加所有已存在的 __HonourMode 子目录到监听列表
	entries, err := os.ReadDir(c.config.SavePath)
	if err != nil {
		log.Printf("读取存档目录失败: %v\n", err)
		return nil // 不返回错误，继续运行
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasSuffix(entry.Name(), "__HonourMode") {
			subDir := filepath.Join(c.config.SavePath, entry.Name())
			if err := watcher.Add(subDir); err != nil {
				log.Printf("添加子目录监听失败 %s: %v\n", subDir, err)
			} else {
				log.Printf("已添加存档目录监听: %s\n", entry.Name())
			}
		}
	}

	return nil
}

func (c *Client) handleSaveFolder(folderPath string) {
	folderName := filepath.Base(folderPath)
	c.statusBar.Set(fmt.Sprintf("正在上传: %s", folderName))

	log.Printf("检测到存档变化: %s\n", folderPath)

	// 打包文件夹为 zip
	zipData, err := zipFolder(folderPath)
	if err != nil {
		log.Printf("打包文件夹失败: %v\n", err)
		c.statusBar.Set(fmt.Sprintf("打包失败: %v", err))
		return
	}

	// 显示压缩后的文件大小
	zipSize := len(zipData)
	log.Printf("压缩包大小: %s\n", formatSize(int64(zipSize)))
	c.statusBar.Set(fmt.Sprintf("正在上传: %s (%s)", folderName, formatSize(int64(zipSize))))

	// 上传 zip 文件
	ctx := context.Background()
	save, err := c.api.UploadSave(ctx, folderName+".zip", zipData)
	if err != nil {
		log.Printf("上传失败: %v\n", err)

		// 检查是否是文件过大错误
		errMsg := err.Error()
		if strings.Contains(errMsg, "413") || strings.Contains(errMsg, "Request Entity Too Large") {
			errMsg = fmt.Sprintf("文件太大 (%s)，请增加 nginx 的 client_max_body_size 配置", formatSize(int64(zipSize)))
		}

		c.statusBar.Set(fmt.Sprintf("上传失败: %s", errMsg))
		c.app.SendNotification(&fyne.Notification{
			Title:   "存档同步",
			Content: "上传失败: " + errMsg,
		})
		return
	}

	// 记录最后一次上传信息，用于处理游戏退出时的自动保存
	c.lastUploadedSave = save
	c.lastUploadTime = time.Now()

	msg := fmt.Sprintf("已备份: %s", folderName)
	c.statusBar.Set(msg)
	log.Printf("上传成功: %s\n", save.ID)

	c.app.SendNotification(&fyne.Notification{
		Title:   "BG3 存档同步",
		Content: msg,
	})
}

func (c *Client) monitorGameProcess(label *widget.Label) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var running bool
		if runtime.GOOS == "windows" {
			running = isProcessRunning("bg3.exe") || isProcessRunning("bg3_dx11.exe")
		} else if runtime.GOOS == "darwin" {
			// macOS 上 BG3 的进程名（可能需要根据实际情况调整）
			running = isProcessRunning("Baldur's Gate 3")
		} else {
			// Linux
			running = isProcessRunning("bg3") || isProcessRunning("bg3.bin")
		}

		if running != c.gameRunning {
			c.gameRunning = running
			if running {
				label.SetText("游戏状态: 运行中")
			} else {
				label.SetText("游戏状态: 未运行")
				// 游戏退出时，删除最近10秒内上传的存档（这是游戏的自动保存）
				if c.lastUploadedSave != nil && time.Since(c.lastUploadTime) < 10*time.Second {
					go c.deleteLastAutoSave()
				}
				// 游戏退出时，询问是否下载最新存档
				if c.config.AutoSync {
					c.checkForNewerSaves()
				}
			}
		}
	}
}

func (c *Client) checkForNewerSaves() {
	// 检查是否开启自动恢复
	if !c.config.AutoRestore {
		return
	}

	go func() {
		// 等待15秒，确保删除自动保存的操作完成
		time.Sleep(15 * time.Second)

		log.Printf("检查云端最新存档...\n")
		c.statusBar.Set("正在检查云端存档...")

		// 获取云端最新存档
		ctx := context.Background()
		latestSave, err := c.api.GetLatestSave(ctx)
		if err != nil {
			log.Printf("获取云端最新存档失败: %v\n", err)
			c.statusBar.Set(fmt.Sprintf("获取云端存档失败: %v", err))
			return
		}

		// 检查是否是刚才被删除的存档（通过比较ID）
		if c.lastUploadedSave != nil && latestSave.ID == c.lastUploadedSave.ID {
			log.Printf("云端最新存档是刚才删除的自动保存，跳过恢复\n")
			c.statusBar.Set("无需恢复")
			return
		}

		log.Printf("发现云端存档: %s (%s)\n", latestSave.FileName, latestSave.Timestamp.Format("2006-01-02 15:04:05"))

		// 自动下载并恢复
		c.statusBar.Set("正在自动恢复云端存档...")
		data, err := c.api.DownloadSave(ctx, latestSave.ID)
		if err != nil {
			log.Printf("下载云端存档失败: %v\n", err)
			c.statusBar.Set(fmt.Sprintf("下载失败: %v", err))
			return
		}

		// 解压到本地
		folderName := strings.TrimSuffix(latestSave.FileName, ".zip")
		saveFolderPath := filepath.Join(c.config.SavePath, folderName)

		// 先删除旧文件夹（如果存在）
		os.RemoveAll(saveFolderPath)

		// 解压
		if err := unzipToFolder(data, saveFolderPath); err != nil {
			log.Printf("解压失败: %v\n", err)
			c.statusBar.Set(fmt.Sprintf("解压失败: %v", err))
			return
		}

		msg := fmt.Sprintf("已自动恢复云端存档: %s", folderName)
		c.statusBar.Set(msg)
		log.Printf("%s\n", msg)

		// 发送通知
		c.app.SendNotification(&fyne.Notification{
			Title:   "BG3 存档同步",
			Content: msg,
		})
	}()
}

func (c *Client) deleteLastAutoSave() {
	if c.lastUploadedSave == nil {
		return
	}

	saveID := c.lastUploadedSave.ID
	fileName := c.lastUploadedSave.FileName
	timeSinceUpload := time.Since(c.lastUploadTime)

	log.Printf("检测到游戏退出，删除最近上传的自动保存: %s (上传于 %.1f 秒前)\n",
		fileName, timeSinceUpload.Seconds())

	ctx := context.Background()
	if err := c.api.DeleteSave(ctx, saveID); err != nil {
		log.Printf("删除自动保存失败: %v\n", err)
		c.statusBar.Set(fmt.Sprintf("删除自动保存失败: %v", err))
		return
	}

	log.Printf("已删除游戏退出时的自动保存: %s\n", fileName)
	c.statusBar.Set("已删除游戏退出时的自动保存")

	// 清空记录
	c.lastUploadedSave = nil
}

func (c *Client) manualSync() {
	c.statusBar.Set("正在手动同步...")
	// 扫描所有 UUID 文件夹并上传
	go func() {
		entries, err := os.ReadDir(c.config.SavePath)
		if err != nil {
			log.Printf("读取存档目录失败: %v\n", err)
			c.statusBar.Set(fmt.Sprintf("读取目录失败: %v", err))
			return
		}

		count := 0
		for _, entry := range entries {
			if entry.IsDir() {
				// 只处理 __HonourMode 后缀的文件夹
				if !strings.HasSuffix(entry.Name(), "__HonourMode") {
					continue
				}

				folderPath := filepath.Join(c.config.SavePath, entry.Name())
				c.handleSaveFolder(folderPath)
				count++
				time.Sleep(500 * time.Millisecond)
			}
		}
		c.statusBar.Set(fmt.Sprintf("手动同步完成，已上传 %d 个存档", count))
	}()
}

func (c *Client) showSettings() {
	win := c.app.NewWindow("设置")
	win.Resize(fyne.NewSize(500, 400))

	// 配置项
	nebulaURL := widget.NewEntry()
	nebulaURL.SetText(c.config.NebulaURL)

	savePath := widget.NewEntry()
	savePath.SetText(c.config.SavePath)

	browseBtn := widget.NewButton("浏览...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if uri != nil {
				savePath.SetText(uri.Path())
			}
		}, win)
	})

	autoUpload := widget.NewCheck("游戏运行时自动上传", nil)
	autoUpload.SetChecked(c.config.AutoUpload)

	autoRestore := widget.NewCheck("游戏退出后自动恢复云端存档", nil)
	autoRestore.SetChecked(c.config.AutoRestore)

	// 保存按钮
	saveBtn := widget.NewButton("保存", func() {
		c.config.NebulaURL = nebulaURL.Text
		c.config.SavePath = savePath.Text
		c.config.AutoUpload = autoUpload.Checked
		c.config.AutoRestore = autoRestore.Checked

		if err := saveConfig(c.config); err != nil {
			dialog.ShowError(err, win)
			return
		}

		dialog.ShowInformation("成功", "设置已保存", win)
		win.Close()
	})

	form := container.NewVBox(
		widget.NewLabel("Nebula 服务器地址:"),
		nebulaURL,
		widget.NewLabel(""),
		widget.NewLabel("存档路径:"),
		container.NewBorder(nil, nil, nil, browseBtn, savePath),
		widget.NewLabel(""),
		autoUpload,
		autoRestore,
		widget.NewLabel(""),
		saveBtn,
	)

	win.SetContent(container.NewPadded(form))
	win.Show()
}
