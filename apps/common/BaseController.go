package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type BaseController struct{}

func (bc *BaseController) JSONOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "data": data})
}

func (bc *BaseController) JSONOKMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 1, "msg": msg})
}

func (bc *BaseController) JSONFail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": msg})
}

func (bc *BaseController) JSONFailMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": msg})
}

func (bc *BaseController) GetAdminUsername(c *gin.Context) string {
	username, _ := GetSession(c, "admin_username").(string)
	return username
}

func (bc *BaseController) GetAdminUID(c *gin.Context) int {
	return GetSessionInt(c, "admin_uid")
}

func (bc *BaseController) GetAdminUcode(c *gin.Context) string {
	ucode, _ := GetSession(c, "admin_ucode").(string)
	return ucode
}

func (bc *BaseController) IsLogin(c *gin.Context) bool {
	return GetSessionInt(c, "admin_uid") > 0
}
