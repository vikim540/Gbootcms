package content

import (
	"fmt"
	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model/content"
	"gbootcms/apps/common"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type ExtFieldController struct {
	common.BaseController
}

func (ef *ExtFieldController) Index(c *gin.Context) {
	// 確保 scode 列存在並遷移舊數據
	content.EnsureExtFieldScodeColumn()
	content.MigrateScodeFromValue()

	page, pageSize, offset := ef.Paginate(c)

	allFields := content.GetAllExtFields()
	total := int64(len(allFields))
	fields := allFields
	if offset >= len(fields) {
		fields = fields[:0]
	} else {
		end := offset + pageSize
		if end > len(fields) {
			end = len(fields)
		}
		fields = fields[offset:end]
	}
	// 為列表預處理：將 scode 字串解析為欄目名稱顯示字串
	listFields := make([]map[string]interface{}, len(fields))
	sorts := content.GetAllContentSorts(c.Request.Context())
	// 建立 scode → name 查詢表
	scodeNameMap := make(map[string]string)
	for _, s := range sorts {
		prefix := ""
		if s.Pcode != "0" {
			prefix = "├ "
		}
		scodeNameMap[s.Scode] = prefix + s.Name
	}
	for i, f := range fields {
		// 預計算適用欄目的顯示字串
		scodeDisplay := ""
		if f.Scode != "" {
			var names []string
			for _, sc := range strings.Split(f.Scode, ",") {
				sc = strings.TrimSpace(sc)
				if name, ok := scodeNameMap[sc]; ok {
					names = append(names, name)
				}
			}
			scodeDisplay = strings.Join(names, "、")
		}
		listFields[i] = map[string]interface{}{
			"ID":           f.ID,
			"Mcode":        f.Mcode,
			"Name":         f.Name,
			"Field":        f.Field,
			"Type":         f.Type,
			"Description":  f.Description,
			"Value":        f.Value,
			"Scode":        f.Scode,
			"ScodeDisplay": scodeDisplay,
			"Required":     f.Required,
			"Sorting":      f.Sorting,
			"Status":       f.Status,
		}
	}
	models := content.GetAllModels()
	baseURL := "/admin/content/extField/index"
	common.Render(c, "content/extfield.html", gin.H{
		"extfields": listFields,
		"models":    models,
		"sorts":     sorts,
		"list":      true,
		"C":         "extField",
		"pagebar":   helper.BuildPagebarHTML(total, page, pageSize, baseURL),
		"pagesize":  pageSize,
	})
}

func (ef *ExtFieldController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		description := c.PostForm("description")
		field := c.PostForm("field")
		if field == "" {
			field = description
		}
		// 對齊 PHP: $name = "ext_" . $name; 強制添加 ext_ 前綴
		if !strings.HasPrefix(field, "ext_") {
			field = "ext_" + field
		}
		typ := c.PostForm("type")
		options := content.NormalizeOptions(c.PostForm("value"))
		// 接收多選 scode（陣列），用逗號 join
		// layui getValue() 可能將 scode[] 重命名為 scode[0]、scode[1] 等，需兼容
		scodeList := c.PostFormArray("scode[]")
		if len(scodeList) == 0 {
			for i := 0; i < 50; i++ {
				if v := c.PostForm(fmt.Sprintf("scode[%d]", i)); v != "" {
					scodeList = append(scodeList, v)
				}
			}
		}
		scode := strings.Join(scodeList, ",")
		// 檢查 field 名稱在同行模型下是否唯一
		if content.CheckFieldUnique(c.PostForm("mcode"), field, 0) {
			ef.JSONFail(c, "字段名稱「"+field+"」在該模型下已存在，同一模型下不允許重複的欄位名稱")
			return
		}
		err := content.AddExtField(
			c.PostForm("mcode"),
			description,
			field,
			typ,
			options,
			scode,
			required,
			sorting,
		)
		if err != nil {
			ef.JSONFail(c, "新增失敗: "+err.Error())
			return
		}
		if field != "" {
			content.EnsureExtColumnExists(field, typ)
		}
		ef.JSONOKMsg(c, common.NoticeAdd)
		return
	}
	content.EnsureExtFieldScodeColumn()
	common.Render(c, "content/extfield.html", gin.H{
		"action": "add",
		"models": content.GetAllModels(),
		"sorts":  content.GetAllContentSorts(c.Request.Context()),
		"list":   true,
		"C":      "extField",
	})
}

func (ef *ExtFieldController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))

	idStr := params["id"]
	if idStr == "" {
		idStr = c.Query("id")
	}
	id, _ := strconv.Atoi(idStr)

	// GET 單欄位快速切換
	fieldName := params["field"]
	if fieldName == "" {
		fieldName = c.Query("field")
	}
	fieldValue := params["value"]
	if fieldValue == "" {
		fieldValue = c.Query("value")
	}
	if c.Request.Method == "GET" && fieldName != "" && fieldValue != "" {
		if fieldName == "status" {
			content.UpdateExtFieldSingleField(id, fieldName, fieldValue)
			ef.JSONOKMsg(c, common.NoticeModify)
			return
		}
		ef.JSONFail(c, "不允許修改的欄位")
		return
	}

	// POST 全量修改
	if c.Request.Method == "POST" {
		sorting, _ := strconv.Atoi(c.DefaultPostForm("sorting", "0"))
		required, _ := strconv.Atoi(c.DefaultPostForm("required", "0"))
		description := c.PostForm("description")
		options := content.NormalizeOptions(c.PostForm("value"))
		// layui getValue() 可能將 scode[] 重命名為 scode[0]、scode[1] 等，需兼容
		scodeList := c.PostFormArray("scode[]")
		if len(scodeList) == 0 {
			for i := 0; i < 50; i++ {
				if v := c.PostForm(fmt.Sprintf("scode[%d]", i)); v != "" {
					scodeList = append(scodeList, v)
				}
			}
		}
		scode := strings.Join(scodeList, ",")
		// 檢查 field 名稱在同行模型下是否唯一（排除自身）
		modField := c.PostForm("field")
		if content.CheckFieldUnique(c.PostForm("mcode"), modField, id) {
			ef.JSONFail(c, "字段名稱「"+modField+"」在該模型下已存在，同一模型下不允許重複的欄位名稱")
			return
		}
		err := content.UpdateExtField(
			id,
			c.PostForm("mcode"),
			description,
			c.PostForm("field"),
			c.PostForm("type"),
			options,
			scode,
			required,
			sorting,
		)
		if err != nil {
			ef.JSONFail(c, "修改失敗: "+err.Error())
			return
		}
		ef.JSONOKMsg(c, common.NoticeModify)
		return
	}

	// GET 顯示修改表單
	content.EnsureExtFieldScodeColumn()
	field := content.GetExtFieldById(id)
	models := content.GetAllModels()
	sorts := content.GetAllContentSorts(c.Request.Context())
	// 將 scode 字串拆成陣列供模板多選回顯
	var scodeArr []string
	if field.Scode != "" {
		scodeArr = strings.Split(field.Scode, ",")
	}
	common.Render(c, "content/extfield.html", gin.H{
		"extfield":      field,
		"extfieldScode": field.Scode,
		"extfieldScodes": scodeArr,
		"models":        models,
		"sorts":         sorts,
		"mod":           true,
		"C":             "extField",
	})
}

func (ef *ExtFieldController) Del(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	err := content.DeleteExtField(id)
	if err != nil {
		ef.JSONFail(c, "刪除失敗: "+err.Error())
		return
	}
	ef.JSONOKMsg(c, common.NoticeDelete)
}
