package common

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"gbootcms/apps/admin/model"
	"gbootcms/config"
	"gbootcms/core/db"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// sessionKey 從配置讀取（可透過環境變數 PBOOTCMS_GO_APP_SESSION_KEY 覆蓋）
var sessionKey []byte

// Session TTL 常量（對齊 PbootCMS PHP session_ticket 機制）
const (
	sessionMaxLifetime     = 24 * time.Hour  // 會話最大生命週期（對齊 cookie MaxAge=86400）
	sessionIdleTimeout     = 2 * time.Hour   // 閒置超時：無活動超過此時間則過期
	sessionCleanupInterval = 10 * time.Minute // 過期清理週期
)

func init() {
	cfg := config.Get()
	if cfg.App.SessionKey != "" {
		sessionKey = []byte(cfg.App.SessionKey)
	} else {
		// 降級：配置未設定時使用默認值
		sessionKey = []byte("gbootcms-session-key-32byte!!!")
	}

	// 啟動過期 session 清理協程（對齊 PbootCMS PHP 的 session 定期清理）
	go cleanupExpiredSessions()
}

type SessionData map[string]interface{}

// sessionEntry 封裝 session 數據和元資訊（TTL 管理）
type sessionEntry struct {
	data         SessionData
	createdAt    time.Time
	lastActivity time.Time
}

var (
	sessionStore = make(map[string]*sessionEntry)
	mu           sync.RWMutex
)

// === SQLite 持久化層 ===
// 雙寫策略：記憶體（速度）+ SQLite（持久化）
// 讀取：記憶體優先，miss 時回 DB
// 寫入：同步寫記憶體 + 異步寫 DB（不阻塞請求）
// 對標 Swoole 6 的常駐記憶體 + DB 備份模式

// persistSessionToDB 將 session 持久化到 DB（upsert）
func persistSessionToDB(sid string, entry *sessionEntry) {
	if db.DB == nil || sid == "" || entry == nil {
		return
	}
	dataBytes, err := json.Marshal(entry.data)
	if err != nil {
		slog.Warn("Session DB 持久化序列化失敗", "sid", sid, "error", err)
		return
	}
	sess := model.Session{
		SID:          sid,
		Data:         string(dataBytes),
		CreatedAt:    entry.createdAt,
		LastActivity: entry.lastActivity,
	}
	if err := db.DB.Save(&sess).Error; err != nil {
		slog.Warn("Session DB 持久化失敗", "sid", sid, "error", err)
	}
}

// loadSessionFromDB 從 DB 載入 session（記憶體 miss 時使用）
func loadSessionFromDB(sid string) *sessionEntry {
	if db.DB == nil || sid == "" {
		return nil
	}
	var sess model.Session
	if err := db.DB.Where("sid = ? AND last_activity > ?", sid, time.Now().Add(-sessionIdleTimeout)).First(&sess).Error; err != nil {
		return nil
	}
	var data SessionData
	if err := json.Unmarshal([]byte(sess.Data), &data); err != nil {
		slog.Warn("Session DB 反序列化失敗", "sid", sid, "error", err)
		return nil
	}
	return &sessionEntry{
		data:         data,
		createdAt:    sess.CreatedAt,
		lastActivity: sess.LastActivity,
	}
}

// deleteSessionFromDB 從 DB 刪除 session
func deleteSessionFromDB(sid string) {
	if db.DB == nil || sid == "" {
		return
	}
	if err := db.DB.Where("sid = ?", sid).Delete(&model.Session{}).Error; err != nil {
		slog.Warn("Session DB 刪除失敗", "sid", sid, "error", err)
	}
}

// LoadSessionsFromDB 啟動時從 DB 載入活躍 session 到記憶體
// 應在 main() 中 DB 初始化後呼叫
func LoadSessionsFromDB() {
	if db.DB == nil {
		return
	}
	var sessions []model.Session
	cutoff := time.Now().Add(-sessionIdleTimeout)
	if err := db.DB.Where("last_activity > ?", cutoff).Find(&sessions).Error; err != nil {
		slog.Warn("啟動時載入 session 失敗", "error", err)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	loaded := 0
	for _, sess := range sessions {
		var data SessionData
		if err := json.Unmarshal([]byte(sess.Data), &data); err != nil {
			continue
		}
		// 檢查最大生命週期
		if time.Since(sess.CreatedAt) > sessionMaxLifetime {
			continue
		}
		sessionStore[sess.SID] = &sessionEntry{
			data:         data,
			createdAt:    sess.CreatedAt,
			lastActivity: sess.LastActivity,
		}
		loaded++
	}

	// 清理 DB 中的過期 session
	db.DB.Where("last_activity < ? OR created_at < ?", cutoff, time.Now().Add(-sessionMaxLifetime)).Delete(&model.Session{})

	slog.Info("Session 從 DB 載入完成", "loaded", loaded, "total_in_memory", len(sessionStore))
}

// isSessionExpired 檢查 session 是否已過期
func isSessionExpired(entry *sessionEntry) bool {
	if entry == nil {
		return true
	}
	now := time.Now()
	if now.Sub(entry.createdAt) > sessionMaxLifetime {
		return true
	}
	if now.Sub(entry.lastActivity) > sessionIdleTimeout {
		return true
	}
	return false
}

// cleanupExpiredSessions 定期清理過期的 session（記憶體 + DB）
func cleanupExpiredSessions() {
	ticker := time.NewTicker(sessionCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		// 清理記憶體
		mu.Lock()
		for sid, entry := range sessionStore {
			if now.Sub(entry.createdAt) > sessionMaxLifetime ||
				now.Sub(entry.lastActivity) > sessionIdleTimeout {
				delete(sessionStore, sid)
			}
		}
		mu.Unlock()

		// 清理 DB
		if db.DB != nil {
			cutoff := now.Add(-sessionIdleTimeout)
			maxCutoff := now.Add(-sessionMaxLifetime)
			db.DB.Where("last_activity < ? OR created_at < ?", cutoff, maxCutoff).Delete(&model.Session{})
		}
	}
}

func getSessionID(c *gin.Context) string {
	if sid, ok := c.Get("sessionID"); ok {
		if s, ok := sid.(string); ok && s != "" {
			return s
		}
	}
	cookie, err := c.Cookie("PbootGo")
	if err != nil || cookie == "" {
		return ""
	}
	return cookie
}

func createSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func SetSession(c *gin.Context, key string, value interface{}) {
	sid := getSessionID(c)
	if sid == "" {
		sid = createSessionID()
		SetSecureCookie(c, "PbootGo", sid, 86400, "/")
		c.Set("sessionID", sid)
	}

	now := time.Now()

	mu.Lock()
	entry, ok := sessionStore[sid]
	if !ok || isSessionExpired(entry) {
		entry = &sessionEntry{
			data:         make(SessionData),
			createdAt:    now,
			lastActivity: now,
		}
		sessionStore[sid] = entry
	}
	entry.lastActivity = now
	entry.data[key] = value
	mu.Unlock()

	// 持久化到 DB（不阻塞請求，錯誤僅記錄）
	persistSessionToDB(sid, entry)
}

func GetSession(c *gin.Context, key string) interface{} {
	sid := getSessionID(c)
	if sid == "" {
		return nil
	}

	mu.RLock()
	entry, ok := sessionStore[sid]
	mu.RUnlock()

	if ok && !isSessionExpired(entry) {
		return entry.data[key]
	}

	// 記憶體 miss：嘗試從 DB 載入
	if !ok {
		dbEntry := loadSessionFromDB(sid)
		if dbEntry != nil && !isSessionExpired(dbEntry) {
			mu.Lock()
			sessionStore[sid] = dbEntry
			mu.Unlock()
			return dbEntry.data[key]
		}
	}
	return nil
}

func GetSessionString(c *gin.Context, key string) string {
	if val := GetSession(c, key); val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func GetSessionInt(c *gin.Context, key string) int {
	if val := GetSession(c, key); val != nil {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case uint:
			return int(v)
		case uint64:
			return int(v)
		case float64:
			return int(v)
		case string:
			n, err := strconv.Atoi(v)
			if err == nil {
				return n
			}
		}
	}
	return 0
}

func DeleteSession(c *gin.Context, key string) {
	sid := getSessionID(c)
	if sid == "" {
		return
	}

	mu.Lock()
	if entry, ok := sessionStore[sid]; ok && !isSessionExpired(entry) {
		delete(entry.data, key)
		// 持久化變更到 DB
		mu.Unlock()
		persistSessionToDB(sid, entry)
		return
	}
	mu.Unlock()
}

func ClearSession(c *gin.Context) {
	sid := getSessionID(c)
	if sid == "" {
		return
	}

	mu.Lock()
	delete(sessionStore, sid)
	mu.Unlock()

	// 從 DB 刪除
	deleteSessionFromDB(sid)
	SetSecureCookie(c, "PbootGo", "", -1, "/")
}

// ClearAllSessions 清除所有會話（排除當前用戶，避免管理員被踢出）
func ClearAllSessions(c *gin.Context) int {
	currentSID := getSessionID(c)

	mu.Lock()
	count := 0
	for sid := range sessionStore {
		if sid != currentSID {
			delete(sessionStore, sid)
			count++
		}
	}
	mu.Unlock()

	// 從 DB 刪除所有非當前 session
	if db.DB != nil {
		db.DB.Where("sid != ?", currentSID).Delete(&model.Session{})
	}
	return count
}

func SetSessionData(c *gin.Context, sid string, data map[string]interface{}) {
	now := time.Now()
	mu.Lock()
	entry := &sessionEntry{
		data:         SessionData(data),
		createdAt:    now,
		lastActivity: now,
	}
	sessionStore[sid] = entry
	mu.Unlock()

	persistSessionToDB(sid, entry)
}

func GetSessionData(c *gin.Context, sid string) map[string]interface{} {
	mu.RLock()
	if entry, ok := sessionStore[sid]; ok && !isSessionExpired(entry) {
		mu.RUnlock()
		return entry.data
	}
	mu.RUnlock()

	// 記憶體 miss：嘗試 DB
	dbEntry := loadSessionFromDB(sid)
	if dbEntry != nil && !isSessionExpired(dbEntry) {
		mu.Lock()
		sessionStore[sid] = dbEntry
		mu.Unlock()
		return dbEntry.data
	}
	return nil
}

func DeleteSessionData(sid string) {
	mu.Lock()
	delete(sessionStore, sid)
	mu.Unlock()

	deleteSessionFromDB(sid)
}

// RegenerateSessionID 重新生成 session ID（防止 Session Fixation 攻擊）
func RegenerateSessionID(c *gin.Context) {
	oldSID := getSessionID(c)
	newSID := createSessionID()
	now := time.Now()

	var newEntry *sessionEntry

	mu.Lock()
	if oldSID != "" {
		if oldEntry, ok := sessionStore[oldSID]; ok && !isSessionExpired(oldEntry) {
			newEntry = &sessionEntry{
				data:         oldEntry.data,
				createdAt:    now,
				lastActivity: now,
			}
			delete(sessionStore, oldSID)
		}
	}
	if newEntry == nil {
		newEntry = &sessionEntry{
			data:         make(SessionData),
			createdAt:    now,
			lastActivity: now,
		}
	}
	sessionStore[newSID] = newEntry
	mu.Unlock()

	// DB：刪舊建新
	if oldSID != "" {
		deleteSessionFromDB(oldSID)
	}
	persistSessionToDB(newSID, newEntry)

	SetSecureCookie(c, "PbootGo", newSID, 86400, "/")
	c.Set("sessionID", newSID)
}

func DeleteSessionKey(c *gin.Context, key string) {
	sid := getSessionID(c)
	if sid == "" {
		return
	}

	mu.Lock()
	if entry, ok := sessionStore[sid]; ok && !isSessionExpired(entry) {
		delete(entry.data, key)
		mu.Unlock()
		persistSessionToDB(sid, entry)
		return
	}
	mu.Unlock()
}
