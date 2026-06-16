// 一次性腳本：按用戶指定清單設置欄目 URL 名稱 (filename)
//
// 用法：go run ./cmd/set_sort_filenames
package main

import (
	"fmt"
	"log"
	"pbootcms-go/apps/admin/model"
	contentmodel "pbootcms-go/apps/admin/model/content"
	"pbootcms-go/config"
	"pbootcms-go/core/db"
)

// 欄目名稱 -> URL 名稱 映射
// 為兼容繁體/簡體，name 用 []string 列出所有可能的寫法
var sortFilenames = []struct {
	Names    []string
	Filename string
}{
	{[]string{"公司簡介", "關於我們", "关于我们"}, "aboutus"},
	{[]string{"新聞中心", "新闻中心"}, "article"},
	{[]string{"公司動態", "公司动态"}, "company"},
	{[]string{"行業動態", "行业动态"}, "industry"},
	{[]string{"產品中心", "产品中心"}, "product"},
	{[]string{"網站建設", "网站建设"}, "website"},
	{[]string{"域名空間", "域名空间"}, "domain"},
	{[]string{"服務案例", "服务案例"}, "case"},
	{[]string{"招賢納士", "招贤纳士"}, "job"},
	{[]string{"在線留言", "在线留言"}, "gbook"},
	{[]string{"聯繫我們", "联系我们"}, "contact"},
}

func main() {
	cfg := config.Load("config/config.json")
	if err := model.InitDB(cfg); err != nil {
		log.Fatalf("init db: %v", err)
	}

	// 先檢查 model.urlname 衝突
	for _, item := range sortFilenames {
		if contentmodel.CheckUrlname(item.Filename) {
			log.Fatalf("filename %q 與 ay_model.urlname 衝突，請換一個名稱", item.Filename)
		}
	}

	updated := 0
	notFound := []string{}
	unchanged := 0

	for _, item := range sortFilenames {
		// 查找欄目（嘗試所有名稱寫法）
		var sort model.ContentSort
		var found bool
		for _, name := range item.Names {
			if err := db.DB.Where("name = ?", name).First(&sort).Error; err == nil {
				found = true
				break
			}
		}
		if !found {
			notFound = append(notFound, item.Names[0])
			fmt.Printf("[SKIP] 找不到名稱為 %v 的欄目\n", item.Names)
			continue
		}

		// 已是目標值則跳過
		if sort.Filename == item.Filename {
			unchanged++
			fmt.Printf("[=] id=%d name=%q filename 已是 %q\n", sort.ID, sort.Name, sort.Filename)
			continue
		}

		// 檢查目標 filename 是否被其他欄目佔用（排除自己）
		if contentmodel.CheckFilename(item.Filename, fmt.Sprintf("id<>%d", sort.ID)) {
			log.Fatalf("filename %q 已被其他欄目佔用", item.Filename)
		}

		// 執行更新
		if err := db.DB.Model(&model.ContentSort{}).
			Where("id = ?", sort.ID).
			Update("filename", item.Filename).Error; err != nil {
			log.Printf("[WARN] id=%d update failed: %v", sort.ID, err)
			continue
		}
		fmt.Printf("[OK] id=%d name=%q scode=%s -> filename=%q\n",
			sort.ID, sort.Name, sort.Scode, item.Filename)
		updated++
	}

	fmt.Printf("\nDone. updated=%d unchanged=%d not_found=%d total=%d\n",
		updated, unchanged, len(notFound), len(sortFilenames))
}
