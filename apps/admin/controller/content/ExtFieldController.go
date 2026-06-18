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
	models := content.GetAllModels()
	list := true
	common.Render(c, "content/extfield.html", gin.H{
		"extfields": fields,
		"models":    models,
		"list":      list,
	})
}

func (ef *ExtFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		name := c.PostForm("name")
		typ := c.PostForm("type")
		err := content.AddExtField(
			c.PostForm("modelcode"),
			name,
			c.PostForm("field"),
			typ,
			required,
			sorting,
		)
		if err != nil {
			ef.JSONFailMsg(c, "新增失敗: "+err.Error())
			return
		}
		// 新增字段後，確保 ay_content_ext 表有對應物理列
		if name != "" {
			content.EnsureExtColumnExists(name, typ)
		}
		ef.JSONOKMsg(c, "新增成功")
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
			ef.JSONOKMsg(c, "修改成功")
			return
		}
		ef.JSONFailMsg(c, "不允許修改的欄位")
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
			ef.JSONFailMsg(c, "修改失敗: "+err.Error())
			return
		}
		ef.JSONOKMsg(c, "修改成功")
		return
	}

	field := content.GetExtFieldById(id)
	models := content.GetAllModels()
	common.Render(c, "content/extfield.html", gin.H{
		"extfield": field,
		"models":   models,
		"mod":      true,
	})
}

func (ef *ExtFieldController) Del(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	err := content.DeleteExtField(id)
	if err != nil {
		ef.JSONFailMsg(c, "刪除失敗: "+err.Error())
		return
	}
	ef.JSONOKMsg(c, "刪除成功")
}
