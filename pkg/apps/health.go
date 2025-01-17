package apps

import "github.com/gin-gonic/gin"

type HealthInterface interface {
	CheckK8sAuth(c *gin.Context)
	CheckK8s(c *gin.Context)
	CheckDatabaseAuth(c *gin.Context)
	CheckDatabase(c *gin.Context)
}
