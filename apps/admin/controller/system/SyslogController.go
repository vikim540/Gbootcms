package system

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"

	"github.com/gin-gonic/gin"
)

// SyslogController - System Log Controller
// Corresponds to PHP: apps/admin/controller/SyslogController.php
type SyslogController struct {
	common.BaseController
}

// Index - Log list
func (sl *SyslogController) Index(c *gin.Context) {
	var logs []model.Syslog
	model.DB.Order("create_time DESC").Limit(100).Find(&logs)
	common.Render(c, "system/syslog.html", gin.H{"logs": logs})
}

// Clear - Clear logs
func (sl *SyslogController) Clear(c *gin.Context) {
	model.DB.Where("1 = 1").Delete(&model.Syslog{})
	sl.JSONOKMsg(c, common.NoticeClean)
}
