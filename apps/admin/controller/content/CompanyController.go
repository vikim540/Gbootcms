package content

import (
	"gbootcms/apps/admin/model"
	"gbootcms/apps/common"

	"github.com/gin-gonic/gin"
)

// CompanyController - Company Information Controller
// Corresponds to PHP: apps/admin/controller/CompanyController.php
type CompanyController struct {
	common.BaseController
}

// Index - Company information page
func (co *CompanyController) Index(c *gin.Context) {
	var company model.Company
	// AcodePlugin 自動按當前區域過濾，取該區域的公司記錄
	model.DB.WithContext(c.Request.Context()).FirstOrCreate(&company)
	common.Render(c, "content/company.html", gin.H{"companys": company})
}

// Mod - Modify company information
func (co *CompanyController) Mod(c *gin.Context) {
	var company model.Company
	// AcodePlugin 自動按當前區域過濾，取該區域的公司記錄
	model.DB.WithContext(c.Request.Context()).FirstOrCreate(&company)

	// 臟檢測：比對提交數據與現有數據
	newName := c.PostForm("name")
	newAddress := c.PostForm("address")
	newPostcode := c.PostForm("postcode")
	newContact := c.PostForm("contact")
	newMobile := c.PostForm("mobile")
	newPhone := c.PostForm("phone")
	newFax := c.PostForm("fax")
	newEmail := c.PostForm("email")
	newQQ := c.PostForm("qq")
	newWeixin := c.PostForm("weixin")
	newIcp := c.PostForm("icp")
	newBlicense := c.PostForm("blicense")
	newOther := c.PostForm("other")

	if company.Name == newName && company.Address == newAddress && company.Postcode == newPostcode &&
		company.Contact == newContact && company.Mobile == newMobile && company.Phone == newPhone &&
		company.Fax == newFax && company.Email == newEmail && company.Qq == newQQ &&
		company.Weixin == newWeixin && company.ICP == newIcp && company.Blicense == newBlicense &&
		company.Other == newOther {
		co.JSONOKMsg(c, common.NoticeNoChange)
		return
	}

	result := model.DB.WithContext(c.Request.Context()).Model(&company).Updates(map[string]interface{}{
		"name":     newName,
		"address":  newAddress,
		"postcode": newPostcode,
		"contact":  newContact,
		"mobile":   newMobile,
		"phone":    newPhone,
		"fax":      newFax,
		"email":    newEmail,
		"qq":       newQQ,
		"weixin":   newWeixin,
		"icp":      newIcp,
		"blicense": newBlicense,
		"other":    newOther,
	})
	if result.Error != nil {
		co.JSONFail(c, "修改失败: "+result.Error.Error())
		return
	}
	co.JSONOKMsg(c, common.NoticeModify)
}
