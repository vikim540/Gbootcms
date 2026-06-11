# Tailwind 可視化編輯器集成計劃

> **備份已保存**: commit `99c6bc7` — single.html 功能全面恢復 + UEditor 靜態資源補全

## 目標

用 **GrapesJS + grapesjs-tailwind** 替換 UEditor,實現:
- 文案人員不懂代碼也能通過點擊/拖拽創建帶 Tailwind 樣式的內容
- 前端渲染時**不加載 Tailwind CSS 文件**,而是將 class 轉為 inline style
- 減少前端 HTTP 請求,頁面加載更快

## 技術方案

### 編輯器: GrapesJS + grapesjs-tailwind

- **GrapesJS**: 專為 CMS 設計的開源頁面構建器(145K/週下載)
- **grapesjs-tailwind**: 集成 Tailblocks.cc 全套預設組件(hero/features/CTA/forms 等)
- 通過 CDN 引入,無需構建步驟
- 內置拖拽、撤銷/重做、響應式預覽

### 保存: HTML + Tailwind class → 數據庫

```
用戶編輯 → GrapesJS 導出 HTML (含 class="flex p-4 bg-blue-500")
         → Go 後端安全過濾(白標籤/屬性)
         → 存入 ay_content.content
```

### 前端渲染: class → inline style

```
讀取 HTML → 正則提取所有 class 屬性
          → tw-to-css 將 Tailwind class 轉為 style 屬性
          → 輸出: <div style="display:flex;padding:1rem;background:#3b82f6">
          → 前端無需加載任何 CSS 文件
```

## 文件結構

```
pbootcms-go/
├── static/admin/extend/grapesjs/          # GrapesJS + Tailwind 插件 (CDN 或本地)
├── templates/admin/content/single.html    # 替換 UEditor 為 GrapesJS
├── apps/admin/controller/content/
│   └── SingleController.go               # 保存邏輯(安全過濾)
├── apps/common/
│   └── tailwind_converter.go              # Tailwind class → inline style 轉換器
└── templates/about.html                   # 前端模板(無需 Tailwind)
```

## 實施步驟

### Phase 1: GrapesJS 編輯器集成
1. 在 single.html 中用 GrapesJS 替換 UEditor textarea
2. 引入 grapesjs + grapesjs-tailwind (CDN)
3. 配置 GrapesJS: 存儲管理關閉(由表單提交處理)、白標籤/屬性

### Phase 2: 保存安全過濾
1. Go 後端 HTML 白名單過濾(允許 div/p/span/h1-h6/a/img/ul/ol 等)
2. class 屬性正則驗證(Tailwind 類名格式)
3. 移除 onclick/onerror/javascript: 等危險屬性

### Phase 3: Tailwind → Inline Style 轉換
1. 實現 Go 版本的 Tailwind-to-inline 轉換器
2. 解析 HTML,提取所有 class 屬性
3. 查表將 Tailwind class 轉為 CSS inline style
4. 輸出純 HTML + inline style(無需外部 CSS)

### Phase 4: 前端渲染
1. `getContentField()` 在返回 content 時自動轉換
2. 或在模板引擎層面做轉換
3. 確保 `{content:content}` 輸出帶 inline style 的 HTML

## 風險與注意

- grapesjs-tailwind 標記為 WIP,需要測試穩定性
- Tailwind class → inline style 需要完整的 class-to-CSS 映射表
- 部分 Tailwind 功能(如響應式前綴 `md:`,hover:)無法完全轉為 inline style
- 首次加載 GrapesJS 稍重(~200KB gzipped)

## 已知限制

- `md:flex`、`hover:bg-blue-600` 等響應式/狀態類無法轉為 inline style
- 這些類會在轉換時被忽略或保留為 class 屬性
- 前端可能仍需少量 CSS 來處理這些情況

---

**下一步**: 確認方案後開始 Phase 1 實施
