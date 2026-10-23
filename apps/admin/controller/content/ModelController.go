package content

import (
	"gbootcms/apps/admin/helper"
	contentModel "gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ModelController struct {
	common.BaseController
}

// Index — 模型列表
func (md *ModelController) Index(c *gin.Context) {
	page, pageSize, offset := md.Paginate(c)

	allModels := contentModel.GetAllModels()
	total := int64(len(allModels))
	// 記憶體內分頁（保留 GetAllModels 的 COALESCE 處理邏輯）
	models := allModels
	if offset >= len(models) {
		models = models[:0]
	} else {
		end := offset + pageSize
		if end > len(models) {
			end = len(models)
		}
		models = models[offset:end]
	}
	baseURL := "/admin/content/model/index"
	common.Render(c, "content/model.html", gin.H{
		"list":     true,
		"models":   models,
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
}

// Add — 新增模型
func (md *ModelController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		if name == "" {
			md.LogAction(c, "新增內容模型失敗")
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
				md.LogAction(c, "新增內容模型失敗")
				md.JSONFail(c, "URL名稱僅允許英文、數字和短橫線")
				return
			}
		}

		// 自動生成 mcode
		mcode := contentModel.GetNextMcode()

		if err := contentModel.AddModel(mcode, name, urlname, listtpl, contenttpl, md.GetAdminUsername(c), typ, status); err != nil {
			md.LogAction(c, "新增內容模型失敗")
			md.JSONFail(c, "新增失敗: "+err.Error())
			return
		}
		md.LogAction(c, "新增內容模型成功")
		md.JSONOKMsg(c, common.NoticeAdd)
		return
	}

	// GET: 渲染新增表單（與列表同頁）
	models := contentModel.GetAllModels()
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
			contentModel.UpdateModelSingleField(id, field, value, md.GetAdminUsername(c))
			md.LogAction(c, "修改內容模型成功")
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
		if conflict := contentModel.CheckUrlnameConflict(urlname, id); conflict != "" {
			md.JSONFail(c, conflict)
			return
		}

		if err := contentModel.UpdateModel(id, name, urlname, listtpl, contenttpl, md.GetAdminUsername(c), typ, status); err != nil {
			md.JSONFail(c, "修改失敗: "+err.Error())
			return
		}
		md.LogAction(c, "修改內容模型成功")
		md.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// === GET: 渲染修改表單 ===
	cm := contentModel.GetModelById(id)
	models := contentModel.GetAllModels()
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
		md.LogAction(c, "刪除內容模型失敗")
		md.JSONFail(c, "缺少模型 ID")
		return
	}
	if err := contentModel.DeleteModel(id); err != nil {
		md.LogAction(c, "刪除內容模型失敗")
		md.JSONFail(c, err.Error())
		return
	}
	md.LogAction(c, "刪除內容模型成功")
	md.JSONOKMsg(c, common.NoticeDelete)
}
