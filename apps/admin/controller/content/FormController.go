package content

import (
	"fmt"
	"strconv"
	"time"

	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	contentModel "gbootcms/apps/admin/model/content"
	"gbootcms/apps/admin/model/system"
	"gbootcms/apps/common"
	"gbootcms/config"

	"github.com/gin-gonic/gin"
)

// FormController 自定義表單控制器（對齊 PHP: FormController.php）
type FormController struct {
	common.BaseController
}

// parseFormParams 從 *action 通配符或 query 中解析參數
func parseFormParams(c *gin.Context) map[string]string {
	params := helper.ParseWildcardAction(c.Param("action"))
	// 補充 query 參數（path 中沒有的）
	for _, key := range []string{"fcode", "action", "id", "field", "value", "export"} {
		if params[key] == "" {
			if v := c.Query(key); v != "" {
				params[key] = v
			}
		}
	}
	return params
}

// injectGetFlat 注入 path 參數為 get_xxx 扁平變量，供模板 [$get.xxx] 使用
func injectGetFlat(data gin.H, params map[string]string) {
	for k, v := range params {
		data["get_"+k] = v
	}
}

// Index 表單管理（列表/查看數據/編輯字段）
func (fm *FormController) Index(c *gin.Context) {
	params := parseFormParams(c)
	fcode := params["fcode"]
	action := params["action"]

	if fcode != "" {
		form := contentModel.GetFormByCode(fcode)
		if form == nil {
			fm.JSONFail(c, "表單不存在")
			return
		}

		fields := contentModel.GetFormFieldByCode(fcode)

		if action == "showdata" {
			// 查看數據
			tableName := form.TableName
			var rawData []map[string]interface{}
			// 動態表（ay_diy_*）無 acode 欄位，不需 WithContext
			model.DB.Raw("SELECT * FROM `" + tableName + "` ORDER BY id DESC").Scan(&rawData)

			// 將 map 鍵名轉為 PascalCase，讓模板引擎 [value->field] 能正確訪問
			dataList := make([]map[string]interface{}, len(rawData))
			for i, row := range rawData {
				pascalRow := make(map[string]interface{})
				for k, v := range row {
					pascalRow[helper.SnakeToPascal(k)] = v
				}
				dataList[i] = pascalRow
			}

			// 建立 fieldNameMap：字段名(lowercase) → PascalCase，供模板動態訪問 {{ val1[FieldNameMap[val2.Name]] }}
			fieldNameMap := make(map[string]string)
			for _, f := range fields {
				fieldNameMap[f.Name] = helper.SnakeToPascal(f.Name)
			}

			page, pageSize, offset := fm.Paginate(c)
			total := int64(len(dataList))
			if offset < len(dataList) {
				end := offset + pageSize
				if end > len(dataList) {
					end = len(dataList)
				}
				dataList = dataList[offset:end]
			} else {
				dataList = []map[string]interface{}{}
			}
			baseURL := fmt.Sprintf("/admin/content/form/index/fcode/%s/action/showdata", fcode)
			data := gin.H{
				"showdata":     true,
				"form":         form,
				"fields":       fields,
				"formdata":     dataList,
				"fieldNameMap": fieldNameMap,
				"pagebar":      helper.BuildPagebarHTML(total, page, pageSize, baseURL),
				"pagesize":     pageSize,
				"get":          params,
			}
			injectGetFlat(data, params)
			common.Render(c, "content/form.html", data)
			return
		}

		if action == "showfield" {
			// 編輯字段
			page, pageSize, _ := fm.Paginate(c)
			var total int64
			model.DB.WithContext(c.Request.Context()).Model(&contentModel.FormField{}).Where("fcode = ?", fcode).Count(&total)
			baseURL := fmt.Sprintf("/admin/content/form/index/fcode/%s/action/showfield", fcode)
			data := gin.H{
				"showfield": true,
				"form":      form,
				"fields":    fields,
				"pagebar":   helper.BuildPagebarHTML(total, page, pageSize, baseURL),
				"pagesize":  pageSize,
				"get":       params,
			}
			injectGetFlat(data, params)
			common.Render(c, "content/form.html", data)
			return
		}
	}

	// 表單列表
	page, pageSize, offset := fm.Paginate(c)
	field := c.Query("field")
	keyword := c.Query("keyword")

	var forms []contentModel.Form
	query := model.DB.WithContext(c.Request.Context()).Model(&contentModel.Form{})
	if field != "" && keyword != "" {
		query = query.Where(field+" LIKE ?", "%"+keyword+"%")
	}
	var total int64
	query.Count(&total)
	query.Order("id ASC").Offset(offset).Limit(pageSize).Find(&forms)

	baseURL := "/admin/content/form/index"
	if field != "" && keyword != "" {
		baseURL += "?field=" + field + "&keyword=" + keyword
	}
	common.Render(c, "content/form.html", gin.H{
		"list":     true,
		"forms":    forms,
		"field":    field,
		"keyword":  keyword,
		"pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize": pageSize,
	})
}

// Add 新增表單/字段
func (fm *FormController) Add(c *gin.Context) {
	params := parseFormParams(c)
	action := params["action"]
	if action == "" {
		action = c.PostForm("action")
	}

	if action == "addform" {
		// 新增表單
		formName := c.PostForm("form_name")
		tableNameInput := c.PostForm("table_name")
		if formName == "" {
			fm.JSONFail(c, "表單名稱不能為空")
			return
		}
		if tableNameInput == "" {
			fm.JSONFail(c, "表名稱不能為空")
			return
		}
		// SQL 注入防護：驗證表名只含字母、數字、下劃線、橫線、點（對齊 PbootCMS PHP var 類型驗證）
		if !common.CheckVarType(tableNameInput) {
			fm.JSONFail(c, "表名稱只能包含字母、數字、下劃線、橫線")
			return
		}

		// 生成 fcode（取最大值+1）
		var maxFcode int
		var existingForms []contentModel.Form
		model.DB.WithContext(c.Request.Context()).Order("id ASC").Find(&existingForms)
		for _, f := range existingForms {
			code, _ := strconv.Atoi(f.Fcode)
			if code > maxFcode {
				maxFcode = code
			}
		}
		newFcode := strconv.Itoa(maxFcode + 1)

		// 對齊 PHP: table_name = 'ay_diy_' + 用戶輸入
		tableName := "ay_diy_" + tableNameInput

		// 創建物理表（DDL 操作，無需 WithContext）
		cfg := config.Get()
		if cfg.Database.Type == "mysql" {
			if err := model.DB.Exec("CREATE TABLE `" + tableName + "` (`id` int(10) unsigned NOT NULL AUTO_INCREMENT,`create_time` datetime NOT NULL,PRIMARY KEY (`id`)) ENGINE=MyISam DEFAULT CHARSET=utf8").Error; err != nil {
				fm.JSONFail(c, "操作失敗: "+err.Error())
				return
			}
		} else {
			if err := model.DB.Exec("CREATE TABLE `" + tableName + "` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,`create_time` TEXT NOT NULL)").Error; err != nil {
				fm.JSONFail(c, "操作失敗: "+err.Error())
				return
			}
		}

		username := fm.GetAdminUsername(c)
		now := time.Now()
		form := contentModel.Form{
			Fcode:      newFcode,
			FormName:   formName,
			TableName:  tableName,
			CreateUser: username,
			UpdateUser: username,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := model.DB.WithContext(c.Request.Context()).Create(&form).Error; err != nil {
			fm.JSONFail(c, "新增失敗")
			return
		}
		fm.LogAction(c, "新增自定義表單成功")
		fm.JSONOKMsg(c, "新增成功")
		return
	}

	// 新增字段
	fcode := c.PostForm("fcode")
	if fcode == "" {
		fm.JSONFail(c, "表單編碼不能為空")
		return
	}
	form := contentModel.GetFormByCode(fcode)
	if form == nil {
		fm.JSONFail(c, "表單不存在")
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	length, _ := strconv.Atoi(c.DefaultPostForm("length", "20"))
	required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
	sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))

	if name == "" {
		fm.JSONFail(c, "字段名稱不能為空")
		return
	}
	// SQL 注入防護：驗證欄位名必須以字母開頭（對齊 PbootCMS PHP MemberField 驗證）
	if !common.CheckColumnName(name) {
		fm.JSONFail(c, "字段名稱必須以字母開頭，只能包含字母、數字、下劃線")
		return
	}
	if description == "" {
		fm.JSONFail(c, "字段描述不能為空")
		return
	}

	tableName := form.TableName
	// 動態建列（DDL 操作，無需 WithContext）
	var columnType string
	if cfg := config.Get(); cfg.Database.Type == "mysql" {
		columnType = fmt.Sprintf("varchar(%d)", length)
	} else {
		columnType = fmt.Sprintf("TEXT(%d)", length)
	}
	if err := model.DB.Exec(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s NULL", tableName, name, columnType)).Error; err != nil {
		fm.JSONFail(c, "操作失敗: "+err.Error())
		return
	}

	username := fm.GetAdminUsername(c)
	now := time.Now()
	formField := contentModel.FormField{
		Fcode:       fcode,
		Name:        name,
		Length:      length,
		Required:    required,
		Description: description,
		Sorting:     sorting,
		CreateUser:  username,
		UpdateUser:  username,
		CreateTime:  now,
		UpdateTime:  now,
	}
	if err := model.DB.WithContext(c.Request.Context()).Create(&formField).Error; err != nil {
		fm.JSONFail(c, "新增字段失敗")
		return
	}
	fm.LogAction(c, "新增表單字段成功")
	fm.JSONOKMsg(c, "新增成功")
}

// Del 刪除表單/字段/數據
func (fm *FormController) Del(c *gin.Context) {
	params := parseFormParams(c)
	idStr := params["id"]
	if idStr == "" {
		fm.JSONFail(c, "參數錯誤")
		return
	}
	id, _ := strconv.Atoi(idStr)
	action := params["action"]

	switch action {
	case "delform":
		if id == 1 {
			fm.JSONFail(c, "留言表單不允許刪除")
			return
		}
		var form contentModel.Form
		if err := model.DB.WithContext(c.Request.Context()).First(&form, id).Error; err != nil {
			fm.JSONFail(c, "表單不存在")
			return
		}
		tableName := form.TableName
		fcode := form.Fcode
		if err := model.DB.WithContext(c.Request.Context()).Where("fcode = ?", fcode).Delete(&contentModel.FormField{}).Error; err != nil {
			fm.JSONFail(c, "刪除失敗："+err.Error())
			return
		}
		// DDL 操作，無需 WithContext
		if err := model.DB.Exec("DROP TABLE IF EXISTS `" + tableName + "`").Error; err != nil {
			fm.JSONFail(c, "操作失敗: "+err.Error())
			return
		}
		if err := model.DB.WithContext(c.Request.Context()).Delete(&contentModel.Form{}, id).Error; err != nil {
			fm.JSONFail(c, "刪除失敗："+err.Error())
			return
		}
		// ay_menu 無 acode 欄位，無需 WithContext
		if err := model.DB.Where("url LIKE ?", "%Form/index/fcode/"+fcode+"/action/showdata%").Delete(&system.Menu{}).Error; err != nil {
			fm.JSONFail(c, "刪除失敗："+err.Error())
			return
		}
		fm.LogAction(c, "刪除自定義表單成功")
		fm.JSONOKMsg(c, "刪除成功")

	case "deldata":
		fcode := params["fcode"]
		if fcode == "" {
			fm.JSONFail(c, "參數 fcode 錯誤")
			return
		}
		tableName := contentModel.GetFormTableByCode(fcode)
		if tableName == "" {
			fm.JSONFail(c, "表單不存在")
			return
		}
		// 動態表（ay_diy_*）無 acode 欄位，無需 WithContext
		if err := model.DB.Exec("DELETE FROM `"+tableName+"` WHERE id = ?", id).Error; err != nil {
			fm.JSONFail(c, "操作失敗: "+err.Error())
			return
		}
		fm.LogAction(c, "刪除表單數據成功")
		fm.JSONOKMsg(c, "刪除成功")

	default:
		var formField contentModel.FormField
		if err := model.DB.WithContext(c.Request.Context()).First(&formField, id).Error; err != nil {
			fm.JSONFail(c, "字段不存在")
			return
		}
		fcode := formField.Fcode
		tableName := contentModel.GetFormTableByCode(fcode)
		// MySQL 刪列，SQLite 不支持（DDL 操作，無需 WithContext）
		cfg := config.Get()
		if cfg.Database.Type == "mysql" && tableName != "" {
			if err := model.DB.Exec("ALTER TABLE `" + tableName + "` DROP COLUMN `" + formField.Name + "`").Error; err != nil {
				fm.JSONFail(c, "操作失敗: "+err.Error())
				return
			}
		}
		if err := model.DB.WithContext(c.Request.Context()).Delete(&formField).Error; err != nil {
			fm.JSONFail(c, "刪除失敗："+err.Error())
			return
		}
		fm.LogAction(c, "刪除表單字段成功")
		fm.JSONOKMsg(c, "刪除成功")
	}
}

// Mod 修改表單/字段
func (fm *FormController) Mod(c *gin.Context) {
	params := parseFormParams(c)
	idStr := params["id"]
	if idStr == "" {
		fm.JSONFail(c, "參數錯誤")
		return
	}
	id, _ := strconv.Atoi(idStr)

	// 單欄位快速切換（GET: /mod/id/5/field/required/value/1）
	fieldName := params["field"]
	value := params["value"]
	if fieldName != "" && value != "" {
		if err := model.DB.WithContext(c.Request.Context()).Model(&contentModel.FormField{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				fieldName:     value,
				"update_time": time.Now(),
				"update_user": fm.GetAdminUsername(c),
			}).Error; err != nil {
			fm.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		fm.JSONOKMsg(c, "修改成功")
		return
	}

	action := params["action"]

	// 添加到菜單
	if action == "addmenu" {
		var form contentModel.Form
		if err := model.DB.WithContext(c.Request.Context()).First(&form, id).Error; err != nil {
			fm.JSONFail(c, "表單不存在")
			return
		}
		menuURL := fmt.Sprintf("/admin/Form/index/fcode/%s/action/showdata", form.Fcode)
		// ay_menu 無 acode 欄位，無需 WithContext
		var existingMenu system.Menu
		model.DB.Where("url LIKE ?", "%"+menuURL+"%").First(&existingMenu)
		if existingMenu.ID > 0 {
			if err := model.DB.Model(&system.Menu{}).Where("id = ?", existingMenu.ID).Update("name", form.FormName).Error; err != nil {
				fm.JSONFail(c, "修改失敗："+err.Error())
				return
			}
			c.JSON(200, gin.H{"code": 1, "data": "菜單已更新", "msg": "菜單已更新", "tourl": "/admin/Form/index"})
			return
		}
		var lastMcode string
		model.DB.Model(&system.Menu{}).Order("mcode DESC").Limit(1).Pluck("mcode", &lastMcode)
		newMcode := "MF" + form.Fcode
		if lastMcode != "" {
			if n, err := strconv.Atoi(lastMcode[1:]); err == nil {
				newMcode = fmt.Sprintf("M%d", n+1)
			}
		}
		if err := model.DB.Create(&system.Menu{
			Mcode:      newMcode,
			Pcode:      "M157",
			Name:       form.FormName,
			URL:        menuURL,
			Sorting:    599,
			Status:     1,
			Shortcut:   0,
			Ico:        "fa-plus-square-o",
			CreateUser: fm.GetAdminUsername(c),
			UpdateUser: fm.GetAdminUsername(c),
		}).Error; err != nil {
			fm.JSONFail(c, "新增失敗："+err.Error())
			return
		}
		fm.LogAction(c, "添加自定義表單到菜單成功")
		c.JSON(200, gin.H{"code": 1, "data": "添加成功", "msg": "添加成功", "tourl": "/admin/Form/index"})
		return
	}

	// POST 修改操作
	if c.Request.Method == "POST" {
		if action == "modform" {
			formName := c.PostForm("form_name")
			if formName == "" {
				fm.JSONFail(c, "表單名稱不能為空")
				return
			}
			if err := model.DB.WithContext(c.Request.Context()).Model(&contentModel.Form{}).Where("id = ?", id).Updates(map[string]interface{}{
				"form_name":   formName,
				"update_time": time.Now(),
				"update_user": fm.GetAdminUsername(c),
			}).Error; err != nil {
				fm.JSONFail(c, "修改失敗："+err.Error())
				return
			}
			fm.LogAction(c, "修改自定義表單成功")
			fm.JSONOKMsg(c, "修改成功")
			return
		}
		description := c.PostForm("description")
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "255"))
		if description == "" {
			fm.JSONFail(c, "字段描述不能為空")
			return
		}
		if err := model.DB.WithContext(c.Request.Context()).Model(&contentModel.FormField{}).Where("id = ?", id).Updates(map[string]interface{}{
			"description": description,
			"required":    required,
			"sorting":     sorting,
			"update_time": time.Now(),
			"update_user": fm.GetAdminUsername(c),
		}).Error; err != nil {
			fm.JSONFail(c, "修改失敗："+err.Error())
			return
		}
		fm.LogAction(c, "修改表單字段成功")
		fm.JSONOKMsg(c, "修改成功")
		return
	}

	// GET 顯示修改表單
	if action == "modform" {
		var form contentModel.Form
		if err := model.DB.WithContext(c.Request.Context()).First(&form, id).Error; err != nil {
			fm.JSONFail(c, "編輯的內容已不存在")
			return
		}
		data := gin.H{
			"mod":  true,
			"form": form,
			"get":  params,
		}
		injectGetFlat(data, params)
		common.Render(c, "content/form.html", data)
		return
	}

	// 顯示字段修改表單
	var field contentModel.FormField
	if err := model.DB.WithContext(c.Request.Context()).First(&field, id).Error; err != nil {
		fm.JSONFail(c, "編輯的內容已不存在")
		return
	}
	data := gin.H{
		"mod":   true,
		"field": field,
		"get":   params,
	}
	injectGetFlat(data, params)
	common.Render(c, "content/form.html", data)
}

// Clear 清空表單數據
func (fm *FormController) Clear(c *gin.Context) {
	params := parseFormParams(c)
	fcode := params["fcode"]
	if fcode == "" {
		fm.JSONFail(c, "參數 fcode 錯誤")
		return
	}
	tableName := contentModel.GetFormTableByCode(fcode)
	if tableName == "" {
		fm.JSONFail(c, "表單不存在")
		return
	}
	// 動態表（ay_diy_*）無 acode 欄位，無需 WithContext
	if err := model.DB.Exec("DELETE FROM `" + tableName + "`").Error; err != nil {
		fm.JSONFail(c, "操作失敗: "+err.Error())
		return
	}
	fm.JSONOKMsg(c, "清空成功")
}
