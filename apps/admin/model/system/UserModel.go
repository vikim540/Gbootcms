package system

import "time"

// AdminUser - Admin User Model
type AdminUser struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Ucode         string    `gorm:"column:ucode" json:"ucode"`
	Username      string    `gorm:"column:username" json:"username"`
	Password      string    `gorm:"column:password" json:"-"`
	Realname      string    `gorm:"column:realname" json:"realname"`
	Rcodes        string    `gorm:"column:rcodes" json:"rcodes"`
	Acodes        string    `gorm:"column:acodes" json:"acodes"`
	Status        int       `gorm:"column:status" json:"status"`
	LoginCount    int       `gorm:"column:login_count" json:"login_count"`
	LastLoginIP   string    `gorm:"column:last_login_ip" json:"last_login_ip"`
	LastLoginTime time.Time `gorm:"column:lastlogintime" json:"lastlogintime"`
	CreateUser    string    `gorm:"column:create_user" json:"create_user"`
	CreateTime    time.Time `gorm:"column:create_time" json:"create_time"`
	UpdateUser    string    `gorm:"column:update_user" json:"update_user"`
	UpdateTime    time.Time `gorm:"column:update_time" json:"update_time"`

	// 非資料庫欄位：預格式化時間字串（pongo2 無 date 過濾器）
	CreateTimeStr    string `gorm:"-" json:"create_time_str"`
	UpdateTimeStr    string `gorm:"-" json:"update_time_str"`
	LastLoginTimeStr string `gorm:"-" json:"last_login_time_str"`

	// 非資料庫欄位：角色名稱（列表顯示用）
	Rolename string `gorm:"-" json:"rolename"`
}

// TableName - Returns table name
func (AdminUser) TableName() string {
	return "ay_user"
}
