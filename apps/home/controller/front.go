package controller

import (
	"net/http"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common/parser"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FrontController struct {
	Store *parser.TemplateStore
}

func NewFrontController(store *parser.TemplateStore) *FrontController {
	return &FrontController{Store: store}
}

func (fc *FrontController) Index(c *gin.Context) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("index.html")
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) ListPage(c *gin.Context) {
	path := c.Param("path")
	path = trimSuffix(path)

	var sort model.ContentSort
	if err := model.DB.Where("urlname = ?", path).First(&sort).Error; err != nil {
		c.String(http.StatusNotFound, "404")
		return
	}

	ctx := fc.buildContext(c)
	ctx.Sort = &sort
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		ctx.CurrentPage = p
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	tpl := sort.ListTpl
	if tpl == "" {
		tpl = "list.html"
	}
	content := fc.Store.Render(tpl)
	content = p.Render(content)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) ContentPage(c *gin.Context) {
	path := c.Request.URL.Path
	path = strings.TrimPrefix(path, "/")
	path = trimSuffix(path)

	if path == "" {
		fc.Index(c)
		return
	}

	var content model.Content
	if err := model.DB.Where("urlname = ? AND status = 1", path).First(&content).Error; err != nil {
		var sort model.ContentSort
		if err2 := model.DB.Where("urlname = ?", path).First(&sort).Error; err2 != nil {
			c.String(http.StatusNotFound, "404")
			return
		}
		fc.renderSortPage(c, &sort)
		return
	}

	ctx := fc.buildContext(c)
	ctx.Content = &content

	var sort model.ContentSort
	if model.DB.Where("scode = ?", content.Scode).First(&sort).Error == nil {
		ctx.Sort = &sort
	}

	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	tpl := "content.html"
	if ctx.Sort != nil && ctx.Sort.ContentTpl != "" {
		tpl = ctx.Sort.ContentTpl
	}
	html := fc.Store.Render(tpl)
	html = p.Render(html)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (fc *FrontController) Search(c *gin.Context) {
	ctx := fc.buildContext(c)
	ctx.Keyword = c.Query("keyword")
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("search.html")
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) Tags(c *gin.Context) {
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("tags.html")
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) Message(c *gin.Context) {
	if c.Request.Method == "POST" {
		msg := model.Message{
			Contacts: c.PostForm("contacts"),
			Mobile:   c.PostForm("mobile"),
			Content:  c.PostForm("content"),
			IP:       c.ClientIP(),
			Status:   0,
		}
		model.DB.Create(&msg)
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "提交成功"})
		return
	}
	ctx := fc.buildContext(c)
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	content := fc.Store.Render("message.html")
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) Visits(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	if id > 0 {
		model.DB.Model(&model.Content{}).Where("id = ?", id).
			UpdateColumn("visits", gorm.Expr("visits + 1"))
	}
	c.String(http.StatusOK, "ok")
}

func (fc *FrontController) renderSortPage(c *gin.Context, sort *model.ContentSort) {
	ctx := fc.buildContext(c)
	ctx.Sort = sort
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		ctx.CurrentPage = p
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)
	tpl := sort.ListTpl
	if tpl == "" {
		tpl = "list.html"
	}
	content := fc.Store.Render(tpl)
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}

func (fc *FrontController) buildContext(c *gin.Context) *parser.Context {
	ctx := &parser.Context{
		Page: make(map[string]interface{}),
	}

	var site model.Site
	if model.DB.First(&site).Error == nil {
		ctx.Site = &site
	}

	var company model.Company
	if model.DB.First(&company).Error == nil {
		ctx.Company = &company
	}

	return ctx
}

func trimSuffix(s string) string {
	return strings.TrimSuffix(strings.TrimSuffix(s, ".html"), ".htm")
}
