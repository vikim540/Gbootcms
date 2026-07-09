package middleware

import (
	"fmt"
	"gbootcms/apps/admin/model"
	"gbootcms/core/db"
	"time"
)

// logSpiderVisitToFile 將蜘蛛訪問記錄寫入 ay_syslog 表（level='spider' 獨立區分）
// 對齊 PHP SpiderController::index → log()
func logSpiderVisitToFile(spider, reqURL, ip string) {
	now := time.Now()
	syslog := model.Syslog{
		// 原始 PbootCMS NOT NULL 欄位
		Level:      "spider",
		Event:      fmt.Sprintf("%s 爬行 %s", spider, reqURL),
		UserIP:     ip,
		UserOS:     "Spider",
		UserBs:     spider,
		CreateUser: spider,
		CreateTime: now.Format("2006-01-02 15:04:05"),
		// GORM 擴展欄位（模板顯示用）
		Username: spider,
		URL:      reqURL,
		Content:  fmt.Sprintf("%s 爬行 %s", spider, reqURL),
		IP:       ip,
		LogTime:  now,
	}
	if db.DB != nil {
		if err := db.DB.Create(&syslog).Error; err != nil {
			fmt.Printf("[SpiderLog] 寫入 ay_syslog 失敗: %v\n", err)
		}
	}
}
