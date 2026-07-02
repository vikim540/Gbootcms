package content

import (
	"pbootcms-go/apps/admin/helper"
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
		"C":         "extField",
	})
}

func (ef *ExtFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		description := c.PostForm("description")
		field := c.PostForm("field")
		if field == "" {
			// 兜底：如果沒有填寫字段名稱，用描述代替
			field = description
		}
		typ := c.PostForm("type")
		val := c.PostForm("value")
		err := content.AddExtField(
			c.PostForm("mcode"),
			description, // name 列存儲描述（與 description 列一致）
			field,
			typ,
			val,
			required,
			sorting,
		)
		if err != nil {
			ef.JSONFail(c, "新增失敗: "+err.Error())
			return
		}
		// 新增字段後，確保 ay_content_ext 表有對應物理列
		if field != "" {
			content.EnsureExtColumnExists(field, typ)
		}
		ef.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "content/extfield.html", gin.H{
		"action": "add",
		"models": content.GetAllModels(),
		"list":   true,
		"C":      "extField",
	})
}

func (ef *ExtFieldController) Mod(c *gin.Context) {
	// 解析萬用 action 參數: /id/42 或 /field/status/value/1
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// === 双重人格路由：GET /field/status/value 模式单字段快速切换 ===
	fieldName := params["field"]
	if fieldName == "" {
		fieldName = c.Query("field")
	}
	fieldValue := params["value"]
	if fieldValue == "" {
		fieldValue = c.Query("value")
	}
	if c.Request.Method == "GET" && fieldName != "" && fieldValue != "" {
		// 白名单：只允许修改 status 字段，防止 SQL 注入
		if fieldName == "status" {
			content.UpdateExtFieldSingleField(id, fieldName, fieldValue)
			ef.JSONOKMsg(c, common.NoticeModify)
			return
		}
		ef.JSONFail(c, "不允許修改的欄位")
		return
	}

	// === POST 全量修改模式 ===
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		description := c.PostForm("description")
		val := c.PostForm("value")
		err := content.UpdateExtField(
			id,
			c.PostForm("mcode"),
			description, // name 列存儲描述
			c.PostForm("field"),
			c.PostForm("type"),
			val,
			required,
			sorting,
		)
		if err != nil {
			ef.JSONFail(c, "修改失敗: "+err.Error())
			return
		}
		ef.JSONOKMsg(c, common.NoticeModify)
		return
	}

	field := content.GetExtFieldById(id)
	models := content.GetAllModels()
	common.Render(c, "content/extfield.html", gin.H{
		"extfield": field,
		"models":   models,
		"mod":      true,
		"C":        "extField",
	})
}

func (ef *ExtFieldController) Del(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	err := content.DeleteExtField(id)
	if err != nil {
		ef.JSONFail(c, "刪除失敗: "+err.Error())
		return
	}
	ef.JSONOKMsg(c, common.NoticeDelete)
}
