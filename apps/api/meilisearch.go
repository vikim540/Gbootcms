package api

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/core/acodeplugin"

	"github.com/meilisearch/meilisearch-go"
)

var (
	meiliService    meilisearch.ServiceManager
	meiliIndex      string = "gbootcms_contents"
	meiliInitMu     sync.Once
	meiliAvailable  bool
)

// MeiliSearchDoc MeiliSearch 索引文檔結構
type MeiliSearchDoc struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle"`
	Content     string `json:"content"`
	Keywords    string `json:"keywords"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	Author      string `json:"author"`
	Source      string `json:"source"`
	Scode       string `json:"scode"`
	Acode       string `json:"acode"`
	Status      int    `json:"status"`
	Ico         string `json:"ico"`
	URL         string `json:"url"`
	Date        string `json:"date"`
	Visits      int    `json:"visits"`
}

// InitMeiliSearch 初始化 MeiliSearch 客戶端
// 如果未配置 meilisearch_url，則跳過（降級到 SQL LIKE）
func InitMeiliSearch() {
	meiliInitMu.Do(func() {
		url := model.GetConfigValue("meilisearch_url", "")
		if url == "" {
			slog.Info("MeiliSearch 未配置，使用 SQL LIKE 作為搜索引擎")
			return
		}
		key := model.GetConfigValue("meilisearch_key", "")
		service := meilisearch.New(url, meilisearch.WithAPIKey(key))

		meiliService = service

		// 確保索引存在（同時驗證連接）
		_, err := service.CreateIndex(&meilisearch.IndexConfig{
			Uid:        meiliIndex,
			PrimaryKey: "id",
		})
		if err != nil && !isIndexAlreadyExistsError(err) {
			slog.Warn("MeiliSearch 連接或建立索引失敗，降級到 SQL LIKE", "err", err, "url", url)
			meiliService = nil
			return
		}

		// 設定可搜索屬性和排序屬性
		searchable := []string{"title", "subtitle", "content", "keywords", "description", "tags"}
		service.Index(meiliIndex).UpdateSearchableAttributes(&searchable)

		filterable := []interface{}{"acode", "scode", "status"}
		service.Index(meiliIndex).UpdateFilterableAttributes(&filterable)

		sortable := []string{"date", "visits"}
		service.Index(meiliIndex).UpdateSortableAttributes(&sortable)

		meiliAvailable = true
		slog.Info("MeiliSearch 初始化成功", "url", url, "index", meiliIndex)

		// 啟動定時同步協程
		go meiliSyncLoop()
	})
}

// isIndexAlreadyExistsError 判斷是否為「索引已存在」錯誤
func isIndexAlreadyExistsError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}

// IsMeiliAvailable 返回 MeiliSearch 是否可用
func IsMeiliAvailable() bool {
	return meiliAvailable
}

// SyncContentToMeili 同步單篇內容到 MeiliSearch
func SyncContentToMeili(ct *model.Content) {
	if !meiliAvailable {
		return
	}
	doc := contentToMeiliDoc(ct)
	_, err := meiliService.Index(meiliIndex).AddDocuments([]interface{}{doc}, nil)
	if err != nil {
		slog.Warn("MeiliSearch 同步文檔失敗", "id", ct.ID, "err", err)
	}
}

// DeleteFromMeili 從 MeiliSearch 刪除文檔
func DeleteFromMeili(id uint) {
	if !meiliAvailable {
		return
	}
	_, err := meiliService.Index(meiliIndex).DeleteDocument(strconv.FormatUint(uint64(id), 10), nil)
	if err != nil {
		slog.Warn("MeiliSearch 刪除文檔失敗", "id", id, "err", err)
	}
}

// SyncAllToMeili 全量同步所有已發佈內容到 MeiliSearch
func SyncAllToMeili() error {
	if !meiliAvailable {
		return fmt.Errorf("MeiliSearch 未啟用")
	}
	var contents []model.Content
	// 全量同步需跳過 acode 隔離，取得所有區域的已發佈內容
	model.DB.WithContext(acodeplugin.SkipAcode(context.Background())).Where("status >= 0").Find(&contents)

	docs := make([]interface{}, 0, len(contents))
	for i := range contents {
		docs = append(docs, contentToMeiliDoc(&contents[i]))
	}

	// 分批上傳（每批 1000 條）
	batchSize := 1000
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		_, err := meiliService.Index(meiliIndex).AddDocuments(docs[i:end], nil)
		if err != nil {
			return fmt.Errorf("同步批次 %d 失敗: %w", i/batchSize, err)
		}
	}
	slog.Info("MeiliSearch 全量同步完成", "count", len(docs))
	return nil
}

// meiliSearch 執行 MeiliSearch 搜索
func meiliSearch(keyword, acode string, page, pagesize int) (*meilisearch.SearchResponse, error) {
	if !meiliAvailable {
		return nil, fmt.Errorf("MeiliSearch 未啟用")
	}
	offset := int64((page - 1) * pagesize)
	resp, err := meiliService.Index(meiliIndex).Search(keyword, &meilisearch.SearchRequest{
		Filter: fmt.Sprintf("acode = \"%s\" AND status = 1", acode),
		Limit:  int64(pagesize),
		Offset: offset,
		Sort:   []string{"date:desc"},
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// contentToMeiliDoc 將 Content 轉換為 MeiliSearch 文檔
func contentToMeiliDoc(ct *model.Content) MeiliSearchDoc {
	dateStr := ""
	if !ct.Date.IsZero() {
		dateStr = ct.Date.Format("2006-01-02 15:04:05")
	}
	return MeiliSearchDoc{
		ID:          ct.ID,
		Title:       ct.Title,
		Subtitle:    ct.Subtitle,
		Content:     ct.Content,
		Keywords:    ct.Keywords,
		Description: ct.Description,
		Tags:        ct.Tags,
		Author:      ct.Author,
		Source:      ct.Source,
		Scode:       ct.Scode,
		Acode:       ct.Acode,
		Status:      ct.Status,
		Ico:         ct.Ico,
		URL:         buildContentURL(ct),
		Date:        dateStr,
		Visits:      ct.Visits,
	}
}

// meiliSyncLoop 定時同步協程
// 每 5 分鐘檢查是否有需要同步的內容（基於 update_time）
func meiliSyncLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if !meiliAvailable {
			return
		}
		// 同步最近 10 分鐘內更新的內容
		since := time.Now().Add(-10 * time.Minute)
		var contents []model.Content
		// 增量同步需跳過 acode 隔離，取得所有區域最近更新的內容
		model.DB.WithContext(acodeplugin.SkipAcode(context.Background())).Where("update_time > ? AND status >= 0", since).Find(&contents)
		if len(contents) == 0 {
			continue
		}
		docs := make([]interface{}, 0, len(contents))
		for i := range contents {
			docs = append(docs, contentToMeiliDoc(&contents[i]))
		}
		_, err := meiliService.Index(meiliIndex).AddDocuments(docs, nil)
		if err != nil {
			slog.Warn("MeiliSearch 定時同步失敗", "err", err, "count", len(docs))
		} else {
			slog.Info("MeiliSearch 定時同步完成", "count", len(docs))
		}
	}
}
