package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	beta "github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/appsbeta"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/consts"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/config"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/db"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/util"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	"github.com/nchc-ai/oauth-provider/pkg/provider"
	rfstackmodel "github.com/nchc-ai/rfstack/model"
	"github.com/nitishm/go-rejson/v4"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ClientSet struct {
	*beta.BetaClient
}

func (c *ClientSet) Beta() *beta.BetaClient {
	return c.BetaClient
}

func NewClientset(KClientSet *kubernetes.Clientset, CourseCrdClient *versioned.Clientset,
	config *config.Config, DB *gorm.DB, provider provider.Provider, rh *rejson.Handler) *ClientSet {
	return &ClientSet{
		BetaClient: beta.NewClient(KClientSet, CourseCrdClient, config, DB, provider, rh),
	}
}

func NewKClients(config *config.Config) (*kubernetes.Clientset, *versioned.Clientset, error) {

	kConfig, err := util.GetConfig(
		config.APIConfig.IsOutsideCluster,
		config.K8SConfig.KUBECONFIG,
	)

	if err != nil {
		log.Fatalf("create kubenetes config fail: %s", err.Error())
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(kConfig)
	if err != nil {
		log.Fatalf("create kubenetes client set fail: %s", err.Error())
		return nil, nil, err
	}

	crdclient, err := versioned.NewForConfig(kConfig)
	if err != nil {
		log.Fatalf("create Course CRD client set fail: %s", err.Error())
		return nil, nil, err
	}

	// create namespace/pvc for teacher & public
	for _, v := range []string{consts.PUBLIC_CLASSROOM, consts.TEACHER_CLASSROOM} {

		_, err = clientset.CoreV1().Namespaces().Get(context.Background(), v, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = clientset.CoreV1().Namespaces().Create(
				context.Background(),
				&v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: v,
					},
				},
				metav1.CreateOptions{})
		}

		if err != nil {
			errStr := fmt.Sprintf("create kubernetes namespace %s fail: %s", v, err.Error())
			log.Error(errStr)
			return nil, nil, err
		}

		// reuse Classroom helper function
		cc := beta.Classroom{
			Config:     config,
			KClientSet: clientset,
		}
		err = cc.CreateDataSetPVC(v)

		if err != nil {
			errStr := fmt.Sprintf("create dataset PVC for namespace %s fail: %s", v, err.Error())
			log.Error(errStr)
			return nil, nil, err
		}
	}

	return clientset, crdclient, nil
}

func NewDBClient(config *config.Config) (*gorm.DB, error) {

	dbArgs := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True",
		config.DBConfig.Username,
		config.DBConfig.Password,
		config.DBConfig.Host,
		config.DBConfig.Port,
		config.DBConfig.Database,
	)

	DB, err := gorm.Open("mysql", dbArgs)

	if err != nil {
		log.Fatalf("create database client fail: %s", err.Error())
		return nil, err
	}

	// create tables
	course := &db.Course{}
	job := &db.Job{}
	dateset := &db.Dataset{}
	port := &db.Port{}
	courseid := &db.CourseID{}
	user := &db.User{}
	audit := &db.Audit{}

	classroomInfo := &db.ClassRoomInfo{}
	classroomInfo1 := &db.ClassRoomInfo{}
	classroomCourse := &db.ClassRoomCourseRelation{}
	classroomSchedule := &db.ClassRoomScheduleRelation{}
	classroomSchedule1 := &db.ClassRoomScheduleRelation{}
	classroomStudent := &db.ClassRoomStudentRelation{}
	classroomTeacher := &db.ClassRoomTeacherRelation{}
	classroomCalendar := &db.ClassRoomCalendarRelation{}
	classroomSelected := &db.ClassRoomSelectedOptionRelation{}

	DB.AutoMigrate(course, job, dateset, port, courseid, user, audit)

	DB.AutoMigrate(classroomInfo, classroomCourse, classroomSchedule, classroomStudent, classroomTeacher,
		classroomCalendar, classroomSelected)

	// Initialize aitrain-public classroom.
	// This classroom can be edited by admin.
	// If this classroom is not found (should be occur at system setup), create a new one with default configuration.
	// If this classroom is already exist and has been edited, DON'T create again.
	if r := DB.First(classroomInfo, db.ClassRoomInfo{
		Model: db.Model{
			ID: consts.PUBLIC_CLASSROOM,
		},
	}); r.Error != nil {
		if r.RecordNotFound() {
			DB.FirstOrCreate(classroomInfo, db.ClassRoomInfo{
				Model: db.Model{
					ID: consts.PUBLIC_CLASSROOM,
				},
				Name:                "Public Course",
				Description:         "Put public course in this Classroom for everyone access",
				IsPublic:            db.TRUE,
				ScheduleDescription: "完全不限時間",
				StartAt:             time.Now().Format("2006-01-02"),
				EndAt:               time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
				SelectedType:        util.Int32Ptr(2),
			})
		} else {
			log.Error(err.Error())
		}
	}

	if r := DB.First(classroomSelected, db.ClassRoomSelectedOptionRelation{
		ClassroomID: consts.PUBLIC_CLASSROOM,
	}); r.Error != nil {
		if r.RecordNotFound() {
			DB.FirstOrCreate(classroomSelected, db.ClassRoomSelectedOptionRelation{
				ClassroomID: consts.PUBLIC_CLASSROOM,
				Label:       "不限時間",
				Value:       "不限時間",
			})
		} else {
			log.Error(err.Error())
		}
	}

	if r := DB.First(classroomSchedule, db.ClassRoomScheduleRelation{
		ClassroomID: consts.PUBLIC_CLASSROOM,
	}); r.Error != nil {
		if r.RecordNotFound() {
			DB.FirstOrCreate(classroomSchedule, db.ClassRoomScheduleRelation{
				ClassroomID: consts.PUBLIC_CLASSROOM,
				Schedule:    "* * * * * *",
			})
		} else {
			log.Error(err.Error())
		}
	}

	// Initialize aitrain-teacher classroom.
	// This classroom is not visible or modified by any user/admin.
	DB.FirstOrCreate(classroomInfo1, db.ClassRoomInfo{
		Model: db.Model{
			ID: consts.TEACHER_CLASSROOM,
		},
		Name:        consts.TEACHER_CLASSROOM,
		Description: "Dummy classroom for teacher launch his job, should not been seen and edited any one",
		IsPublic:    db.FALSE,
	})

	DB.FirstOrCreate(classroomSchedule1, db.ClassRoomScheduleRelation{
		ClassroomID: consts.TEACHER_CLASSROOM,
		Schedule:    "* * * * * *",
	})

	// add foreign key
	DB.Model(job).
		AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT").
		AddForeignKey("classroom_id", "classroomInfo(id)", "CASCADE", "RESTRICT")
	DB.Model(dateset).AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT")
	DB.Model(port).AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT")
	DB.Model(course).AddForeignKey("id", "courseid(id)", "CASCADE", "RESTRICT")

	DB.Model(classroomCourse).
		AddForeignKey("classroom_id", "classroomInfo(id)", "CASCADE", "RESTRICT").
		AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT")

	DB.Model(classroomSchedule).AddForeignKey("classroom_id", "classroomInfo(id)", "CASCADE", "RESTRICT")
	DB.Model(classroomTeacher).AddForeignKey("classroom_id", "classroomInfo(id)", "CASCADE", "RESTRICT")
	DB.Model(classroomStudent).AddForeignKey("classroom_id", "classroomInfo(id)", "CASCADE", "RESTRICT")
	DB.Model(classroomSelected).AddForeignKey("classroom_id", "classroomInfo(id)", "CASCADE", "RESTRICT")
	DB.Model(classroomCalendar).AddForeignKey("classroom_id", "classroomInfo(id)", "CASCADE", "RESTRICT")

	// vmCourse & vmJob Table should be created by rfstack, we create the tables here to make sure
	// they available when query for classroom.
	vmCourse := &rfstackmodel.Course{}
	vmJob := &rfstackmodel.Job{}
	DB.AutoMigrate(vmCourse, vmJob)
	DB.Model(vmCourse).AddForeignKey("id", "courseid(id)", "CASCADE", "RESTRICT")
	DB.Model(vmJob).AddForeignKey("course_id", "courseid(id)", "CASCADE", "RESTRICT")

	uid := strings.Split(config.APIConfig.UidRange, "/")
	DB.Exec(fmt.Sprintf("ALTER TABLE %s AUTO_INCREMENT = %s", user.TableName(), uid[0]))
	return DB, nil
}
