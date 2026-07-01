package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// MessageController - 留言管理控制器
// 對應 PHP: apps/admin/controller/content/MessageController.php
type MessageController struct {
	common.BaseController
}

// Index - 留言列表
func (ms *MessageController) Index(c *gin.Context) {
	// 單字段切換（AJAX 審核狀態切換）
	if field := c.Query("field"); field != "" {
		idStr := c.Param("id")
		if idStr == "" {
			idStr = c.Query("id")
		}
		id, _ := strconv.Atoi(idStr)
		value := c.Query("value")
		if id > 0 && field != "" {
			model.DB.Model(&model.Message{}).Where("id = ?", id).Update(field, value)
			ms.JSONOK(c, "")
			return
		}
	}

	var messages []model.Message
	model.DB.Order("id DESC").Find(&messages)

	// 獲取留言字段定義（fcode=1）
	fields := content.GetFormFieldByCode("1")

	// 處理時間格式
	type msgRow struct {
		model.Message
		CreateTimeStr string
		UpdateTimeStr string
	}
	var rows []msgRow
	for _, m := range messages {
		row := msgRow{Message: m}
		if !m.CreateTime.IsZero() {
			row.CreateTimeStr = m.CreateTime.Format("2006-01-02 15:04:05")
		}
		if !m.UpdateTime.IsZero() {
			row.UpdateTimeStr = m.UpdateTime.Format("2006-01-02 15:04:05")
		}
		rows = append(rows, row)
	}

	common.Render(c, "content/message.html", gin.H{
		"messages": rows,
		"fields":   fields,
		"list":     len(rows) > 0,
	})
}

// Mod - 回覆留言
func (ms *MessageController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		model.DB.Model(&model.Message{}).Where("id = ?", id).Updates(map[string]interface{}{
			"recontent":   c.PostForm("replycontent"),
			"status":      c.PostForm("status"),
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
			"update_user": "admin",
		})
		ms.JSONOKMsg(c, common.NoticeReply)
		return
	}

	var msg model.Message
	model.DB.First(&msg, id)
	common.Render(c, "content/message.html", gin.H{
		"message": msg,
		"mod":     true,
		"get_id":  idStr,
	})
}

// Del - 刪除留言
func (ms *MessageController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Message{}, idStr)
	ms.JSONOKMsg(c, common.NoticeDelete)
}

// Clear - 清空留言
func (ms *MessageController) Clear(c *gin.Context) {
	ids := c.Query("ids")
	if ids != "" {
		// 批量刪除選中記錄（ids 用 "and" 分隔，與 PbootCMS 一致）
		idList := strings.Split(ids, "and")
		for _, idStr := range idList {
			idStr = strings.TrimSpace(idStr)
			if idStr != "" {
				model.DB.Delete(&model.Message{}, idStr)
			}
		}
		ms.JSONOKMsg(c, common.NoticeDelete)
		return
	}
	// 清空全部
	model.DB.Where("1 = 1").Delete(&model.Message{})
	ms.JSONOKMsg(c, common.NoticeClean)
}
