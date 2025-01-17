package apps

import "github.com/gin-gonic/gin"

type ClassroomInterface interface {
	List(c *gin.Context)
	ListAll(c *gin.Context)
	Add(c *gin.Context)
	Delete(ctx *gin.Context)
	Get(c *gin.Context)
	Update(c *gin.Context)
	UploadUserAccount(c *gin.Context)
}
