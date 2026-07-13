package common

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"gbootcms/config"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// sessionKey 從配置讀取（可透過環境變數 PBOOTCMS_GO_APP_SESSION_KEY 覆蓋）
var sessionKey []byte

// Session TTL 常量（對齊 PbootCMS PHP session_ticket 機制）
// PbootCMS: 初始 1 小時，後台操作延長至 3 小時，每 30 分鐘清理
const (
	sessionMaxLifetime = 24 * time.Hour // 會話最大生命週期（對齊 cookie MaxAge=86400）
	sessionIdleTimeout = 2 * time.Hour  // 閒置超時：無活動超過此時間則過期
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

// isSessionExpired 檢查 session 是否已過期
func isSessionExpired(entry *sessionEntry) bool {
	if entry == nil {
		return true
	}
	now := time.Now()
	// 超過最大生命週期
	if now.Sub(entry.createdAt) > sessionMaxLifetime {
		return true
	}
	// 閒置超時
	if now.Sub(entry.lastActivity) > sessionIdleTimeout {
		return true
	}
	return false
}

// cleanupExpiredSessions 定期清理過期的 session（後台協程）
func cleanupExpiredSessions() {
	ticker := time.NewTicker(sessionCleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		mu.Lock()
		now := time.Now()
		for sid, entry := range sessionStore {
			if now.Sub(entry.createdAt) > sessionMaxLifetime ||
				now.Sub(entry.lastActivity) > sessionIdleTimeout {
				delete(sessionStore, sid)
			}
		}
		mu.Unlock()
	}
}

func getSessionID(c *gin.Context) string {
	// 優先從 gin context 取（同一請求內 SetSession 創建後的 session ID）
	if sid, ok := c.Get("sessionID"); ok {
		if s, ok := sid.(string); ok && s != "" {
			return s
		}
	}
	// 從 cookie 取
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
		c.SetCookie("PbootGo", sid, 86400, "/", "", false, true) // HttpOnly=true 防止 XSS 竊取
		c.Set("sessionID", sid) // 存入 context，後續同請求內復用
	}

	now := time.Now()

	mu.Lock()
	defer mu.Unlock()

	entry, ok := sessionStore[sid]
	if !ok || isSessionExpired(entry) {
		// 新 session 或已過期，重新建立
		entry = &sessionEntry{
			data:         make(SessionData),
			createdAt:    now,
			lastActivity: now,
		}
		sessionStore[sid] = entry
	}
	entry.lastActivity = now // 更新活動時間
	entry.data[key] = value
}

func GetSession(c *gin.Context, key string) interface{} {
	sid := getSessionID(c)
	if sid == "" {
		return nil
	}

	mu.RLock()
	defer mu.RUnlock()

	entry, ok := sessionStore[sid]
	if !ok || isSessionExpired(entry) {
		return nil
	}
	return entry.data[key]
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
	defer mu.Unlock()

	if entry, ok := sessionStore[sid]; ok && !isSessionExpired(entry) {
		delete(entry.data, key)
	}
}

func ClearSession(c *gin.Context) {
	sid := getSessionID(c)
	if sid == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	delete(sessionStore, sid)
	c.SetCookie("PbootGo", "", -1, "/", "", false, true)
}

// ClearAllSessions 清除所有會話（排除當前用戶，避免管理員被踢出）
// 用於後台「清理會話」功能，讓所有其他用戶強制重新登入
func ClearAllSessions(c *gin.Context) int {
	currentSID := getSessionID(c)

	mu.Lock()
	defer mu.Unlock()

	count := 0
	for sid := range sessionStore {
		if sid != currentSID {
			delete(sessionStore, sid)
			count++
		}
	}
	return count
}

func SetSessionData(c *gin.Context, sid string, data map[string]interface{}) {
	now := time.Now()
	mu.Lock()
	defer mu.Unlock()
	sessionStore[sid] = &sessionEntry{
		data:         SessionData(data),
		createdAt:    now,
		lastActivity: now,
	}
}

func GetSessionData(c *gin.Context, sid string) map[string]interface{} {
	mu.RLock()
	defer mu.RUnlock()
	if entry, ok := sessionStore[sid]; ok && !isSessionExpired(entry) {
		return entry.data
	}
	return nil
}

func DeleteSessionData(sid string) {
	mu.Lock()
	defer mu.Unlock()
	delete(sessionStore, sid)
}

// RegenerateSessionID 重新生成 session ID（防止 Session Fixation 攻擊）
// 將舊 session 的數據遷移到新 session，刪除舊 session，設置新 cookie
// 用於會員登入時（對齊後台管理員登入的 generateSessionID 邏輯）
func RegenerateSessionID(c *gin.Context) {
	oldSID := getSessionID(c)

	newSID := createSessionID()
	now := time.Now()

	mu.Lock()
	// 遷移舊 session 數據到新 session
	if oldSID != "" {
		if oldEntry, ok := sessionStore[oldSID]; ok && !isSessionExpired(oldEntry) {
			sessionStore[newSID] = &sessionEntry{
				data:         oldEntry.data,
				createdAt:    now,          // 重置創建時間（新 session）
				lastActivity: now,
			}
			delete(sessionStore, oldSID)
		} else {
			sessionStore[newSID] = &sessionEntry{
				data:         make(SessionData),
				createdAt:    now,
				lastActivity: now,
			}
		}
	} else {
		sessionStore[newSID] = &sessionEntry{
			data:         make(SessionData),
			createdAt:    now,
			lastActivity: now,
		}
	}
	mu.Unlock()

	c.SetCookie("PbootGo", newSID, 86400, "/", "", false, true)
	c.Set("sessionID", newSID)
}

func DeleteSessionKey(c *gin.Context, key string) {
	sid := getSessionID(c)
	if sid == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if entry, ok := sessionStore[sid]; ok && !isSessionExpired(entry) {
		delete(entry.data, key)
	}
}

func encodeCookie(data SessionData) string {
	b, _ := json.Marshal(data)
	return base64.URLEncoding.EncodeToString(b)
}

func decodeCookie(encoded string) SessionData {
	b, _ := base64.URLEncoding.DecodeString(encoded)
	var data SessionData
	json.Unmarshal(b, &data)
	return data
}
