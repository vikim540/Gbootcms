package common

import (
	"net/http"
	"strconv"

	"pbootcms-go/core/db"
	"pbootcms-go/core/mediaplugin"

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
func (bc *BaseController) IsBatchSort(c *gin.Context) bool {
	return c.Request.Method == "POST" && c.PostForm("submit") == "sorting"
}

func (bc *BaseController) BatchSort(c *gin.Context, modelPtr interface{}, sortColumn string, defaultSort int) {
	idList := c.PostFormArray("listall[]")
	sortList := c.PostFormArray("sorting[]")

	if len(idList) == 0 {
		bc.JSONFail(c, "没有排序数据")
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
		bc.JSONOKMsg(c, "排序未變化 ("+strconv.Itoa(unchanged)+" 條)")
		return
	}
	if unchanged > 0 {
		bc.JSONOKMsg(c, "排序已保存 ("+strconv.Itoa(updated)+" 條，"+strconv.Itoa(unchanged)+" 條無變化)")
		return
	}
	bc.JSONOKMsg(c, "排序已保存 ("+strconv.Itoa(updated)+" 條)")
}

func (bc *BaseController) JSONOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": data})
}

func (bc *BaseController) JSONOKMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "msg": msg})
}

func (bc *BaseController) JSONFail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": msg})
}

func (bc *BaseController) JSONFailMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": msg})
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
