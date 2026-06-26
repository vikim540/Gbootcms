package content

import (
	"pbootcms-go/apps/admin/model"
	"pbootcms-go/apps/common"

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
	model.DB.FirstOrCreate(&company, model.Company{ID: 1})
	common.Render(c, "content/company.html", gin.H{"companys": company})
}

// Mod - Modify company information
func (co *CompanyController) Mod(c *gin.Context) {
	var company model.Company
	model.DB.FirstOrCreate(&company, model.Company{ID: 1})

	result := model.DB.Model(&company).Updates(map[string]interface{}{
		"name":     c.PostForm("name"),
		"address":  c.PostForm("address"),
		"postcode": c.PostForm("postcode"),
		"contact":  c.PostForm("contact"),
		"mobile":   c.PostForm("mobile"),
		"phone":    c.PostForm("phone"),
		"fax":      c.PostForm("fax"),
		"email":    c.PostForm("email"),
		"qq":       c.PostForm("qq"),
		"weixin":   c.PostForm("weixin"),
		"icp":      c.PostForm("icp"),
		"blicense": c.PostForm("blicense"),
		"other":    c.PostForm("other"),
	})
	if result.Error != nil {
		co.JSONFail(c, "修改失败: "+result.Error.Error())
		return
	}
	co.JSONOKMsg(c, "修改成功")
}
