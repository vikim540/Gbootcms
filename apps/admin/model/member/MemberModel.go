package member

import "time"

// Member 對應 PbootCMS ay_member 表（24 個字段，完整對齊原版數據庫）
type Member struct {
	ID            uint      `gorm:"primaryKey;column:id" json:"id"`
	Ucode         string    `gorm:"column:ucode" json:"ucode"`
	Username      string    `gorm:"column:username" json:"username"`
	Useremail     string    `gorm:"column:useremail" json:"useremail"`
	Usermobile    string    `gorm:"column:usermobile" json:"usermobile"`
	Nickname      string    `gorm:"column:nickname" json:"nickname"`
	Password      string    `gorm:"column:password" json:"-"`
	Headpic       string    `gorm:"column:headpic" json:"headpic"`
	Status        int       `gorm:"column:status" json:"status"`
	Activation    int       `gorm:"column:activation" json:"activation"`
	GID           string    `gorm:"column:gid" json:"gid"`
	Wxid          string    `gorm:"column:wxid" json:"wxid"`
	Qqid          string    `gorm:"column:qqid" json:"qqid"`
	Wbid          string    `gorm:"column:wbid" json:"wbid"`
	Score         int       `gorm:"column:score" json:"score"`
	RegisterTime  time.Time `gorm:"column:register_time" json:"register_time"`
	LoginCount    int       `gorm:"column:login_count" json:"login_count"`
	LastLoginIP   string    `gorm:"column:last_login_ip" json:"last_login_ip"`
	LastLoginTime string    `gorm:"column:last_login_time" json:"last_login_time"`
	Sex           string    `gorm:"column:sex" json:"sex"`
	Birthday      string    `gorm:"column:birthday" json:"birthday"`
	Telephone     string    `gorm:"column:telephone" json:"telephone"`
	Email         string    `gorm:"column:email" json:"email"`
	QQ            string    `gorm:"column:qq" json:"qq"`
	// 非資料庫欄位：用於列表顯示等級名稱
	Gname          string `gorm:"-" json:"gname"`
	RegisterTimeStr string `gorm:"-" json:"register_time_str"`
}

func (Member) TableName() string { return "ay_member" }
