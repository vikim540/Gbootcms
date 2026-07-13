package content

import (
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LabelController struct {
	common.BaseController
}

func (lb *LabelController) Index(c *gin.Context) {
	// === POST 批量更新標籤值 ===
	if c.Request.Method == "POST" {
		postForm := make(map[string]string)
		// 支援 multipart/form-data（AJAX FormData 提交）和 application/x-www-form-urlencoded
		// 先嘗試 ParseMultipartForm，再呼叫 ParseForm 處理 urlencoded
		// ParseMultipartForm 對非 multipart 請求會返回 ErrNotMultipart 但不消費 body
		c.Request.ParseMultipartForm(32 << 20)
		c.Request.ParseForm()

		// 從 PostForm 收集（urlencoded 值或已從 multipart 複製的值）
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				postForm[k] = v[0]
			}
		}
		// 補充 MultipartForm 值（multipart/form-data 的情況下 PostForm 可能未完整填充）
		if c.Request.MultipartForm != nil {
			for k, v := range c.Request.MultipartForm.Value {
				if len(v) > 0 {
					postForm[k] = v[0]
				}
			}
		}

		log.Printf("[LabelController.Index] POST fields count: %d", len(postForm))

		updated := content.BatchUpdateLabelValues(postForm, lb.GetAdminUsername(c))
		lb.JSONOKMsg(c, common.NoticeLabelSaved(updated))
		return
	}

	labels := content.GetAllLabels()
	common.Render(c, "content/label.html", gin.H{"list": true, "labels": labels})
}

func (lb *LabelController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		typ := 1
		if t := c.PostForm("type"); t != "" {
			if v, err := strconv.Atoi(t); err == nil {
				typ = v
			}
		}
		err := content.AddLabelFull(
		c.PostForm("name"),
		c.PostForm("value"),
		c.PostForm("description"),
		lb.GetAdminUsername(c),
		typ,
	)
		if err != nil {
			lb.JSONFail(c, "Add failed: "+err.Error())
			return
		}
		lb.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	common.Render(c, "content/label.html", gin.H{"action": "add"})
}

func (lb *LabelController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		err := content.UpdateLabel(
			id,
			c.PostForm("name"),
			c.PostForm("value"),
			lb.GetAdminUsername(c),
		)
		if err != nil {
			lb.JSONFail(c, "Modify failed: "+err.Error())
			return
		}
		lb.JSONOKMsg(c, common.NoticeModify)
		return
	}

	label := content.GetLabelById(id)
	common.Render(c, "content/label.html", gin.H{"label": label, "action": "mod"})
}

func (lb *LabelController) Del(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	err := content.DeleteLabel(id)
	if err != nil {
		lb.JSONFail(c, "Delete failed: "+err.Error())
		return
	}
	lb.JSONOKMsg(c, common.NoticeDelete)
}
