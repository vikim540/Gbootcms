package controller

import (
	"fmt"
	"net/http"
	"strings"

	"gbootcms/apps/admin/model"
	"gbootcms/core/acodeplugin"

	"github.com/gin-gonic/gin"
)

// LLMS 生成 /llms.txt 文件，為 LLM 爬蟲提供站點內容概覽
// 格式規範：https://llmstxt.org/
func (fc *FrontController) LLMS(c *gin.Context) {
	skipCtx := acodeplugin.SkipAcode(c.Request.Context())
	domain := fc.getDomain(c)

	var sb strings.Builder

	// 標題：站點名稱
	var site model.Site
	model.DB.WithContext(skipCtx).First(&site)
	title := site.Title
	if title == "" {
		title = "Gbootcms"
	}
	sb.WriteString("# " + title + "\n\n")

	// 簡介
	if site.Description != "" {
		sb.WriteString("> " + site.Description + "\n\n")
	}

	// 站點基本信息
	sb.WriteString("## 站點資訊\n\n")
	if site.Subtitle != "" {
		sb.WriteString(fmt.Sprintf("- 站點副標題: %s\n", site.Subtitle))
	}
	if site.Keywords != "" {
		sb.WriteString(fmt.Sprintf("- 站點關鍵字: %s\n", site.Keywords))
	}
	sb.WriteString(fmt.Sprintf("- 站點地址: %s/\n", domain))
	sb.WriteString("\n")

	// 主要欄目
	sb.WriteString("## 主要欄目\n\n")
	var sorts []model.ContentSort
	model.DB.WithContext(skipCtx).Where("status = 1 AND pcode = ? OR pcode = ''", "0").
		Order("sorting ASC, id ASC").Find(&sorts)
	if len(sorts) == 0 {
		// 回退：查詢所有啟用的欄目
		model.DB.WithContext(skipCtx).Where("status = 1").
			Order("sorting ASC, id ASC").Find(&sorts)
	}
	for _, s := range sorts {
		if s.Name == "" {
			continue
		}
		sortURL := buildSortURL(&s)
		if sortURL != "" {
			desc := s.Description
			if desc == "" {
				desc = s.Name
			}
			sb.WriteString(fmt.Sprintf("- [%s](%s%s): %s\n", s.Name, domain, sortURL, desc))
		}
	}
	sb.WriteString("\n")

	// 最新內容（取前 20 條）
	sb.WriteString("## 最新內容\n\n")
	var contents []model.Content
	model.DB.WithContext(skipCtx).Where("status = 1").
		Order("date DESC").Limit(20).Find(&contents)
	for _, ct := range contents {
		if ct.Title == "" {
			continue
		}
		contentURL := buildSitemapContentURL(skipCtx, &ct)
		if contentURL != "" {
			desc := ct.Description
			if desc == "" {
				// 截取內容前 100 字
				plain := stripHTMLTags(ct.Content)
				if len(plain) > 100 {
					plain = plain[:100] + "..."
				}
				desc = plain
			}
			sb.WriteString(fmt.Sprintf("- [%s](%s%s): %s\n", ct.Title, domain, contentURL, desc))
		}
	}
	sb.WriteString("\n")

	// 公司聯繫資訊
	var company model.Company
	model.DB.WithContext(skipCtx).First(&company)
	if company.Name != "" {
		sb.WriteString("## 聯繫資訊\n\n")
		sb.WriteString(fmt.Sprintf("- 公司名稱: %s\n", company.Name))
		if company.Address != "" {
			sb.WriteString(fmt.Sprintf("- 地址: %s\n", company.Address))
		}
		if company.Phone != "" {
			sb.WriteString(fmt.Sprintf("- 電話: %s\n", company.Phone))
		}
		if company.Email != "" {
			sb.WriteString(fmt.Sprintf("- 郵箱: %s\n", company.Email))
		}
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, sb.String())
}

// stripHTMLTags 移除 HTML 標籤，返回純文字
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}
