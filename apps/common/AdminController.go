package common

import (
	"github.com/gin-gonic/gin"
)

// AdminController - Backend Management Base Controller
// This class provides basic functionality for backend management pages,
// including login check, permission verification, and menu retrieval
type AdminController struct {
	BaseController
}

// InitAdmin - Initialize backend controller, check login and permission
func (ac *AdminController) InitAdmin(c *gin.Context) bool {
	// Check login status - already handled in middleware
	// Additional initialization checks can be done here

	// Get menu for current URL
	// ...

	return true
}

// GetSecondMenu - Get sibling menus of current menu (used for sidebar highlight)
func (ac *AdminController) GetSecondMenu(c *gin.Context) gin.H {
	// Get menu data from session/middleware
	// Return sibling menus of current menu
	return gin.H{}
}

// NoAuthCheck - Controllers that skip permission check
var NoAuthCheckControllers = []string{}

// IsNoAuthCheck - Check if controller skips permission check
func IsNoAuthCheck(controller string) bool {
	for _, v := range NoAuthCheckControllers {
		if v == controller {
			return true
		}
	}
	return false
}
