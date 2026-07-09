package common

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"gbootcms/config"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// sessionKey 從配置讀取（可透過環境變數 PBOOTCMS_GO_APP_SESSION_KEY 覆蓋）
var sessionKey []byte

func init() {
	cfg := config.Get()
	if cfg.App.SessionKey != "" {
		sessionKey = []byte(cfg.App.SessionKey)
	} else {
		// 降級：配置未設定時使用默認值
		sessionKey = []byte("gbootcms-session-key-32byte!!!")
	}
}

type SessionData map[string]interface{}

var (
	sessionStore = make(map[string]SessionData)
	mu           sync.RWMutex
)

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
		c.SetCookie("PbootGo", sid, 86400, "/", "", false, false)
		c.Set("sessionID", sid) // 存入 context，後續同請求內復用
	}

	mu.Lock()
	defer mu.Unlock()

	if sessionStore[sid] == nil {
		sessionStore[sid] = make(SessionData)
	}
	sessionStore[sid][key] = value
}

func GetSession(c *gin.Context, key string) interface{} {
	sid := getSessionID(c)
	if sid == "" {
		return nil
	}

	mu.RLock()
	defer mu.RUnlock()

	if data, ok := sessionStore[sid]; ok {
		return data[key]
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
	defer mu.Unlock()

	if data, ok := sessionStore[sid]; ok {
		delete(data, key)
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
	c.SetCookie("PbootGo", "", -1, "/", "", false, false)
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
	mu.Lock()
	defer mu.Unlock()
	sessionStore[sid] = data
}

func GetSessionData(c *gin.Context, sid string) map[string]interface{} {
	mu.RLock()
	defer mu.RUnlock()
	if data, ok := sessionStore[sid]; ok {
		return data
	}
	return nil
}

func DeleteSessionData(sid string) {
	mu.Lock()
	defer mu.Unlock()
	delete(sessionStore, sid)
}

func DeleteSessionKey(c *gin.Context, key string) {
	sid := getSessionID(c)
	if sid == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if data, ok := sessionStore[sid]; ok {
		delete(data, key)
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
