# 單模板多語言架構 (Multilingual & i18n)

傳統 CMS 往往透過將整個 `template/` 目錄複製多份（如 `template/en/`、`template/sc/`）來支援多語言，導致修改一個樣式或標籤需要手動同步數十個 HTML 檔案。

Gbootcms 踐行**奧卡姆剃刀原則**，徹底刪除冗餘的物理克隆，統一收斂為**單套模板 + 動態字典架構**。

---

## 1. 目錄架構

```
template/default/
├── index.html               # 首頁模板（全語言共用）
├── list.html                # 列表頁模板
├── content.html             # 詳情頁模板
├── i18n/                    # 字典目錄（JSON 格式）
│   ├── en.json              # 英文語言字典
│   ├── sc.json              # 簡體中文語言字典
│   └── tc.json              # 繁體中文語言字典
└── comm/                    # 公共模組
    ├── head.html
    └── foot.html
```

---

## 2. 字典定義範例 (`template/default/i18n/en.json`)

```json
{
  "Home": "Home",
  "Search": "Search",
  "SearchPlaceholder": "Enter keyword...",
  "Prev": "Previous",
  "Next": "Next",
  "ReadMore": "Read More",
  "Submit": "Submit",
  "Author": "Author",
  "Date": "Date",
  "Hits": "Views"
}
```

---

## 3. 前台模板調用範例

在 HTML 模板中，只需書寫 `{i18n:Key}` 標籤：

```html
<nav class="navbar">
    <a href="/">{i18n:Home}</a>
</nav>

<form action="/search" method="get">
    <input type="text" name="keyword" placeholder="{i18n:SearchPlaceholder}">
    <button type="submit">{i18n:Search}</button>
</form>

<div class="pager">
    <span>{i18n:Prev}</span>
    <span>{i18n:Next}</span>
</div>
```

---

## 4. 路由與數據庫隔離

- **URL 前綴**：
  - 繁體（預設）：`https://domain.com/news/15.html`
  - 簡體：`https://domain.com/sc/news/15.html`
  - 英文：`https://domain.com/en/news/15.html`
- **GORM AcodePlugin**：
  - 後台和前台查詢透過 GORM 插件自動注入 `acode = ?` 條件，開發者在業務層寫 SQL 時無需手動拼接語言過濾。

---

## 5. 精確雙向跨語言關聯 (`rel_id`)

跨語言文章映射透過 `ay_content.rel_id` 明確綁定：
- 嚴禁根據欄目文章發布順序或位次硬連。
- 在輸出 `<link rel="alternate" hreflang="...">` 時，僅精準輸出確有對應關聯的文章語言版本，杜絕跨語言 404 與 SEO 懲罰。
