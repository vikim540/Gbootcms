package content

import (
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ExtFieldController struct {
	common.BaseController
}

func (ef *ExtFieldController) Index(c *gin.Context) {
	fields := content.GetAllExtFields()
	common.Render(c, "content/extfield.html", gin.H{"fields": fields})
}

func (ef *ExtFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		err := content.AddExtField(
			c.PostForm("modelcode"),
			c.PostForm("name"),
			c.PostForm("field"),
			c.PostForm("type"),
			required,
			sorting,
		)
		if err != nil {
			ef.JSONFailMsg(c, "Add failed: "+err.Error())
			return
		}
		ef.JSONOKMsg(c, "Added successfully")
		return
	}
	common.Render(c, "content/extfield.html", gin.H{"action": "add"})
}

func (ef *ExtFieldController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// === 双重人格路由：GET /field/status/value 模式单字段快速切换 ===
	fieldName := c.Query("field")
	fieldValue := c.Query("value")
	if c.Request.Method == "GET" && fieldName != "" && fieldValue != "" {
		// 白名单：只允许修改 status 字段，防止 SQL 注入
		if fieldName == "status" {
			content.UpdateExtFieldSingleField(id, fieldName, fieldValue)
			ef.JSONOKMsg(c, "Modified successfully")
			return
		}
		ef.JSONFailMsg(c, "Invalid field")
		return
	}

	// === POST 全量修改模式 ===
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		err := content.UpdateExtField(
			id,
			c.PostForm("modelcode"),
			c.PostForm("name"),
			c.PostForm("field"),
			c.PostForm("type"),
			required,
			sorting,
		)
		if err != nil {
			ef.JSONFailMsg(c, "Modify failed: "+err.Error())
			return
		}
		ef.JSONOKMsg(c, "Modified successfully")
		return
	}

	field := content.GetExtFieldById(id)
	common.Render(c, "content/extfield.html", gin.H{"field": field, "action": "mod"})
}

func (ef *ExtFieldController) Del(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	err := content.DeleteExtField(id)
	if err != nil {
		ef.JSONFailMsg(c, "Delete failed: "+err.Error())
		return
	}
	ef.JSONOKMsg(c, "Deleted successfully")
}
