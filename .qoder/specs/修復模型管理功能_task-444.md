# 完善擴展字段與內容管理

## Context

PbootCMS 的內容管理通過 `mcode`（模型編碼）串聯三個維度：模型定義 → 擴展字段 → 內容數據。Go 版本已有基本框架，但擴展字段的**數據保存和加載完全缺失**：

- `ay_content_ext` 表可能不存在或無動態列
- `ExtFieldController.Add()` 新增字段時不執行 `ALTER TABLE` 添加物理列
- `ContentController.Add/Mod()` POST 不讀取 ext_ 前綴表單值
- `ContentService.GetContent()` 不 JOIN `ay_content_ext` 表
- `content.html` 模板中 `{php}` 代碼塊被轉譯器直接刪除（單選/多選/下拉選項全部丟失）

## Task 1: 數據庫基礎 — `ay_content_ext` 表與動態列

**文件**: `apps/admin/model/content/ExtFieldModel.go`

新增函數：
```go
func EnsureContentExtTable()           // CREATE TABLE IF NOT EXISTS ay_content_ext
func ColumnExistsInContentExt(col string) bool  // PRAGMA table_info 檢查
func AddColumnToContentExt(col, colType string) error  // ALTER TABLE ADD COLUMN
func SqliteColumnTypeForExtType(typ string) string  // 類型映射
```

類型映射（與 PHP 一致）：
| type | 用途 | SQLite 列類型 |
|------|------|-------------|
| 2 | 多行文本 | TEXT(1000) |
| 7 | 日期 | TEXT |
| 8 | 編輯器 | TEXT(10000) |
| 10 | 多圖 | TEXT(1000) |
| 其他 | — | TEXT(200) |

**文件**: `apps/admin/seed/seed.go`

在 `SeedData()` 中調用 `EnsureContentExtTable()`（冪等操作）。

**文件**: `apps/admin/controller/content/ExtFieldController.go`

`Add()` POST 成功後，檢查列是否存在 → 不存在則 ALTER TABLE 添加物理列。

## Task 2: 模型層 — `ContentExtModel.go`（新建）

**文件**: `apps/admin/model/content/ContentExtModel.go`

使用 `map[string]interface{}` 操作（因列是動態的）：
```go
func GetContentExtByContentID(contentID uint) map[string]interface{}
func InsertContentExt(data map[string]interface{}) error
func UpdateContentExt(contentID uint, data map[string]interface{}) error
func UpsertContentExt(contentID uint, data map[string]interface{}) error
func DeleteContentExt(contentID uint) error
```

## Task 3: 服務層 — ContentService 集成擴展字段

**文件**: `apps/admin/service/content/ContentService.go`

修改 `CreateContent` 簽名，增加 `extData` 參數：
```go
func (s *ContentService) CreateContent(doc *model.Content, extData map[string]interface{}) error
```
- 先插入主表 → 得到 doc.ID → 插入 ay_content_ext

修改 `UpdateContent` 簽名，增加 `extData` 參數：
```go
func (s *ContentService) UpdateContent(id int, updates map[string]interface{}, extData map[string]interface{}) error
```
- 更新主表 → UpsertContentExt

修改 `DeleteContent`：
- 刪除主表記錄 → 同步刪除 ay_content_ext

新增 `GetContentWithExt`：
```go
func (s *ContentService) GetContentWithExt(id int) (map[string]interface{}, error)
```
- 查 ay_content → 轉 map → 注入 ext 字段數據

同步修改 `CopyContent`：複製時也複製擴展數據。

## Task 4: 控制器層 — 收集與傳遞 ext 數據

**文件**: `apps/admin/controller/content/ContentController.go`

`Add()` POST：
- 調用 `GetExtFieldsByMcode(mcode)` 獲取字段名列表
- 遍歷 `c.PostForm(name)` 收集 ext_ 值（多選用 `c.PostFormArray`）
- 傳入 `svc.CreateContent(&doc, extData)`

`Mod()` POST：
- 同樣收集 ext 數據
- 傳入 `svc.UpdateContent(id, updates, extData)`

`Mod()` GET：
- 改用 `svc.GetContentWithExt(id)` 返回 map（含擴展數據）
- 傳入模板

`contentTemplateData()`：
- 改為傳入預處理的 ext field 數據（包含 `CurrentValue`、`Options`、`SelectedValues` 等）

新增 `buildExtFieldTemplateData()`：
- 為每個 ext field 構建模板友好的 map：
  - `Name`, `Type`, `Description`, `Required`, `Value`
  - `CurrentValue` — 編輯時的當前值
  - `Options` — 單選/多選/下拉的選項切片
  - `SelectedValues` — 多選的已選值切片
  - `Pics` — 多圖的圖片路徑切片

## Task 5: 模板改造 — 消除 `{php}` 依賴

**文件**: `apps/admin/view/content/content.html`

核心問題：`{php}` 塊被轉譯器 `rePhpBlock` 正則直接刪除，導致單選/多選/下拉的選項循環完全不存在。

策略：將 ext field 區域的 `{php}` 塊替換為 pongo2 原生語法。

### Add 表單（行 201-332）：
| 類型 | 原 PHP | 替換為 pongo2 |
|------|--------|--------------|
| 3 單選 | `$radios=explode(',')...` | `{% for opt in val1.Options %}<input type="radio"...>{% endfor %}` |
| 4 多選 | `$checkboxs=explode(',')...` | hidden + `{% for opt in val1.Options %}<input type="checkbox"...>{% endfor %}` |
| 9 下拉 | `$selects=explode(',')...` | `{% for opt in val1.Options %}<option...>{% endfor %}` |

### Mod 表單（行 553-721）：
| 類型 | 原 PHP 回顯 | 替換為 pongo2 |
|------|-----------|--------------|
| 1 單行 | `{$content->{$value->name}}` | `{{ val1.CurrentValue }}` |
| 2 多行 | `{php}...str_replace('<br>'...){/php}` | `{{ val1.CurrentValueMultiline }}` |
| 3 單選 | `{php}if($content->$name==...){/php}` | `{% if val1.CurrentValue == opt %} checked{% endif %}` |
| 4 多選 | `{php}if(in_array(...)){/php}` | `{% if opt in val1.SelectedValues %} checked{% endif %}` |
| 5 圖片 | `{php}$name=...{/php}` | `{% if val1.CurrentValue %}<img...>{% endif %}` |
| 8 編輯器 | `{fun=decode_string([$content->$name])}` | `{{ val1.CurrentValue|safe }}` |
| 9 下拉 | `{php}if($content->$name==...){/php}` | `{% if val1.CurrentValue == opt %} selected{% endif %}` |
| 10 多圖 | `{php}$pics=explode(...){/php}` | `{% for pic in val1.Pics %}<dl>...{% endfor %}` |

**注意**：pongo2 原生 `{% for %}` 和 `{% if %}` 標籤不會被轉譯器破壞（轉譯器只處理 PbootCMS 標籤）。

## Task 6: 輔助函數

**文件**: `apps/admin/helper/template_helpers.go`

新增 `SnakeToPascal`（導出版本），供 `ContentController` 和 `ContentService` 使用。

## Task 7: 消息繁體化

**文件**: `apps/admin/controller/content/ContentController.go`

所有英文消息改為繁體中文：
- "Added successfully" → "新增成功"
- "Modified successfully" → "修改成功"
- "Deleted successfully" → "刪除成功"
- "Copied successfully" → "複製成功"
- "Moved successfully" → "移動成功"
- "Submitted successfully" → "提交成功"
- "Title cannot be empty" → "標題不能為空"
- "No items selected" → "未選擇任何項目"
- "target sort cannot be empty" → "目標欄目不能為空"
- "content does not exist" → "內容不存在"

**文件**: `apps/admin/service/content/ContentService.go`

- "title and sort cannot be empty" → "標題和欄目不能為空"
- "content does not exist" → "內容不存在"
- "target sort cannot be empty" → "目標欄目不能為空"
- "field not allowed" → "不允許修改的欄位"

**文件**: `content.html`

- "您确定要删除选中的内容么？" → "您確定要刪除選中的內容麼？"

## 涉及文件

| 文件 | 操作 | 描述 |
|------|------|------|
| `apps/admin/model/content/ContentExtModel.go` | 新建 | ay_content_ext 表 CRUD |
| `apps/admin/model/content/ExtFieldModel.go` | 修改 | 增加 EnsureContentExtTable/AddColumn 等函數 |
| `apps/admin/controller/content/ExtFieldController.go` | 修改 | Add 時 ALTER TABLE |
| `apps/admin/service/content/ContentService.go` | 修改 | CreateContent/UpdateContent/DeleteContent 集成 ext，新增 GetContentWithExt |
| `apps/admin/controller/content/ContentController.go` | 修改 | Add/Mod 收集 ext 數據，Mod GET 加載 ext 數據 |
| `apps/admin/helper/template_helpers.go` | 修改 | 導出 SnakeToPascal |
| `apps/admin/view/content/content.html` | 修改 | ext field 區域替換 {php} 為 pongo2 |
| `apps/admin/seed/seed.go` | 修改 | 調用 EnsureContentExtTable |

## 驗證步驟

1. 編譯 `go build -o bin\pbootcms-go.exe .`
2. 管理後台 → 模型字段 → 新增一個 type=1（單行文本）字段
3. 用 SQLite 工具確認 `ay_content_ext` 表已新增對應列
4. 內容列表 → 新增一條帶擴展字段值的內容 → 確認 ay_content_ext 表有新行
5. 編輯該內容 → 確認擴展字段值正確回顯
6. 測試單選、多選、下拉類型的選項渲染和選中狀態
7. 刪除內容 → 確認 ay_content_ext 記錄同步清除
# 修復模型管理功能

## Context

`/admin/Model/index` 模型管理頁面已有基本框架但存在多處缺陷，導致功能不完整：
- 狀態切換按鈕 URL (`/mod/id/3/field/status/value/0`) 與路由 (`/mod/:id`) 不匹配 → **狀態切換完全失效**
- 新增不傳 `listtpl`/`contenttpl` → 新增的模型缺少模板配置
- 修改不更新 `type`/`status`/`listtpl`/`contenttpl` → **修改表單形同虛設**
- 模板期望 `$list`/`$mod` 變量但控制器傳 `action` → 頁面空白或顯示錯誤
- 刪除不檢查 `issystem` 和欄目關聯 → 可能誤刪

## Task 1: 路由改為 `*action` 通配（route.go）

將 Model 路由從 `:id` 改為 `*action` 通配，與 ContentSort、Slide 等控制器統一風格：

```go
// 現行
adminGroup.GET("/content/model/mod/:id", md.Mod)
adminGroup.POST("/content/model/mod/:id", md.Mod)

// 改為
adminGroup.Any("/content/model/mod/*action", md.Mod)
adminGroup.Any("/content/model/del/*action", md.Del)
```

## Task 2: 重寫 ModelController.go

使用 `helper.ParseWildcardAction` 解析 URL 參數，與 ContentSortController 保持一致的編碼風格：

**Index**: 傳 `list: true` + `models`

**Add** (POST):
- 讀取全部表單欄位：`name`, `type`, `urlname`, `listtpl`, `contenttpl`, `status`
- 驗證 name 非空
- 驗證 urlname 正則 `^[a-zA-Z0-9\-]+$`（僅非空時）
- 自動生成 mcode（`GetNextMcode` 遞增）
- 傳 `list: true` 給 GET 渲染

**Mod** (GET):
- 解析 wildcard action，支持 `/id/3/field/status/value/0` 快速切換
- 支持 `?id=3` 查詢參數
- GET 單條模型 → 渲染修改表單，傳 `mod: true` + `model`

**Mod** (POST):
- 讀取全部表單欄位：`name`, `type`, `urlname`, `listtpl`, `contenttpl`, `status`
- 驗證同 Add
- 更新全部欄位

**Del**:
- 檢查 `issystem=1` → 拒絕刪除
- 檢查 `ay_content_sort` 有無關聯該 mcode → 有則拒絕

## Task 3: 修復 ModelModel.go

**AddModel** — 新增 `listtpl`, `contenttpl` 參數：
```go
func AddModel(mcode, name, urlname, listtpl, contenttpl, updateUser string, typ, status int) error
```

**UpdateModel** — 新增 `typ`, `status`, `listtpl`, `contenttpl`：
```go
func UpdateModel(id int, mcode, name, urlname, listtpl, contenttpl, updateUser string, typ, status int) error
```

**DeleteModel** — 增加關聯檢查：
```go
func DeleteModel(id int) error {
    // 1. 檢查 issystem
    // 2. 檢查 ay_content_sort 關聯
    // 3. 執行刪除
}
```

**GetNextMcode** — 實現遞增邏輯：
```go
func GetNextMcode() string {
    // 查詢最大 mcode → 提取數字部分 → +1 → 格式化
}
```

新增 **CheckUrlnameConflict** — 檢查 urlname 跨模型衝突及與欄目 filename 衝突。

## Task 4: 模板文字繁體化（model.html）

模板中修改涉及的後台顯示文字改為香港繁體：
- `模型列表` → `模型列表`
- `模型新增` → `模型新增`
- `模型修改` → `模型修改`
- `请输入模型名称` → `請輸入模型名稱`
- `请选择模型类型` → `請選擇模型類型`
- `单页` → `單頁`
- `列表` → `列表`
- 等等...

模板結構不變，僅改動中文文字。

## 涉及文件

| 文件 | 操作 |
|------|------|
| `apps/route/route.go` | 改路由為 `*action` |
| `apps/admin/controller/content/ModelController.go` | 重寫 |
| `apps/admin/model/content/ModelModel.go` | 修復 AddModel/UpdateModel/DeleteModel/GetNextMcode，新增 CheckUrlnameConflict |
| `apps/admin/view/content/model.html` | 繁體化顯示文字 |

## 驗證步驟

1. 編譯：`go build -o bin\pbootcms-go.exe .`
2. 重啟服務器
3. 訪問 `/admin/Model/index` → 確認列表正常顯示
4. 點擊狀態切換 → 確認 AJAX 成功切換
5. 新增模型（含模板欄位）→ 確認數據完整寫入
6. 修改模型全部欄位 → 確認全部更新
7. 刪除系統模型 → 確認被拒絕
8. 刪除有欄目關聯的模型 → 確認被拒絕
