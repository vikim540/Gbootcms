package content

import (
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

type LabelController struct {
	common.BaseController
}

func (lb *LabelController) Index(c *gin.Context) {
	// === POST 批量更新标签值 ===
	if c.Request.Method == "POST" {
		postForm := make(map[string]string)
		if err := c.Request.ParseForm(); err == nil {
			for k, v := range c.Request.PostForm {
				if len(v) > 0 {
					postForm[k] = v[0]
				}
			}
		}
		updated := content.BatchUpdateLabelValues(postForm, "admin")
		lb.JSONOKMsg(c, common.NoticeLabelSaved(updated))
		return
	}

	labels := content.GetAllLabels()
	common.Render(c, "content/label.html", gin.H{"labels": labels})
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
			"admin",
			typ,
		)
		if err != nil {
			lb.JSONFailMsg(c, "Add failed: "+err.Error())
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
			"admin",
		)
		if err != nil {
			lb.JSONFailMsg(c, "Modify failed: "+err.Error())
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
		lb.JSONFailMsg(c, "Delete failed: "+err.Error())
		return
	}
	lb.JSONOKMsg(c, common.NoticeDelete)
}
