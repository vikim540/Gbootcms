package content

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
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
	Files      []MediaFile
	UsedPaths  map[string]bool
	MarkedPaths map[string]bool
	ScanTime   time.Time
	Scanning   bool // 是否正在掃描中
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
var fileRefs = []refTable{
	{"ay_content",      "id", "title", []refColumn{{"ico", "ico(封面)"}, {"pics", "pics(多圖)"}, {"enclosure", "enclosure(附件)"}}},
	{"ay_content_sort", "id", "name",  []refColumn{{"ico", "ico(圖標)"}, {"pic", "pic(圖片)"}}},
	{"ay_slide",        "id", "name",  []refColumn{{"pic", "pic(輪播圖)"}, {"pic_mobile", "pic_mobile(移動端)"}}},
	{"ay_link",         "id", "name",  []refColumn{{"logo", "logo(Logo)"}}},
	{"ay_company",      "id", "name",  []refColumn{{"weixin", "weixin(微信)"}, {"blicense", "blicense(證照)"}}},
	{"ay_site",         "id", "name",  []refColumn{{"logo", "logo(Logo)"}}},
	{"ay_member",       "id", "name",  []refColumn{{"headpic", "headpic(頭像)"}}},
}

// validateOnce 確保 PRAGMA 校驗只運行壹次
var validateOnce sync.Once

// validateFileRefs 啟動校驗：PRAGMA table_info 檢查 fileRefs 中所有表-列是否存在
func validateFileRefs() {
	validateOnce.Do(func() {
		byTable := make(map[string][]refColumn)
		for _, rt := range fileRefs {
			byTable[rt.table] = rt.columns
		}
		for tableName, cols := range byTable {
			type colInfo struct {
				Name string `gorm:"column:name"`
			}
			var actualCols []colInfo
			if err := model.DB.Raw(fmt.Sprintf("PRAGMA table_info(`%s`)", tableName)).Scan(&actualCols).Error; err != nil {
				log.Printf("[WARN] MediaController: 無法查詢 %s: %v", tableName, err)
				continue
			}
			actualSet := make(map[string]bool, len(actualCols))
			for _, c := range actualCols {
				actualSet[c.Name] = true
			}
			for _, c := range cols {
				if !actualSet[c.column] {
					log.Printf("[WARN] MediaController: 檔案引用欄位 %s.%s（%s）在數據庫中不存在", tableName, c.column, c.label)
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
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pagesize", "40"))
	filter := ctx.DefaultQuery("filter", "all")
	search := strings.ToLower(ctx.DefaultQuery("search", ""))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 40
	}

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
	offset := (page - 1) * pageSize
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

	var mark content.MediaMark
	if err := model.DB.Where("path = ?", path).First(&mark).Error; err == nil {
		model.DB.Delete(&mark)
		c.JSONOKMsg(ctx, "已取消標記")
	} else {
		model.DB.Create(&content.MediaMark{Path: path})
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
			"files":     files,
			"total":     total,
			"total_size": formatSize(totalSize),
		},
	})
}

// Restore 從回收站還原文件
func (c *MediaController) Restore(ctx *gin.Context) {
	path := ctx.PostForm("path")
	if path == "" {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "缺少文件路徑"})
		return
	}

	backupDir := filepath.Join("static", "backup", "media")
	srcPath := filepath.Join(backupDir, filepath.FromSlash(path))

	// 防止路徑穿越
	if !strings.HasPrefix(filepath.Clean(srcPath), filepath.Clean(backupDir)) {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "無效的路徑"})
		return
	}

	if _, err := os.Stat(srcPath); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "文件不存在於回收站"})
		return
	}

	// 還原到 static/upload/ 下的原始位置
	dstPath := filepath.Join("static", "upload", filepath.FromSlash(path))
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
			"total":       len(result.Files),
			"total_size":  formatSize(totalSize),
			"scan_time":   result.ScanTime.Format("2006-01-02 15:04:05"),
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
	ext := strings.ToLower(filepath.Ext(info.Name()))
	mimeMap := map[string]string{
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".bmp": "image/bmp", ".webp": "image/webp",
		".avif": "image/avif", ".svg": "image/svg+xml", ".ico": "image/x-icon",
		".doc": "application/msword", ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls": "application/vnd.ms-excel", ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".pdf": "application/pdf", ".txt": "text/plain", ".csv": "text/csv",
		".mp4": "video/mp4", ".avi": "video/x-msvideo", ".webm": "video/webm",
	}
	if mt, ok := mimeMap[ext]; ok {
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
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		cfg, _, err := image.DecodeConfig(f)
		if err != nil {
			return 0, 0
		}
		return cfg.Width, cfg.Height
	case ".png":
		f.Seek(0, 0)
		cfg, _, err := image.DecodeConfig(f)
		if err != nil {
			return 0, 0
		}
		return cfg.Width, cfg.Height
	case ".gif":
		f.Seek(0, 0)
		cfg, _, err := image.DecodeConfig(f)
		if err != nil {
			return 0, 0
		}
		return cfg.Width, cfg.Height
	case ".bmp":
		f.Seek(0, 0)
		cfg, _, err := image.DecodeConfig(f)
		if err != nil {
			return 0, 0
		}
		return cfg.Width, cfg.Height
	case ".webp":
		f.Seek(0, 0)
		cfg, _, err := image.DecodeConfig(f)
		if err != nil {
			return 0, 0
		}
		return cfg.Width, cfg.Height
	}
	return 0, 0
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

	return usages
}

// pathMatchField 判斷字段值是否包含目標路徑
func pathMatchField(fieldVal, np, base string) bool {
	if fieldVal == "" {
		return false
	}
	fv := normalizePath(fieldVal)
	return fv == np || fv == base || fv == "/"+np || fv == "/"+base
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

// normalizePath 標準化路徑為 static/upload/... 格式
func normalizePath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "./")
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

func addPaths(set map[string]bool, val string) {
	if val == "" {
		return
	}
	np := normalizePath(val)
	set[np] = true
	set["/"+np] = true
}

func getCategory(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".avif", ".svg", ".ico":
		return "image"
	case ".doc", ".docx", ".xls", ".xlsx", ".pdf", ".txt", ".csv":
		return "document"
	case ".mp4", ".avi", ".mov", ".wmv", ".flv", ".webm":
		return "video"
	default:
		return "other"
	}
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
