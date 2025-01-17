package db

import (
	"context"
	"fmt"
	"strings"

	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/common"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	"github.com/nitishm/go-rejson/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Job struct {
	Model
	OauthUser
	// foreign key
	CourseID string `gorm:"size:36"`
	// foreign key
	ClassroomID *string `gorm:"size:72"`
	Status      string  `gorm:"not null"`
}

func (Job) TableName() string {
	return "containerJobs"
}

func (j *Job) NewEntry(DB *gorm.DB) error {

	if err := DB.Create(j).Error; err != nil {
		return err
	}

	return nil
}

func (j *Job) GetCourse(DB *gorm.DB) (*Course, error) {

	course, err := GetCourse(DB, j.CourseID)
	if err != nil {
		return nil, err
	}

	datasetsArray, err := course.GetDataset(DB)
	if err != nil {
		return nil, err
	}

	datasets := []common.LabelValue{}
	for _, v := range datasetsArray {
		dataset_name := strings.SplitN(v, "-", 2)
		if len(dataset_name) != 2 || dataset_name[0] != "dataset" {
			log.Warning(fmt.Sprintf("%s doesn't start with 'dataset-', NOT valided dataset name, skip", dataset_name))
			continue
		}

		datasets = append(datasets, common.LabelValue{
			Label: dataset_name[1],
			Value: v,
		})
	}

	return &Course{
		Model: Model{
			ID:        course.ID,
			CreatedAt: course.CreatedAt,
		},
		Name:         course.Name,
		Introduction: course.Introduction,
		Image:        course.Image,
		Level:        course.Level,
		Gpu:          course.Gpu,
		Datasets:     &datasets,
	}, nil
}

func (j *Job) DeleteCourseCRD(db *gorm.DB, redis *rejson.Handler, crdClient *versioned.Clientset, ns string) (string, error) {

	deletePolicy := metav1.DeletePropagationForeground
	if err := crdClient.NchcV1alpha1().Courses(ns).
		Delete(context.Background(), j.ID, metav1.DeleteOptions{PropagationPolicy: &deletePolicy}); err != nil {
		return fmt.Sprintf("Failed to delete Course CRD {%s}: %s", j.ID, err.Error()), err
	}

	if err := db.Unscoped().Delete(j).Error; err != nil {
		return fmt.Sprintf("Failed to delete job {%s} information : %s", j.ID, err.Error()), err
	}

	redisKey := fmt.Sprintf("%s:%s", j.Provider, j.User)
	if _, err := redis.JSONDel(redisKey, "."); err != nil {
		return fmt.Sprintf("Delete cache key {%s} fail for delete job: %s", redisKey, err.Error()), err
	}

	return "", nil
}

func (j *Job) GetJobOwnByUser(db *gorm.DB) ([]Job, error) {

	resultJobs := []Job{}
	if err := db.Where(&j).Find(&resultJobs).Error; err != nil {
		return nil, err
	}

	return resultJobs, nil
}

func (j *Job) UserJobCount(db *gorm.DB) (int, error) {
	count := 0
	if err := db.Model(&Job{}).Where("user = ?", j.User).Count(&count).Error; err != nil {
		return count, err
	}
	return count, nil
}
