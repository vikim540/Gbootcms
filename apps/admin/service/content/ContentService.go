package content

import (
	"errors"
	"fmt"
	"pbootcms-go/apps/admin/helper"
	"pbootcms-go/apps/admin/model"
	contentModel "pbootcms-go/apps/admin/model/content"
	"pbootcms-go/apps/common"
	"strconv"
	"strings"
	"time"
)

// ContentService handles content business logic
type ContentService struct{}

// ListContents returns paginated content list and total count
func (s *ContentService) ListContents(mcode, scode, keyword string, page, pageSize int) ([]model.Content, int64, error) {
	if page < 1 {
		page = 1
	}
	var total int64
	query := model.DB.Model(&model.Content{}).Where("status >= 0")
	if mcode != "" {
		query = query.Where("scode IN (SELECT scode FROM ay_content_sort WHERE mcode = ?)", mcode)
	}
	if scode != "" {
		query = query.Where("scode = ? OR subscode = ?", scode, scode)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR tags LIKE ?", like, like)
	}
	query.Count(&total)

	var contents []model.Content
	err := query.Order("date DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&contents).Error
	return contents, total, err
}

// GetContent returns a single content by ID
func (s *ContentService) GetContent(id int) (*model.Content, error) {
	var doc model.Content
	err := model.DB.First(&doc, id).Error
	if err != nil {
		return nil, errors.New("內容不存在")
	}
	return &doc, nil
}

// GetContentWithExt 返回內容及其擴展數據，合併為一個 map（用於編輯表單回填）
func (s *ContentService) GetContentWithExt(id int) (map[string]interface{}, error) {
	doc, err := s.GetContent(id)
	if err != nil {
		return nil, err
	}

	// 將 Content struct 轉為 map
	result := helper.StructToMap(doc)

	// 加載擴展數據並合併
	extData := contentModel.GetContentExtByContentID(doc.ID)
	if extData != nil {
		for k, v := range extData {
			if k != "extid" && k != "contentid" {
				// 將 ext 字段名轉為 PascalCase 以匹配模板約定
				result[common.SnakeToPascal(k)] = v
			}
		}
	}

	return result, nil
}

// CollectExtFieldData 從表單收集擴展字段數據。
// extFields: 該模型的擴展字段定義列表
// postForm: 從 gin.Context 獲取單值表單參數的函數
// postFormArray: 從 gin.Context 獲取多值表單參數的函數
func (s *ContentService) CollectExtFieldData(
	extFields []contentModel.ExtField,
	postForm func(key string) string,
	postFormArray func(key string) []string,
) map[string]interface{} {
	data := make(map[string]interface{})
	for _, ef := range extFields {
		fieldName := ef.Field // DB 列名
		if fieldName == "" {
			fieldName = ef.Name // 向後兼容舊數據
		}
		if fieldName == "" {
			continue
		}
		// 多選 checkbox 提交時帶 [] 後綴
		arr := postFormArray(fieldName + "[]")
		if len(arr) > 0 {
			data[fieldName] = strings.Join(arr, ",")
		} else {
			val := postForm(fieldName)
			// 多行文本：換行符替換為 <br>（與 PHP 一致）
			if ef.Type == "2" {
				data[fieldName] = strings.ReplaceAll(val, "\r\n", "<br>")
			} else {
				data[fieldName] = val
			}
		}
	}
	return data
}

// CreateContent creates a new content record with optional ext field data
func (s *ContentService) CreateContent(doc *model.Content, extData map[string]interface{}) error {
	if doc.Title == "" || doc.Scode == "" {
		return errors.New("標題和欄目不能為空")
	}
	if doc.Date.IsZero() {
		doc.Date = time.Now()
	}
	if doc.Status == 0 {
		doc.Status = 1
	}
	if err := model.DB.Create(doc).Error; err != nil {
		return err
	}
	// 如果有擴展數據，插入 ay_content_ext
	if len(extData) > 0 {
		extData["contentid"] = doc.ID
		if err := contentModel.InsertContentExt(extData); err != nil {
			// 回滾：刪除已插入的內容
			model.DB.Delete(&model.Content{}, doc.ID)
			return errors.New("創建擴展數據失敗: " + err.Error())
		}
	}
	return nil
}

// UpdateContent updates a content record with a map of fields and optional ext data
func (s *ContentService) UpdateContent(id int, updates map[string]interface{}, extData map[string]interface{}) error {
	if err := model.DB.Model(&model.Content{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	if len(extData) > 0 {
		if err := contentModel.UpsertContentExt(uint(id), extData); err != nil {
			return errors.New("更新擴展數據失敗: " + err.Error())
		}
	}
	return nil
}

// DeleteContent deletes contents by comma-separated IDs (also deletes ext data)
func (s *ContentService) DeleteContent(ids []string) error {
	for _, id := range ids {
		if err := model.DB.Delete(&model.Content{}, id).Error; err != nil {
			return err
		}
		// 同步刪除擴展數據
		if idInt, err := strconv.Atoi(id); err == nil {
			contentModel.DeleteContentExt(uint(idInt))
		}
	}
	return nil
}

// CopyContent copies content to a target scode (also copies ext data)
func (s *ContentService) CopyContent(id int, targetScode string) error {
	if targetScode == "" {
		return errors.New("目標欄目不能為空")
	}
	var src model.Content
	if err := model.DB.First(&src, id).Error; err != nil {
		return errors.New("內容不存在")
	}
	copyDoc := model.Content{
		Scode:       targetScode,
		Subscode:    src.Subscode,
		Title:       src.Title,
		Subtitle:    src.Subtitle,
		Keywords:    src.Keywords,
		Description: src.Description,
		Content:     src.Content,
		Ico:         src.Ico,
		Pics:        src.Pics,
		Source:      src.Source,
		Author:      src.Author,
		Visits:      0,
		IsTop:       src.IsTop,
		IsRecommend: src.IsRecommend,
		IsHeadline:  src.IsHeadline,
		Date:        time.Now(),
		Sorting:     src.Sorting,
		Status:      1,
	}
	if err := model.DB.Create(&copyDoc).Error; err != nil {
		return err
	}
	// 複製擴展數據
	extData := contentModel.GetContentExtByContentID(src.ID)
	if extData != nil {
		newExt := make(map[string]interface{})
		for k, v := range extData {
			if k != "extid" && k != "contentid" {
				newExt[k] = v
			}
		}
		if len(newExt) > 0 {
			newExt["contentid"] = copyDoc.ID
			contentModel.InsertContentExt(newExt)
		}
	}
	return nil
}

// MoveContent moves content to a target scode
func (s *ContentService) MoveContent(id int, targetScode string) error {
	if targetScode == "" {
		return errors.New("目標欄目不能為空")
	}
	return model.DB.Model(&model.Content{}).Where("id = ?", id).Update("scode", targetScode).Error
}

// UpdateSorting updates sorting for multiple contents
func (s *ContentService) UpdateSorting(idSortingMap map[string]int) error {
	for idStr, sorting := range idSortingMap {
		if err := model.DB.Model(&model.Content{}).Where("id = ?", idStr).Update("sorting", sorting).Error; err != nil {
			return err
		}
	}
	return nil
}

// allowedSingleFields defines the whitelist for single-field updates
var allowedSingleFields = map[string]bool{
	"status":      true,
	"istop":       true,
	"isrecommend": true,
	"isheadline":  true,
	"sorting":     true,
	"title":       true,
}

// UpdateSingleField updates a single field with whitelist validation
func (s *ContentService) UpdateSingleField(id int, field, value string) error {
	if !allowedSingleFields[field] {
		return errors.New("不允許修改的欄位: " + field)
	}
	return model.DB.Model(&model.Content{}).Where("id = ?", id).Update(field, value).Error
}

// GetAllSorts returns all active sorts ordered by sorting
func (s *ContentService) GetAllSorts() ([]model.ContentSort, error) {
	var sorts []model.ContentSort
	err := model.DB.Where("status = 1").Order("sorting ASC").Find(&sorts).Error
	return sorts, err
}

// BuildExtFieldTemplateData 為每個擴展字段構建模板友好的數據結構，
// 包含當前值（編輯時）、選項列表（單選/多選/下拉/多圖）、已選值等。
func (s *ContentService) BuildExtFieldTemplateData(mcode string, contentMap map[string]interface{}) []map[string]interface{} {
	fields := helper.GetExtFieldsByMcode(mcode)
	if len(fields) == 0 {
		return []map[string]interface{}{}
	}
	result := make([]map[string]interface{}, len(fields))
	for i, ef := range fields {
		item := map[string]interface{}{
			"Name":           ef.Name,
			"Field":          ef.Field,
			"Type":           ef.Type,
			"Description":    ef.Description,
			"Required":       ef.Required,
			"Value":          ef.Value,
			"CurrentValue":   "",
			"Options":        []string{},
			"SelectedValues": []string{},
			"Pics":           []string{},
		}

		// 從 contentMap 中取當前值（編輯時）
		if contentMap != nil {
			fieldName := ef.Field
			if fieldName == "" {
				fieldName = ef.Name // 向後兼容
			}
			pascalName := common.SnakeToPascal(fieldName)
			if v, ok := contentMap[pascalName]; ok && v != nil {
				item["CurrentValue"] = fmt.Sprintf("%v", v)
			}
		}

		// 多行文本：將 <br> 轉回 \r\n 以便 textarea 顯示
		if ef.Type == "2" {
			if cv, ok := item["CurrentValue"].(string); ok && cv != "" {
				item["CurrentValueMultiline"] = strings.ReplaceAll(cv, "<br>", "\r\n")
			} else {
				item["CurrentValueMultiline"] = ""
			}
		}

		// 選項類字段：預處理選項列表
		if ef.Type == "3" || ef.Type == "4" || ef.Type == "9" {
			if ef.Value != "" {
				item["Options"] = strings.Split(ef.Value, ",")
			}
		}

		// 多選：預處理已選值
		if ef.Type == "4" {
			if cv, ok := item["CurrentValue"].(string); ok && cv != "" {
				item["SelectedValues"] = strings.Split(cv, ",")
			}
		}

		// 多圖：預處理圖片路徑
		if ef.Type == "10" {
			if cv, ok := item["CurrentValue"].(string); ok && cv != "" {
				item["Pics"] = strings.Split(cv, ",")
			}
		}

		result[i] = item
	}
	return result
}
