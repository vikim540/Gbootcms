package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"pbootcms-go/config"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FormController - Custom Form Controller
// Corresponds to PHP: apps/admin/controller/FormController.php
type FormController struct {
	common.BaseController
}

// Index - Form management
func (fm *FormController) Index(c *gin.Context) {
	fcode := c.Query("fcode")
	action := c.Query("action")

	if fcode == "" {
		field := c.Query("field")
		keyword := c.Query("keyword")

		var forms []model.Form
		query := model.DB.Model(&model.Form{})
		if field != "" && keyword != "" {
			query = query.Where(field+" LIKE ?", "%"+keyword+"%")
		}
		query.Order("id ASC").Find(&forms)

		common.Render(c, "content/form.html", gin.H{
			"forms":   forms,
			"field":   field,
			"keyword": keyword,
		})
		return
	}

	switch action {
	case "showdata":
		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err != nil {
			fm.JSONFail(c, "Form does not exist")
			return
		}

		var fields []model.FormField
		model.DB.Where("fcode = ?", fcode).Order("sorting ASC").Find(&fields)

		var dataList []map[string]interface{}
		tableName := form.Table
		if tableName == "" {
			tableName = model.TableName("form_data_" + fcode)
		}
		model.DB.Raw("SELECT * FROM " + tableName + " ORDER BY id DESC").Scan(&dataList)

		common.Render(c, "content/form.html", gin.H{
			"form":     form,
			"fields":   fields,
			"dataList": dataList,
			"action":   "showdata",
		})

	case "showfield":
		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err != nil {
			fm.JSONFail(c, "Form does not exist")
			return
		}

		var fields []model.FormField
		model.DB.Where("fcode = ?", fcode).Order("sorting ASC").Find(&fields)

		common.Render(c, "content/form.html", gin.H{
			"form":   form,
			"fields": fields,
			"action": "showfield",
		})

	default:
		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err != nil {
			fm.JSONFail(c, "Form does not exist")
			return
		}

		var fields []model.FormField
		model.DB.Where("fcode = ?", fcode).Order("sorting ASC").Find(&fields)

		common.Render(c, "content/form.html", gin.H{
			"form":   form,
			"fields": fields,
		})
	}
}

// Add - Add form/field
func (fm *FormController) Add(c *gin.Context) {
	action := c.PostForm("action")

	switch action {
	case "addform":
		formName := c.PostForm("form_name")
		if formName == "" {
			fm.JSONFail(c, "Form name cannot be empty")
			return
		}

		var maxFcode int
		var existingForms []model.Form
		model.DB.Order("id ASC").Find(&existingForms)
		for _, f := range existingForms {
			code, _ := strconv.Atoi(f.Fcode)
			if code > maxFcode {
				maxFcode = code
			}
		}
		newFcode := strconv.Itoa(maxFcode + 1)

		tableName := c.PostForm("table_name")
		if tableName == "" {
			tableName = model.TableName("form_data_" + newFcode)
		}

		form := model.Form{
			Fcode:    newFcode,
			FormName: formName,
			Table:    tableName,
			Status:   1,
		}
		model.DB.Create(&form)

		cfg := config.Get()
		if cfg.Database.Type == "mysql" {
			model.DB.Exec("CREATE TABLE `" + tableName + "` (" +
				"`id` INT UNSIGNED NOT NULL AUTO_INCREMENT," +
				"`createtime` DATETIME NULL DEFAULT NULL," +
				"PRIMARY KEY (`id`)" +
				") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
		}

		fm.JSONOKMsg(c, "表單新增成功")

	case "addfield":
		fcode := c.PostForm("fcode")
		if fcode == "" {
			fm.JSONFail(c, "Form code cannot be empty")
			return
		}

		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err != nil {
			fm.JSONFail(c, "Form does not exist")
			return
		}

		name := c.PostForm("name")
		fieldName := c.PostForm("field")
		length := c.DefaultPostForm("length", "255")
		if name == "" || fieldName == "" {
			fm.JSONFail(c, "Field name and identifier cannot be empty")
			return
		}

		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))

		formField := model.FormField{
			Fcode:    fcode,
			Name:     name,
			Field:    fieldName,
			Type:     c.DefaultPostForm("type", "text"),
			Required: required,
			Sorting:  sorting,
			Status:   1,
		}
		model.DB.Create(&formField)

		tableName := form.Table
		if tableName == "" {
			tableName = model.TableName("form_data_" + fcode)
		}
		model.DB.Exec("ALTER TABLE `" + tableName + "` ADD COLUMN `" + fieldName + "` VARCHAR(" + length + ") NULL")

		fm.JSONOKMsg(c, "字段新增成功")

	default:
		fm.JSONFail(c, "Unknown operation")
	}
}

// Del - Delete form/field/data
func (fm *FormController) Del(c *gin.Context) {
	action := c.Query("action")

	switch action {
	case "delform":
		fcode := c.Query("fcode")
		if fcode == "" {
			fm.JSONFail(c, "Form code cannot be empty")
			return
		}

		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err != nil {
			fm.JSONFail(c, "Form does not exist")
			return
		}

		tableName := form.Table
		if tableName == "" {
			tableName = model.TableName("form_data_" + fcode)
		}
		model.DB.Exec("DROP TABLE IF EXISTS `" + tableName + "`")
		model.DB.Where("fcode = ?", fcode).Delete(&model.FormField{})
		model.DB.Where("fcode = ?", fcode).Delete(&model.Form{})

		fm.JSONOKMsg(c, "表單刪除成功")

	case "deldata":
		fcode := c.Query("fcode")
		id := c.Query("id")
		if fcode == "" || id == "" {
			fm.JSONFail(c, "Incomplete parameters")
			return
		}

		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err != nil {
			fm.JSONFail(c, "Form does not exist")
			return
		}

		tableName := form.Table
		if tableName == "" {
			tableName = model.TableName("form_data_" + fcode)
		}
		model.DB.Exec("DELETE FROM `"+tableName+"` WHERE id = ?", id)

		fm.JSONOKMsg(c, "數據刪除成功")

	default:
		idStr := c.Query("id")
		if idStr == "" {
			fm.JSONFail(c, "Field ID cannot be empty")
			return
		}
		id, _ := strconv.Atoi(idStr)

		var formField model.FormField
		if err := model.DB.First(&formField, id).Error; err != nil {
			fm.JSONFail(c, "Field does not exist")
			return
		}

		fcode := formField.Fcode
		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err == nil {
			tableName := form.Table
			if tableName == "" {
				tableName = model.TableName("form_data_" + fcode)
			}
			cfg := config.Get()
			if cfg.Database.Type == "mysql" {
				model.DB.Exec("ALTER TABLE `" + tableName + "` DROP COLUMN `" + formField.Field + "`")
			}
		}

		model.DB.Delete(&formField)
		fm.JSONOKMsg(c, "字段刪除成功")
	}
}

// Mod - Modify form/field
func (fm *FormController) Mod(c *gin.Context) {
	field := c.PostForm("field")
	value := c.PostForm("value")

	if field != "" && value != "" {
		idStr := c.PostForm("id")
		if idStr == "" {
			fm.JSONFail(c, "ID cannot be empty")
			return
		}

		target := c.PostForm("target")
		if target == "form" {
			model.DB.Model(&model.Form{}).Where("id = ?", idStr).Update(field, value)
		} else if target == "field" {
			model.DB.Model(&model.FormField{}).Where("id = ?", idStr).Update(field, value)
		} else {
			model.DB.Model(&model.FormField{}).Where("id = ?", idStr).Update(field, value)
		}
		fm.JSONOKMsg(c, "修改成功")
		return
	}

	action := c.PostForm("action")

	switch action {
	case "addmenu":
		fcode := c.PostForm("fcode")
		if fcode == "" {
			fm.JSONFail(c, "Form code cannot be empty")
			return
		}

		var form model.Form
		if err := model.DB.Where("fcode = ?", fcode).First(&form); err.Error != nil {
			fm.JSONFail(c, "Form does not exist")
			return
		}

		var menu model.Menu
		model.DB.Where("url = ?", "/admin/content/form/index?fcode="+fcode).First(&menu)
		if menu.ID > 0 {
			fm.JSONFail(c, "Menu already exists")
			return
		}

		var maxSort int
		model.DB.Model(&model.Menu{}).Where("pcode = 'M130'").
			Select("MAX(sorting)").Scan(&maxSort)

		model.DB.Create(&model.Menu{
			Mcode:   "MF" + fcode,
			Pcode:   "M130",
			Name:    form.FormName,
			URL:     "/admin/content/form/index?fcode=" + fcode,
			Ico:     "fa-wpforms",
			Sorting: maxSort + 1,
			Status:  1,
		})

		fm.JSONOKMsg(c, "菜單新增成功")

	case "modform":
		fcode := c.PostForm("fcode")
		if fcode == "" {
			fm.JSONFail(c, "Form code cannot be empty")
			return
		}

		formName := c.PostForm("form_name")
		if formName == "" {
			fm.JSONFail(c, "Form name cannot be empty")
			return
		}

		model.DB.Model(&model.Form{}).Where("fcode = ?", fcode).Update("form_name", formName)
		fm.JSONOKMsg(c, "表單修改成功")

	default:
		idStr := c.PostForm("id")
		if idStr == "" {
			fm.JSONFail(c, "Field ID cannot be empty")
			return
		}
		id, _ := strconv.Atoi(idStr)

		updates := map[string]interface{}{
			"name":     c.PostForm("name"),
			"required": c.DefaultPostForm("required", "0"),
			"sorting":  c.DefaultPostForm("sorting", "0"),
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("required", "0")); err == nil {
			updates["required"] = v
		}
		if v, err := strconv.Atoi(c.DefaultPostForm("sorting", "0")); err == nil {
			updates["sorting"] = v
		}

		model.DB.Model(&model.FormField{}).Where("id = ?", id).Updates(updates)
		fm.JSONOKMsg(c, "字段修改成功")
	}
}

// Clear - Clear form data
func (fm *FormController) Clear(c *gin.Context) {
	fcode := c.Query("fcode")
	if fcode == "" {
		fcode = c.PostForm("fcode")
	}
	if fcode == "" {
		fm.JSONFail(c, "Form code cannot be empty")
		return
	}

	var form model.Form
	if err := model.DB.Where("fcode = ?", fcode).First(&form).Error; err != nil {
		fm.JSONFail(c, "Form does not exist")
		return
	}

	tableName := form.Table
	if tableName == "" {
		tableName = model.TableName("form_data_" + fcode)
	}
	model.DB.Exec("DELETE FROM `" + tableName + "`")

	fm.JSONOKMsg(c, "表單數據清理成功")
}
