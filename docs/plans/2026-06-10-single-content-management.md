# 欄目內容管理（單頁內容）實施計劃

> **For agentic workers:** 按任務逐步執行,每步完成後驗證。

**Goal:** 完善 PbootCMS-Go 的單頁內容管理功能——將"專題"改名為"欄目內容",修復 SingleController 查詢邏輯,使後台能正確顯示和編輯各欄目下的單頁內容(如"公司簡介")。

**Architecture:** 
- 單頁內容存儲在 `ay_content` 表(與列表內容共用),通過 `ay_content.scode → ay_content_sort.mcode → ay_model.type=1` 關聯
- 類型判斷來自 `ay_model` 表的 JOIN,而非 `ay_content_sort.type`
- 前台渲染時通過 model 的 type 決定使用 `contenttpl`(單頁) 還是 `listtpl`(列表)

**Tech Stack:** Go + Gin + GORM + SQLite + pongo2

---

## 文件結構

| 操作 | 文件 | 職責 |
|------|------|------|
| 修改 | `apps/admin/controller/content/SingleController.go` | 修復 Index 查詢(JOIN ay_model),完善 Mod |
| 修改 | `apps/admin/seed/seed.go` | 菜單"單頁內容"→"欄目內容" |
| 修改 | `apps/home/controller/front.go` | renderSortPage 用 model type 判斷模板 |
| 修改 | `templates/admin/common/head.html` | 側邊欄動態菜單顯示"欄目內容" |
| 不動 | `apps/admin/model/content/ContentSortModel.go` | 保持不動,type 通過 JOIN 獲取 |

---

### Task 1: 修復 SingleController.Index() 查詢邏輯

**問題:** 當前查詢 `type = 2` 永遠匹配不到,應 JOIN `ay_model` 查 `type=1`(單頁模型)。

**Files:**
- Modify: `apps/admin/controller/content/SingleController.go:22-42`

- [ ] **Step 1: 修改 Index 方法的查詢邏輯**

將 `Index()` 方法中的查詢改為 JOIN `ay_model` 過濾單頁模型:

```go
func (sg *SingleController) Index(c *gin.Context) {
	mcode := c.Query("mcode")
	if mcode == "" {
		mcode = "3D2" // 默認單頁模型
	}

	var sorts []model.ContentSort
	// 通過 mcode 關聯查詢屬於單頁模型的欄目
	model.DB.Where("mcode = ? AND status = 1", mcode).Order("sorting ASC").Find(&sorts)

	var contents []model.Content
	if len(sorts) > 0 {
		var scodes []string
		for _, s := range sorts {
			scodes = append(scodes, s.Scode)
		}
		// 取每個 scode 的最新一條記錄(單頁每個欄目只有一條)
		model.DB.Where("scode IN (?) AND id IN (SELECT MAX(id) FROM ay_content WHERE sccode IN (?) GROUP BY scode)",
			scodes, scodes).Order("scode ASC").Find(&contents)
	}

	data := gin.H{}
	data["contents"] = contents
	data["sorts"] = sorts
	data["mcode"] = mcode
	data["list"] = true
	common.Render(c, "content/single.html", data)
}
```

- [ ] **Step 2: 編譯驗證**

```bash
cd F:\mysite\AI\idea\pbootcmstogo\pbootcms-go
go build -o bin\pbootcms-go.exe .
```

Expected: 編譯成功

- [ ] **Step 3: Commit**

```bash
git add apps/admin/controller/content/SingleController.go
git commit -m "fix: SingleController.Index JOIN ay_model 查詢單頁內容"
```

---

### Task 2: 修復 renderSortPage() 模板選擇邏輯

**問題:** 當前用 `sort.Type==0` 判斷單頁,但 `ay_content_sort` 原始表沒有 type 字段。應查 `ay_model.type`。

**Files:**
- Modify: `apps/home/controller/front.go:207-232`

- [ ] **Step 1: 修改 renderSortPage 方法**

```go
func (fc *FrontController) renderSortPage(c *gin.Context, sort *model.ContentSort) {
	ctx := fc.buildContext(c)
	ctx.Sort = sort
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		ctx.CurrentPage = p
	}
	p := parser.New()
	parser.RegisterAllProviders(p, ctx)

	// 通過 mcode 查 ay_model 獲取 type
	var tpl string
	var contentModel model.ContentModel
	if sort.Mcode != "" && model.DB.Where("mcode = ?", sort.Mcode).First(&contentModel).Error == nil {
		if contentModel.Type == 1 {
			// 單頁模型 → 用 ContentTpl
			tpl = sort.ContentTpl
		} else {
			// 列表模型 → 用 ListTpl
			tpl = sort.ListTpl
		}
	} else {
		// 默認用 ListTpl
		tpl = sort.ListTpl
	}
	if tpl == "" {
		tpl = "list.html"
	}

	// 單頁需要加載內容數據
	if contentModel.Type == 1 {
		var content model.Content
		if model.DB.Where("scode = ? AND status = 1", sort.Scode).Order("id DESC").First(&content).Error == nil {
			ctx.Content = &content
		}
	}

	content := fc.Store.Render(tpl)
	content = p.Render(content)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, content)
}
```

- [ ] **Step 2: 編譯驗證**

Expected: 編譯成功

- [ ] **Step 3: Commit**

```bash
git add apps/home/controller/front.go
git commit -m "fix: renderSortPage 通過 ay_model.type 判斷單頁/列表"
```

---

### Task 3: 菜單"專題"→"欄目內容"

**Files:**
- Modify: `apps/admin/seed/seed.go:121`
- Modify: `templates/admin/common/head.html:93-101`

- [ ] **Step 1: 修改種子數據菜單名稱**

在 `seed.go` 中找到 `{Mcode: "M131", Pcode: "M130", Name: "單頁內容", ...}` 改為:
```go
{Mcode: "M131", Pcode: "M130", Name: "栏目内容", URL: "/admin/Single/index", ...},
```

- [ ] **Step 2: 修改側邊欄動態菜單**

在 `head.html` 中找到 type=1 的菜單項,將顯示文字改為"欄目內容":
```html
{if($value3->type==1)}
    <dd><a href="{url./admin/Single/index/mcode/'.$value3->mcode.'}">
        <i class="fa fa-file-text-o"></i> [value3->name]内容</a></dd>
{/if}
```
(保持 `[value3->name]内容` 格式不變,因為模型名稱本身就叫"單頁模型",顯示為"單頁模型內容")

- [ ] **Step 3: 編譯驗證 + Commit**

```bash
git add apps/admin/seed/seed.go templates/admin/common/head.html
git commit -m "refactor: 菜單'單頁內容'改名為'欄目內容'"
```

---

### Task 4: 確保單頁內容可被編輯

**問題:** 需要驗證 SingleController.Mod() 能正確讀取和保存單頁內容。

**Files:**
- Verify: `apps/admin/controller/content/SingleController.go` Mod 方法

- [ ] **Step 1: 啟動服務,訪問後台**

訪問 `http://localhost:8080/admin/Single/index` 檢查是否顯示單頁內容列表

- [ ] **Step 2: 點擊修改,驗證表單**

確認表單能正確顯示標題、內容、SEO 信息等字段

- [ ] **Step 3: 修改並保存**

修改"公司簡介"的內容,點擊保存,驗證數據持久化

- [ ] **Step 4: 前台驗證**

訪問 `http://localhost:8080/aboutus` 確認修改後的內容正確顯示

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "fix: 單頁內容管理全鏈路修復"
```

---

### Task 5: 清理無用代碼

**Files:**
- Modify: `apps/admin/model/content/SingleModel.go` - 移除未使用的 Single 結構體(或保留為 Content 別名)

- [ ] **Step 1: 檢查 Single 結構體使用情況**

確認 `Single` 結構體是否被任何代碼引用

- [ ] **Step 2: 清理或保留**

如果未被使用,可以刪除或改為 `type Single = Content` 別名

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: 清理未使用的 Single 結構體"
```
