package storage

import (
	"log/slog"
	"sync"
	"time"

	"gbootcms/apps/admin/model"
)

// Storage 存儲抽象接口
type Storage interface {
	// Upload 上傳本地檔案到雲端，返回公開 URL
	Upload(localPath, objectKey string) (string, error)
	// Delete 刪除雲端檔案
	Delete(objectKey string) error
	// GetURL 取得檔案的公開 URL
	GetURL(objectKey string) string
	// Exists 檢查檔案是否存在於雲端
	Exists(objectKey string) bool
	// IsEnabled 是否啟用雲存儲
	IsEnabled() bool
}

// cacheEntry 快取項
type cacheEntry struct {
	url       string
	exists    bool
	checkedAt time.Time
}

// cacheManager 檔案快取管理器
type cacheManager struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

var (
	defaultStorage Storage
	cache          *cacheManager
	initOnce       sync.Once
)

func initCache() {
	ttlSec := 300 // 預設 5 分鐘
	if v := model.GetConfigValue("r2_cache_ttl", "300"); v != "" {
		if n := parseIntSafe(v); n > 0 {
			ttlSec = n
		}
	}
	cache = &cacheManager{
		entries: make(map[string]*cacheEntry),
		ttl:     time.Duration(ttlSec) * time.Second,
	}
}

func parseIntSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// InitStorage 初始化存儲（根據配置選擇本地或 R2）
func InitStorage() {
	initOnce.Do(func() {
		initCache()

		if model.GetConfigValue("r2_enabled", "0") == "1" {
			r2, err := NewR2Storage()
			if err != nil {
				slog.Error("R2 存儲初始化失敗，降級為本地存儲", "error", err)
				defaultStorage = &LocalStorage{}
				return
			}
			defaultStorage = r2
			slog.Info("Cloudflare R2 雲存儲已啟用", "bucket", model.GetConfigValue("r2_bucket", ""))
		} else {
			defaultStorage = &LocalStorage{}
		}
	})
}

// GetStorage 取得當前存儲實例
func GetStorage() Storage {
	if defaultStorage == nil {
		InitStorage()
	}
	return defaultStorage
}

// RefreshStorage 重新初始化存儲（管理員修改配置後呼叫）
func RefreshStorage() {
	// 重置 initOnce 以允許重新初始化
	initOnce = sync.Once{}
	cache = nil
	InitStorage()
}

// --- 快取方法 ---

// getCached 從快取中取得項目（過期返回 nil）
func (cm *cacheManager) getCached(key string) *cacheEntry {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	entry := cm.entries[key]
	if entry == nil {
		return nil
	}
	if time.Since(entry.checkedAt) > cm.ttl {
		return nil
	}
	return entry
}

// setCached 設定快取項目
func (cm *cacheManager) setCached(key string, url string, exists bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.entries[key] = &cacheEntry{
		url:       url,
		exists:    exists,
		checkedAt: time.Now(),
	}
}

// invalidate 使快取項目失效
func (cm *cacheManager) invalidate(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.entries, key)
}

// clear 清除所有快取
func (cm *cacheManager) clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.entries = make(map[string]*cacheEntry)
}

// ClearCache 清除存儲快取（外部呼叫）
func ClearCache() {
	if cache != nil {
		cache.clear()
	}
	slog.Info("存儲快取已清除")
}
