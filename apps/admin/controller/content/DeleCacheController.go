package content

import (
	"fmt"
	"os"
	"path/filepath"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DeleCacheController - Cache Update Controller
// Corresponds to PHP: apps/admin/controller/DeleCacheController.php
type DeleCacheController struct {
	common.BaseController
}

// Index - Cache update page
func (dc *DeleCacheController) Index(c *gin.Context) {
	typeStr := c.Query("type")
	idMinStr := c.Query("idzuixiao")
	idMaxStr := c.Query("idzuida")
	scode := c.Query("scode")

	if typeStr != "" {
		cacheDir := filepath.Join("runtime", "cache")
		switch typeStr {
		case "1":
			dc.deleIndex(cacheDir)
			dc.deleSort(cacheDir, "")
			dc.JSONOK(c, common.NoticeCacheHomepage)
		case "2":
			dc.deleSortAll(cacheDir, scode)
			dc.JSONOK(c, common.NoticeCacheSortList)
		case "3":
			dc.deleContent(cacheDir, idMinStr, idMaxStr)
			dc.JSONOK(c, common.NoticeCacheContent)
		default:
			dc.JSONFail(c, "Invalid parameter")
		}
		return
	}

	var sorts []model.ContentSort
	model.DB.Order("sorting ASC, id ASC").Find(&sorts)

	dc.render(c, "content/delecache.html", gin.H{
		"sorts":        sorts,
		"sort_select":  helper.BuildSortSelectHTML(sorts, ""),
	})
}

// render - delegates to common.Render for template rendering
func (dc *DeleCacheController) render(c *gin.Context, tpl string, data gin.H) {
	common.Render(c, tpl, data)
}

func (dc *DeleCacheController) deleIndex(cacheDir string) {
	os.MkdirAll(cacheDir, 0755)
	entries, _ := os.ReadDir(cacheDir)
	for _, entry := range entries {
		if !entry.IsDir() && hasSuffix(entry.Name(), ".html") {
			fullPath := filepath.Join(cacheDir, entry.Name())
			data, err := os.ReadFile(fullPath)
			if err == nil {
				content := string(data)
				if contains(content, "Gbootcms") || contains(content, "ay_") {
					os.Remove(fullPath)
				}
			}
		}
	}
}

func (dc *DeleCacheController) deleSort(cacheDir string, scode string) {
	var scodes []string
	if scode == "" {
		var sorts []model.ContentSort
		model.DB.Where("type IN (1,2)").Find(&sorts)
		for _, s := range sorts {
			scodes = append(scodes, s.Scode)
		}
	} else {
		scodes = append(scodes, scode)
	}

	os.MkdirAll(cacheDir, 0755)
	entries, _ := os.ReadDir(cacheDir)
	for _, entry := range entries {
		if !entry.IsDir() && hasSuffix(entry.Name(), ".html") {
			fullPath := filepath.Join(cacheDir, entry.Name())
			os.Remove(fullPath)
		}
	}
}

func (dc *DeleCacheController) deleSortAll(cacheDir string, scode string) {
	os.MkdirAll(cacheDir, 0755)
	entries, _ := os.ReadDir(cacheDir)
	for _, entry := range entries {
		if !entry.IsDir() && hasSuffix(entry.Name(), ".html") {
			fullPath := filepath.Join(cacheDir, entry.Name())
			os.Remove(fullPath)
		}
	}
}

func (dc *DeleCacheController) deleContent(cacheDir string, idMin string, idMax string) {
	os.MkdirAll(cacheDir, 0755)

	if idMin != "" && idMax != "" {
		minID, _ := strconv.Atoi(idMin)
		maxID, _ := strconv.Atoi(idMax)
		for i := minID; i <= maxID; i++ {
			var content model.Content
			if err := model.DB.First(&content, i).Error; err == nil {
				cacheKey := fmt.Sprintf("%x", []byte(content.URLName+content.Title))
				cacheFile := filepath.Join(cacheDir, cacheKey+".html")
				os.Remove(cacheFile)
			}
		}
	} else {
		entries, _ := os.ReadDir(cacheDir)
		for _, entry := range entries {
			if !entry.IsDir() && hasSuffix(entry.Name(), ".html") {
				fullPath := filepath.Join(cacheDir, entry.Name())
				os.Remove(fullPath)
			}
		}
	}
}

// String helper function
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
