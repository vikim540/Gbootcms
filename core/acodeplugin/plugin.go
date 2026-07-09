// Package acodeplugin 提供 GORM 自動區域隔離插件。
//
// 設計目標：
//   - 一次註冊，所有含 acode 欄位的表自動隔離，無需開發者手動加 WHERE
//   - 與 mediaplugin 同一模式（GORM Plugin），零架構增量
//   - 通過 context 傳遞 acode，請求級隔離，線程安全
//   - 提供 SkipAcode() 明確跳過隔離（如區域管理本身需跨區查詢）
//
// 使用方法：
//
//	// 1. 初始化時註冊（core/db/db.go）
//	db.Use(&acodeplugin.AcodePlugin{})
//
//	// 2. 中間件注入 acode 到 context
//	ctx := acodeplugin.WithAcode(c.Request.Context(), "cn")
//	c.Request = c.Request.WithContext(ctx)
//
//	// 3. 控制器中使用 .WithContext()
//	model.DB.WithContext(c.Request.Context()).Find(&links)
//	// → SELECT * FROM ay_link WHERE acode = 'cn'
//
//	// 4. 跨區查詢（區域管理等）
//	model.DB.WithContext(acodeplugin.SkipAcode(c.Request.Context())).Find(&areas)
//	// → SELECT * FROM ay_area（無 acode 過濾）
package acodeplugin

import (
	"context"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// ──────────────────────────────────────────────
// Context 輔助函數
// ──────────────────────────────────────────────

// ctxKey 私有類型，避免 context key 衝突
type ctxKey struct{}

// skipKey 標記跳過 acode 隔離
type skipKey struct{}

// WithAcode 將 acode 注入 context
func WithAcode(ctx context.Context, acode string) context.Context {
	return context.WithValue(ctx, ctxKey{}, acode)
}

// GetAcode 從 context 提取 acode，未設置時返回空字串
func GetAcode(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// SkipAcode 標記此 context 跳過 acode 隔離
// 用於區域管理、系統配置等需要跨區查詢的場景
func SkipAcode(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipKey{}, true)
}

// isSkipAcode 判斷 context 是否標記為跳過隔離
func isSkipAcode(ctx context.Context) bool {
	v, _ := ctx.Value(skipKey{}).(bool)
	return v
}

// ──────────────────────────────────────────────
// GORM Plugin 實現
// ──────────────────────────────────────────────

// AcodePlugin 是 GORM 多區域自動隔離插件
//
// 工作原理：
//   - Query/Update/Delete：檢查 Schema 是否有 acode 欄位，有則自動加 WHERE acode = ?
//   - Create：檢查 Schema 是否有 acode 欄位，有則自動填充（僅當值為空時）
//   - 通過 context 傳遞 acode，無 context 時不過濾（向後兼容）
type AcodePlugin struct{}

// Name 插件名稱（GORM Plugin 接口要求）
func (p *AcodePlugin) Name() string {
	return "AcodePlugin"
}

// Initialize 在 GORM 初始化時註冊回調
func (p *AcodePlugin) Initialize(db *gorm.DB) error {
	// 查詢：自動加 WHERE acode = ?
	if err := db.Callback().Query().Before("gorm:query").
		Register("acode:query_filter", p.injectWhere); err != nil {
		return err
	}
	// 新增：自動填充 acode
	if err := db.Callback().Create().Before("gorm:before_create").
		Register("acode:fill_create", p.fillAcode); err != nil {
		return err
	}
	// 更新：自動加 WHERE acode = ?（防止跨區修改）
	if err := db.Callback().Update().Before("gorm:before_update").
		Register("acode:update_filter", p.injectWhere); err != nil {
		return err
	}
	// 刪除：自動加 WHERE acode = ?（防止跨區刪除）
	if err := db.Callback().Delete().Before("gorm:before_delete").
		Register("acode:delete_filter", p.injectWhere); err != nil {
		return err
	}
	return nil
}

// injectWhere 在查詢/更新/刪除前自動注入 WHERE acode = ?
func (p *AcodePlugin) injectWhere(db *gorm.DB) {
	if db.Statement == nil || db.Statement.Schema == nil {
		return
	}

	// 檢查此表是否有 acode 欄位，無則跳過
	field, hasAcode := db.Statement.Schema.FieldsByDBName["acode"]
	if !hasAcode {
		return
	}
	_ = field

	ctx := db.Statement.Context
	if ctx == nil {
		return
	}

	// 跳過模式（區域管理等跨區查詢）
	if isSkipAcode(ctx) {
		return
	}

	acode := GetAcode(ctx)
	if acode == "" {
		// 無 acode 時不過濾（向後兼容，未遷移的代碼不受影響）
		return
	}

	// 注入 WHERE acode = ?
	// 使用 AddClause 合併現有 WHERE 條件（AND 語義）
	db.Statement.AddClause(clause.Where{
		Exprs: []clause.Expression{
			clause.Eq{
				Column: clause.Column{Table: db.Statement.Table, Name: "acode"},
				Value:  acode,
			},
		},
	})
}

// fillAcode 在新增前自動填充 acode（僅當值為空時）
func (p *AcodePlugin) fillAcode(db *gorm.DB) {
	if db.Statement == nil || db.Statement.Schema == nil {
		return
	}

	// 檢查此表是否有 acode 欄位
	field, hasAcode := db.Statement.Schema.FieldsByDBName["acode"]
	if !hasAcode {
		return
	}

	ctx := db.Statement.Context
	if ctx == nil {
		return
	}

	if isSkipAcode(ctx) {
		return
	}

	acode := GetAcode(ctx)
	if acode == "" {
		return
	}

	// 處理單條記錄和批量記錄
	rv := db.Statement.ReflectValue
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			fillAcodeIfEmpty(field, ctx, rv.Index(i), acode)
		}
	case reflect.Struct:
		fillAcodeIfEmpty(field, ctx, rv, acode)
	}
}

// fillAcodeIfEmpty 當 acode 欄位為空時填充值
func fillAcodeIfEmpty(field *schema.Field, ctx context.Context, rv reflect.Value, acode string) {
	val, isZero := field.ValueOf(ctx, rv)
	if isZero || val == "" || val == nil {
		_ = field.Set(ctx, rv, acode)
	}
}
