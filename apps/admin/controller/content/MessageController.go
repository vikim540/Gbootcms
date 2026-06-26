package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MessageController - Message Management Controller
// Corresponds to PHP: apps/admin/controller/MessageController.php
type MessageController struct {
	common.BaseController
}

// Index - Message list
func (ms *MessageController) Index(c *gin.Context) {
	var messages []model.Message
	model.DB.Order("askdate DESC").Find(&messages)
	common.Render(c, "content/message.html", gin.H{"messages": messages})
}

// Mod - Reply message
func (ms *MessageController) Mod(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	if c.Request.Method == "POST" {
		model.DB.Model(&model.Message{}).Where("id = ?", id).Updates(map[string]interface{}{
			"replycontent": c.PostForm("replycontent"),
			"replydate":    time.Now(),
			"status":       1,
		})
		ms.JSONOKMsg(c, "回覆成功")
		return
	}

	var msg model.Message
	model.DB.First(&msg, id)
	common.Render(c, "content/exmessage.html", gin.H{"message": msg})
}

// Del - Delete message
func (ms *MessageController) Del(c *gin.Context) {
	idStr := c.Query("id")
	model.DB.Delete(&model.Message{}, idStr)
	ms.JSONOKMsg(c, "刪除成功")
}

// Clear - Clear messages
func (ms *MessageController) Clear(c *gin.Context) {
	model.DB.Where("1 = 1").Delete(&model.Message{})
	ms.JSONOKMsg(c, "清理成功")
}
