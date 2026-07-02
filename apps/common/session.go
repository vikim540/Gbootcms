package common

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

var sessionKey = []byte("pbootcms-go-session-key-32byte!")

type SessionData map[string]interface{}

var (
	sessionStore = make(map[string]SessionData)
	mu           sync.RWMutex
)

func getSessionID(c *gin.Context) string {
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
