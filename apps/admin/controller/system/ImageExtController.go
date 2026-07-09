package system

import (
	"os"
	"path/filepath"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/core/acodeplugin"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// ImageExtController - Image Extension Management Controller
// Corresponds to PHP: apps/admin/controller/ImageExtController.php
type ImageExtController struct {
	common.BaseController
}

// Index - 已棄用，重定向到媒體庫
func (ie *ImageExtController) Index(c *gin.Context) {
	c.Redirect(302, "/admin/content/media/index")
}

// CheckDataFile - Check data file
func (ie *ImageExtController) CheckDataFile(c *gin.Context) {
	_, pageSize, offset := ie.Paginate(c)
	scanCtx := acodeplugin.SkipAcode(c.Request.Context())

	var dbImages []string

	var contents []model.Content
	model.DB.WithContext(scanCtx).Limit(2000).Find(&contents)
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
	model.DB.WithContext(scanCtx).Find(&sorts)
	for _, s := range sorts {
		if s.Ico != "" {
			dbImages = append(dbImages, s.Ico)
		}
		if s.Pic != "" {
			dbImages = append(dbImages, s.Pic)
		}
	}

	var slides []model.Slide
	model.DB.WithContext(scanCtx).Find(&slides)
	for _, s := range slides {
		if s.Pic != "" {
			dbImages = append(dbImages, s.Pic)
		}
	}

	var links []model.Link
	model.DB.WithContext(scanCtx).Find(&links)
	for _, l := range links {
		if l.Logo != "" {
			dbImages = append(dbImages, l.Logo)
		}
	}

	var sites []model.Site
	model.DB.WithContext(scanCtx).Find(&sites)
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
	end := offset + pageSize
	if end > total {
		end = total
	}
	var pageList []gin.H
	for i := offset; i < end; i++ {
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
		ie.JSONOKMsg(c, common.NoticeClean)
		return
	}

	if extType == "1" {
		scanCtx := acodeplugin.SkipAcode(c.Request.Context())
		var dbImages []string
		var contents []model.Content
		model.DB.WithContext(scanCtx).Limit(2000).Find(&contents)
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
		model.DB.WithContext(scanCtx).Find(&slides)
		for _, s := range slides {
			if s.Pic != "" {
				dbImages = append(dbImages, s.Pic)
			}
		}
		var links []model.Link
		model.DB.WithContext(scanCtx).Find(&links)
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
		ie.JSONOKMsg(c, common.NoticeCleanAll)
		return
	}

	ie.JSONFail(c, "Unknown operation type")
}
