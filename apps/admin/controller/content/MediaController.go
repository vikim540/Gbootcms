package content

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// Index 媒體庫頁面（僅渲染統計外殼，文件列表由 AJAX 加載）
func (c *MediaController) Index(ctx *gin.Context) {
	files := scanFiles()
	usedPaths := getUsedPaths()
	markedPaths := getMarkedPaths()

	totalSize := int64(0)
	usedCount, unusedCount, markedCount := 0, 0, 0
	for _, f := range files {
		np := normalizePath(f.Path)
		f.Used = usedPaths[np] || usedPaths["/"+np]
		f.Marked = markedPaths[np]
		totalSize += f.Size
		if f.Marked {
			markedCount++
		} else if f.Used {
			usedCount++
		} else {
			unusedCount++
		}
	}

	common.Render(ctx, "content/media.html", gin.H{
		"Total":       len(files),
		"TotalSize":   formatSize(totalSize),
		"UsedCount":   usedCount,
		"UnusedCount": unusedCount,
		"MarkedCount": markedCount,
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

	files := scanFiles()
	usedPaths := getUsedPaths()
	markedPaths := getMarkedPaths()

	// Enrich files with used/marked/category
	var items []MediaFile
	for _, f := range files {
		np := normalizePath(f.Path)
		f.Used = usedPaths[np] || usedPaths["/"+np]
		f.Marked = markedPaths[np]
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

// Clean 清理未使用的文件
func (c *MediaController) Clean(ctx *gin.Context) {
	force := ctx.PostForm("force") == "1"
	files := scanFiles()
	usedPaths := getUsedPaths()
	markedPaths := getMarkedPaths()

	deleted := 0
	skipped := 0
	var errors []string

	for _, f := range files {
		np := normalizePath(f.Path)
		isUsed := usedPaths[np] || usedPaths["/"+np]
		isMarked := markedPaths[np]

		if isUsed {
			continue
		}
		if isMarked && !force {
			skipped++
			continue
		}

		fullPath := filepath.Join("static", strings.TrimPrefix(np, "static/"))
		if err := os.Remove(fullPath); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", f.Name, err.Error()))
		} else {
			deleted++
		}
	}

	msg := fmt.Sprintf("已清理 %d 個文件", deleted)
	if skipped > 0 {
		msg += fmt.Sprintf("，跳過 %d 個已標記文件", skipped)
	}
	if len(errors) > 0 {
		msg += fmt.Sprintf("，%d 個失敗", len(errors))
	}

	c.JSONOKMsg(ctx, msg)
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

	// ay_content: ico, pics, enclosure
	type ContentRow struct {
		Ico, Pics, Enclosure string
	}
	var contents []ContentRow
	model.DB.Table("ay_content").Select("ico, pics, enclosure").Find(&contents)
	for _, row := range contents {
		addPaths(used, row.Ico)
		addPaths(used, row.Pics)
		addPaths(used, row.Enclosure)
	}

	// ay_content_sort: ico, pic
	type SortRow struct {
		Ico, Pic string
	}
	var sorts []SortRow
	model.DB.Table("ay_content_sort").Select("ico, pic").Find(&sorts)
	for _, row := range sorts {
		addPaths(used, row.Ico)
		addPaths(used, row.Pic)
	}

	// ay_slide: pic
	type SlideRow struct{ Pic string }
	var slides []SlideRow
	model.DB.Table("ay_slide").Select("pic").Find(&slides)
	for _, row := range slides {
		addPaths(used, row.Pic)
	}

	// ay_link: logo
	type LinkRow struct{ Logo string }
	var links []LinkRow
	model.DB.Table("ay_link").Select("logo").Find(&links)
	for _, row := range links {
		addPaths(used, row.Logo)
	}

	// ay_company: weixin, license
	type CompanyRow struct {
		Weixin, License string
	}
	var companies []CompanyRow
	model.DB.Table("ay_company").Select("weixin, license").Find(&companies)
	for _, row := range companies {
		addPaths(used, row.Weixin)
		addPaths(used, row.License)
	}

	// ay_site: logo
	type SiteRow struct{ Logo string }
	var sites []SiteRow
	model.DB.Table("ay_site").Select("logo").Find(&sites)
	for _, row := range sites {
		addPaths(used, row.Logo)
	}

	// ay_content.content HTML 中的 img src 引用
	type ContentHTML struct{ Content string }
	var htmls []ContentHTML
	model.DB.Table("ay_content").Select("content").Find(&htmls)
	for _, row := range htmls {
		extractSrcPaths(row.Content, used)
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
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".ico":
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
