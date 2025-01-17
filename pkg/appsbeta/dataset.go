package beta

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/consts"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/common"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Dataset struct {
	Config     *config.Config
	KClientSet *kubernetes.Clientset
}

// @Summary  List all shared data set stored in PV
// @Description  List all shared data set stored in PV
// @Tags DataSet
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.DatasetsListResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/datasets [get]
func (d *Dataset) List(c *gin.Context) {

	// This only to be done before user create job in corresponding classroom namespace.
	// There is no need to block and wait for sync finish at this time.
	// Actually, syncDataSetPVC() create soft link on file system, it does not waste a lot of time.
	go d.syncDataSetPVC()

	pvcNameList := []common.LabelValue{}
	pvcs, err := d.KClientSet.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil {
		RespondWithError(c, http.StatusInternalServerError,
			"List Kubernetes default namespace PVC fail: %s", err.Error())
		return
	}

	for _, pvc := range pvcs.Items {
		// dataset pvc name should start with "dataset-", pvc name will be stored in database
		//https://gitlab.com/nchc-ai/AI-Eduational-Platform/issues/18#note_86408557
		if strings.HasPrefix(pvc.Name, "dataset-") {
			r := strings.SplitN(pvc.Name, "-", 2)
			if len(r) != 2 || r[0] != "dataset" {
				log.Warning(fmt.Sprintf("%s doesn't start with 'dataset-', NOT valided dataset name, skip", pvc.Name))
				continue
			}
			lbval := common.LabelValue{
				Label: r[1],
				Value: pvc.Name,
			}
			pvcNameList = append(pvcNameList, lbval)
		}
	}

	c.JSON(http.StatusOK, model.DatasetsListResponse{
		Error:    false,
		Datasets: pvcNameList,
	})

}

func (d *Dataset) syncDataSetPVC() {

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			consts.NamespaceLabelInstance: d.Config.APIConfig.NamespacePrefix,
		},
	}

	allNS, err := d.KClientSet.CoreV1().Namespaces().List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
		})
	if err != nil {
		log.Warningf("List all namespace fail: %s", err.Error())
		return
	}

	cc := Classroom{
		Config:     d.Config,
		KClientSet: d.KClientSet,
	}

	for _, ns := range allNS.Items {
		err1 := cc.CreateDataSetPVC(ns.Name)
		if err1 != nil {
			log.Warningf("sync pvc in namespace {%s} fail: %s ", ns.Name, err1.Error())
		}

		err2 := cc.RemoveDataSetPVC(ns.Name)
		if err2 != nil {
			log.Warningf("sync pvc in namespace {%s} fail: %s ", ns.Name, err2.Error())
		}
	}

}
