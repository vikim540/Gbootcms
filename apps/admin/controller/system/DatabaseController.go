package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"
	"gbootcms/config"

	"github.com/gin-gonic/gin"
)

// DatabaseController 資料庫管理控制器
// 對應PHP: apps/admin/controller/DatabaseController.php
type DatabaseController struct {
	common.BaseController
}

// backupFileInfo 備份檔案資訊
type backupFileInfo struct {
	Name string
	Size string
	Time string
	Path string
}

// Index 資料庫管理頁面
func (db *DatabaseController) Index(c *gin.Context) {
	cfg := config.Get()
	dbType := cfg.Database.Type
	var tables []gin.H

	if dbType == "sqlite" {
		var tableNames []string
		model.DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&tableNames)
		for _, t := range tableNames {
			var rowCount int64
			model.DB.Raw("SELECT COUNT(*) FROM \"" + t + "\"").Scan(&rowCount)
			tables = append(tables, gin.H{"Name": t, "Rows": rowCount})
		}
	} else {
		var tableNames []string
		var dbName string
		model.DB.Raw("SELECT DATABASE()").Scan(&dbName)
		model.DB.Raw("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME", dbName).Scan(&tableNames)
		for _, t := range tableNames {
			var rowCount int64
			model.DB.Raw("SELECT COUNT(*) FROM `" + t + "`").Scan(&rowCount)
			tables = append(tables, gin.H{"Name": t, "Rows": rowCount})
		}
	}

	// 取得備份檔案列表
	backups := db.listBackups()

	// 取得當前資料庫檔案大小（SQLite）
	dbSize := ""
	if dbType == "sqlite" {
		if info, err := os.Stat(cfg.Database.DBName); err == nil {
			dbSize = formatFileSize(info.Size())
		}
	}

	common.Render(c, "system/database.html", gin.H{
		"tables":  tables,
		"db":      dbType,
		"backups": backups,
		"db_size": dbSize,
	})
}

// listBackups 掃描備份目錄，返回已排序的備份檔案列表
func (db *DatabaseController) listBackups() []backupFileInfo {
	backupDir := filepath.Join("static", "backup", "sql")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil
	}
	var backups []backupFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 只列出 .db 和 .sql 備份檔案
		if !strings.HasSuffix(name, ".db") && !strings.HasSuffix(name, ".sql") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backupFileInfo{
			Name: name,
			Size: formatFileSize(info.Size()),
			Time: info.ModTime().Format("2006-01-02 15:04:05"),
			Path: name,
		})
	}
	// 按修改時間降序排列（最新的在最前面）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Time > backups[j].Time
	})
	return backups
}

// formatFileSize 格式化檔案大小為人類可讀格式
func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
}

// Mod 資料庫操作（優化/修復/備份）
func (db *DatabaseController) Mod(c *gin.Context) {
	action := c.PostForm("action")
	switch action {
	case "yh":
		db.optimizeTables(c)
	case "xf":
		db.repairTables(c)
	case "bf":
		db.backupTableAction(c)
	case "bfdb":
		db.backupDBAction(c)
	case "bfsqlite":
		db.backupSQLiteAction(c)
	case "delbackup":
		db.deleteBackup(c)
	default:
		db.JSONFail(c, "未知操作！")
	}
}

// DownloadBackup 下載備份檔案
func (db *DatabaseController) DownloadBackup(c *gin.Context) {
	filename := c.Query("file")
	// 安全驗證：只允許檔案名，不允許路徑分隔符
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		db.JSONFail(c, "非法檔案名！")
		return
	}
	if !strings.HasSuffix(filename, ".db") && !strings.HasSuffix(filename, ".sql") {
		db.JSONFail(c, "不支援的檔案類型！")
		return
	}
	filePath := filepath.Join("static", "backup", "sql", filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		db.JSONFail(c, "檔案不存在！")
		return
	}
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.File(filePath)
}

// deleteBackup 刪除備份檔案
func (db *DatabaseController) deleteBackup(c *gin.Context) {
	filename := c.PostForm("file")
	// 安全驗證
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		db.JSONFail(c, "非法檔案名！")
		return
	}
	if !strings.HasSuffix(filename, ".db") && !strings.HasSuffix(filename, ".sql") {
		db.JSONFail(c, "不支援的檔案類型！")
		return
	}
	filePath := filepath.Join("static", "backup", "sql", filename)
	if err := os.Remove(filePath); err != nil {
		db.JSONFail(c, "刪除失敗："+err.Error())
		return
	}
	db.JSONOKMsg(c, "刪除成功")
}

func (db *DatabaseController) optimizeTables(c *gin.Context) {
	tables, err := db.getTableList(c)
	if err != nil {
		db.JSONFail(c, err.Error())
		return
	}
	cfg := config.Get()

	if cfg.Database.Type == "sqlite" {
		model.DB.Exec("VACUUM")
	} else {
		for _, t := range tables {
			model.DB.Exec("OPTIMIZE TABLE " + t)
		}
	}
	db.JSONOKMsg(c, common.NoticeOptimize)
}

func (db *DatabaseController) repairTables(c *gin.Context) {
	tables, err := db.getTableList(c)
	if err != nil {
		db.JSONFail(c, err.Error())
		return
	}
	cfg := config.Get()

	if cfg.Database.Type == "sqlite" {
		db.JSONFail(c, "SQLite 不支援修復表操作！")
		return
	}

	for _, t := range tables {
		model.DB.Exec("REPAIR TABLE " + t)
	}
	db.JSONOKMsg(c, common.NoticeRepair)
}

func (db *DatabaseController) backupTableAction(c *gin.Context) {
	tables, err := db.getTableList(c)
	if err != nil {
		db.JSONFail(c, err.Error())
		return
	}

	backupDir := filepath.Join("static", "backup", "sql")
	os.MkdirAll(backupDir, 0755)

	sqlContent := db.header()
	for _, t := range tables {
		sqlContent += db.tableSql(t)
		sqlContent += db.dataSql(t)
	}

	filename := fmt.Sprintf("pbootcms_%s_%s.sql", strings.Join(tables, "_"), time.Now().Format("20060102150405"))
	filePath := filepath.Join(backupDir, filename)
	db.writeFile(filePath, sqlContent)
	db.JSONOKMsg(c, common.NoticeBackup)
}

func (db *DatabaseController) backupDBAction(c *gin.Context) {
	cfg := config.Get()
	backupDir := filepath.Join("static", "backup", "sql")
	os.MkdirAll(backupDir, 0755)

	var tables []string
	if cfg.Database.Type == "sqlite" {
		model.DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&tables)
	} else {
		var dbName string
		model.DB.Raw("SELECT DATABASE()").Scan(&dbName)
		model.DB.Raw("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME", dbName).Scan(&tables)
	}

	sqlContent := db.header()
	for _, t := range tables {
		sqlContent += db.tableSql(t)
		sqlContent += db.dataSql(t)
	}

	filename := fmt.Sprintf("pbootcms_db_%s.sql", time.Now().Format("20060102150405"))
	filePath := filepath.Join(backupDir, filename)
	db.writeFile(filePath, sqlContent)
	db.JSONOKMsg(c, common.NoticeBackup)
}

func (db *DatabaseController) backupSQLiteAction(c *gin.Context) {
	cfg := config.Get()
	if cfg.Database.Type != "sqlite" {
		db.JSONFail(c, "僅 SQLite 支援此操作！")
		return
	}

	backupDir := filepath.Join("static", "backup", "sql")
	os.MkdirAll(backupDir, 0755)

	srcPath := cfg.Database.DBName
	srcFile, err := os.Open(srcPath)
	if err != nil {
		db.JSONFail(c, "開啟資料庫檔案失敗！")
		return
	}
	defer srcFile.Close()

	filename := fmt.Sprintf("pbootcms_%s.db", time.Now().Format("20060102150405"))
	dstPath := filepath.Join(backupDir, filename)

	dstFile, err := os.Create(dstPath)
	if err != nil {
		db.JSONFail(c, "建立備份檔案失敗！")
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		db.JSONFail(c, "複製資料庫檔案失敗！")
		return
	}
	db.JSONOKMsg(c, common.NoticeBackup)
}

func (db *DatabaseController) header() string {
	var sb strings.Builder
	sb.WriteString("-- Gbootcms 資料庫備份檔案\n")
	sb.WriteString(fmt.Sprintf("-- 產生時間: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("-- -------------------------------------------\n\n")
	sb.WriteString("SET FOREIGN_KEY_CHECKS=0;\n\n")
	return sb.String()
}

func (db *DatabaseController) tableSql(tableName string) string {
	cfg := config.Get()
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("-- ----------------------------\n"))
	sb.WriteString(fmt.Sprintf("-- 表結構 `%s`\n", tableName))
	sb.WriteString(fmt.Sprintf("-- ----------------------------\n"))

	var createSQL string
	if cfg.Database.Type == "sqlite" {
		model.DB.Raw("SELECT sql FROM sqlite_master WHERE name=? AND type='table'", tableName).Scan(&createSQL)
		if createSQL != "" {
			sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n", tableName))
			sb.WriteString(createSQL + ";\n\n")
		}
	} else {
		rows, err := model.DB.Raw("SHOW CREATE TABLE `" + tableName + "`").Rows()
		if err == nil {
			cols, _ := rows.Columns()
			vals := make([][]byte, len(cols))
			ptrs := make([]interface{}, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if rows.Next() {
				rows.Scan(ptrs...)
				if len(vals) > 1 {
					createSQL = string(vals[1])
				}
			}
			rows.Close()
		}
		if createSQL != "" {
			sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS `%s`;\n", tableName))
			sb.WriteString(createSQL + ";\n\n")
		}
	}

	return sb.String()
}

func (db *DatabaseController) dataSql(tableName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("-- ----------------------------\n"))
	sb.WriteString(fmt.Sprintf("-- 表資料 `%s`\n", tableName))
	sb.WriteString(fmt.Sprintf("-- ----------------------------\n"))

	rows, err := model.DB.Raw("SELECT * FROM \"" + tableName + "\"").Rows()
	if err != nil {
		return sb.String()
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return sb.String()
	}

	colCount := len(columns)
	values := make([]interface{}, colCount)
	valuePtrs := make([]interface{}, colCount)
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	colNames := make([]string, colCount)
	for i, col := range columns {
		colNames[i] = fmt.Sprintf("`%s`", col)
	}
	colList := strings.Join(colNames, ", ")

	for rows.Next() {
		rows.Scan(valuePtrs...)

		valStrs := make([]string, colCount)
		for i, val := range values {
			switch v := val.(type) {
			case nil:
				valStrs[i] = "NULL"
			case []byte:
				escaped := strings.ReplaceAll(string(v), "'", "''")
				escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
				valStrs[i] = fmt.Sprintf("'%s'", escaped)
			case string:
				escaped := strings.ReplaceAll(v, "'", "''")
				escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
				valStrs[i] = fmt.Sprintf("'%s'", escaped)
			default:
				valStrs[i] = fmt.Sprintf("%v", v)
			}
		}

		sb.WriteString(fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s);\n",
			tableName, colList, strings.Join(valStrs, ", ")))
	}
	sb.WriteString("\n")

	return sb.String()
}

func (db *DatabaseController) writeFile(filePath string, content string) {
	os.WriteFile(filePath, []byte(content), 0644)
}

func (db *DatabaseController) getTableList(c *gin.Context) ([]string, error) {
	tableList := c.PostFormArray("list[]")
	if len(tableList) == 0 {
		tableList = strings.Split(c.PostForm("list"), ",")
	}

	if len(tableList) == 0 {
		return nil, fmt.Errorf("請選擇要操作的表！")
	}

	validPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	var validTables []string
	for _, t := range tableList {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !validPattern.MatchString(t) {
			return nil, fmt.Errorf("表名包含非法字元！%s", t)
		}
		validTables = append(validTables, t)
	}

	if len(validTables) == 0 {
		return nil, fmt.Errorf("請選擇要操作的表！")
	}
	return validTables, nil
}

// RestoreSQLite 從備份檔案恢復 SQLite 資料庫
func (db *DatabaseController) RestoreSQLite(c *gin.Context) {
	filename := c.PostForm("file")
	if filename == "" {
		db.JSONFail(c, "請選擇備份檔案！")
		return
	}
	// 安全驗證
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..") {
		db.JSONFail(c, "非法檔案名！")
		return
	}
	if !strings.HasSuffix(filename, ".db") {
		db.JSONFail(c, "只能從 .db 備份檔案恢復！")
		return
	}

	cfg := config.Get()
	if cfg.Database.Type != "sqlite" {
		db.JSONFail(c, "僅 SQLite 支援此操作！")
		return
	}

	backupPath := filepath.Join("static", "backup", "sql", filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		db.JSONFail(c, "備份檔案不存在！")
		return
	}

	// 先備份當前資料庫（安全網）
	currentBackup := cfg.Database.DBName + ".before_restore"
	os.Rename(cfg.Database.DBName, currentBackup)

	// 複製備份檔案到當前資料庫路徑
	srcFile, err := os.Open(backupPath)
	if err != nil {
		// 恢復失敗，還原
		os.Rename(currentBackup, cfg.Database.DBName)
		db.JSONFail(c, "開啟備份檔案失敗！")
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(cfg.Database.DBName)
	if err != nil {
		os.Rename(currentBackup, cfg.Database.DBName)
		db.JSONFail(c, "建立資料庫檔案失敗！")
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		os.Rename(currentBackup, cfg.Database.DBName)
		db.JSONFail(c, "恢復失敗：" + err.Error())
		return
	}

	// 成功後刪除臨時備份
	os.Remove(currentBackup)

	db.JSONOKMsg(c, "資料庫恢復成功，請重新啟動服務以生效")
}
