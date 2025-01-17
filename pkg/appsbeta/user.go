package beta

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/common"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/db"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/proxy"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/util"
)

type User struct {
	db *gorm.DB
}

// @Summary Get all users' id have the same role
// @Description Get all users' id have the same role
// @Tags User
// @Accept  json
// @Produce  json
// @Param roleid path string true "student or teacher"
// @Success 200 {object} docs.RoleListResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/user/role/{roleid} [get]
func (u *User) RoleList(c *gin.Context) {
	provider, exist := c.Get("Provider")
	if exist == false {
		provider = db.DEFAULT_PROVIDER
	}

	id := c.Param("roleid")

	if id == "" {
		log.Errorf("Empty role id")
		RespondWithError(c, http.StatusBadRequest, "Empty role id")
		return
	}

	user := db.User{
		Provider: util.StringPtr(provider.(string)),
		Role:     id,
	}

	result, err := user.GetRoleList(u.db)
	if err != nil {
		log.Errorf("Get role %s list fail: %s", id, err.Error())
		RespondWithError(c, http.StatusInternalServerError, err.Error())
	}

	userList := []common.LabelValue{}

	for _, n := range result {
		lbval := common.LabelValue{
			Label: n,
			Value: n,
		}
		userList = append(userList, lbval)
	}

	c.JSON(http.StatusOK, proxy.RoleListResponse{
		Error: false,
		Users: userList,
	})

}
