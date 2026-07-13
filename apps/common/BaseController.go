package common

import (
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/core/db"
	"gbootcms/core/mediaplugin"

	"github.com/gin-gonic/gin"
)

type BaseController struct{}

// BatchSort is a generic batch-sorting handler for all admin list pages.
//
// It reads two parallel form arrays submitted by the standard PbootCMS template pattern:
//   <input type="hidden" name="listall[]" value="{id}">
//   <input type="text"   name="sorting[]" value="{sort}">
//   <button type="submit" name="submit" value="sorting">保存排序</button>
//
// Usage in any controller:
//
//	if bc.IsBatchSort(c) {
//	    bc.BatchSort(c, &model.Slide{}, "sorting", 255)
//	    return
//	}
//
// The table name and primary-key column are auto-resolved from the model via GORM schema,
// so callers only need to pass the sort column name and a default value.
// LogAction 記錄操作日誌到 ay_syslog（對齊 PbootCMS Controller::log()）
// 供所有 Controller 繼承使用，無需各自實現
func (bc *BaseController) LogAction(c *gin.Context, msg string) {
	username, _ := GetSession(c, "admin_username").(string)
	if username == "" {
		username = c.PostForm("username")
	}
	if username == "" {
		username = "unknown"
	}
	ua := c.Request.UserAgent()
	chPlatformVer := c.GetHeader("Sec-CH-UA-Platform-Version")
	osName, browser := ParseUserAgent(ua, chPlatformVer)
	now := time.Now()
	entry := model.Syslog{
		Level:      "admin",
		Event:      msg,
		UserIP:     c.ClientIP(),
		UserOS:     osName,
		UserBs:     browser,
		CreateUser: username,
		CreateTime: now.Format("2006-01-02 15:04:05"),
		Username:   username,
		URL:        c.Request.URL.Path,
		Content:    msg,
		IP:         c.ClientIP(),
		LogTime:    now,
	}
	if err := model.DB.Create(&entry).Error; err != nil {
		slog.Error("[Syslog] 寫入失敗", "err", err, "msg", msg)
	}
}

// Paginate 組件化分頁（對齊 PbootCMS Controller::page() + Paging::limit()）
// 從 query 讀取 page（默認1）和 pagesize（默認15，可透過下拉選擇器覆蓋）
//
// 用法：
//
//	page, pageSize, offset := bc.Paginate(c)
//	db.Offset(offset).Limit(pageSize).Find(&items)
//	data["pagebar"] = helper.BuildPagebarHTML(total, page, pageSize, baseURL)
//	data["pagesize"] = pageSize
func (bc *BaseController) Paginate(c *gin.Context) (page, pageSize, offset int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	// 默認分頁大小從資料庫配置讀取（對齊 PbootCMS pagenum 配置），默認 15
	pageSize = 15
	if ps := c.Query("pagesize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	} else {
		if cfgVal := model.GetConfigValue("pagesize", ""); cfgVal != "" {
			if v, err := strconv.Atoi(cfgVal); err == nil && v > 0 {
				pageSize = v
			}
		}
	}
	offset = (page - 1) * pageSize
	return
}

func (bc *BaseController) IsBatchSort(c *gin.Context) bool {
	return c.Request.Method == "POST" && c.PostForm("submit") == "sorting"
}

func (bc *BaseController) BatchSort(c *gin.Context, modelPtr interface{}, sortColumn string, defaultSort int) {
	// 兼容 listall[] 和 listall[0] 兩種鍵名格式
	idList := extractIndexedArray(c, "listall")
	sortList := extractIndexedArray(c, "sorting")

	if len(idList) == 0 {
		bc.JSONFail(c, "沒有排序資料")
		return
	}

	updated := 0
	unchanged := 0
	for i, idStr := range idList {
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			continue
		}
		sortVal := defaultSort
		if i < len(sortList) && sortList[i] != "" {
			if v, err := strconv.Atoi(sortList[i]); err == nil {
				sortVal = v
			}
		}
		// 读取当前值，只在變化時才寫庫（避免無謂的 Update 请求）
		var current int
		db.DB.Model(modelPtr).Select(sortColumn).Where("id = ?", id).Scan(&current)
		if current == sortVal {
			unchanged++
			continue
		}
		db.DB.Model(modelPtr).Where("id = ?", id).UpdateColumn(sortColumn, sortVal)
		updated++
	}

	if updated == 0 {
		bc.JSONOKMsg(c, NoticeSortNoChange(unchanged))
		return
	}
	if unchanged > 0 {
		bc.JSONOKMsg(c, NoticeSortSavedPartial(updated, unchanged))
		return
	}
	bc.JSONOKMsg(c, NoticeSortSaved(updated))
}

// extractIndexedArray 從 POST 表單提取數組字段，兼容 name[] 和 name[0] 兩種格式
func extractIndexedArray(c *gin.Context, field string) []string {
	// 先嘗試標準格式 name[]
	vals := c.PostFormArray(field + "[]")
	if len(vals) > 0 {
		return vals
	}
	// 回退：收集 name[0], name[1], ... 並按索引排序
	idxMap := map[int]string{}
	for key, vs := range c.Request.PostForm {
		prefix := field + "["
		if strings.HasPrefix(key, prefix) && len(vs) > 0 {
			idxStr := key[len(prefix) : len(key)-1]
			if idx, err := strconv.Atoi(idxStr); err == nil {
				idxMap[idx] = vs[0]
			}
		}
	}
	if len(idxMap) == 0 {
		return nil
	}
	indices := make([]int, 0, len(idxMap))
	for k := range idxMap {
		indices = append(indices, k)
	}
	sort.Ints(indices)
	result := make([]string, 0, len(indices))
	for _, idx := range indices {
		result = append(result, idxMap[idx])
	}
	return result
}

func (bc *BaseController) JSONOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": data, "tourl": ""})
}

func (bc *BaseController) JSONOKMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": msg, "msg": msg, "tourl": ""})
}

// JSONOKMsgTourl 成功響應帶消息和跳轉 URL
// 用於新增成功後跳轉到列表頁，防止頁面停留在新增表單導致重複新增
func (bc *BaseController) JSONOKMsgTourl(c *gin.Context, msg string, tourl string) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": msg, "msg": msg, "tourl": tourl})
}

func (bc *BaseController) JSONFail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": msg, "msg": msg, "tourl": ""})
}

func (bc *BaseController) GetAdminUsername(c *gin.Context) string {
	username, _ := GetSession(c, "admin_username").(string)
	return username
}

func (bc *BaseController) GetAdminUID(c *gin.Context) int {
	return GetSessionInt(c, "admin_uid")
}

func (bc *BaseController) GetAdminUcode(c *gin.Context) string {
	ucode, _ := GetSession(c, "admin_ucode").(string)
	return ucode
}

func (bc *BaseController) IsLogin(c *gin.Context) bool {
	return GetSessionInt(c, "admin_uid") > 0
}

// MarkMediaCacheDirty 標記媒體庫緩存為臟。
//
// 建議在所有「會改變數據庫中文件引用」的寫操作後調用。
// 但實際上，通過 GORM Plugin（core/mediaplugin.MediaDirtyPlugin）
// 已經會自動處理大部分場景。此函數保留作為手動觸發的後備接口。
//
// 為什麼需要這樣做？
//
// 媒體庫有一個掃描緩存，避免每次訪問都全表掃描。但緩存會導致一個問題：
// 當你修改幻燈片換上一張新圖，緩存中這張新圖還是「未用」狀態。
// 如果你再換回原圖，緩存中這張原圖的「已用」狀態已經丟失。
//
// 通過「臟標記」機制：任何寫操作後立即標記緩存失效，下次訪問自動重掃，
// 既保證準確性，又不會對每個讀請求都重新掃描。
//
// 內部轉發到 core/mediaplugin 包以避免循環引用。
func MarkMediaCacheDirty() {
	mediaplugin.MarkDirty()
}

// IsMediaCacheDirty 供 MediaController 內部讀取
func IsMediaCacheDirty() bool {
	return mediaplugin.IsDirty()
}

// ClearMediaCacheDirty 在掃描成功後清除標記
func ClearMediaCacheDirty() {
	mediaplugin.ClearDirty()
}
