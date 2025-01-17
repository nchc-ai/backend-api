package apps

import "github.com/gin-gonic/gin"

type ProxyInterface interface {
	// Required: These must be implemented by all oauth service
	GetToken(c *gin.Context)
	RefreshToken(c *gin.Context)
	Introspection(c *gin.Context)
	Logout(c *gin.Context)
	QueryUser(c *gin.Context)

	// Optional:
	// Not all oauth service expose user related functionality
	// provider should return Not implemented error if some function is not support
	// provider will return "not supported" error
	RegisterUser(c *gin.Context)
	UpdateUserBasicInfo(c *gin.Context)
	ChangeUserPassword(c *gin.Context)

}
