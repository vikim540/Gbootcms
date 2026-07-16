package content

// SlideGroup - 輪播圖分組 Model
// DB table: ay_slide_group
// 用途：為 ay_slide.gid 數字提供人類可讀的名稱映射（如 gid=1 → "簡體首頁"）
// 設計原則：不改動原版 ay_slide 表結構，分組名稱獨立儲存
type SlideGroup struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	GID        int    `gorm:"column:gid" json:"gid"`           // 對應 ay_slide.gid
	Name       string `gorm:"column:name" json:"name"`         // 分組名稱（如 "簡體首頁"）
	Sorting    int    `gorm:"column:sorting" json:"sorting"`   // 排序
	Acode      string `gorm:"column:acode" json:"acode"`       // 區域代碼（多語言隔離）
	CreateTime string `gorm:"column:create_time" json:"create_time"`
	UpdateTime string `gorm:"column:update_time" json:"update_time"`

	// 非資料庫欄位
	Count int64 `gorm:"-" json:"count"` // 該分組下的輪播圖數量（控制器填充）
}
