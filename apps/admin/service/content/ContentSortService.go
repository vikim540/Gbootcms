package content

import (
	"errors"
	"fmt"
	"pbootcms-go/apps/admin/model"
	"strings"
	"time"
)

// ContentSortService handles content sort business logic
type ContentSortService struct{}

// ListSorts returns all sorts ordered by sorting
func (s *ContentSortService) ListSorts() ([]model.ContentSort, error) {
	var sorts []model.ContentSort
	err := model.DB.Order("sorting ASC, id ASC").Find(&sorts).Error
	return sorts, err
}

// GetSort returns a single sort by ID
func (s *ContentSortService) GetSort(id int) (*model.ContentSort, error) {
	var sort model.ContentSort
	err := model.DB.First(&sort, id).Error
	if err != nil {
		return nil, errors.New("sort does not exist")
	}
	return &sort, nil
}

// GetSortByScode returns a single sort by scode
func (s *ContentSortService) GetSortByScode(scode string) (*model.ContentSort, error) {
	var sort model.ContentSort
	err := model.DB.Where("scode = ?", scode).First(&sort).Error
	if err != nil {
		return nil, errors.New("sort does not exist")
	}
	return &sort, nil
}

// BatchAddSorts creates multiple sorts from comma-separated names
func (s *ContentSortService) BatchAddSorts(multiplename, pcode string) error {
	names := splitAndTrim(multiplename)
	if len(names) == 0 {
		return nil
	}
	var lastSort model.ContentSort
	model.DB.Order("id DESC").First(&lastSort)
	lastCodeNum := 0
	fmt.Sscanf(lastSort.Scode, "%d", &lastCodeNum)

	for _, name := range names {
		if name == "" {
			continue
		}
		lastCodeNum++
		newScode := fmt.Sprintf("%d", lastCodeNum)
		model.DB.Create(&model.ContentSort{
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
func (s *ContentSortService) CreateSort(sort *model.ContentSort) error {
	if sort.Name == "" || sort.Scode == "" {
		return errors.New("name and code cannot be empty")
	}
	if sort.Status == 0 {
		sort.Status = 1
	}
	if err := model.DB.Create(sort).Error; err != nil {
		return err
	}
	// If type=1 (list) and no outlink, create initial content
	if sort.Type == 1 && sort.Outlink == "" {
		model.DB.Create(&model.Content{
			Scode:  sort.Scode,
			Title:  sort.Name,
			Status: 1,
			Date:   time.Now(),
		})
	}
	return nil
}

// UpdateSort updates a sort record
func (s *ContentSortService) UpdateSort(id int, updates map[string]interface{}) error {
	return model.DB.Model(&model.ContentSort{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateSortByScode updates a sort record by scode
func (s *ContentSortService) UpdateSortByScode(scode string, updates map[string]interface{}) error {
	return model.DB.Model(&model.ContentSort{}).Where("scode = ?", scode).Updates(updates).Error
}

// UpdateSortByScodeField updates a single field by scode with whitelist validation
func (s *ContentSortService) UpdateSortByScodeField(scode, field, value string) error {
	if !allowedSortSingleFields[field] {
		return errors.New("field not allowed: " + field)
	}
	return model.DB.Model(&model.ContentSort{}).Where("scode = ?", scode).Update(field, value).Error
}

// UpdateSortSorting updates sorting order for multiple sorts
func (s *ContentSortService) UpdateSortSorting(idSortingMap map[string]int) error {
	for idStr, sorting := range idSortingMap {
		if err := model.DB.Model(&model.ContentSort{}).Where("id = ?", idStr).Update("sorting", sorting).Error; err != nil {
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
func (s *ContentSortService) UpdateSortSingleField(id int, field, value string) error {
	if !allowedSortSingleFields[field] {
		return errors.New("field not allowed: " + field)
	}
	return model.DB.Model(&model.ContentSort{}).Where("id = ?", id).Update(field, value).Error
}

// DeleteSort deletes a sort by ID
func (s *ContentSortService) DeleteSort(idStr string) error {
	return model.DB.Delete(&model.ContentSort{}, idStr).Error
}

// DeleteSortByScode deletes a sort by scode
func (s *ContentSortService) DeleteSortByScode(scode string) error {
	return model.DB.Where("scode = ?", scode).Delete(&model.ContentSort{}).Error
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
