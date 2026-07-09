package content

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	"gbootcms/apps/admin/model/member"
	"gbootcms/apps/common"

	"github.com/gin-gonic/gin"
)

// fieldDisplay 用於模板中動態顯示留言字段
type fieldDisplay struct {
	Description string
	Value       string
}

// formFieldDef 留言字段定義（含描述文本）
type formFieldDef struct {
	Name        string
	Description string
}

// msgRow 留言列表行，嵌入 Message 並添加計算字段
type msgRow struct {
	model.Message
	CreateTimeStr string        // 格式化後的創建時間
	UserIPStr     string        // 轉換後的 IP 地址（整數→點分格式）
	UserBs        string        // 別名: [value->user_bs] → val1.UserBs（匹配 SnakeToPascal 輸出）
	UserOs        string        // 別名: [value->user_os] → val1.UserOs（匹配 SnakeToPascal 輸出）
	Recontent     string        // 別名: [value->recontent] → val1.Recontent（匹配 SnakeToPascal 輸出）
	Username      string        // 會員帳號（LEFT JOIN ay_member）
	Nickname      string        // 會員暱稱（LEFT JOIN ay_member）
	Fields        []fieldDisplay // 動態字段列表
}

// MessageController 留言管理控制器
// 對應 PHP: apps/admin/controller/content/MessageController.php
type MessageController struct {
	common.BaseController
}

// getFormFields 查詢留言字段定義（含 description 列）
func getFormFields(c *gin.Context, fcode string) []formFieldDef {
	var defs []struct {
		Name        string `gorm:"column:name"`
		Description string `gorm:"column:description"`
	}
	model.DB.WithContext(c.Request.Context()).Table("ay_form_field").
		Select("name, description").
		Where("fcode = ?", fcode).
		Order("sorting ASC, id ASC").
		Scan(&defs)

	result := make([]formFieldDef, len(defs))
	for i, d := range defs {
		result[i] = formFieldDef{
			Name:        d.Name,
			Description: d.Description,
		}
	}
	return result
}

// getMsgFieldValue 根據列名獲取 Message 字段值
func getMsgFieldValue(m model.Message, columnName string) string {
	switch columnName {
	case "contacts":
		return m.Contacts
	case "mobile":
		return m.Mobile
	case "content":
		return m.Content
	case "user_ip", "ip":
		return long2ip(m.IP)
	case "user_bs", "bs", "browser":
		return m.Browser
	case "user_os", "os":
		return m.OS
	case "recontent", "replycontent":
		return m.ReContent
	case "status":
		return strconv.Itoa(m.Status)
	case "create_time", "askdate":
		if !m.CreateTime.IsZero() {
			return m.CreateTime.Format("2006-01-02 15:04:05")
		}
		return ""
	default:
		return ""
	}
}

// long2ip 將整數 IP 轉換為點分格式
// PHP 的 ip2long 將 IP 存為整數（如 2130706433 = 127.0.0.1）
// Go 前端存的是字串 IP（如 "127.0.0.1"），此函數同時相容兩種格式
func long2ip(ipVal string) string {
	if ipVal == "" {
		return ""
	}
	// 已經是 IP 地址格式（含小數點）
	if strings.Contains(ipVal, ".") {
		return ipVal
	}
	// 整數格式（PHP ip2long 存儲）
	n, err := strconv.ParseInt(ipVal, 10, 64)
	if err != nil {
		return ipVal
	}
	// 負數處理（PHP 32 位無符號整數在 64 位系統上可能為負數）
	if n < 0 {
		n += 1 << 32
	}
	return fmt.Sprintf("%d.%d.%d.%d",
		(n>>24)&0xFF,
		(n>>16)&0xFF,
		(n>>8)&0xFF,
		n&0xFF)
}

// Index 留言列表
func (ms *MessageController) Index(c *gin.Context) {
	// 導出 Excel
	if c.Query("export") != "" {
		ms.exportMessages(c)
		return
	}

	// 分頁參數
	page, pageSize, offset := ms.Paginate(c)

	// 查詢總數
	var total int64
	model.DB.WithContext(c.Request.Context()).Model(&model.Message{}).Count(&total)

	// 查詢留言列表（分頁）
	var messages []model.Message
	model.DB.WithContext(c.Request.Context()).Order("id DESC").Offset(offset).Limit(pageSize).Find(&messages)

	// 批量查詢會員信息（避免 N+1 查詢）
	uidSet := make(map[int]bool)
	for _, m := range messages {
		if m.UID > 0 {
			uidSet[m.UID] = true
		}
	}
	memberMap := make(map[int]member.Member)
	if len(uidSet) > 0 {
		uids := make([]int, 0, len(uidSet))
		for uid := range uidSet {
			uids = append(uids, uid)
		}
		var members []member.Member
		model.DB.WithContext(c.Request.Context()).Where("id IN ?", uids).Find(&members)
		for _, mem := range members {
			memberMap[int(mem.ID)] = mem
		}
	}

	// 查詢字段定義
	fieldDefs := getFormFields(c, "1")

	// 構建行數據
	rows := make([]msgRow, 0, len(messages))
	for _, m := range messages {
		row := msgRow{
			Message:   m,
			UserIPStr: long2ip(m.IP),
			UserBs:    m.Browser,
			UserOs:    m.OS,
			Recontent: m.ReContent,
		}
		if !m.CreateTime.IsZero() {
			row.CreateTimeStr = m.CreateTime.Format("2006-01-02 15:04:05")
		}
		// 會員信息
		if m.UID > 0 {
			if mem, ok := memberMap[m.UID]; ok {
				row.Username = mem.Username
				row.Nickname = mem.Nickname
			}
		}
		// 動態字段
		row.Fields = make([]fieldDisplay, 0, len(fieldDefs))
		for _, fd := range fieldDefs {
			row.Fields = append(row.Fields, fieldDisplay{
				Description: fd.Description,
				Value:       getMsgFieldValue(m, fd.Name),
			})
		}
		rows = append(rows, row)
	}

	// 構建分頁 HTML
	pagebar := helper.BuildPagebarHTML(total, page, pageSize, "/admin/content/message/index")

	common.Render(c, "content/message.html", gin.H{
		"messages": rows,
		"list":     true,
		"pagebar":  pagebar,
		"pagesize": pageSize,
		"C":        "content/message",
	})
}

// Mod 回覆留言（支持狀態切換和回覆表單）
func (ms *MessageController) Mod(c *gin.Context) {
	// 解析 ID（兼容 :id 和 *action 兩種路由）
	idStr := c.Param("id")
	action := c.Param("action")
	params := helper.ParseWildcardAction(action)
	if idStr == "" {
		idStr = params["id"]
	}
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// 狀態切換: /mod/id/1/field/status/value/0
	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}
	if field != "" && value != "" && id > 0 {
		model.DB.WithContext(c.Request.Context()).Model(&model.Message{}).Where("id = ?", id).Update(field, value)
		c.Redirect(http.StatusFound, "/admin/content/message/index")
		return
	}

	if c.Request.Method == "POST" {
		// 回覆提交
		recontent := c.PostForm("recontent")
		status := c.PostForm("status")
		model.DB.WithContext(c.Request.Context()).Model(&model.Message{}).Where("id = ?", id).Updates(map[string]interface{}{
			"recontent":   recontent,
			"status":      status,
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
			"update_user": ms.GetAdminUsername(c),
		})
		ms.JSONOKMsg(c, common.NoticeReply)
		return
	}

	// 顯示回覆表單
	var msg model.Message
	model.DB.WithContext(c.Request.Context()).First(&msg, id)
	// 使用 msgRow 包裝，提供 Recontent 別名字段（匹配模板轉換器 SnakeToPascal 輸出）
	modRow := msgRow{
		Message:   msg,
		Recontent: msg.ReContent,
	}
	common.Render(c, "content/message.html", gin.H{
		"message": modRow,
		"mod":     true,
		"get_id":  idStr,
		"C":       "content/message",
	})
}

// Del 刪除留言（AJAX JSON 響應）
func (ms *MessageController) Del(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		action := c.Param("action")
		params := helper.ParseWildcardAction(action)
		idStr = params["id"]
	}
	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		if id > 0 {
			model.DB.WithContext(c.Request.Context()).Delete(&model.Message{}, id)
		}
	}
	ms.JSONOKMsg(c, common.NoticeDelete)
}

// Clear 清空留言（支持批量刪除和全部清空）
func (ms *MessageController) Clear(c *gin.Context) {
	ids := c.Query("ids")
	if ids != "" {
		// 批量刪除選中記錄（ids 用 "and" 分隔，與 PbootCMS 一致）
		idList := strings.Split(ids, "and")
		for _, s := range idList {
			s = strings.TrimSpace(s)
			if s != "" {
				id, _ := strconv.Atoi(s)
				if id > 0 {
					model.DB.WithContext(c.Request.Context()).Delete(&model.Message{}, id)
				}
			}
		}
		c.Redirect(http.StatusFound, "/admin/content/message/index")
		return
	}
	// 清空全部（僅 ucode==10001 有權限）
	if common.GetSessionInt(c, "admin_ucode") == 10001 {
		model.DB.WithContext(c.Request.Context()).Where("1 = 1").Delete(&model.Message{})
	}
	c.Redirect(http.StatusFound, "/admin/content/message/index")
}

// exportMessages 導出留言記錄為 Excel（HTML 表格格式）
func (ms *MessageController) exportMessages(c *gin.Context) {
	var messages []model.Message
	model.DB.WithContext(c.Request.Context()).Order("id DESC").Find(&messages)

	fieldDefs := getFormFields(c, "1")

	var sb strings.Builder
	sb.WriteString("<html><head><meta charset='utf-8'></head><body>")
	sb.WriteString("<table border='1'>")

	// 表頭
	sb.WriteString("<tr>")
	for _, fd := range fieldDefs {
		sb.WriteString(fmt.Sprintf("<th>%s</th>", fd.Description))
	}
	sb.WriteString("<th>時間</th>")
	sb.WriteString("<th>IP</th>")
	sb.WriteString("<th>瀏覽器</th>")
	sb.WriteString("<th>操作系統</th>")
	sb.WriteString("<th>回覆內容</th>")
	sb.WriteString("</tr>")

	// 數據行
	for _, m := range messages {
		sb.WriteString("<tr>")
		for _, fd := range fieldDefs {
			sb.WriteString(fmt.Sprintf("<td>%s</td>", getMsgFieldValue(m, fd.Name)))
		}
		timeStr := ""
		if !m.CreateTime.IsZero() {
			timeStr = m.CreateTime.Format("2006-01-02 15:04:05")
		}
		sb.WriteString(fmt.Sprintf("<td>%s</td>", timeStr))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", long2ip(m.IP)))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", m.Browser))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", m.OS))
		sb.WriteString(fmt.Sprintf("<td>%s</td>", m.ReContent))
		sb.WriteString("</tr>")
	}

	sb.WriteString("</table></body></html>")

	filename := fmt.Sprintf("留言記錄-%s.xls", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "application/vnd.ms-excel")
	c.Header("Cache-Control", "max-age=0")
	c.Header("Content-Disposition", fmt.Sprintf("attachment;filename=%s", filename))
	c.String(http.StatusOK, sb.String())
}
