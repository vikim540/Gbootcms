package content

import (
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ModelController struct {
	common.BaseController
}

func (md *ModelController) Index(c *gin.Context) {
	models := content.GetAllModels()
	common.Render(c, "content/model.html", gin.H{"models": models})
}

func (md *ModelController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		typ, _ := strconv.Atoi(c.DefaultPostForm("type", "1"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))
		err := content.AddModel(
			c.PostForm("mcode"),
			c.PostForm("name"),
			c.PostForm("urlname"),
			"admin",
			typ,
			status,
		)
		if err != nil {
			md.JSONFailMsg(c, "Add failed: "+err.Error())
			return
		}
		md.JSONOKMsg(c, "Added successfully")
		return
	}
	common.Render(c, "content/model.html", gin.H{"action": "add"})
}

func (md *ModelController) Mod(c *gin.Context) {
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
			content.UpdateModelSingleField(id, fieldName, fieldValue, "admin")
			md.JSONOKMsg(c, "Modified successfully")
			return
		}
		md.JSONFailMsg(c, "Invalid field")
		return
	}

	// === POST 全量修改模式 ===
	if c.Request.Method == "POST" {
		err := content.UpdateModel(
			id,
			c.PostForm("mcode"),
			c.PostForm("name"),
			c.PostForm("urlname"),
			"admin",
		)
		if err != nil {
			md.JSONFailMsg(c, "Modify failed: "+err.Error())
			return
		}
		md.JSONOKMsg(c, "Modified successfully")
		return
	}

	cm := content.GetModelById(id)
	common.Render(c, "content/model.html", gin.H{"model": cm, "action": "mod"})
}

func (md *ModelController) Del(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	err := content.DeleteModel(id)
	if err != nil {
		md.JSONFailMsg(c, "Delete failed: "+err.Error())
		return
	}
	md.JSONOKMsg(c, "Deleted successfully")
}
