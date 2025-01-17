package apps

import "github.com/gin-gonic/gin"

type UserInterface interface {
	RoleList(c *gin.Context)
}
