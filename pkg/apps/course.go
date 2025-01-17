package apps

import (
	"github.com/gin-gonic/gin"
)

type CourseInterface interface {
	Delete(c *gin.Context)
	Update(c *gin.Context)
	Add(c *gin.Context)
	Get(c *gin.Context)
	ListUserCourse(c *gin.Context)
	ListLevelCourse(c *gin.Context)
	ListAllCourse(c *gin.Context)
	SearchCourse(c *gin.Context)
	CourseType(c *gin.Context)
	CourseNameList(c *gin.Context)
}
