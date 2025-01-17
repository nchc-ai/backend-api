package db

import (
	"errors"
	"fmt"

	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/common"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/util"
	"github.com/nchc-ai/course-crd/pkg/apis/coursecontroller/v1alpha1"
)

const VM = "VM"
const CONTAINER = "CONTAINER"
const DEFAULT_PROVIDER = "default-provider"

const (
	ERROR_COURSE_NOINTRO_FMT   = "課程 {%s} 課程介紹沒有填寫"
	ERROR_COURSE_NOGPU_FMT     = "課程 {%s} GPU欄位沒有填寫"
	ERROR_COURSE_NOPATH_FMT    = "課程 {%s} 工作目錄沒有填寫"
	ERROR_COURSE_NOIMAGE_FMT   = "課程 {%s} 映像檔欄位沒有填寫"
	ERROR_COURSE_NODATASET_FMT = "課程 {%s} 資料集欄位沒有填寫"
	ERROR_COURSE_NOPORT_FMT    = "課程 {%s} 存取端口沒有填寫"
)

type CourseID struct {
	Model
}

func (CourseID) TableName() string {
	return "courseid"
}

type Course struct {
	Model
	OauthUser
	Name         string                `gorm:"not null" json:"name"`
	Level        string                `gorm:"not null;default:'basic';size:10" json:"level"`
	Introduction *string               `gorm:"size:3000" json:"introduction,omitempty"`
	Image        string                `gorm:"not null" json:"-"`
	AccessType   v1alpha1.AccessType   `gorm:"not null;default:'NodePort'" json:"accessType,omitempty"`
	Gpu          *int32                `gorm:"not null;default:0" json:"-"`
	WritablePath *string               `gorm:"not null" json:"writablePath,omitempty"`
	ImageLV      *common.LabelValue    `gorm:"-" json:"image,omitempty"`
	GpuLV        *common.LabelIntValue `gorm:"-" json:"gpu,omitempty"`
	Datasets     *[]common.LabelValue  `gorm:"-" json:"datasets,omitempty"`
	Ports        *[]Port               `gorm:"-" json:"ports,omitempty"`
	CourseType   *string               `gorm:"-" json:"type,omitempty"`
	ClasroomID   *string               `gorm:"-" json:"roomId,omitempty"`
}

func (Course) TableName() string {
	return "containerCourses"
}

func GetCourse(db *gorm.DB, id string) (*Course, error) {
	course := Course{
		Model: Model{
			ID: id,
		},
	}
	err := db.First(&course).Error

	if err != nil {
		return nil, err
	}
	return &course, nil
}

func (course *Course) CheckNilField(happenWhen string) (bool, []error) {

	if course.Introduction == nil {
		return true, []error{
			errors.New("Introduction field is nil"),
			errors.New(fmt.Sprintf(happenWhen+ERROR_COURSE_NOINTRO_FMT, course.Name)),
		}
	}

	if course.GpuLV == nil {
		return true, []error{
			errors.New("gpu field is nil"),
			errors.New(fmt.Sprintf(happenWhen+ERROR_COURSE_NOGPU_FMT, course.Name)),
		}
	}

	if course.WritablePath == nil {
		return true, []error{
			errors.New("writablepath field is nil"),
			errors.New(fmt.Sprintf(happenWhen+ERROR_COURSE_NOPATH_FMT, course.Name)),
		}
	}

	if course.ImageLV == nil {
		return true, []error{
			errors.New("image field is nil"),
			errors.New(fmt.Sprintf(happenWhen+ERROR_COURSE_NOIMAGE_FMT, course.Name)),
		}
	}

	if course.Datasets == nil {
		return true, []error{
			errors.New("dataset field is nil"),
			errors.New(fmt.Sprintf(happenWhen+ERROR_COURSE_NODATASET_FMT, course.Name)),
		}
	}

	if course.Ports == nil {
		return true, []error{
			errors.New("port field is nil"),
			errors.New(fmt.Sprintf(happenWhen+ERROR_COURSE_NOPORT_FMT, course.Name)),
		}
	}

	return false, nil
}

func (course *Course) GetDataset(DB *gorm.DB) ([]string, error) {

	dataset := Dataset{
		CourseID: course.ID,
	}
	datasetResult := []Dataset{}
	err := DB.Where(&dataset).Find(&datasetResult).Error
	if err != nil {
		return nil, err
	}

	courseDataset := []string{}

	for _, s := range datasetResult {
		courseDataset = append(courseDataset, s.DatasetName)
	}

	return courseDataset, nil
}

func (course *Course) GetPort(DB *gorm.DB) ([]Port, error) {
	port := Port{
		CourseID: course.ID,
	}

	portResult := []Port{}
	if err := DB.Where(&port).Find(&portResult).Error; err != nil {
		return nil, err
	}

	return portResult, nil
}

func (course *Course) Type(DB *gorm.DB) (string, error) {

	co := Course{
		Model: Model{
			ID: course.ID,
		},
	}

	if first := DB.Find(&co); first.Error != nil {
		if first.RecordNotFound() {
			// if not found in container course DB, this course type is VM
			return VM, nil
		}
		return "", first.Error
	}

	//find course in container course db, course type is container
	return CONTAINER, nil
}

func (course *Course) IsOwner(DB *gorm.DB, user string, provider string) (bool, error) {
	c := Course{
		Model: Model{
			ID: course.ID,
		},
		OauthUser: OauthUser{
			User:     user,
			Provider: provider,
		},
	}

	if result := DB.Where(c).Find(&c); result.Error != nil {
		return false, result.Error
	}

	log.Infof(fmt.Sprintf("course {%s} is owned by {%s:%s}", course.ID, user, provider))
	return true, nil

}

func QueryCourse(DB *gorm.DB, query interface{}, args ...interface{}) ([]Course, error) {
	// query course based on course condition
	results := []Course{}

	if err := DB.Where(query, args).Find(&results).Error; err != nil {
		log.Errorf("Query courses table fail: %s", err.Error())
		return nil, err
	}

	finalResult := []Course{}
	for _, c := range results {
		finalResult = append(finalResult, Course{
			Model: Model{
				ID:        c.ID,
				CreatedAt: c.CreatedAt,
			},
			Name:       c.Name,
			Level:      c.Level,
			CourseType: util.StringPtr(CONTAINER),
		})
	}

	return finalResult, nil
}
