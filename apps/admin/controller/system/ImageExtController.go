package system

import (
	"os"
	"path/filepath"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ImageExtController - Image Extension Management Controller
// Corresponds to PHP: apps/admin/controller/ImageExtController.php
type ImageExtController struct {
	common.BaseController
}

// Index - Image extension page
func (ie *ImageExtController) Index(c *gin.Context) {
	common.Render(c, "system/extimage.html", nil)
}

// CheckDataFile - Check data file
func (ie *ImageExtController) CheckDataFile(c *gin.Context) {
	count := 30
	pageStr := c.Query("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	start := (page - 1) * count

	var dbImages []string

	var contents []model.Content
	model.DB.Limit(2000).Find(&contents)
	imgRe := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)
	for _, ct := range contents {
		if ct.Ico != "" {
			dbImages = append(dbImages, ct.Ico)
		}
		if ct.Pics != "" {
			for _, pic := range strings.Split(ct.Pics, ",") {
				if pic != "" {
					dbImages = append(dbImages, pic)
				}
			}
		}
		matches := imgRe.FindAllStringSubmatch(ct.Content, -1)
		for _, m := range matches {
			if len(m) > 1 && m[1] != "" {
				dbImages = append(dbImages, m[1])
			}
		}
	}

	var sorts []model.ContentSort
	model.DB.Find(&sorts)
	for _, s := range sorts {
		if s.Ico != "" {
			dbImages = append(dbImages, s.Ico)
		}
		if s.Pic != "" {
			dbImages = append(dbImages, s.Pic)
		}
	}

	var slides []model.Slide
	model.DB.Find(&slides)
	for _, s := range slides {
		if s.Pic != "" {
			dbImages = append(dbImages, s.Pic)
		}
	}

	var links []model.Link
	model.DB.Find(&links)
	for _, l := range links {
		if l.Logo != "" {
			dbImages = append(dbImages, l.Logo)
		}
	}

	var sites []model.Site
	model.DB.Find(&sites)
	for _, s := range sites {
		if s.Logo != "" {
			dbImages = append(dbImages, s.Logo)
		}
	}

	dbImageSet := make(map[string]bool)
	for _, img := range dbImages {
		dbImageSet[img] = true
	}

	uploadDir := filepath.Join("static", "upload")
	var fileArr []string
	filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".bmp" || ext == ".ico" || ext == ".svg" || ext == ".webp" || ext == ".avif" {
			relPath := "/" + strings.ReplaceAll(path, "\\", "/")
			fileArr = append(fileArr, relPath)
		}
		return nil
	})

	var difference []string
	for _, f := range fileArr {
		if !dbImageSet[f] {
			difference = append(difference, f)
		}
	}

	total := len(difference)
	end := start + count
	if end > total {
		end = total
	}
	var pageList []gin.H
	for i := start; i < end; i++ {
		pageList = append(pageList, gin.H{
			"real_path":   difference[i],
			"static_path": difference[i],
		})
	}

	c.JSON(200, gin.H{
		"code":  0,
		"msg":   "",
		"count": total,
		"data":  pageList,
	})
}

// DoExt - Execute image extension operation
func (ie *ImageExtController) DoExt(c *gin.Context) {
	extType := c.PostForm("type")
	backupDir := filepath.Join("static", "backup", "ImageExt")
	os.MkdirAll(backupDir, 0755)

	if extType == "0" {
		list := c.PostFormArray("list")
		for _, item := range list {
			if item == "" {
				continue
			}
			srcPath := filepath.Join(".", strings.TrimPrefix(item, "/"))
			if _, err := os.Stat(srcPath); err == nil {
				dstPath := filepath.Join(backupDir, filepath.Base(srcPath))
				os.Rename(srcPath, dstPath)
			}
		}
		ie.JSONOKMsg(c, "Cleaned successfully")
		return
	}

	if extType == "1" {
		var dbImages []string
		var contents []model.Content
		model.DB.Limit(2000).Find(&contents)
		imgRe := regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)
		for _, ct := range contents {
			if ct.Ico != "" {
				dbImages = append(dbImages, ct.Ico)
			}
			if ct.Pics != "" {
				for _, pic := range strings.Split(ct.Pics, ",") {
					if pic != "" {
						dbImages = append(dbImages, pic)
					}
				}
			}
			matches := imgRe.FindAllStringSubmatch(ct.Content, -1)
			for _, m := range matches {
				if len(m) > 1 && m[1] != "" {
					dbImages = append(dbImages, m[1])
				}
			}
		}
		var slides []model.Slide
		model.DB.Find(&slides)
		for _, s := range slides {
			if s.Pic != "" {
				dbImages = append(dbImages, s.Pic)
			}
		}
		var links []model.Link
		model.DB.Find(&links)
		for _, l := range links {
			if l.Logo != "" {
				dbImages = append(dbImages, l.Logo)
			}
		}

		dbImageSet := make(map[string]bool)
		for _, img := range dbImages {
			dbImageSet[img] = true
		}

		uploadDir := filepath.Join("static", "upload")
		filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".bmp" || ext == ".ico" || ext == ".svg" || ext == ".webp" || ext == ".avif" {
				relPath := "/" + strings.ReplaceAll(path, "\\", "/")
				if !dbImageSet[relPath] {
					dstPath := filepath.Join(backupDir, filepath.Base(path))
					os.Rename(path, dstPath)
				}
			}
			return nil
		})
		ie.JSONOKMsg(c, "All cleaned successfully")
		return
	}

	ie.JSONFail(c, "Unknown operation type")
}
