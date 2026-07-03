package common

import "strconv"

// notice.go — 統一通知提示文案
// 所有 controller 的 JSONOKMsg/JSONFail 應引用此處常量，禁止硬編碼字符串。
// 帶變量的消息用函數，固定消息用常量。

// ─── 通用操作 ───
const (
	NoticeModify    = "修改成功"
	NoticeAdd       = "新增成功"
	NoticeDelete    = "刪除成功"
	NoticeSave      = "保存成功"
	NoticeNoChange  = "無變更"
	NoticeOperation = "操作成功"
)

// ─── 批量操作 ───
const (
	NoticeBatchAdd    = "批量新增成功"
	NoticeBatchDelete = "批量刪除成功"
)

// ─── 內容操作 ───
const (
	NoticeCopy    = "複製成功"
	NoticeMove    = "移動成功"
	NoticeSubmit  = "提交成功"
	NoticeReply   = "回覆成功"
	NoticePublish = "發布成功"
	NoticeOffline = "下線成功"
)

// ─── 系統操作 ───
const (
	NoticeClean        = "清理成功"
	NoticeCleanAll     = "全部清理成功"
	NoticeCacheCleaned = "緩存清理成功"
	NoticeSwitch       = "切換成功"
	NoticePassword     = "密碼修改成功"
	NoticeOptimize     = "優化成功"
	NoticeRepair       = "修復成功"
	NoticeBackup       = "備份成功"
)

// ─── 驗證碼 ───
const (
	NoticeCaptchaEmpty = "驗證碼不能為空"
	NoticeCaptchaError = "驗證碼錯誤"
)

// ─── 通用錯誤 ───
const (
	NoticeFailAdd     = "新增失敗"
	NoticeFailModify  = "修改失敗"
	NoticeFailDelete  = "刪除失敗"
	NoticeFailSave    = "保存失敗"
	NoticeFailSubmit  = "提交失敗"
	NoticeFailUpload  = "上傳失敗"
	NoticeFailRequest = "請求發生錯誤"
	NoticeParamEmpty  = "參數不能為空"
	NoticeNoSelect    = "未選擇任何項目"
	NoticeNoExist     = "數據不存在"
)

// ─── 緩存更新 ───
const (
	NoticeCacheHomepage = "首頁及欄目已更新"
	NoticeCacheSortList = "全部欄目列表已更新"
	NoticeCacheContent  = "內容已更新"
)

// ─── 郵件測試 ───
const (
	NoticeMailTestEmpty     = "請先填寫測試收件郵箱"
	NoticeMailTestBadFormat = "郵箱格式不正確"
	NoticeMailTestNoSMTP    = "SMTP 配置不完整，請先填寫伺服器地址和發件帳號"
	NoticeMailTestFail      = "郵件發送失敗，請檢查 SMTP 配置後重試"
	NoticeMailTestRequest   = "請求異常，請檢查網絡或重新登入"
)

// NoticeMailTestSent 測試郵件已發送
func NoticeMailTestSent(to string) string {
	return "測試郵件已發送至 " + to + "，請查收"
}

// ─── 表單系統 ───
const (
	NoticeFormAdd      = "表單新增成功"
	NoticeFormModify   = "表單修改成功"
	NoticeFormDelete   = "表單刪除成功"
	NoticeFormDataClear = "表單數據清理成功"
	NoticeFieldAdd     = "字段新增成功"
	NoticeFieldModify  = "字段修改成功"
	NoticeFieldDelete  = "字段刪除成功"
	NoticeMenuAdd      = "菜單新增成功"
	NoticeDataDelete   = "數據刪除成功"
)

// ─── 帶變量的消息（函數） ───

// NoticeSortSaved 排序已保存
func NoticeSortSaved(updated int) string {
	return "排序已保存 (" + strconv.Itoa(updated) + " 條)"
}

// NoticeSortSavedPartial 排序已保存（部分無變化）
func NoticeSortSavedPartial(updated, unchanged int) string {
	return "排序已保存 (" + strconv.Itoa(updated) + " 條，" + strconv.Itoa(unchanged) + " 條無變化)"
}

// NoticeSortNoChange 排序未變化
func NoticeSortNoChange(unchanged int) string {
	return "排序未變化 (" + strconv.Itoa(unchanged) + " 條)"
}

// NoticeLabelSaved 標籤已保存
func NoticeLabelSaved(updated int) string {
	return "標籤已保存 (" + strconv.Itoa(updated) + " 個)"
}
