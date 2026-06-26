package content

import (
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ModelController struct {
	common.BaseController
}

// Index — 模型列表
func (md *ModelController) Index(c *gin.Context) {
	models := content.GetAllModels()
	common.Render(c, "content/model.html", gin.H{
		"list":   true,
		"models": models,
	})
}

// Add — 新增模型
func (md *ModelController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		if name == "" {
			md.JSONFail(c, "模型名稱不能為空")
			return
		}

		typ, _ := strconv.Atoi(c.DefaultPostForm("type", "1"))
		urlname := c.PostForm("urlname")
		listtpl := c.PostForm("listtpl")
		contenttpl := c.PostForm("contenttpl")
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		// 驗證 urlname 格式
		if urlname != "" {
			if matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-]+$`, urlname); !matched {
				md.JSONFail(c, "URL名稱僅允許英文、數字和短橫線")
				return
			}
		}

		// 自動生成 mcode
		mcode := content.GetNextMcode()

		if err := content.AddModel(mcode, name, urlname, listtpl, contenttpl, "admin", typ, status); err != nil {
			md.JSONFail(c, "新增失敗: "+err.Error())
			return
		}
		md.JSONOKMsg(c, common.NoticeAdd)
		return
	}

	// GET: 渲染新增表單（與列表同頁）
	models := content.GetAllModels()
	common.Render(c, "content/model.html", gin.H{
		"list":   true,
		"models": models,
	})
}

// Mod — 修改模型
func (md *ModelController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}

	// === 單字段快速切換（如狀態切換） ===
	if field != "" && value != "" {
		if field == "status" {
			content.UpdateModelSingleField(id, field, value, "admin")
			md.JSONOKMsg(c, common.NoticeModify)
			return
		}
		md.JSONFail(c, "不允許修改的字段")
		return
	}

	// === POST 全量修改 ===
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		if name == "" {
			md.JSONFail(c, "模型名稱不能為空")
			return
		}

		typ, _ := strconv.Atoi(c.DefaultPostForm("type", "1"))
		urlname := c.PostForm("urlname")
		listtpl := c.PostForm("listtpl")
		contenttpl := c.PostForm("contenttpl")
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		// 驗證 urlname 格式
		if urlname != "" {
			if matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-]+$`, urlname); !matched {
				md.JSONFail(c, "URL名稱僅允許英文、數字和短橫線")
				return
			}
		}

		// 檢查 urlname 衝突
		if conflict := content.CheckUrlnameConflict(urlname, id); conflict != "" {
			md.JSONFail(c, conflict)
			return
		}

		if err := content.UpdateModel(id, name, urlname, listtpl, contenttpl, "admin", typ, status); err != nil {
			md.JSONFail(c, "修改失敗: "+err.Error())
			return
		}
		md.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// === GET: 渲染修改表單 ===
	cm := content.GetModelById(id)
	models := content.GetAllModels()
	common.Render(c, "content/model.html", gin.H{
		"list":   true,
		"models": models,
		"mod":    true,
		"model":  cm,
	})
}

// Del — 刪除模型
func (md *ModelController) Del(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)
	if id <= 0 {
		md.JSONFail(c, "缺少模型 ID")
		return
	}
	if err := content.DeleteModel(id); err != nil {
		md.JSONFail(c, err.Error())
		return
	}
	md.JSONOKMsg(c, common.NoticeDelete)
}
