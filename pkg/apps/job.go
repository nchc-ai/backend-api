package apps

import "github.com/gin-gonic/gin"

type JobInterface interface {
	Launch(c *gin.Context)
	Delete(c *gin.Context)
	List(c *gin.Context)
}
