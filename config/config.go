package config

import (
	"encoding/json"
	"os"
	"sync"
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
	Debug             bool   `json:"debug"`
	Port              int    `json:"port"`
	TemplateDir       string `json:"template_dir"`
	AdminTemplateDir  string `json:"admin_template_dir"`
	StaticDir         string `json:"static_dir"`
	RuntimeDir    string `json:"runtime_dir"`
	URLSuffix     string `json:"url_suffix"`
	PageSize      int    `json:"page_size"`
	PageNum       int    `json:"page_num"`
	AdminPath     string `json:"admin_path"`
	CacheEnabled  bool   `json:"cache_enabled"`
	CacheTime     int    `json:"cache_time"`
}

type Config struct {
	App      App      `json:"app"`
	Database Database `json:"database"`
}

var (
	globalConfig *Config
	once        sync.Once
)

func Load(path string) *Config {
	once.Do(func() {
		globalConfig = &Config{
			App: App{
				Debug:            true,
				Port:             8080,
				TemplateDir:       "templates",
				AdminTemplateDir: "apps/admin/view",
				StaticDir:        "static",
				RuntimeDir:   "runtime",
				URLSuffix:    ".html",
				PageSize:     15,
				PageNum:      5,
				AdminPath:    "admin",
				CacheEnabled: false,
				CacheTime:    900,
			},
			Database: Database{
				Type:   "sqlite",
				DBName: "data/pbootcms.db",
				Prefix: "ay_",
			},
		}

		data, err := os.ReadFile(path)
		if err == nil {
			json.Unmarshal(data, globalConfig)
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
