package member

import (
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	memberModel "gbootcms/apps/admin/model/member"
	"gbootcms/apps/common"
	"gbootcms/config"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MemberFieldController - 會員欄位控制器
// 對應 PHP: apps/admin/controller/member/MemberFieldController.php
type MemberFieldController struct {
	common.BaseController
}

// Index - 會員欄位列表（含新增Tab）
func (mf *MemberFieldController) Index(c *gin.Context) {
	// 分頁處理
	page, pageSize, offset := mf.Paginate(c)
	baseURL := "/admin/member/field/index"

	// 統計總記錄數
	var total int64
	model.DB.WithContext(c.Request.Context()).Model(&model.MemberField{}).Count(&total)

	var fields []model.MemberField
	model.DB.WithContext(c.Request.Context()).Order("sorting ASC, id ASC").Offset(offset).Limit(pageSize).Find(&fields)
	common.Render(c, "member/field.html", gin.H{
		"list":     true,
		"fields":   fields,
		"C":        "member/field",
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
}

// Add - 新增會員欄位（對齊 PHP MemberFieldController::add()）
func (mf *MemberFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		description := c.PostForm("description")

		if name == "" {
			mf.JSONFail(c, "欄位名稱不能為空")
			return
		}

		// 欄位名稱必須以字母開頭（對齊 PHP: preg_match('/^[a-zA-Z][\w]+$/', $name)）
		matched, _ := regexp.MatchString(`^[a-zA-Z][\w]+$`, name)
		if !matched {
			mf.JSONFail(c, "欄位名稱必須以字母開頭，且只能包含字母、數字、下劃線")
			return
		}

		if description == "" {
			mf.JSONFail(c, "欄位描述不能為空")
			return
		}

		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		length, _ := strconv.Atoi(c.DefaultPostForm("length", "20"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		// 欄位不存在時創建物理列（對齊 PHP: isExistField → ALTER TABLE）
		if !memberModel.ColumnExistsInMember(name) {
			if err := memberModel.AddColumnToMember(name, length); err != nil {
				mf.JSONFail(c, "創建欄位失敗："+err.Error())
				return
			}
		} else if memberModel.IsFieldRegistered(name) {
			// 欄位存在且已登記則報錯（對齊 PHP: checkField）
			mf.JSONFail(c, "欄位已經存在，不能重複添加")
			return
		}

		now := time.Now()
		username := mf.GetAdminUsername(c)
		if err := model.DB.WithContext(c.Request.Context()).Create(&model.MemberField{
			Name:        name,
			Length:      length,
			Required:    required,
			Description: description,
			Sorting:     sorting,
			Status:      status,
			CreateUser:  username,
			UpdateUser:  username,
			CreateTime:  now,
			UpdateTime:  now,
		}).Error; err != nil {
			mf.JSONFail(c, "新增失敗："+err.Error())
			return
		}
		mf.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	// GET 請求重定向到列表頁
	c.Redirect(302, "/admin/member/field/index")
}

// Mod - 修改會員欄位（支援狀態切換 + 完整修改）
// 路由: /admin/member/field/mod/*action
func (mf *MemberFieldController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// 單欄位切換（狀態/必填開關）
	field := params["field"]
	if field == "" {
		field = c.Query("field")
	}
	value := params["value"]
	if value == "" {
		value = c.Query("value")
	}

	if field != "" && value != "" {
		if err := model.DB.WithContext(c.Request.Context()).Model(&model.MemberField{}).Where("id = ?", id).Update(field, value).Error; err != nil {
			mf.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		c.Redirect(302, "/admin/member/field/index")
		return
	}

	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		status, _ := strconv.Atoi(c.DefaultPostForm("status", "1"))

		if err := model.DB.WithContext(c.Request.Context()).Model(&model.MemberField{}).Where("id = ?", id).Updates(map[string]interface{}{
			"description": c.PostForm("description"),
			"required":    required,
			"sorting":     sorting,
			"status":      status,
			"update_user": mf.GetAdminUsername(c),
			"update_time": time.Now(),
		}).Error; err != nil {
			mf.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		mf.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET 載入修改頁面
	var field1 model.MemberField
	model.DB.WithContext(c.Request.Context()).First(&field1, id)
	common.Render(c, "member/field.html", gin.H{
		"mod":   true,
		"field": field1,
		"C":     "member/field",
	})
}

// Del - 刪除會員欄位（對齊 PHP MemberFieldController::del()）
func (mf *MemberFieldController) Del(c *gin.Context) {
	// 支援 *action 通配符路徑: /del/id/123
	params := helper.ParseWildcardAction(c.Param("action"))
	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	if idStr == "" {
		idStr = c.PostForm("id")
	}
	if idStr == "" {
		mf.JSONFail(c, "缺少刪除目標ID")
		return
	}

	// 取得欄位名稱（刪除前查詢，用於 DROP COLUMN）
	fieldName := memberModel.GetFieldNameByID(idStr)

	if err := model.DB.WithContext(c.Request.Context()).Delete(&model.MemberField{}, idStr).Error; err != nil {
		mf.JSONFail(c, "刪除失敗："+err.Error())
		return
	}

	// MySQL 執行欄位刪除，SQLite 暫不支援（對齊 PHP: get_db_type() == 'mysql'）
	if fieldName != "" {
		cfg := config.Get()
		if cfg.Database.Type == "mysql" {
			memberModel.DropColumnFromMember(fieldName)
		}
	}

	mf.JSONOKMsg(c, common.NoticeDelete)
}
