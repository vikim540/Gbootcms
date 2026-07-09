package system

import (
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SyslogController - System Log Controller
// 三類日誌獨立分頁：系統日誌（admin）、蜘蛛日誌（spider）、通知日誌（mail_/webhook_）
type SyslogController struct {
	common.BaseController
}

// Index - Log list with pagination
func (sl *SyslogController) Index(c *gin.Context) {
	// 系統日誌分頁（使用 Paginate 統一處理 page/pagesize 參數）
	sysPage, pageSize, sysOffset := sl.Paginate(c)

	var sysTotal int64
	model.DB.Model(&model.Syslog{}).
		Where("level NOT IN ('spider') AND level NOT LIKE 'mail_%' AND level NOT LIKE 'webhook_%'").
		Count(&sysTotal)
	var syslogs []model.Syslog
	model.DB.Where("level NOT IN ('spider') AND level NOT LIKE 'mail_%' AND level NOT LIKE 'webhook_%'").
		Order("id DESC").Offset(sysOffset).Limit(pageSize).Find(&syslogs)
	sysPagebar := helper.BuildPagebarHTML(sysTotal, sysPage, pageSize, "/admin/system/syslog/index")

	// 蜘蛛日誌分頁
	spiderPage, _ := strconv.Atoi(c.Query("spage"))
	if spiderPage < 1 {
		spiderPage = 1
	}
	var spiderTotal int64
	model.DB.Model(&model.Syslog{}).Where("level = 'spider'").Count(&spiderTotal)
	var spiderLogs []model.Syslog
	model.DB.Where("level = 'spider'").
		Order("id DESC").Offset((spiderPage - 1) * pageSize).Limit(pageSize).Find(&spiderLogs)
	spiderPagebar := helper.BuildPagebarHTMLEx(spiderTotal, spiderPage, pageSize, "/admin/system/syslog/index", "spage")

	// 通知日誌分頁
	notifyPage, _ := strconv.Atoi(c.Query("npage"))
	if notifyPage < 1 {
		notifyPage = 1
	}
	var notifyTotal int64
	model.DB.Model(&model.NotifyLog{}).
		Where("level LIKE 'mail_%' OR level LIKE 'webhook_%'").Count(&notifyTotal)
	notifyLogs := model.GetNotifyLogsPaged(pageSize, (notifyPage-1)*pageSize)
	notifyPagebar := helper.BuildPagebarHTMLEx(notifyTotal, notifyPage, pageSize, "/admin/system/syslog/index", "npage")

	common.Render(c, "system/syslog.html", gin.H{
		"list":             true,
		"syslogs":          syslogs,
		"spiderLogs":       spiderLogs,
		"notifyLogs":       notifyLogs,
		"sysPagebar":       sysPagebar,
		"spiderPagebar":    spiderPagebar,
		"notifyPagebar":    notifyPagebar,
		"sysPageStart":     sysOffset,
		"spiderPageStart":  (spiderPage - 1) * pageSize,
		"notifyPageStart":  (notifyPage - 1) * pageSize,
	})
}

// applyPathAction 將 /key/value 路徑參數轉為 query 參數（復用 helper.ParseWildcardAction）
func (sl *SyslogController) applyPathAction(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	for k, v := range params {
		if c.Request.URL.RawQuery != "" {
			c.Request.URL.RawQuery += "&"
		}
		c.Request.URL.RawQuery += k + "=" + v
	}
}

// IndexCatchAll 處理 /system/syslog/index/*action 路徑（如 /pagesize/20/spage/2）
func (sl *SyslogController) IndexCatchAll(c *gin.Context) {
	sl.applyPathAction(c)
	sl.Index(c)
}

// Clear - Clear system logs (non-spider, non-notify)
func (sl *SyslogController) Clear(c *gin.Context) {
	model.DB.Where("level NOT IN ('spider') AND level NOT LIKE 'mail_%' AND level NOT LIKE 'webhook_%'").
		Delete(&model.Syslog{})
	sl.LogAction(c, "清空系統日誌")
	sl.JSONOKMsg(c, common.NoticeClean)
}

// ClearSpider - Clear spider logs only
func (sl *SyslogController) ClearSpider(c *gin.Context) {
	model.DB.Where("level = 'spider'").Delete(&model.Syslog{})
	sl.LogAction(c, "清空蜘蛛日誌")
	sl.JSONOKMsg(c, common.NoticeClean)
}

// ClearNotify - Clear notification logs only
func (sl *SyslogController) ClearNotify(c *gin.Context) {
	model.ClearNotifyLogs()
	sl.LogAction(c, "清空通知日誌")
	sl.JSONOKMsg(c, common.NoticeClean)
}
