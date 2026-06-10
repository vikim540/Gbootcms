# ContentSort CRUD 全面修復設計

## 問題清單

### P0 - 嚴重 (功能不可用)
1. **表單 action scode 丟失**: `[$get.scode]` → `{{ get_scode }}`,但 `get_scode` 未注入(路徑參數)
2. **模板變量大小寫不匹配**: `flattenData` 把 `formcheck`→`Formcheck`,模板用 `{{ formcheck }}`
3. **模板下拉框字面量**: `[value]` 未轉譯為 `{{ val1 }}`
4. **條件判斷失效**: `{% if value!=Sort.Listtpl %}` 中 `value` 不是 pongo2 變量

### P1 - 中等 (功能異常)
5. **DeleteSortByScode 缺少 id fallback**: scode 為空時刪除靜默失敗

## 修復方案

### 1. 移除 flattenData PascalCase 轉換

**文件**: `apps/common/Render.go`

將 `flattenData` 改為**保留原始 key**,不做大小寫轉換:
```go
// Before: result[SnakeToPascal(k)] = v
// After:  result[k] = v  (保持原始 key)
```

同時修改 transpiler 中所有引用 PascalCase 的地方,確保生成的 pongo2 變量使用 snake_case。

### 2. 注入路徑參數到模板變量

**文件**: `apps/common/Render.go`

在 `Render` 函數中,從 `X-Original-Path` 或 `c.Request.URL.Path` 解析路徑段,
注入 `get_scode`、`get_id` 等變量(模擬 PbootCMS 的 `$_GET` 行為):
```go
// 從 /admin/contentsort/mod/scode/1 解析出 scode=1
pathParams := parsePathParams(pageURL)
for k, v := range pathParams {
    data["get_"+k] = v
}
```

### 3. transpiler 修復 `[value]` 轉譯

**文件**: `core/basic/view.go`

在 `convertPbootToPongo2` 中添加處理:
- `{foreach $list(key,value)}` 循環內的 `[value]` → `{{ val1 }}`
- `{foreach $list(key,value)}` 循環內的 `[key]` → `{{ key1 }}`
- `{foreach $list(key,value)}` 中的 `value` 變量 → `val1`(用於條件判斷)

### 4. DeleteSortByScode 添加 id fallback

**文件**: `apps/admin/service/content/ContentSortService.go`

與 `GetSortByScode` 和 `UpdateSortByScode` 保持一致。

## 驗證清單

- [ ] 後台列表頁:按鈕 URL 正確 (`/mod/scode/1`)
- [ ] 修改頁面:表單 action 包含 scode (`/mod/scode/1`)
- [ ] 修改保存:POST 後數據持久化
- [ ] 模板下拉框:顯示實際文件名而非 `[value]`
- [ ] 切換狀態:正常工作
- [ ] 前台 URL:有 urlname 時用 `/{urlname}.html`,無時用 `/sort/{scode}`
