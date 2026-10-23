# 多語言與路徑劃分架構 (Multilingual Architecture)

Gbootcms 遵循 PbootCMS 原生架構思想，採用**路徑前綴劃分 + 語言獨立模板**的多語言體系，杜絕非標準的中間層自造標籤，保持 HTML 模板語句自然流暢與標準標籤生態相容。

---

## 1. 路由路徑劃分

前台路由依據 URL 前綴自動匹配與切換當前語言上下文（`acode`）：

| 語言區域 (`acode`) | URL 路徑前綴 | 說明 |
| :--- | :--- | :--- |
| **預設區域 (`tc`)** | `/` (如 `/website/74`) | 預設繁體中文站點 |
| **英文區域 (`en`)** | `/en/` (如 `/en/website/74`) | 英文站點 |
| **簡體中文 (`sc`)** | `/sc/` (如 `/sc/website/74`) | 簡體中文站點 |

系統內部由 `acodeplugin` 自動解析路徑前綴並注入 `Request Context`，GORM 查詢自動完成多語言數據過濾。

---

## 2. 模板目錄結構

各語言擁有專屬的模板目錄，直接以自然語言書寫靜態 UI，並使用標準 `{gboot:xxx}`、`{content:xxx}`、`{sort:xxx}` 標籤進行動態資料綁定：

```
template/
├── default/                 # 預設模板（繁體中文）
│   ├── index.html
│   ├── product.html
│   └── comm/
│       ├── head.html
│       └── foot.html
├── en/                      # 英文專屬模板
│   ├── index.html
│   ├── product.html
│   └── comm/
│       ├── head.html
│       └── foot.html
└── sc/                      # 簡體中文專屬模板
    ├── index.html
    ├── product.html
    └── comm/
        ├── head.html
        └── foot.html
```

> **靜態資源優化**：所有語言模板的公共 CSS/JS 與字型檔案由 `template/default/static/` 統一維護，並透過 `/template/{acode}/static` 自動映射，避免磁碟空間重複佔用。

---

## 3. 模板套用標籤範例

在各語言模板中，直接套用標準標籤與原生文案：

### 預設繁體模板 (`template/default/product.html`)
```html
<div class="text-secondary my-3 border-bottom-dashed lh-3">
    上架時間：{content:date style=Y-m-d}
</div>
<div class="text-secondary my-3 border-bottom-dashed lh-3">
    瀏覽次數：{content:visits}
</div>
<div class="text-secondary my-3 border-bottom-dashed lh-3">
    產品類型：{content:ext_type}
</div>
<p>上一篇：{content:precontent}</p>
<p>下一篇：{content:nextcontent}</p>
```

### 英文專屬模板 (`template/en/product.html`)
```html
<div class="text-secondary my-3 border-bottom-dashed lh-3">
    Date Posted: {content:date style=Y-m-d}
</div>
<div class="text-secondary my-3 border-bottom-dashed lh-3">
    Views: {content:visits}
</div>
<div class="text-secondary my-3 border-bottom-dashed lh-3">
    Product Type: {content:ext_type}
</div>
<p>Previous: {content:precontent}</p>
<p>Next: {content:nextcontent}</p>
```

---

## 4. 跨語言 SEO 雙向關聯

跨語言文章透過資料表中的 `rel_id` 欄位實現精準綁定：
- 渲染詳情頁時，`{gboot:hreflang}` 自動檢索具備相同 `rel_id` 的跨語言目標頁面，生成標準 `<link rel="alternate" hreflang="..." href="...">`。
- 無跨語言關聯時僅輸出當前頁面的自引用與 `x-default`，不進行位次猜測，確保搜尋引擎索引完全正確。
