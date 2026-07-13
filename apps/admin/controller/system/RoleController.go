package system

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gbootcms/apps/admin/helper"
	"gbootcms/apps/admin/model"
	sysModel "gbootcms/apps/admin/model/system"
	"gbootcms/apps/common"
	"github.com/gin-gonic/gin"
)

// RoleController 系統角色控制器
// 對應 PHP: apps/admin/controller/system/RoleController.php
type RoleController struct {
	common.BaseController
}

// Index 角色列表
func (rc *RoleController) Index(c *gin.Context) {
	var roles []sysModel.Role
	model.DB.Order("id ASC").Find(&roles)

	// 預格式化時間
	for i := range roles {
		if !roles[i].CreateTime.IsZero() {
			roles[i].CreateTimeStr = roles[i].CreateTime.Format("2006-01-02 15:04:05")
		}
		if !roles[i].UpdateTime.IsZero() {
			roles[i].UpdateTimeStr = roles[i].UpdateTime.Format("2006-01-02 15:04:05")
		}
	}

	data := gin.H{
		"list":  true,
		"roles": roles,
	}
	common.Render(c, "system/role.html", data)
}

// Add 新增角色
func (rc *RoleController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		description := c.PostForm("description")
		levels := c.PostFormArray("levels[]")
		acodes := c.PostFormArray("acodes[]")

		if name == "" {
			rc.JSONFail(c, "角色名稱不能為空！")
			return
		}

		// 自動生成 rcode（R101, R102, ...）
		lastRcode := sysModel.GetLastRcode()
		newRcode := autoRcode(lastRcode)

		username := rc.GetAdminUsername(c)
		now := time.Now()
		role := &sysModel.Role{
			Rcode:       newRcode,
			Name:        name,
			Description: description,
			CreateUser:  username,
			UpdateUser:  username,
			CreateTime:  now,
			UpdateTime:  now,
		}
		if err := model.DB.Create(role).Error; err != nil {
			rc.JSONFail(c, "新增失敗！")
			return
		}

		// 寫入權限到 ay_role_level（先刪後插，確保乾淨）
		sysModel.DelRoleLevels(newRcode)
		sysModel.AddRoleLevels(newRcode, levels)

		// 寫入區域到 ay_role_area
		sysModel.DelRoleAreas(newRcode)
		sysModel.AddRoleAreas(newRcode, acodes)

		rc.LogAction(c, "新增角色"+newRcode+"成功")
		c.JSON(200, gin.H{"code": 1, "data": common.NoticeAdd, "msg": common.NoticeAdd, "tourl": "/admin/Role/index"})
		return
	}

	// GET: 顯示新增表單
	menuList := buildMenuList(nil)
	areaCheckbox := buildAreaCheckbox(nil)

	data := gin.H{
		"add":          true,
		"menu_list":    menuList,
		"area_checkbox": areaCheckbox,
	}
	common.Render(c, "system/role.html", data)
}

// Mod 修改角色
func (rc *RoleController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	rcode := params["rcode"]
	if rcode == "" {
		rcode = params["id"]
	}
	if rcode == "" {
		rcode = c.Query("rcode")
	}

	if rcode == "" {
		rc.JSONFail(c, "傳遞的參數值錯誤！")
		return
	}

	if c.Request.Method == "POST" {
		name := c.PostForm("name")
		description := c.PostForm("description")
		levels := c.PostFormArray("levels[]")
		acodes := c.PostFormArray("acodes[]")

		if name == "" {
			rc.JSONFail(c, "角色名稱不能為空！")
			return
		}

		updates := map[string]interface{}{
			"name":        name,
			"description": description,
			"update_user": rc.GetAdminUsername(c),
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
		}
		if err := model.DB.Model(&sysModel.Role{}).Where("rcode = ?", rcode).Updates(updates).Error; err != nil {
			rc.JSONFail(c, "修改失敗！")
			return
		}

		// 重寫權限（先刪後插）
		sysModel.DelRoleLevels(rcode)
		sysModel.AddRoleLevels(rcode, levels)

		// 重寫區域
		sysModel.DelRoleAreas(rcode)
		sysModel.AddRoleAreas(rcode, acodes)

		rc.LogAction(c, "修改角色"+rcode+"成功")
		c.JSON(200, gin.H{"code": 1, "data": common.NoticeModify, "msg": common.NoticeModify, "tourl": "/admin/Role/index"})
		return
	}

	// GET: 顯示修改表單
	role := sysModel.GetRoleByRcode(rcode)
	if role == nil {
		rc.JSONFail(c, "編輯的內容已經不存在！")
		return
	}

	// 預格式化時間
	if !role.CreateTime.IsZero() {
		role.CreateTimeStr = role.CreateTime.Format("2006-01-02 15:04:05")
	}
	if !role.UpdateTime.IsZero() {
		role.UpdateTimeStr = role.UpdateTime.Format("2006-01-02 15:04:05")
	}

	menuList := buildMenuList(role.LevelList)
	areaCheckbox := buildAreaCheckbox(role.AreaList)

	data := gin.H{
		"mod":          true,
		"role":         role,
		"menu_list":    menuList,
		"area_checkbox": areaCheckbox,
	}
	common.Render(c, "system/role.html", data)
}

// Del 刪除角色
func (rc *RoleController) Del(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	rcode := params["id"]
	if rcode == "" {
		rcode = params["rcode"]
	}
	if rcode == "" {
		rcode = c.Query("rcode")
	}

	if rcode == "" {
		rc.JSONFail(c, "傳遞的參數值錯誤！")
		return
	}

	if err := sysModel.DelRoleByRcode(rcode); err != nil {
		rc.JSONFail(c, "刪除失敗！")
		return
	}

	rc.LogAction(c, "刪除角色"+rcode+"成功")
	c.JSON(200, gin.H{"code": 1, "data": common.NoticeDelete, "msg": common.NoticeDelete, "tourl": "/admin/Role/index"})
}

// autoRcode 自動生成角色編碼（R101, R102, ...）
func autoRcode(lastCode string) string {
	if lastCode == "" {
		return "R101"
	}
	// 提取數字部分並遞增
	prefix := ""
	numStr := ""
	for i, ch := range lastCode {
		if ch >= '0' && ch <= '9' {
			prefix = lastCode[:i]
			numStr = lastCode[i:]
			break
		}
	}
	if numStr == "" {
		return lastCode + "1"
	}
	num, _ := strconv.Atoi(numStr)
	return fmt.Sprintf("%s%03d", prefix, num+1)
}

// buildMenuList 生成權限複選框 HTML
// 對齊 PbootCMS PHP RoleController::makeLevelList()
// checkedLevels 是已選中的權限 URL 列表
func buildMenuList(checkedLevels []string) string {
	// 構建已選中 map（不區分大小寫，與 injectCheckLevelPermissions 保持一致）
	checkedMap := make(map[string]bool)
	for _, l := range checkedLevels {
		checkedMap[strings.ToLower(strings.TrimSpace(l))] = true
	}

	// 查詢所有啟用的菜單（一次查詢）
	var menus []sysModel.Menu
	model.DB.Where("status = ?", 1).Order("sorting ASC, id ASC").Find(&menus)

	// 查詢所有菜單操作按鈕（一次查詢）
	var actions []sysModel.MenuAction
	model.DB.Order("sorting ASC, id ASC").Find(&actions)

	// 查詢 ay_type 表獲取操作按鈕中文名稱（對齊 PHP 的三表 JOIN）
	type actionName struct {
		Value string `gorm:"column:value"`
		Item  string `gorm:"column:item"`
	}
	var typeActions []actionName
	model.DB.Table("ay_type").Where("tcode = ?", "T101").Scan(&typeActions)
	actionTextMap := make(map[string]string)
	for _, t := range typeActions {
		actionTextMap[t.Value] = t.Item
	}

	// 按 mcode 分組操作
	actionMap := make(map[string][]sysModel.MenuAction)
	for _, a := range actions {
		actionMap[a.Mcode] = append(actionMap[a.Mcode], a)
	}

	// 在記憶體中構建菜單樹（消除 N+1 查詢）
	// 注意：根菜單的 pcode 為空字串 ""（非 "0"），對齊 buildMenuTree() in Render.go
	menuTree := buildMenuTreeInMemory(menus, "")

	// 遞迴生成 HTML
	var sb strings.Builder
	renderMenuLevel(&sb, menuTree, actionMap, actionTextMap, checkedMap, 0)
	return sb.String()
}

// menuNode 記憶體中的菜單樹節點
type menuNode struct {
	Menu     sysModel.Menu
	Children []menuNode
}

// buildMenuTreeInMemory 在記憶體中構建菜單樹（不做 DB 查詢）
func buildMenuTreeInMemory(menus []sysModel.Menu, parentCode string) []menuNode {
	var tree []menuNode
	for _, m := range menus {
		if m.Pcode == parentCode {
			node := menuNode{Menu: m}
			node.Children = buildMenuTreeInMemory(menus, m.Mcode)
			tree = append(tree, node)
		}
	}
	return tree
}

// renderMenuLevel 遞迴渲染權限複選框（不做 DB 查詢）
func renderMenuLevel(sb *strings.Builder, nodes []menuNode, actionMap map[string][]sysModel.MenuAction, actionTextMap map[string]string, checkedMap map[string]bool, depth int) {
	indent := strings.Repeat("&nbsp;&nbsp;&nbsp;&nbsp;", depth)
	for _, node := range nodes {
		m := node.Menu
		// 瀏覽權限（不區分大小寫比對）
		checked := ""
		if checkedMap[strings.ToLower(m.URL)] {
			checked = "checked"
		}
		sb.WriteString(fmt.Sprintf("%s<input type='checkbox' name='levels[]' value='%s' title='瀏覽' %s> <b>%s</b>",
			indent, m.URL, checked, m.Name))

		// 操作權限（add/mod/del/export/import 等）
		if actions, ok := actionMap[m.Mcode]; ok {
			preURL := getPreURL(m.URL)
			for _, a := range actions {
				url := preURL + a.Action
				checked := ""
				if checkedMap[strings.ToLower(url)] {
					checked = "checked"
				}
				btnText := actionTextWithMap(a.Action, actionTextMap)
				sb.WriteString(fmt.Sprintf(" <input type='checkbox' name='levels[]' value='%s' title='%s' %s>",
					url, btnText, checked))
			}
		}
		sb.WriteString("<br>\n")

		// 遞迴子菜單（使用記憶體樹，不做 DB 查詢）
		if len(node.Children) > 0 {
			renderMenuLevel(sb, node.Children, actionMap, actionTextMap, checkedMap, depth+1)
		}
	}
}

// getPreURL 從菜單 URL 提取前綴（如 /admin/Role/index → /admin/Role/）
func getPreURL(url string) string {
	count := 0
	for i, c := range url {
		if c == '/' {
			count++
			if count == 3 {
				return url[:i+1]
			}
		}
	}
	return url + "/"
}

// actionTextWithMap 操作按鈕文字（從 ay_type 表查詢，fallback 到硬編碼）
func actionTextWithMap(action string, actionTextMap map[string]string) string {
	if text, ok := actionTextMap[action]; ok && text != "" {
		return text
	}
	// fallback（繁體中文）
	switch action {
	case "add":
		return "新增"
	case "mod":
		return "修改"
	case "del":
		return "刪除"
	case "export":
		return "匯出"
	case "import":
		return "匯入"
	default:
		return action
	}
}

// buildAreaCheckbox 生成區域複選框 HTML
// checkedAreas 是已選中的區域編碼列表
func buildAreaCheckbox(checkedAreas []string) string {
	checkedMap := make(map[string]bool)
	for _, a := range checkedAreas {
		checkedMap[strings.TrimSpace(a)] = true
	}

	// 查詢所有區域
	areaTree := sysModel.GetAreaList()
	var sb strings.Builder
	renderAreaCheckbox(&sb, areaTree, checkedMap, 0)
	return sb.String()
}

// renderAreaCheckbox 遞迴渲染區域複選框
func renderAreaCheckbox(sb *strings.Builder, tree []sysModel.AreaTreeNode, checkedMap map[string]bool, depth int) {
	indent := strings.Repeat("&nbsp;&nbsp;&nbsp;&nbsp;", depth)
	for _, node := range tree {
		checked := ""
		if checkedMap[node.Acode] {
			checked = "checked"
		}
		sb.WriteString(fmt.Sprintf("%s<input type='checkbox' name='acodes[]' value='%s' title='%s' %s> %s<br>\n",
			indent, node.Acode, node.Name, checked, node.Name))
		if len(node.Son) > 0 {
			renderAreaCheckbox(sb, node.Son, checkedMap, depth+1)
		}
	}
}
