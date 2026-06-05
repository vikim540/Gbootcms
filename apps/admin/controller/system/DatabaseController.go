package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"
	"pbootcms-go/config"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// DatabaseController 数据库管理控制器
// 对应PHP: apps/admin/controller/DatabaseController.php
type DatabaseController struct {
	common.BaseController
}

// Index 数据库管理页面
func (db *DatabaseController) Index(c *gin.Context) {
	cfg := config.Get()
	dbType := cfg.Database.Type
	var tables []string

	if dbType == "sqlite" {
		model.DB.Raw("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").Scan(&tables)
	} else {
		var dbName string
		model.DB.Raw("SELECT DATABASE()").Scan(&dbName)
		model.DB.Raw("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME", dbName).Scan(&tables)
	}

	var tableInfo []gin.H
	for _, t := range tables {
		var rowCount int64
		model.DB.Raw("SELECT COUNT(*) FROM " + t).Scan(&rowCount)
		tableInfo = append(tableInfo, gin.H{"name": t, "rows": rowCount})
	}

	common.Render(c, "system/database.html", gin.H{
		"tables":  tableInfo,
		"db_type": dbType,
	})
}

// Mod 数据库操作（优化/修复/备份）
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
	default:
		db.JSONFail(c, "未知操作！")
	}
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
	db.JSONOKMsg(c, "优化成功！")
}

func (db *DatabaseController) repairTables(c *gin.Context) {
	tables, err := db.getTableList(c)
	if err != nil {
		db.JSONFail(c, err.Error())
		return
	}
	cfg := config.Get()

	if cfg.Database.Type == "sqlite" {
		db.JSONFail(c, "SQLite不支持修复表操作！")
		return
	}

	for _, t := range tables {
		model.DB.Exec("REPAIR TABLE " + t)
	}
	db.JSONOKMsg(c, "修复成功！")
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
	db.JSONOKMsg(c, "备份成功！")
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
	db.JSONOKMsg(c, "备份成功！")
}

func (db *DatabaseController) backupSQLiteAction(c *gin.Context) {
	cfg := config.Get()
	if cfg.Database.Type != "sqlite" {
		db.JSONFail(c, "仅SQLite支持此操作！")
		return
	}

	backupDir := filepath.Join("static", "backup", "sql")
	os.MkdirAll(backupDir, 0755)

	srcPath := cfg.Database.DBName
	srcFile, err := os.Open(srcPath)
	if err != nil {
		db.JSONFail(c, "打开数据库文件失败！")
		return
	}
	defer srcFile.Close()

	filename := fmt.Sprintf("pbootcms_%s.db", time.Now().Format("20060102150405"))
	dstPath := filepath.Join(backupDir, filename)

	dstFile, err := os.Create(dstPath)
	if err != nil {
		db.JSONFail(c, "创建备份文件失败！")
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		db.JSONFail(c, "复制数据库文件失败！")
		return
	}
	db.JSONOKMsg(c, "备份成功！")
}

func (db *DatabaseController) header() string {
	var sb strings.Builder
	sb.WriteString("-- PbootCMS-Go 数据库备份文件\n")
	sb.WriteString("-- 版本: 1.8.1\n")
	sb.WriteString(fmt.Sprintf("-- 生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("-- -------------------------------------------\n\n")
	sb.WriteString("SET FOREIGN_KEY_CHECKS=0;\n\n")
	return sb.String()
}

func (db *DatabaseController) tableSql(tableName string) string {
	cfg := config.Get()
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("-- ----------------------------\n"))
	sb.WriteString(fmt.Sprintf("-- 表结构 `%s`\n", tableName))
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
	sb.WriteString(fmt.Sprintf("-- 表数据 `%s`\n", tableName))
	sb.WriteString(fmt.Sprintf("-- ----------------------------\n"))

	rows, err := model.DB.Raw("SELECT * FROM " + tableName).Rows()
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
		return nil, fmt.Errorf("请选择要操作的表！")
	}

	validPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	var validTables []string
	for _, t := range tableList {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !validPattern.MatchString(t) {
			return nil, fmt.Errorf("表名包含非法字符！%s", t)
		}
		validTables = append(validTables, t)
	}

	if len(validTables) == 0 {
		return nil, fmt.Errorf("请选择要操作的表！")
	}
	return validTables, nil
}
