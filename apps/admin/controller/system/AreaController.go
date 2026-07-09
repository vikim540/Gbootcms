package system

import (
	"fmt"
	"gbootcms/apps/admin/helper"
	sysModel "gbootcms/apps/admin/model/system"
	"gbootcms/apps/common"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AreaController 數據區域控制器
// 對應 PHP: apps/admin/controller/system/AreaController.php
type AreaController struct {
	common.BaseController
}

// areaListRow 列表行（帶縮進和子節點標記）
type areaListRow struct {
	ID         uint   `json:"id"`
	Blank      string `json:"blank"`
	Name       string `json:"name"`
	Acode      string `json:"acode"`
	Domain     string `json:"domain"`
	IsDefault  string `json:"is_default"`
	CreateUser string `json:"create_user"`
	UpdateTime string `json:"update_time"`
	Son        bool   `json:"son"`
}

// areaOption 下拉選項
type areaOption struct {
	Acode    string `json:"acode"`
	Name     string `json:"name"`
	Indent   string `json:"indent"`
	Selected bool   `json:"selected"`
}

// Index 區域列表（樹形）
func (ar *AreaController) Index(c *gin.Context) {
	tree := sysModel.GetAreaList()
	var rows []areaListRow
	makeAreaList(tree, "", &rows)

	areaSelectTree := sysModel.GetAreaSelect()
	var opts []areaOption
	makeAreaOptions(areaSelectTree, "", "", &opts)

	data := gin.H{
		"list":         true,
		"areas":        rows,
		"area_options": opts,
	}
	common.Render(c, "system/area.html", data)
}

// makeAreaList 遞迴生成帶縮進的扁平列表
func makeAreaList(tree []sysModel.AreaTreeNode, blank string, out *[]areaListRow) {
	for _, node := range tree {
		row := areaListRow{
			ID:         node.ID,
			Blank:      blank,
			Name:       node.Name,
			Acode:      node.Acode,
			Domain:     node.Domain,
			IsDefault:  node.IsDefault,
			CreateUser: node.CreateUser,
			UpdateTime: formatAreaTime(node.UpdateTime),
			Son:        len(node.Son) > 0,
		}
		*out = append(*out, row)
		if len(node.Son) > 0 {
			makeAreaList(node.Son, blank+"　　", out)
		}
	}
}

// makeAreaOptions 遞迴生成下拉選項（結構化數據，非 HTML 字串）
func makeAreaOptions(tree []sysModel.AreaTreeNode, selectid string, excludeAcode string, out *[]areaOption) {
	for _, node := range tree {
		if node.Acode == excludeAcode {
			continue
		}
		opt := areaOption{
			Acode:    node.Acode,
			Name:     node.Name,
			Indent:   indentPrefix(node.Pcode),
			Selected: selectid == node.Acode,
		}
		*out = append(*out, opt)
		if len(node.Son) > 0 {
			makeAreaOptions(node.Son, selectid, excludeAcode, out)
		}
	}
}

// indentPrefix 根據 pcode 生成縮進前綴
func indentPrefix(pcode string) string {
	if pcode != "0" {
		return "　　"
	}
	return ""
}

// formatAreaTime 格式化時間字串
func formatAreaTime(t interface{}) string {
	if t == nil {
		return ""
	}
	type fmter interface {
		Format(layout string) string
	}
	if f, ok := t.(fmter); ok {
		return f.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%v", t)
}

// domainRegex 用於提取域名
var domainRegex = regexp.MustCompile(`^(https?://)?([\w\-.]+)(/+)?$`)

// extractDomain 從輸入中提取純域名
func extractDomain(input string) (string, bool) {
	if input == "" {
		return "", true
	}
	if domainRegex.MatchString(input) {
		return domainRegex.ReplaceAllString(input, "$2"), true
	}
	return "", false
}

// Add 新增區域
func (ar *AreaController) Add(c *gin.Context) {
	if c.Request.Method == "POST" {
		acode := c.PostForm("acode")
		pcode := c.PostForm("pcode")
		name := c.PostForm("name")
		domain := c.PostForm("domain")
		isDefault := c.DefaultPostForm("is_default", "0")

		if acode == "" {
			ar.JSONFail(c, "編碼不能為空！")
			return
		}
		if pcode == "" {
			pcode = "0"
		}
		if name == "" {
			ar.JSONFail(c, "區域名稱不能為空！")
			return
		}

		if domain != "" {
			cleanDomain, ok := extractDomain(domain)
			if !ok {
				ar.JSONFail(c, "要綁定的域名輸入有錯！")
				return
			}
			domain = cleanDomain
			if sysModel.CheckAreaDomainExists(domain, "") {
				ar.JSONFail(c, "該域名已經綁定其他區域，不能再使用！")
				return
			}
		}

		if sysModel.CheckAreaAcodeExists(acode, "") {
			ar.JSONFail(c, "該區域編號已經存在，不能再使用！")
			return
		}

		username := ar.GetAdminUsername(c)
		area := &sysModel.Area{
			Acode:      acode,
			Pcode:      pcode,
			Name:       name,
			Domain:     domain,
			IsDefault:  isDefault,
			CreateUser: username,
			UpdateUser: username,
		}
		if err := sysModel.AddArea(area); err != nil {
			ar.JSONFail(c, "新增失敗！")
			return
		}
		ar.LogAction(c, "新增數據區域"+acode+"成功")
		c.JSON(200, gin.H{"code": 1, "data": common.NoticeAdd, "msg": common.NoticeAdd, "tourl": "/admin/Area/index"})
		return
	}

	// GET: 顯示新增表單
	areaSelectTree := sysModel.GetAreaSelect()
	var opts []areaOption
	makeAreaOptions(areaSelectTree, "", "", &opts)
	common.Render(c, "system/area.html", gin.H{
		"list":         true,
		"area_options": opts,
	})
}

// Mod 修改區域
func (ar *AreaController) Mod(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	acode := params["acode"]
	if acode == "" {
		acode = c.Query("acode")
	}
	if acode == "" {
		acode = c.Param("id")
	}

	if acode == "" {
		ar.JSONFail(c, "傳遞的參數值錯誤！")
		return
	}

	if c.Request.Method == "POST" {
		newAcode := c.PostForm("acode")
		pcode := c.PostForm("pcode")
		name := c.PostForm("name")
		domain := c.PostForm("domain")
		isDefault := c.DefaultPostForm("is_default", "0")

		if newAcode == "" {
			ar.JSONFail(c, "編碼不能為空！")
			return
		}
		if pcode == "" {
			pcode = "0"
		}
		if name == "" {
			ar.JSONFail(c, "區域名稱不能為空！")
			return
		}

		if domain != "" {
			cleanDomain, ok := extractDomain(domain)
			if !ok {
				ar.JSONFail(c, "要綁定的域名輸入有錯！")
				return
			}
			domain = cleanDomain
			if sysModel.CheckAreaDomainExists(domain, acode) {
				ar.JSONFail(c, "該域名已經綁定其他區域，不能再使用！")
				return
			}
		}

		if sysModel.CheckAreaAcodeExists(newAcode, acode) {
			ar.JSONFail(c, "該區域編號已經存在，不能再使用！")
			return
		}

		updates := map[string]interface{}{
			"acode":       newAcode,
			"pcode":       pcode,
			"name":        name,
			"domain":      domain,
			"is_default":  isDefault,
			"update_user": ar.GetAdminUsername(c),
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
		}
		if err := sysModel.ModArea(acode, updates); err != nil {
			ar.JSONFail(c, "修改失敗！")
			return
		}
		ar.LogAction(c, "修改數據區域"+acode+"成功")
		c.JSON(200, gin.H{"code": 1, "data": common.NoticeModify, "msg": common.NoticeModify, "tourl": "/admin/Area/index"})
		return
	}

	// GET: 顯示修改表單
	area := sysModel.GetAreaByAcode(acode)
	if area == nil {
		ar.JSONFail(c, "編輯的內容已經不存在！")
		return
	}

	areaSelectTree := sysModel.GetAreaSelect()
	var opts []areaOption
	makeAreaOptions(areaSelectTree, area.Pcode, acode, &opts)

	// 預格式化供模板顯示
	areaMap := map[string]interface{}{
		"Acode":      area.Acode,
		"Pcode":      area.Pcode,
		"Name":       area.Name,
		"Domain":     area.Domain,
		"IsDefault":  area.IsDefault,
		"CreateUser": area.CreateUser,
	}

	data := gin.H{
		"mod":          true,
		"area":         areaMap,
		"area_options": opts,
		"get_acode":    acode,
	}
	injectGetFlat(data, params)
	common.Render(c, "system/area.html", data)
}

// Del 刪除區域
func (ar *AreaController) Del(c *gin.Context) {
	params := helper.ParseWildcardAction(c.Param("action"))
	acode := params["acode"]
	if acode == "" {
		acode = c.Query("acode")
	}

	if acode == "" {
		ar.JSONFail(c, "傳遞的參數值錯誤！")
		return
	}
	if acode == "cn" {
		ar.JSONFail(c, "系統內置區域不允許刪除！")
		return
	}

	if err := sysModel.DelArea(acode); err != nil {
		ar.JSONFail(c, "刪除失敗，請核對是否為默認區域！")
		return
	}
	ar.LogAction(c, "刪除數據區域"+acode+"成功")
	c.JSON(200, gin.H{"code": 1, "data": common.NoticeDelete, "msg": common.NoticeDelete, "tourl": "/admin/Area/index"})
}

// injectGetFlat 注入 path 參數為 get_xxx 扁平變量
func injectGetFlat(data gin.H, params map[string]string) {
	for k, v := range params {
		data["get_"+k] = v
	}
}

// unused import guard
var _ = strings.TrimSpace
