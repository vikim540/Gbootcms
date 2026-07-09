package system

import (
	"gbootcms/apps/common"

	"github.com/gin-gonic/gin"
)

// UpgradeController - Online Upgrade Controller
// Corresponds to PHP: apps/admin/controller/UpgradeController.php
type UpgradeController struct {
	common.BaseController
}

// Index - Upgrade page
func (up *UpgradeController) Index(c *gin.Context) {
	common.Render(c, "system/upgrade.html", nil)
}
