package content

import (
	"errors"
	"pbootcms-go/apps/admin/model"
	"time"
)

// ContentService handles content business logic
type ContentService struct{}

// ListContents returns paginated content list and total count
func (s *ContentService) ListContents(scode string, page, pageSize int) ([]model.Content, int64, error) {
	if page < 1 {
		page = 1
	}
	var total int64
	query := model.DB.Model(&model.Content{}).Where("status >= 0")
	if scode != "" {
		query = query.Where("scode = ? OR subscode = ?", scode, scode)
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
		return nil, errors.New("content does not exist")
	}
	return &doc, nil
}

// CreateContent creates a new content record
func (s *ContentService) CreateContent(doc *model.Content) error {
	if doc.Title == "" || doc.Scode == "" {
		return errors.New("title and sort cannot be empty")
	}
	if doc.Date.IsZero() {
		doc.Date = time.Now()
	}
	if doc.Status == 0 {
		doc.Status = 1
	}
	return model.DB.Create(doc).Error
}

// UpdateContent updates a content record with a map of fields
func (s *ContentService) UpdateContent(id int, updates map[string]interface{}) error {
	return model.DB.Model(&model.Content{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteContent deletes contents by comma-separated IDs
func (s *ContentService) DeleteContent(ids []string) error {
	for _, id := range ids {
		if err := model.DB.Delete(&model.Content{}, id).Error; err != nil {
			return err
		}
	}
	return nil
}

// CopyContent copies content to a target scode
func (s *ContentService) CopyContent(id int, targetScode string) error {
	if targetScode == "" {
		return errors.New("target sort cannot be empty")
	}
	var src model.Content
	if err := model.DB.First(&src, id).Error; err != nil {
		return errors.New("content does not exist")
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
	return model.DB.Create(&copyDoc).Error
}

// MoveContent moves content to a target scode
func (s *ContentService) MoveContent(id int, targetScode string) error {
	if targetScode == "" {
		return errors.New("target sort cannot be empty")
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
		return errors.New("field not allowed: " + field)
	}
	return model.DB.Model(&model.Content{}).Where("id = ?", id).Update(field, value).Error
}

// GetAllSorts returns all active sorts ordered by sorting
func (s *ContentService) GetAllSorts() ([]model.ContentSort, error) {
	var sorts []model.ContentSort
	err := model.DB.Where("status = 1").Order("sorting ASC").Find(&sorts).Error
	return sorts, err
}
