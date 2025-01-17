package beta

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Health struct {
	KClientSet *kubernetes.Clientset
	DB         *gorm.DB
}

// @Summary check backend kubernetes is running, token required
// @Description check backend kubernetes is running, token required
// @Tags HealthCheck
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.HealthKubernetesResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/health/kubernetesAuth [get]
func (h *Health) CheckK8sAuth(c *gin.Context) {
	h.checkK8s(c)
}

// @Summary check backend kubernetes is running
// @Description check backend kubernetes is running
// @Tags HealthCheck
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.HealthKubernetesResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/health/kubernetes [get]
func (h *Health) CheckK8s(c *gin.Context) {
	h.checkK8s(c)
}

// @Summary check backend database is running, token required
// @Description check backend database is running, token required
// @Tags HealthCheck
// @Accept  json
// @Produce  json
// @Param db_name body docs.GenericDBRequest true "show tables in db"
// @Success 200 {object} docs.HealthDatabaseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/health/databaseAuth [post]
func (h *Health) CheckDatabaseAuth(c *gin.Context) {
	h.CheckDatabase(c)
}

// @Summary check backend database is running
// @Description check backend database is running
// @Tags HealthCheck
// @Accept  json
// @Produce  json
// @Param db_name body docs.GenericDBRequest true "show tables in db"
// @Success 200 {object} docs.HealthDatabaseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/health/database [post]
func (h *Health) CheckDatabase(c *gin.Context) {
	h.checkDatabase(c)
}

// PRIVATE function

func (h *Health) checkK8s(c *gin.Context) {
	statusList := []model.Node{}
	nList, err := h.KClientSet.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})

	if err != nil {
		log.Errorf("List Node fail: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, "List Node fail: %s", err.Error())
		return
	}

	for _, n := range nList.Items {
		a := n.Status.Conditions[len(n.Status.Conditions)-1]
		statusList = append(statusList, model.Node{
			Name:   n.Name,
			Status: a.Type,
		})
	}

	resp := model.HealthKubernetesResponse{
		Error:   false,
		Message: statusList,
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Health) checkDatabase(c *gin.Context) {
	var req model.GenericRequest
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}
	msg := req.Message

	tNameList := []string{}

	rows, err := h.DB.Raw("show tables").Rows()

	if err != nil {
		log.Errorf("Show all table name fail: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Query all table name fail: %s", err.Error())
		return
	}

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Errorf("Scan table name fail: %s", err.Error())
			RespondWithError(c, http.StatusInternalServerError, "Scan table name fail: %s", err.Error())
			return
		}
		tNameList = append(tNameList, name)
	}

	resp := model.HealthDatabaseResponse{
		GenericResponse: model.GenericResponse{
			Error:   false,
			Message: msg,
		},
		Tables: tNameList,
	}

	c.JSON(http.StatusOK, resp)
}
