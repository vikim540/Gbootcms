package content

import (
	"fmt"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ─── 緩存機制 ──────────────────────────────────────────────────
// 媒體庫掃描結果緩存。
// 設計原則：緩存盡可能短（5 分鐘），並在所有改變文件引用的寫操作後自動失效。

type mediaCacheData struct {
	Files       []MediaFile
	UsedPaths   map[string]bool
	MarkedPaths map[string]bool
	ScanTime    time.Time
	Scanning    bool // 是否正在掃描中
}

var (
	mediaCache   *mediaCacheData
	mediaCacheMu sync.RWMutex
	// cacheTTL 縮短為 5 分鐘，避免長時間持有過時的「已用」狀態
	cacheTTL = 5 * time.Minute
)

// isDirty 檢查緩存是否被標記為臟（轉發到 common package 的全局標記）
func isDirty() bool {
	return common.IsMediaCacheDirty()
}

// clearDirty 清除臟標記
func clearDirty() {
	common.ClearMediaCacheDirty()
}

// getCache 獲取緩存數據，如果過期、為空、或被標記為臟則返回 nil
func getCache() *mediaCacheData {
	mediaCacheMu.RLock()
	defer mediaCacheMu.RUnlock()
	if mediaCache == nil || mediaCache.Scanning {
		return nil
	}
	// 任何寫操作後都會標記臟，強制重掃
	if isDirty() {
		return nil
	}
	if time.Since(mediaCache.ScanTime) > cacheTTL {
		return nil
	}
	return mediaCache
}

// ensureCache 確保緩存存在，不存在則同步掃描（首次訪問或被標記為臟時觸發）
func ensureCache() *mediaCacheData {
	validateFileRefs()
	if c := getCache(); c != nil {
		return c
	}
	// 首次訪問、或被標記為臟、或過期 → 重新掃描
	return doScan()
}

// refreshCache 強制刷新緩存（手動觸發）
func refreshCache() *mediaCacheData {
	clearDirty()
	return doScan()
}

// invalidateCache 使緩存失效（清理文件後觸發）
func invalidateCache() {
	mediaCacheMu.Lock()
	defer mediaCacheMu.Unlock()
	mediaCache = nil
	common.MarkMediaCacheDirty()
}

// doScan 執行實際掃描，帶超時保護
func doScan() *mediaCacheData {
	mediaCacheMu.Lock()
	// 標記正在掃描
	if mediaCache != nil {
		mediaCache.Scanning = true
	} else {
		mediaCache = &mediaCacheData{Scanning: true}
	}
	mediaCacheMu.Unlock()

	// 帶超時保護的掃描（最多 120 秒）
	type scanResult struct {
		files   []MediaFile
		used    map[string]bool
		marked  map[string]bool
		elapsed time.Duration
	}
	ch := make(chan scanResult, 1)
	go func() {
		start := time.Now()
		files := scanFiles()
		used := getUsedPaths()
		marked := getMarkedPaths()
		ch <- scanResult{files, used, marked, time.Since(start)}
	}()

	select {
	case r := <-ch:
		mediaCacheMu.Lock()
		mediaCache = &mediaCacheData{
			Files:       r.files,
			UsedPaths:   r.used,
			MarkedPaths: r.marked,
			ScanTime:    time.Now(),
		}
		mediaCacheMu.Unlock()
		clearDirty() // 掃描完成，清除臟標記
		return mediaCache
	case <-time.After(120 * time.Second):
		mediaCacheMu.Lock()
		mediaCache.Scanning = false
		mediaCacheMu.Unlock()
		return nil // 超時，返回空
	}
}

type MediaController struct {
	common.BaseController
}

// MediaFile 媒體文件信息
type MediaFile struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	SizeStr  string `json:"size_str"`
	ModTime  string `json:"mod_time"`
	Used     bool   `json:"used"`
	Marked   bool   `json:"marked"`
	Category string `json:"category"` // image, document, video, other
}

// ─── 檔案引用欄位定義（getUsedPaths + findUsages 的唯壹資料來源）──────
type refColumn struct {
	column string // 列名
	label  string // 人類可讀描述（findUsages 顯示用）
}

type refTable struct {
	table   string      // 完整表名（含 ay_ 前綴）
	idCol   string      // ID 列名
	nameCol string      // 顯示名稱列名（title / name）
	columns []refColumn // 文件引用列列表
}

// fileRefs 是整個媒體庫中「哪些欄位可能含有文件路徑」的唯壹權威定義。
// getUsedPaths() 和 findUsages() 都從這裏讀取，修改欄位只需改此壹處。
// 注意：nameCol 必須與實際 DB 欄位名一致（ay_slide 用 title 非 name，ay_member 用 username 非 name）。
// 新增含文件引用的表時，必須同步更新 core/mediaplugin/dirty.go 的 MediaReferencingTables 白名單。
var fileRefs = []refTable{
	{"ay_content", "id", "title", []refColumn{{"ico", "ico(封面)"}, {"pics", "pics(多圖)"}, {"enclosure", "enclosure(附件)"}}},
	{"ay_content_sort", "id", "name", []refColumn{{"ico", "ico(圖標)"}, {"pic", "pic(圖片)"}}},
	{"ay_slide", "id", "title", []refColumn{{"pic", "pic(輪播圖)"}, {"pic_mobile", "pic_mobile(移動端)"}}},
	{"ay_link", "id", "name", []refColumn{{"logo", "logo(Logo)"}}},
	{"ay_company", "id", "name", []refColumn{{"weixin", "weixin(微信)"}, {"blicense", "blicense(證照)"}}},
	{"ay_site", "id", "name", []refColumn{{"logo", "logo(Logo)"}}},
	{"ay_member", "id", "username", []refColumn{{"headpic", "headpic(頭像)"}}},
}

// validateOnce 確保 PRAGMA 校驗只運行壹次
var validateOnce sync.Once

// validateFileRefs 啟動校驗：PRAGMA table_info 檢查 fileRefs 中所有表-列是否存在
// 校驗範圍包含 idCol、nameCol 和所有文件引用列，確保 nameCol 與實際 DB 欄位名一致
func validateFileRefs() {
	validateOnce.Do(func() {
		for _, rt := range fileRefs {
			type colInfo struct {
				Name string `gorm:"column:name"`
			}
			var actualCols []colInfo
			if err := model.DB.Raw(fmt.Sprintf("PRAGMA table_info(`%s`)", rt.table)).Scan(&actualCols).Error; err != nil {
				slog.Warn("[MediaController] 無法查詢表結構", "table", rt.table, "err", err)
				continue
			}
			actualSet := make(map[string]bool, len(actualCols))
			for _, c := range actualCols {
				actualSet[c.Name] = true
			}
			// 校驗 idCol
			if !actualSet[rt.idCol] {
				slog.Warn("[MediaController] idCol 在數據庫中不存在", "table", rt.table, "idCol", rt.idCol)
			}
			// 校驗 nameCol
			if !actualSet[rt.nameCol] {
				slog.Warn("[MediaController] nameCol 在數據庫中不存在", "table", rt.table, "nameCol", rt.nameCol)
			}
			// 校驗文件引用列
			for _, c := range rt.columns {
				if !actualSet[c.column] {
					slog.Warn("[MediaController] 檔案引用欄位在數據庫中不存在", "table", rt.table, "column", c.column, "label", c.label)
				}
			}
		}
	})
}

// Index 媒體庫頁面（僅渲染統計外殼，文件列表由 AJAX 加載）
func (c *MediaController) Index(ctx *gin.Context) {
	cache := ensureCache()

	total := 0
	totalSize := int64(0)
	usedCount, unusedCount, markedCount := 0, 0, 0
	if cache != nil {
		total = len(cache.Files)
		for _, f := range cache.Files {
			np := normalizePath(f.Path)
			if np == "" {
				continue
			}
			isUsed := cache.UsedPaths[np] || cache.UsedPaths["/"+np]
			isMarked := cache.MarkedPaths[np]
			totalSize += f.Size
			if isMarked {
				markedCount++
			} else if isUsed {
				usedCount++
			} else {
				unusedCount++
			}
		}
	}

	common.Render(ctx, "content/media.html", gin.H{
		"Total":       total,
		"TotalSize":   formatSize(totalSize),
		"UsedCount":   usedCount,
		"UnusedCount": unusedCount,
		"MarkedCount": markedCount,
		"CacheTime": func() string {
			if cache != nil {
				return cache.ScanTime.Format("2006-01-02 15:04")
			}
			return "未掃描"
		}(),
	})
}

// List 分頁 JSON API（支持篩選 + 搜索）
func (c *MediaController) List(ctx *gin.Context) {
	page, pageSize, offset := c.Paginate(ctx)
	filter := ctx.DefaultQuery("filter", "all")
	search := strings.ToLower(ctx.DefaultQuery("search", ""))

	cache := ensureCache()
	if cache == nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 1, "data": gin.H{
			"files": []MediaFile{}, "total": 0, "page": 1,
			"pagesize": pageSize, "total_pages": 1,
		}})
		return
	}

	// Enrich files with used/marked/category
	var items []MediaFile
	for _, f := range cache.Files {
		np := normalizePath(f.Path)
		if np == "" {
			continue
		}
		f.Used = cache.UsedPaths[np] || cache.UsedPaths["/"+np]
		f.Marked = cache.MarkedPaths[np]
		f.Category = getCategory(f.Name)
		items = append(items, f)
	}

	// Filter
	if filter != "all" {
		var filtered []MediaFile
		for _, f := range items {
			switch filter {
			case "image", "document", "video":
				if f.Category == filter {
					filtered = append(filtered, f)
				}
			case "unused":
				if !f.Used && !f.Marked {
					filtered = append(filtered, f)
				}
			case "marked":
				if f.Marked {
					filtered = append(filtered, f)
				}
			}
		}
		items = filtered
	}

	// Search (filename substring match)
	if search != "" {
		var matched []MediaFile
		for _, f := range items {
			if strings.Contains(strings.ToLower(f.Name), search) {
				matched = append(matched, f)
			}
		}
		items = matched
	}

	// Sort by mod_time desc
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModTime > items[j].ModTime
	})

	// Pagination
	total := len(items)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if offset > total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	pageItems := items[offset:end]

	ctx.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": gin.H{
			"files":       pageItems,
			"total":       total,
			"page":        page,
			"pagesize":    pageSize,
			"total_pages": totalPages,
		},
	})
}

// Mark 標記/取消標記文件（GORM model）
func (c *MediaController) Mark(ctx *gin.Context) {
	path := ctx.PostForm("path")
	if path == "" {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Path required"})
		return
	}

	// 標準化路徑，確保與 scanFiles 的路徑格式一致
	np := normalizePath(path)
	if np == "" {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "無效的文件路徑"})
		return
	}

	var mark content.MediaMark
	if err := model.DB.WithContext(ctx.Request.Context()).Where("path = ?", np).First(&mark).Error; err == nil {
		if err := model.DB.WithContext(ctx.Request.Context()).Delete(&mark).Error; err != nil {
			c.JSONFail(ctx, "取消標記失敗："+err.Error())
			return
		}
		c.JSONOKMsg(ctx, "已取消標記")
	} else {
		if err := model.DB.WithContext(ctx.Request.Context()).Create(&content.MediaMark{Path: np}).Error; err != nil {
			c.JSONFail(ctx, "標記失敗："+err.Error())
			return
		}
		c.JSONOKMsg(ctx, "已標記為保護")
	}
}

// Clean 清理未使用的文件（移至 static/backup/media/，保留目錄結構）
func (c *MediaController) Clean(ctx *gin.Context) {
	force := ctx.PostForm("force") == "1"
	cache := ensureCache()
	if cache == nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "掃描數據不可用"})
		return
	}

	backupDir := filepath.Join("static", "backup", "media")
	os.MkdirAll(backupDir, 0755)

	deleted := 0
	skipped := 0
	var errors []string

	for _, f := range cache.Files {
		np := normalizePath(f.Path)
		if np == "" {
			continue
		}
		isUsed := cache.UsedPaths[np] || cache.UsedPaths["/"+np]
		isMarked := cache.MarkedPaths[np]

		if isUsed {
			continue
		}
		if isMarked && !force {
			skipped++
			continue
		}

		fullPath := filepath.Join("static", strings.TrimPrefix(np, "static/"))

		// 在備份目錄下重建相同的子目錄結構（如 backup/media/upload/202606/xxx.jpg）
		relPath := strings.TrimPrefix(np, "static/")
		dstPath := filepath.Join(backupDir, relPath)
		os.MkdirAll(filepath.Dir(dstPath), 0755)

		if err := os.Rename(fullPath, dstPath); err != nil {
			slog.Error("[MediaController] 清理文件失敗", "file", f.Name, "src", fullPath, "dst", dstPath, "err", err)
			errors = append(errors, fmt.Sprintf("%s: %s", f.Name, err.Error()))
		} else {
			deleted++
		}
	}

	// 清理後使緩存失效，下次訪問重新掃描
	if deleted > 0 {
		invalidateCache()
	}

	msg := fmt.Sprintf("已清理 %d 個文件（已移至回收站）", deleted)
	if skipped > 0 {
		msg += fmt.Sprintf("，跳過 %d 個已標記文件", skipped)
	}
	if len(errors) > 0 {
		msg += fmt.Sprintf("，%d 個失敗", len(errors))
	}

	c.JSONOKMsg(ctx, msg)
}

// BackupList 回收站文件列表
func (c *MediaController) BackupList(ctx *gin.Context) {
	backupDir := filepath.Join("static", "backup", "media")
	type BackupFile struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Size    int64  `json:"size"`
		SizeStr string `json:"size_str"`
		ModTime string `json:"mod_time"`
	}
	var files []BackupFile
	filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(backupDir, path)
		files = append(files, BackupFile{
			Name:    info.Name(),
			Path:    filepath.ToSlash(rel),
			Size:    info.Size(),
			SizeStr: formatSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		return nil
	})

	total := len(files)
	totalSize := int64(0)
	for _, f := range files {
		totalSize += f.Size
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": gin.H{
			"files":      files,
			"total":      total,
			"total_size": formatSize(totalSize),
		},
	})
}

// Restore 從回收站還原文件
//
// 路徑約定（與 Clean/BackupList 保持一致）：
//   - backupDir = static/backup/media/
//   - 備份時保留 static/ 下的相對結構，所以 backup 中的路徑形如 upload/202606/xxx.jpg
//   - 還原時直接拼回 static/ 即可：static/upload/202606/xxx.jpg
func (c *MediaController) Restore(ctx *gin.Context) {
	relPath := ctx.PostForm("path")
	if relPath == "" {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "缺少文件路徑"})
		return
	}

	backupDir := filepath.Join("static", "backup", "media")
	srcPath := filepath.Join(backupDir, filepath.FromSlash(relPath))

	// 防止路徑穿越
	if !strings.HasPrefix(filepath.Clean(srcPath), filepath.Clean(backupDir)) {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "無效的路徑"})
		return
	}

	if _, err := os.Stat(srcPath); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "文件不存在於回收站"})
		return
	}

	// 還原到原始位置：relPath 已含 upload/ 前綴，直接拼 static/
	dstPath := filepath.Join("static", filepath.FromSlash(relPath))
	os.MkdirAll(filepath.Dir(dstPath), 0755)

	if err := os.Rename(srcPath, dstPath); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "還原失敗：" + err.Error()})
		return
	}

	invalidateCache()
	c.JSONOKMsg(ctx, "文件已還原")
}

// BackupClear 清空回收站（永久刪除）
func (c *MediaController) BackupClear(ctx *gin.Context) {
	backupDir := filepath.Join("static", "backup", "media")
	count := 0
	filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if os.Remove(path) == nil {
			count++
		}
		return nil
	})

	// 清理空目錄
	filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == backupDir {
			return nil
		}
		os.Remove(path) // 僅刪除空目錄
		return nil
	})

	c.JSONOKMsg(ctx, fmt.Sprintf("回收站已清空，共刪除 %d 個文件", count))
}

// Refresh 手動刷新掃描緩存（耗時操作，前端需彈窗確認）
func (c *MediaController) Refresh(ctx *gin.Context) {
	result := refreshCache()
	if result == nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "掃描超時，請稍後重試"})
		return
	}

	totalSize := int64(0)
	for _, f := range result.Files {
		totalSize += f.Size
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code": 1,
		"msg":  fmt.Sprintf("掃描完成，共 %d 個文件", len(result.Files)),
		"data": gin.H{
			"total":      len(result.Files),
			"total_size": formatSize(totalSize),
			"scan_time":  result.ScanTime.Format("2006-01-02 15:04:05"),
		},
	})
}

// UsageInfo 文件使用位置信息
type UsageInfo struct {
	Table string `json:"table"` // 表名（ay_content 等）
	ID    int    `json:"id"`    // 記錄 ID
	Name  string `json:"name"`  // 記錄標題
	Field string `json:"field"` // 使用的字段名
}

// Detail 文件詳細信息 API（圖片尺寸 + 使用位置）
func (c *MediaController) Detail(ctx *gin.Context) {
	path := ctx.Query("path")
	if path == "" {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "缺少文件路徑"})
		return
	}

	np := normalizePath(path)
	if np == "" {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "無效的文件路徑"})
		return
	}

	// 安全檢查：只允許訪問 static/upload/ 目錄下的文件
	// 用正斜線統一比較，避免 Windows 反斜線差異
	absPath, _ := filepath.Abs(np)
	uploadRoot, _ := filepath.Abs(filepath.Join("static", "upload"))
	absPathSlash := filepath.ToSlash(absPath) + "/"
	uploadRootSlash := filepath.ToSlash(uploadRoot) + "/"
	if !strings.HasPrefix(absPathSlash, uploadRootSlash) {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "拒絕訪問此路徑"})
		return
	}

	fullPath := np

	// 基礎信息
	info, err := os.Stat(fullPath)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "文件不存在"})
		return
	}

	result := gin.H{
		"name":     info.Name(),
		"path":     np,
		"size":     info.Size(),
		"size_str": formatSize(info.Size()),
		"mod_time": info.ModTime().Format("2006-01-02 15:04:05"),
		"category": getCategory(info.Name()),
		"ext":      strings.ToLower(filepath.Ext(info.Name())),
	}

	// 圖片尺寸
	width, height := getImageDimension(fullPath)
	if width > 0 {
		result["width"] = width
		result["height"] = height
		result["dimension"] = fmt.Sprintf("%d × %d px", width, height)
	}

	// MIME 類型
	if mt := getMime(info.Name()); mt != "" {
		result["mime"] = mt
	}

	// 使用位置
	usages := findUsages(np)
	result["usages"] = usages
	result["usage_count"] = len(usages)

	ctx.JSON(http.StatusOK, gin.H{"code": 1, "data": result})
}

// getImageDimension 讀取圖片寬高
func getImageDimension(path string) (int, int) {
	ext := strings.ToLower(filepath.Ext(path))
	if fileTypes[ext].category != "image" {
		return 0, 0
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// findUsages 查找文件在數據庫中的使用位置
func findUsages(filePath string) []UsageInfo {
	var usages []UsageInfo
	np := normalizePath(filePath)
	base := strings.TrimPrefix(np, "static/")

	for _, rt := range fileRefs {
		// 構建 SELECT: idCol, nameCol, col1, col2, ...
		colCount := 2 + len(rt.columns)
		colNames := make([]string, 0, colCount)
		colNames = append(colNames, rt.idCol, rt.nameCol)
		for _, c := range rt.columns {
			colNames = append(colNames, c.column)
		}
		selectStr := strings.Join(colNames, ", ")

		rows, err := model.DB.Table(rt.table).Select(selectStr).Rows()
		if err != nil {
			continue
		}

		for rows.Next() {
			vals := make([]*string, colCount)
			ptrs := make([]interface{}, colCount)
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			rows.Scan(ptrs...)

			if vals[0] == nil || vals[1] == nil {
				continue
			}
			id, _ := strconv.Atoi(*vals[0])
			name := *vals[1]

			for i, c := range rt.columns {
				if vals[i+2] != nil && pathMatchField(*vals[i+2], np, base) {
					usages = append(usages, UsageInfo{rt.table, id, name, c.label})
				}
			}
		}
		rows.Close()
	}

	// 特殊處理：ay_content.content 正文中的 HTML img src
	type ContentRow struct {
		ID      int
		Title   string
		Content string
	}
	var contents []ContentRow
	model.DB.Table("ay_content").Select("id, title, content").Find(&contents)
	for _, row := range contents {
		if containsImgSrc(row.Content, np, base) {
			usages = append(usages, UsageInfo{"ay_content", row.ID, row.Title, "content(正文)"})
		}
	}

	// 特殊處理：ay_label.value 中的 HTML img src（自定義標籤可能含圖片）
	type LabelRow struct {
		ID    int
		Name  string
		Value string
	}
	var labels []LabelRow
	model.DB.Table("ay_label").Select("id, name, value").Find(&labels)
	for _, row := range labels {
		decoded := strings.ReplaceAll(row.Value, "&quot;", "\"")
		if containsImgSrc(decoded, np, base) {
			usages = append(usages, UsageInfo{"ay_label", row.ID, row.Name, "value(標籤值)"})
		}
	}

	return usages
}

// pathMatchField 判斷欄位值是否包含目標路徑
// 逗號分隔的多值欄位（如 ay_content.pics）需逐一拆分比對
func pathMatchField(fieldVal, np, base string) bool {
	if fieldVal == "" {
		return false
	}
	for _, p := range strings.Split(fieldVal, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		fv := normalizePath(p)
		if fv == np || fv == base || fv == "/"+np || fv == "/"+base {
			return true
		}
	}
	return false
}

// containsImgSrc 判斷 HTML 內容中是否引用了目標圖片
func containsImgSrc(html, np, base string) bool {
	if html == "" {
		return false
	}
	// 簡單搜索路徑片段
	return strings.Contains(html, base) || strings.Contains(html, np)
}

// ─── Helpers ────────────────────────────────────────────────────

// normalizePath 標準化路徑為 static/upload/... 格式（正斜線）
// 包含路徑穿越防護：拒絕包含 ../ 的路徑逃逸到上層目錄
// 注意：不使用 filepath.Clean，因為它在 Windows 上會將正斜線轉為反斜線，
// 導致 map key 不匹配。純字串操作確保跨平台行為一致。
func normalizePath(path string) string {
	if path == "" {
		return ""
	}
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "./")
	// 路徑穿越防護：拒絕包含 ../ 的路徑（攻擊者可能用各種變體如 ..\ 或 ..%2f）
	if strings.Contains(path, "..") {
		return ""
	}
	return path
}

// scanFiles 掃描 static/upload/ 下所有上傳文件
func scanFiles() []MediaFile {
	var files []MediaFile
	uploadDir := filepath.Join("static", "upload")
	filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		files = append(files, MediaFile{
			Name:    info.Name(),
			Path:    normalizePath(path),
			Size:    info.Size(),
			SizeStr: formatSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		return nil
	})
	return files
}

// getUsedPaths 獲取數據庫中引用的所有文件路徑
func getUsedPaths() map[string]bool {
	used := make(map[string]bool)

	for _, rt := range fileRefs {
		cols := make([]string, len(rt.columns))
		for i, c := range rt.columns {
			cols[i] = c.column
		}
		selectStr := strings.Join(cols, ", ")

		rows, err := model.DB.Table(rt.table).Select(selectStr).Rows()
		if err != nil {
			continue
		}
		for rows.Next() {
			vals := make([]*string, len(rt.columns))
			ptrs := make([]interface{}, len(rt.columns))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			rows.Scan(ptrs...)
			for _, v := range vals {
				if v != nil && *v != "" {
					addPaths(used, *v)
				}
			}
		}
		rows.Close()
	}

	// ay_content.content HTML 中的 img src 引用（特殊處理）
	type ContentHTML struct{ Content string }
	var htmls []ContentHTML
	model.DB.Table("ay_content").Select("content").Find(&htmls)
	for _, row := range htmls {
		extractSrcPaths(row.Content, used)
	}

	// ay_label.value HTML 中的 img src 引用（自定義標籤可能含圖片）
	type LabelHTML struct{ Value string }
	var labels []LabelHTML
	model.DB.Table("ay_label").Select("value").Find(&labels)
	for _, row := range labels {
		// 標籤值可能含 HTML 實體編碼的 &quot;，先解碼再提取
		decoded := strings.ReplaceAll(row.Value, "&quot;", "\"")
		extractSrcPaths(decoded, used)
	}

	return used
}

// extractSrcPaths 從 HTML 內容中提取 img src 引用路徑
func extractSrcPaths(html string, used map[string]bool) {
	idx := 0
	for {
		start := strings.Index(html[idx:], "src=")
		if start == -1 {
			break
		}
		start += idx + 4
		if start >= len(html) {
			break
		}
		quote := html[start]
		if quote != '"' && quote != '\'' {
			idx = start
			continue
		}
		end := strings.IndexByte(html[start+1:], quote)
		if end == -1 {
			break
		}
		src := html[start+1 : start+1+end]
		if strings.Contains(src, "upload/") {
			np := normalizePath(src)
			used[np] = true
			used["/"+np] = true
		}
		idx = start + 1 + end
	}
}

// getMarkedPaths 獲取已標記保護的文件（GORM model）
func getMarkedPaths() map[string]bool {
	marked := make(map[string]bool)
	var marks []content.MediaMark
	model.DB.Find(&marks)
	for _, m := range marks {
		marked[m.Path] = true
	}
	return marked
}

// addPaths 將文件路徑加入 used 集合
// 對齊 PbootCMS PHP 原版 explode(',', $value['pics']) 邏輯：
// 逗號分隔的多值欄位（如 ay_content.pics）需逐一路徑處理
func addPaths(set map[string]bool, val string) {
	if val == "" {
		return
	}
	for _, p := range strings.Split(val, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		np := normalizePath(p)
		if np == "" {
			continue
		}
		set[np] = true
		set["/"+np] = true
	}
}

// fileTypes 是副檔名 → 分類 + MIME 的單一數據源。
// getCategory / getMime / getImageDimension 均從此查詢，改一處即全局生效。
type fileTypeInfo struct {
	category string
	mime     string
}

var fileTypes = map[string]fileTypeInfo{
	".jpg":  {"image", "image/jpeg"},
	".jpeg": {"image", "image/jpeg"},
	".png":  {"image", "image/png"},
	".gif":  {"image", "image/gif"},
	".bmp":  {"image", "image/bmp"},
	".webp": {"image", "image/webp"},
	".avif": {"image", "image/avif"},
	".svg":  {"image", "image/svg+xml"},
	".ico":  {"image", "image/x-icon"},
	".doc":  {"document", "application/msword"},
	".docx": {"document", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	".xls":  {"document", "application/vnd.ms-excel"},
	".xlsx": {"document", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
	".pdf":  {"document", "application/pdf"},
	".txt":  {"document", "text/plain"},
	".csv":  {"document", "text/csv"},
	".mp4":  {"video", "video/mp4"},
	".avi":  {"video", "video/x-msvideo"},
	".mov":  {"video", "video/quicktime"},
	".wmv":  {"video", "video/x-ms-wmv"},
	".flv":  {"video", "video/x-flv"},
	".webm": {"video", "video/webm"},
}

func getCategory(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if info, ok := fileTypes[ext]; ok {
		return info.category
	}
	return "other"
}

func getMime(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	if info, ok := fileTypes[ext]; ok {
		return info.mime
	}
	return ""
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}
