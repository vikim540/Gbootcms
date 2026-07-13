# AGENTS.md

> 本文件供 AI 代理（Claude / GPT / Cursor / Trae 等）在接手 Gbootcms 項目時快速建立上下文。
> 詳細技術文檔見 `docs/pbootcms-go-dev-guide.md`，AI 開發約定見 `docs/AI_GUIDELINES.md`。

---

## 項目簡介

Gbootcms 是基於 PbootCMS 3.2.12 的 Go 語言移植版。保留原版數據庫結構（`ay_` 前綴）、模板語法、URL 路由規則和後台 UI（Layui），用 Go 技術棧替換 PHP 後端。

- **模組名**：`gbootcms`（go.mod）
- **Go 版本**：1.25.0
- **數據庫**：SQLite（glebarez 純 Go 驅動，無需 CGO），文件路徑 `data/pbootcms.db`
- **前台**：http://localhost:8080/
- **後台**：http://localhost:8080/admin
- **預設帳號**：`admin` / `123456`

### 編譯與運行

```powershell
# 編譯（產物必須輸出到 bin/）
go build -o bin/gbootcms.exe .

# 運行
.\bin\gbootcms.exe

# 或用構建腳本
.\build-run.bat
```

---

## 技術棧

| 層次 | 選型 | 說明 |
|------|------|------|
| 語言 | Go 1.25 | 單二進制部署，無需 CGO |
| Web 框架 | Gin v1.12 | 路由、中間件、請求處理 |
| ORM | GORM v1.31 | AutoMigrate，`ay_` 前綴 |
| 數據庫 | SQLite (glebarez 純 Go) | 無需安裝 CGO/GCC |
| 後台模板 | Pongo2 v6.1 | Django 風格 + PbootCMS 語法轉換器 |
| 前台模板 | 自研 TagParser | `{gboot:xxx}` 標籤 + fsnotify 熱重載 |
| 後台 UI | Layui 2.5.4 + jQuery 1.12.4 | 與 PbootCMS 原版一致 |
| 前台 UI | Bootstrap 4 + Swiper 4 | 前台模板自帶 |
| 配置 | Viper v1.21 | 環境變數前綴 `PBOOTCMS_GO_` |
| 日誌 | log/slog (stdlib) | 結構化日誌 |

---

## 目錄結構

```
gbootcms/
├── main.go                          # 程序入口
├── config/
│   ├── config.go                    # 配置結構體與載入（Viper 單例）
│   └── config.json                  # 配置文件（端口、DB 路徑等）
├── core/
│   ├── db/db.go                     # GORM + SQLite 初始化
│   ├── basic/view.go                # 後台模板引擎（pongo2 + PHP 語法轉換）
│   └── mediaplugin/plugin.go        # GORM 媒體快取失效插件
├── apps/
│   ├── route/route.go               # 路由集中註冊
│   ├── common/
│   │   ├── BaseController.go        # 基礎控制器（JSON 回應、批量排序等）
│   │   ├── Render.go                # 後台模板渲染入口
│   │   ├── session.go               # 自實現記憶體 Session
│   │   ├── notice.go                # 通知訊息常量
│   │   ├── captcha.go               # 驗證碼生成與校驗
│   │   ├── turnstile.go             # Cloudflare Turnstile 驗證
│   │   ├── middleware/              # 認證/壓縮/快取/IP過濾/URL重寫等中間件
│   │   └── parser/                  # 前台模板標籤解析引擎
│   ├── admin/
│   │   ├── controller/              # 後台控制器
│   │   │   ├── content/             # 內容管理（Content/ContentSort/Single/Slide/Link/Form/ExtField 等）
│   │   │   ├── system/              # 系統管理（Config/Menu/Role/User/Syslog 等）
│   │   │   └── member/              # 會員管理（Member/MemberGroup/MemberField/MemberComment）
│   │   ├── model/                   # 數據模型
│   │   │   ├── db.go                # 全域 DB 實例 + 類型別名
│   │   │   ├── content/             # 內容模型
│   │   │   ├── system/              # 系統模型
│   │   │   └── member/              # 會員模型
│   │   ├── service/content/         # 業務服務層
│   │   ├── helper/                  # 模板輔助函數
│   │   └── view/                    # 後台 HTML 模板
│   └── home/
│       └── controller/
│           ├── front.go             # 前台控制器
│           ├── member.go            # 會員前台控制器
│           └── comment.go           # 評論控制器
├── template/default/                # 前台模板目錄
├── static/                          # 全域靜態資源
├── data/pbootcms.db                 # SQLite 數據庫
├── bin/                             # 編譯產物
└── docs/                            # 文檔
```

---

## 雙模板引擎架構

這是新人最容易混淆的一點：**前台走自研 `gboot` 解析器，後台走 pongo2**。

| 場景 | 引擎 | 語法 | 轉換器 |
|------|------|------|--------|
| 後台 admin view | pongo2 | `{if([$list])}` → `{% if List %}` | `core/basic/view.go` convertPbootToPongo2() |
| 前台 template/default | 自研 TagParser | `{gboot:xxx}` + `[prefix:field]` | `apps/common/parser/tags.go` |

後台模板用 `{$var->field}` / `[value->field]` 語法（pongo2 轉譯），前台模板用 `{gboot:xxx}` 標籤。

### 模板變量語法速查

| 語法 | 說明 | 範例 |
|------|------|------|
| `[$var]` | 扁平變量 | `[$formcheck]` → `{{ Formcheck }}` |
| `[$var->field]` | 物件屬性 | `[$form->fcode]` → `{{ Form.Fcode }}` |
| `[value->field]` | 循環變量屬性 | `[value->name]` → `{{ value.Name }}` |
| `{$var->field}` | 模板變量 | `{$link->name}` → `{{ link.Name }}` |
| `{if([$list])}` | 條件判斷 | 轉為 `{% if List %}` |
| `{foreach $items(key,value)}` | 循環 | 轉為 `{% for ... %}` |

---

## 寫代碼前的強制檢查流程

```
1. 讀 docs/AI_GUIDELINES.md 和 docs/pbootcms-go-dev-guide.md — 檢查硬約束
2. Grep 一個已實現的同類控制器 — 參考其模式（如 ContentSort、Slide）
3. 確認方法/函數存在 — Grep 函數名，不要猜測
4. 確認模型欄位名 — 讀 Model struct，不要猜測
5. 確認 DB 表結構 — sqlite3 data/pbootcms.db ".schema ay_xxx"
```

---

## 硬約束（不可違反）

1. **數據庫零改動** — 不修改、不刪除原版表結構和字段，表前綴 `ay_`
2. **自定義字段用 `mcode`** — 查詢和插入時用 `mcode` 而非 `modelcode`
3. **表單 action URL 用小寫** — `/admin/content/mod` 而非 `/admin/Content/mod`，防止 POST 數據丟失
4. **模板字段類型用整數比較** — `val1.Type==1` 而非字符串比較
5. **Layui 多選框不用 `lay-skin="primary"`** — 保持原版 UI 和值提交
6. **日期字串解析** — `time.Parse("2006-01-02 15:04:05", v)` 後存入 DB
7. **Pongo2 模板用預格式化 DateStr** — 不存在 `date` 過濾器，控制器中預格式化
8. **通知消息用常量** — `common.NoticeAdd` / `common.NoticeModify` / `common.NoticeDelete`，禁止硬編碼
9. **編輯表單做髒檢查** — 無變更時不寫 DB、不發成功通知
10. **構建產物輸出 `bin/`** — 不滯留根目錄
11. **文檔放 `docs/`** — 所有 `.md` 文件放 docs 文件夾，根目錄僅保留必要文件
12. **模板清除 BOM** — 所有模板文件開頭的 `U+FEFF` 必須清除
13. **狀態切換鏈接加 `class="switch"`** — comm.js 用此選擇器攔截 AJAX，否則瀏覽器直接顯示 JSON
14. **前台 include 路徑用 `common/`** — `{include file='common/head.html'}` 而非 `comm/`
15. **繁體中文** — 所有修改的代碼和模板使用繁體中文
16. **前台欄目/內容權限檢查** — 必須呼叫 `checkSortPermission` / `checkContentPermission`
17. **PbootCMS message 表字段** — `user_ip` 非 `ip`，`create_time` 非 `askdate`，`ReContent` 非 `ReCode`
18. **首頁 banner 高度限制** — `.swiper-container` 最大高度 400px + `overflow: hidden`
19. **自定義表單 table_name 前綴** — `ay_diy_` 而非 `form_data_`
20. **前台模板標籤前綴** — `{gboot:xxx}` 而非 `{pboot:xxx}`
21. **用戶輸入必須過濾** — 富文字內容經 `common.FilterUserInput()` 處理（XSS 防護）
22. **動態 SQL 標識符必須驗證** — 表名用 `CheckVarType()`，欄位名用 `CheckColumnName()` 白名單驗證
23. **會員登入必須重新生成 Session ID** — 呼叫 `common.RegenerateSessionID(c)` 防 Session Fixation
24. **刪除操作必須用 POST** — GET 刪除有 CSRF 風險
25. **AJAX 攔截回應雙欄位** — `auth.go` 中 AJAX 回應必須同時包含 `data` + `msg`
26. **通知文案不加感嘆號** — 統一風格，所有通知訊息不帶 `！`

---

## 後台控制器標準模式

### 列表頁

```go
common.Render(c, "module/template.html", gin.H{
    "list":     true,        // 必須！模板用 {if([$list])} 判斷
    "items":    items,
    "C":        "module",    // 當前控制器路徑
    "pagebar":  helper.BuildPagebarHTML(total, page, pageSize, baseURL),
    "pagesize": pageSize,
})
```

### 修改頁

```go
common.Render(c, "module/template.html", gin.H{
    "mod":    true,          // 必須！模板用 {if([$mod])} 判斷
    "item":   item,
    "C":      "module",
})
```

遺忘 `list`/`mod` 標誌會導致頁面空白。

### 路由模式

```go
// 有狀態切換的控制器必須用 *action 通配符
adminGroup.Any("/admin/xxx/mod/*action", ctrl.Mod)
adminGroup.GET("/admin/xxx/del/*action", ctrl.Del)

// 無狀態切換可用固定路由
adminGroup.GET("/admin/xxx/index", ctrl.Index)
adminGroup.POST("/admin/xxx/add", ctrl.Add)
```

### Mod 方法解析 *action

```go
func (ctrl *Controller) Mod(c *gin.Context) {
    params := helper.ParseWildcardAction(c.Param("action"))
    idStr := params["id"]       // /id/123 → "123"
    field := params["field"]    // /field/status → "status"
    value := params["value"]    // /value/0 → "0"

    // 單欄位切換（狀態開關）
    if field != "" && value != "" {
        model.DB.Model(&Model{}).Where("id = ?", id).Update(field, value)
        ctrl.JSONOKMsg(c, common.NoticeModify)
        return
    }
    // ... 完整修改邏輯
}
```

---

## API 速查表

### BaseController 方法（apps/common/BaseController.go）

| 方法 | 用途 |
|------|------|
| `JSONOK(c, data)` | 成功響應 `{"code":1,"data":...}` |
| `JSONOKMsg(c, msg)` | 成功響應帶消息 |
| `JSONFail(c, msg)` | 失敗響應 `{"code":0,"msg":...}` |
| `BatchSort(c, modelPtr, sortCol, defaultSort)` | 通用批量排序 |
| `GetAdminUsername(c) string` | 從 Session 獲取管理員用戶名 |
| `GetAdminUID(c) int` | 從 Session 獲取管理員 UID |
| `IsLogin(c) bool` | 判斷登錄態 |
| `IsBatchSort(c) bool` | 判斷是否批量排序請求 |
| `Paginate(c) (page, pageSize, offset)` | 分頁參數 |

> **不存在的方法**：`JSONErrMsg`、`GetAutoCode`、`AdminController.InitAdmin`

### 通知消息常量（apps/common/notice.go）

```go
common.NoticeAdd      // "新增成功"
common.NoticeModify   // "修改成功"
common.NoticeDelete   // "刪除成功"
common.NoticeNoChange // "內容未發生變化"
```

### 雙 MD5 密碼雜湊（兩行寫法）

```go
// 兩行寫法，不要嵌套
firstMd5 := fmt.Sprintf("%x", md5.Sum([]byte(password)))
encPwd := fmt.Sprintf("%x", md5.Sum([]byte(firstMd5)))
```

### time.Time 模板顯示

```go
// 在 Model 中加非 DB 欄位，控制器中預格式化
type Member struct {
    RegisterTime   time.Time
    RegisterTimeStr string `gorm:"-" json:"register_time_str"`
}
// 控制器
if !member.RegisterTime.IsZero() {
    member.RegisterTimeStr = member.RegisterTime.Format("2006-01-02 15:04:05")
}
// 模板
<td>[value->register_time_str]</td>
```

---

## 狀態切換（class="switch"）

後台模板的狀態切換圖標**必須**加 `class="switch"`，否則 `comm.js` 無法攔截點擊，瀏覽器直接顯示 JSON。

```html
<!-- 正確：comm.js 攔截 AJAX，切換圖標，不跳轉 -->
<a href="/admin/xxx/mod/id/[value->id]/field/status/value/0" class="switch">
  <i class='fa fa-toggle-on'></i>
</a>
```

適用字段：status / required / istop / isrecommend 等所有開關字段。

---

## Session 鍵名

### 後台管理員

| 鍵名 | 用途 |
|------|------|
| `pboot_admin_uid` | 管理員 ID |
| `pboot_admin_ucode` | 管理員編號 |
| `pboot_admin_username` | 用戶名 |

### 前台會員

| 鍵名 | 用途 |
|------|------|
| `pboot_uid` | 會員 ID |
| `pboot_ucode` | 會員編號 |
| `pboot_username` | 用戶名 |
| `pboot_useremail` | 郵箱 |
| `pboot_usermobile` | 手機 |
| `pboot_gid` | 等級 ID |
| `pboot_gcode` | 等級編號（int 類型） |
| `pboot_gname` | 等級名稱 |

---

## 配置系統

`config/config.json` 主要配置項：

```json
{
  "app": {
    "debug": true,
    "port": 8080,
    "template_dir": "template/default",
    "admin_template_dir": "apps/admin/view",
    "url_suffix": ".html",
    "page_size": 15,
    "admin_path": "admin"
  },
  "database": {
    "type": "sqlite",
    "dbname": "data/pbootcms.db",
    "prefix": "ay_"
  }
}
```

環境變數覆蓋（前綴 `PBOOTCMS_GO_`）：
`PBOOTCMS_GO_DATABASE_TYPE=mysql` 覆蓋 `database.type`

後台動態配置存於 `ay_config` 表，用 `model.GetConfigValue(key, default)` 讀取。

---

## 反模式清單（高頻錯誤）

| # | 錯誤行為 | 正確做法 |
|---|---------|---------|
| 1 | 不讀 docs 直接寫代碼 | 先讀 AI_GUIDELINES.md 和 dev-guide |
| 2 | 猜測方法名 | Grep 確認方法存在 |
| 3 | 猜測模型欄位名 | 讀 Model struct 定義 |
| 4 | 用 `:id` 路由而非 `*action` | 檢查模板是否有狀態切換鏈接 |
| 5 | 忘記傳遞 `list`/`mod` 標誌 | Render 時必須傳 `gin.H{"list": true}` |
| 6 | 狀態切換鏈接缺少 `class="switch"` | 必須加此 class |
| 7 | 嵌套 md5.Sum 導致編譯失敗 | 兩行寫法 |
| 8 | 直接渲染 time.Time | 加 `gorm:"-"` 字串欄位預格式化 |
| 9 | 硬編碼通知消息 | 引用 notice.go 常量 |
| 10 | 用簡體中文寫代碼/模板 | 全部繁體化 |
| 11 | backurl 未 URL-encode | 用 `url.QueryEscape` |
| 12 | backurl 開放重定向 | 用 `isSafeRedirectURL` 驗證相對路徑 |
| 13 | pongo2 `{foreach $value->Field(key,value)}` | 不支援 `->`，在 controller 預計算為字串 |
| 14 | ExtField 同模型 field 名稱重複 | 用 `CheckFieldUnique` 檢查 |
| 15 | 用 `c.Query()` 在 IndexCatchAll 後讀參數 | 改用 `c.Request.URL.Query().Get()` |
| 16 | 前台缺少權限檢查 | 必須呼叫 `checkSortPermission` / `checkContentPermission` |
| 17 | POST body 中用 `c.Query` 讀 backurl | 用 `c.DefaultPostForm` 讀取 |
| 18 | Layui `form.on('submit')` 攔截非 `lay-submit` 按鈕 | 改用 jQuery `$(document).on('submit')` |
| 19 | 用戶輸入未過濾直接存入 DB | 必須經過 `common.FilterUserInput()`（XSS 防護） |
| 20 | 動態 SQL 表名/欄位名未驗證 | 必須用 `CheckVarType()` / `CheckColumnName()` 白名單驗證 |
| 21 | 會員登入未重新生成 Session ID | 必須呼叫 `common.RegenerateSessionID(c)` |
| 22 | AJAX 攔截回應只有 `msg` 缺少 `data` | 必須同時包含 `data` + `msg` |
| 23 | 刪除操作用 GET 請求 | 刪除操作必須用 POST（防 CSRF） |
| 24 | 通知文案帶感嘆號 | 通知文案**不加感嘆號** |

---

## 已實現功能

### 後台管理

- 管理員登入/登出（密碼 bcrypt + 雙 MD5 向後相容 + 驗證碼 + 登入鎖定）
- 安全防護：XSS 過濾（FilterUserInput）、SQL 注入防護（標識符白名單）、CSRF token、Session TTL、Session Fixation 防護
- 內容管理：文章/產品增刪改查、批量排序、擴展字段、ext_ 前綴自定義字段
- 欄目管理：樹形結構、自定義 URL、模板選擇
- 單頁管理、站點信息、公司信息
- 幻燈片、友情鏈接、自定義標籤、內鏈標籤
- 內容模型、擴展字段、自定義表單（含 Turnstile 人機驗證）
- 媒體庫（圖片管理）、留言管理
- 系統配置（70+ 配置項）、菜單管理、角色權限
- 用戶管理、數據庫備份、系統日誌、區域管理
- 會員管理：會員列表/等級/欄位/評論（含批量操作、Excel 匯出）

### 前台展示

- 首頁（輪播圖、導航、產品展示）
- 列表頁（分頁、ext_ 篩選、子分類導航）
- 內容詳情頁（擴展字段、麵包屑、上一篇/下一篇、訪問量統計）
- 搜索頁、標籤頁、留言頁、單頁
- 模板熱重載（fsnotify）、.html 偽靜態 URL
- 會員系統：登入/註冊/登出/個人中心/資料修改/密碼找回
- 評論系統：提交/列表/我的評論/刪除
- Cloudflare Turnstile 人機驗證整合

---

## 開發環境

- Go 1.25+
- GOPROXY：`https://goproxy.cn,direct`
- 無需 CGO（純 Go SQLite 驅動）
- PowerShell 7（構建腳本）

---

## Git 工作流

### 克隆與首次配置

```powershell
# 1. 克隆倉庫
git clone https://github.com/vikim540/Gbootcms.git
cd Gbootcms

# 2. 設定你的提交者身份（項目級，不影響全域配置）
git config --local user.name "你的名字"
git config --local user.email "你的郵箱"

# 3. 安裝依賴
go mod download

# 4. 編譯運行
go build -o bin/gbootcms.exe .
.\bin\gbootcms.exe
```

### 認證方式

倉庫使用 HTTPS 協議，推送時需要 GitHub 認證：

- **Personal Access Token（推薦）**：在 GitHub Settings → Developer settings → Personal access tokens 建立令牌，推送時用令牌代替密碼
- **GitHub CLI**：`gh auth login` 按提示完成認證
- **Credential Manager**：Windows 上 Git 會自動使用 Git Credential Manager 快取憑證

### 日常提交與推送

```powershell
# 查看變更
git status
git diff

# 暫存所有改動（排除 runtime 產物）
git add -A

# 提交（建議使用 Conventional Commits 格式）
git commit -m "fix: 簡述修復內容"
git commit -m "feat: 簡述新增功能"
git commit -m "refactor: 簡述重構內容"

# 推送
git push origin main
```

### 代理注意事項

如果你使用 VPN 或代理訪問 GitHub，需要配置 git 代理：

```powershell
# 設定代理（替換為你的代理地址和端口）
git config --global http.proxy http://127.0.0.1:你的端口
git config --global https.proxy http://127.0.0.1:你的端口

# 移除代理（不用代理時必須移除，否則推送會失敗）
git config --global --unset http.proxy
git config --global --unset https.proxy
```

**新人接手注意**：如果克隆後 `git push` 報連接超時或拒絕連接，先檢查是否有遺留的代理配置：
```powershell
git config --global --get http.proxy   # 查看當前代理
```

### 換行符規則

項目使用 `.gitattributes` 統一換行符為 **LF**。Windows 用戶無需手動處理，git 會自動轉換：

- 入庫時：CRLF → LF（自動）
- 檢出時：保持 LF（`core.autocrlf=input`）

如果克隆後發現所有文件都被標記為「已修改」，執行：
```powershell
git add --renormalize .
git commit -m "chore: normalize line endings"
```

### 已入庫的特殊文件

| 文件 | 說明 |
|------|------|
| `data/pbootcms.db` | 開發用 SQLite 數據庫（預設帳號 admin/123456），方便新人克隆後直接運行 |
| `config/config.json` | 配置文件（端口、DB 路徑等），入庫方便快速啟動 |
| `static/backup/` | 備份目錄，**已加入 .gitignore**，不應入庫 |

### 分支策略

- `main`：穩定分支，所有提交直接推送到 `main`
- 功能開發時可臨時建立 `feature/xxx` 分支，完成後合併回 `main`

---

## 文檔索引

| 文檔 | 路徑 | 說明 |
|------|------|------|
| 開發技術文檔 | `docs/pbootcms-go-dev-guide.md` | 完整技術文檔（含防遺忘清單、API 速查、反模式、會員系統、ExtField 系統） |
| AI 開發指南 | `docs/AI_GUIDELINES.md` | AI 輔助開發約定（硬約束、反模式、狀態切換速查） |
| README | `README.md` | 項目簡介、快速開始、功能清單 |
