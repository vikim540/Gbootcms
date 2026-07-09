package content

import (
	"context"
	"errors"
	"fmt"
	"gbootcms/apps/admin/model"
	contentmodel "gbootcms/apps/admin/model/content"
	"strings"
	"time"
)

// ContentSortService handles content sort business logic
type ContentSortService struct{}

// ListSorts returns all sorts ordered by sorting
func (s *ContentSortService) ListSorts(ctx context.Context) ([]model.ContentSort, error) {
	var sorts []model.ContentSort
	err := model.DB.WithContext(ctx).Order("sorting ASC, id ASC").Find(&sorts).Error
	return sorts, err
}

// GetSort returns a single sort by ID
func (s *ContentSortService) GetSort(ctx context.Context, id int) (*model.ContentSort, error) {
	var sort model.ContentSort
	err := model.DB.WithContext(ctx).First(&sort, id).Error
	if err != nil {
		return nil, errors.New("欄目不存在")
	}
	return &sort, nil
}

// GetSortByScode returns a single sort by scode.
// If scode looks like a numeric id and no scode match is found,
// fall back to id-based lookup (covers the case where the DB
// was seeded without scode values).
func (s *ContentSortService) GetSortByScode(ctx context.Context, scode string) (*model.ContentSort, error) {
	var sort model.ContentSort
	err := model.DB.WithContext(ctx).Where("scode = ?", scode).First(&sort).Error
	if err == nil {
		return &sort, nil
	}
	// Fallback: try id-based lookup
	var byID model.ContentSort
	if err2 := model.DB.WithContext(ctx).Where("id = ?", scode).First(&byID).Error; err2 == nil {
		return &byID, nil
	}
	return nil, errors.New("欄目不存在")
}

// BatchAddSorts creates multiple sorts from comma-separated names
func (s *ContentSortService) BatchAddSorts(ctx context.Context, multiplename, pcode string) error {
	names := splitAndTrim(multiplename)
	if len(names) == 0 {
		return nil
	}
	var lastSort model.ContentSort
	model.DB.WithContext(ctx).Order("id DESC").First(&lastSort)
	lastCodeNum := 0
	fmt.Sscanf(lastSort.Scode, "%d", &lastCodeNum)

	for _, name := range names {
		if name == "" {
			continue
		}
		lastCodeNum++
		newScode := fmt.Sprintf("%d", lastCodeNum)
		model.DB.WithContext(ctx).Create(&model.ContentSort{
			Scode:  newScode,
			Pcode:  pcode,
			Name:   name,
			Type:   1,
			Sort:   lastCodeNum,
			Status: 1,
		})
	}
	return nil
}

// CreateSort creates a new sort
func (s *ContentSortService) CreateSort(ctx context.Context, sort *model.ContentSort) error {
	if sort.Name == "" {
		return errors.New("欄目名稱不能為空")
	}
	// scode 為空時自動生成（與 BatchAddSorts 一致）
	if sort.Scode == "" {
		var lastSort model.ContentSort
		model.DB.WithContext(ctx).Order("id DESC").First(&lastSort)
		lastCodeNum := 0
		fmt.Sscanf(lastSort.Scode, "%d", &lastCodeNum)
		sort.Scode = fmt.Sprintf("%d", lastCodeNum+1)
	}
	if sort.Status == 0 {
		sort.Status = 1
	}

	// URL 名稱驗證 + 衝突處理（與 PbootCMS PHP 一致）
	sort.Filename = strings.Trim(sort.Filename, "/")
	if !contentmodel.IsValidFilename(sort.Filename) {
		return errors.New("URL名稱只允許字母、數字、橫線、斜線組成")
	}
	if contentmodel.CheckUrlname(sort.Filename) {
		return errors.New("URL名稱與模型URL名稱衝突，請換一個名稱")
	}
	if sort.Filename != "" {
		sort.Filename = contentmodel.GenerateUniqueFilename(ctx, sort.Filename)
	}

	if err := model.DB.WithContext(ctx).Create(sort).Error; err != nil {
		return err
	}
	// If type=1 (list) and no outlink, create initial content
	if sort.Type == 1 && sort.Outlink == "" {
		model.DB.WithContext(ctx).Create(&model.Content{
			Scode:  sort.Scode,
			Title:  sort.Name,
			Status: 1,
			Date:   time.Now(),
		})
	}
	return nil
}

// UpdateSort updates a sort record
func (s *ContentSortService) UpdateSort(ctx context.Context, id int, updates map[string]interface{}) error {
	if err := validateAndNormalizeFilenameUpdate(ctx, updates, "id="+fmt.Sprintf("%d", id)); err != nil {
		return err
	}
	return model.DB.WithContext(ctx).Model(&model.ContentSort{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateSortByScode updates a sort record by scode, with id fallback
func (s *ContentSortService) UpdateSortByScode(ctx context.Context, scode string, updates map[string]interface{}) error {
	if err := validateAndNormalizeFilenameUpdate(ctx, updates, "scode <> '"+scode+"'"); err != nil {
		return err
	}
	res := model.DB.WithContext(ctx).Model(&model.ContentSort{}).Where("scode = ?", scode).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// scode not found in DB — try id-based fallback
		res2 := model.DB.WithContext(ctx).Model(&model.ContentSort{}).Where("id = ?", scode).Updates(updates)
		if res2.Error != nil {
			return res2.Error
		}
		if res2.RowsAffected == 0 {
			return errors.New("欄目不存在")
		}
	}
	return nil
}

// validateAndNormalizeFilenameUpdate 對 updates 中的 filename 做完整 PbootCMS 校驗鏈
// excludeWhere 為排除自身的 WHERE 條件
func validateAndNormalizeFilenameUpdate(ctx context.Context, updates map[string]interface{}, excludeWhere string) error {
	raw, ok := updates["filename"]
	if !ok {
		return nil
	}
	filename, _ := raw.(string)
	filename = strings.Trim(filename, "/")
	updates["filename"] = filename

	if !contentmodel.IsValidFilename(filename) {
		return errors.New("URL名稱只允許字母、數字、橫線、斜線組成")
	}
	if contentmodel.CheckUrlname(filename) {
		return errors.New("URL名稱與模型URL名稱衝突，請換一個名稱")
	}
	if filename != "" {
		updates["filename"] = contentmodel.GenerateUniqueFilename(ctx, filename, excludeWhere)
	}
	return nil
}

// UpdateSortByScodeField updates a single field by scode with whitelist validation, with id fallback
func (s *ContentSortService) UpdateSortByScodeField(ctx context.Context, scode, field, value string) error {
	if !allowedSortSingleFields[field] {
		return errors.New("不允許修改的欄位: " + field)
	}
	res := model.DB.WithContext(ctx).Model(&model.ContentSort{}).Where("scode = ?", scode).Update(field, value)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// scode not found — try id-based fallback
		res2 := model.DB.WithContext(ctx).Model(&model.ContentSort{}).Where("id = ?", scode).Update(field, value)
		if res2.Error != nil {
			return res2.Error
		}
		if res2.RowsAffected == 0 {
			return errors.New("欄目不存在")
		}
	}
	return nil
}

// UpdateSortSorting updates sorting order for multiple sorts
func (s *ContentSortService) UpdateSortSorting(ctx context.Context, idSortingMap map[string]int) error {
	for idStr, sorting := range idSortingMap {
		if err := model.DB.WithContext(ctx).Model(&model.ContentSort{}).Where("id = ?", idStr).Update("sorting", sorting).Error; err != nil {
			return err
		}
	}
	return nil
}

// allowedSortSingleFields defines the whitelist for sort single-field updates
var allowedSortSingleFields = map[string]bool{
	"status":      true,
	"istop":       true,
	"isrecommend": true,
	"isheadline":  true,
	"sorting":     true,
	"name":        true,
}

// UpdateSortSingleField updates a single field with whitelist validation
func (s *ContentSortService) UpdateSortSingleField(ctx context.Context, id int, field, value string) error {
	if !allowedSortSingleFields[field] {
		return errors.New("不允許修改的欄位: " + field)
	}
	return model.DB.WithContext(ctx).Model(&model.ContentSort{}).Where("id = ?", id).Update(field, value).Error
}

// DeleteSort deletes a sort by ID
func (s *ContentSortService) DeleteSort(ctx context.Context, idStr string) error {
	return model.DB.WithContext(ctx).Delete(&model.ContentSort{}, idStr).Error
}

// DeleteSortByScode deletes a sort record by scode, with id fallback
func (s *ContentSortService) DeleteSortByScode(ctx context.Context, scode string) error {
	res := model.DB.WithContext(ctx).Where("scode = ?", scode).Delete(&model.ContentSort{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// scode not found — try id-based fallback
		res2 := model.DB.WithContext(ctx).Where("id = ?", scode).Delete(&model.ContentSort{})
		if res2.Error != nil {
			return res2.Error
		}
		if res2.RowsAffected == 0 {
			return errors.New("欄目不存在")
		}
	}
	return nil
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
