package apps

import "github.com/gin-gonic/gin"

type ImageInterface interface {
	List(c *gin.Context)
	Commit(c *gin.Context)
}
