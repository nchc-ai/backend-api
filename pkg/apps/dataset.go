package apps

import "github.com/gin-gonic/gin"

type DatasetInterface interface {
	List(c *gin.Context)
}
