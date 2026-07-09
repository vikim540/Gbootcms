package config

import (
	"sync"

	"github.com/spf13/viper"
)

type Database struct {
	Type   string `json:"type"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	User   string `json:"user"`
	Passwd string `json:"passwd"`
	DBName string `json:"dbname"`
	Prefix string `json:"prefix"`
}

type App struct {
	Debug            bool   `json:"debug"`
	Port             int    `json:"port"`
	TemplateDir      string `json:"template_dir"`
	AdminTemplateDir string `json:"admin_template_dir"`
	StaticDir        string `json:"static_dir"`
	RuntimeDir       string `json:"runtime_dir"`
	URLSuffix        string `json:"url_suffix"`
	PageSize         int    `json:"page_size"`
	PageNum          int    `json:"page_num"`
	AdminPath        string `json:"admin_path"`
	CacheEnabled     bool   `json:"cache_enabled"`
	CacheTime        int    `json:"cache_time"`
	SessionKey       string `json:"session_key"`
}

type Config struct {
	App      App      `json:"app"`
	Database Database `json:"database"`
}

var (
	globalConfig *Config
	once         sync.Once
)

func Load(path string) *Config {
	once.Do(func() {
		v := viper.New()

		// 預設值
		v.SetDefault("app.debug", true)
		v.SetDefault("app.port", 8080)
		v.SetDefault("app.template_dir", "template/default")
		v.SetDefault("app.admin_template_dir", "apps/admin/view")
		v.SetDefault("app.static_dir", "static")
		v.SetDefault("app.runtime_dir", "runtime")
		v.SetDefault("app.url_suffix", ".html")
		v.SetDefault("app.page_size", 15)
		v.SetDefault("app.page_num", 5)
		v.SetDefault("app.admin_path", "admin")
		v.SetDefault("app.cache_enabled", false)
		v.SetDefault("app.cache_time", 900)
		v.SetDefault("app.session_key", "gbootcms-session-key-32byte!!!")
		v.SetDefault("database.type", "sqlite")
		v.SetDefault("database.dbname", "data/pbootcms.db")
		v.SetDefault("database.prefix", "ay_")

		// 從 JSON 配置檔讀取
		v.SetConfigFile(path)
		v.SetConfigType("json")
		_ = v.ReadInConfig()

		// 環境變數覆蓋（PBOOTCMS_GO_ 前綴，例如 PBOOTCMS_GO_APP_SESSION_KEY）
		v.SetEnvPrefix("PBOOTCMS_GO")
		v.AutomaticEnv()

		globalConfig = &Config{
			App: App{
				Debug:            v.GetBool("app.debug"),
				Port:             v.GetInt("app.port"),
				TemplateDir:      v.GetString("app.template_dir"),
				AdminTemplateDir: v.GetString("app.admin_template_dir"),
				StaticDir:        v.GetString("app.static_dir"),
				RuntimeDir:       v.GetString("app.runtime_dir"),
				URLSuffix:        v.GetString("app.url_suffix"),
				PageSize:         v.GetInt("app.page_size"),
				PageNum:          v.GetInt("app.page_num"),
				AdminPath:        v.GetString("app.admin_path"),
				CacheEnabled:     v.GetBool("app.cache_enabled"),
				CacheTime:        v.GetInt("app.cache_time"),
				SessionKey:       v.GetString("app.session_key"),
			},
			Database: Database{
				Type:   v.GetString("database.type"),
				Host:   v.GetString("database.host"),
				Port:   v.GetInt("database.port"),
				User:   v.GetString("database.user"),
				Passwd: v.GetString("database.passwd"),
				DBName: v.GetString("database.dbname"),
				Prefix: v.GetString("database.prefix"),
			},
		}
	})
	return globalConfig
}

func Get() *Config {
	if globalConfig == nil {
		return Load("config/config.json")
	}
	return globalConfig
}
