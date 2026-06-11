package content

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type MediaController struct {
}

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

// Index 媒體庫列表頁
func (c *MediaController) Index(ctx *gin.Context) {
	files := c.scanFiles()
	usedPaths := c.getUsedPaths()
	markedPaths := c.getMarkedPaths()

	var items []MediaFile
	for _, f := range files {
		relPath := strings.TrimPrefix(f.Path, "static/")
		relPath = "static/" + relPath
		f.Used = usedPaths[relPath] || usedPaths["/"+relPath]
		f.Marked = markedPaths[relPath]
		f.Category = getCategory(f.Name)
		items = append(items, f)
	}

	// Sort by time desc
	sort.Slice(items, func(i, j int) bool {
		return items[i].ModTime > items[j].ModTime
	})

	totalSize := int64(0)
	usedCount, unusedCount, markedCount := 0, 0, 0
	for _, f := range items {
		totalSize += f.Size
		if f.Used {
			usedCount++
		} else {
			unusedCount++
		}
		if f.Marked {
			markedCount++
		}
	}

	common.Render(ctx, "content/media.html", gin.H{
		"Files":       items,
		"Total":       len(items),
		"TotalSize":   formatSize(totalSize),
		"UsedCount":   usedCount,
		"UnusedCount": unusedCount,
		"MarkedCount": markedCount,
		"Formcheck":   ctx.GetString("formcheck"),
	})
}

// Mark 標記/取消標記文件
func (c *MediaController) Mark(ctx *gin.Context) {
	path := ctx.PostForm("path")
	if path == "" {
		ctx.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Path required"})
		return
	}
	// Check if already marked
	var count int64
	model.DB.Table("ay_media_mark").Where("path = ?", path).Count(&count)
	if count > 0 {
		model.DB.Table("ay_media_mark").Where("path = ?", path).Delete(nil)
		ctx.JSON(http.StatusOK, gin.H{"code": 1, "msg": "已取消標記"})
	} else {
		model.DB.Table("ay_media_mark").Create(map[string]interface{}{
			"path":       path,
			"create_time": time.Now(),
		})
		ctx.JSON(http.StatusOK, gin.H{"code": 1, "msg": "已標記為保護"})
	}
}

// Clean 清理未使用的文件
func (c *MediaController) Clean(ctx *gin.Context) {
	force := ctx.PostForm("force") == "1"
	files := c.scanFiles()
	usedPaths := c.getUsedPaths()
	markedPaths := c.getMarkedPaths()

	deleted := 0
	skipped := 0
	var errors []string

	for _, f := range files {
		relPath := strings.TrimPrefix(f.Path, "static/")
		relPath = "static/" + relPath
		isUsed := usedPaths[relPath] || usedPaths["/"+relPath]
		isMarked := markedPaths[relPath]

		if isUsed {
			continue
		}
		if isMarked && !force {
			skipped++
			continue
		}

		fullPath := filepath.Join("static", strings.TrimPrefix(relPath, "static/"))
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

	ctx.JSON(http.StatusOK, gin.H{"code": 1, "msg": msg})
}

// scanFiles 掃描所有上傳文件
func (c *MediaController) scanFiles() []MediaFile {
	var files []MediaFile
	uploadDir := "static/upload"
	filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		files = append(files, MediaFile{
			Name:    info.Name(),
			Path:    filepath.ToSlash(path),
			Size:    info.Size(),
			SizeStr: formatSize(info.Size()),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})
		return nil
	})
	return files
}

// getUsedPaths 獲取數據庫中引用的所有文件路徑
func (c *MediaController) getUsedPaths() map[string]bool {
	used := make(map[string]bool)

	// Scan ay_content fields: ico, pics, enclosure
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

	// Scan ay_content_sort: ico, pic
	type SortRow struct {
		Ico, Pic string
	}
	var sorts []SortRow
	model.DB.Table("ay_content_sort").Select("ico, pic").Find(&sorts)
	for _, row := range sorts {
		addPaths(used, row.Ico)
		addPaths(used, row.Pic)
	}

	// Scan ay_slide: pic
	type SlideRow struct{ Pic string }
	var slides []SlideRow
	model.DB.Table("ay_slide").Select("pic").Find(&slides)
	for _, row := range slides {
		addPaths(used, row.Pic)
	}

	// Scan ay_link: logo
	type LinkRow struct{ Logo string }
	var links []LinkRow
	model.DB.Table("ay_link").Select("logo").Find(&links)
	for _, row := range links {
		addPaths(used, row.Logo)
	}

	// Scan ay_company: weixin, license
	type CompanyRow struct {
		Weixin, License string
	}
	var companies []CompanyRow
	model.DB.Table("ay_company").Select("weixin, license").Find(&companies)
	for _, row := range companies {
		addPaths(used, row.Weixin)
		addPaths(used, row.License)
	}

	// Scan ay_site: logo
	type SiteRow struct{ Logo string }
	var sites []SiteRow
	model.DB.Table("ay_site").Select("logo").Find(&sites)
	for _, row := range sites {
		addPaths(used, row.Logo)
	}

	// Scan content HTML for image src references
	type ContentHTML struct{ Content string }
	var htmls []ContentHTML
	model.DB.Table("ay_content").Select("content").Find(&htmls)
	for _, row := range htmls {
		// Extract src="..." patterns
		idx := 0
		for {
			start := strings.Index(row.Content[idx:], "src=")
			if start == -1 {
				break
			}
			start += idx + 4
			quote := row.Content[start]
			if quote != '"' && quote != '\'' {
				idx = start
				continue
			}
			end := strings.IndexByte(row.Content[start+1:], quote)
			if end == -1 {
				break
			}
			src := row.Content[start+1 : start+1+end]
			if strings.Contains(src, "upload/") {
				used[src] = true
				used["/"+src] = true
			}
			idx = start + 1 + end
		}
	}

	return used
}

// getMarkedPaths 獲取已標記保護的文件
func (c *MediaController) getMarkedPaths() map[string]bool {
	marked := make(map[string]bool)
	// Auto-create table if not exists
	model.DB.Exec(`CREATE TABLE IF NOT EXISTS ay_media_mark (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT NOT NULL UNIQUE,
		create_time DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	type MarkRow struct{ Path string }
	var rows []MarkRow
	model.DB.Table("ay_media_mark").Select("path").Find(&rows)
	for _, r := range rows {
		marked[r.Path] = true
	}
	return marked
}

func addPaths(set map[string]bool, val string) {
	if val == "" {
		return
	}
	set[val] = true
	set["/"+val] = true
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
