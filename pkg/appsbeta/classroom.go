package beta

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/consts"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/config"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/db"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/util"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Classroom struct {
	DB              *gorm.DB
	Config          *config.Config
	KClientSet      *kubernetes.Clientset
	CourseCrdClient *versioned.Clientset
}

// @Summary Upload account csv file
// @Description Upload account csv file
// @Tags Classroom
// @Accept  multipart/form-data
// @Produce  json
// @Param file formData file true "account file"
// @Success 200 {object} docs.UploadUserResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/classroom/upload [post]
func (cm *Classroom) UploadUserAccount(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		errStr := fmt.Sprintf("Upload account csv file fail: %s", err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusBadRequest, errStr)
		return
	}

	f, err := file.Open()
	if err != nil {
		errStr := fmt.Sprintf("Open account csv file fail: %s", err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusBadRequest, errStr)
		return
	}
	reader := csv.NewReader(bufio.NewReader(f))

	users := []model.UserLabelValue{}
	for {
		// csv format is name,email
		line, err := reader.Read()

		if err == io.EOF {
			break
		} else if err != nil {
			errStr := fmt.Sprintf("Read account csv file fail: %s", err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusBadRequest, errStr)
			return
		}
		users = append(users, model.UserLabelValue{
			Name:  line[0],
			Email: line[1],
		})
	}

	c.JSON(http.StatusOK, model.UploaduserResponse{
		Error: false,
		Users: users,
	})
}

// @Summary Add new Classroom information
// @Description Add new Classroom information into database
// @Tags Classroom
// @Accept  json
// @Produce json
// @Param classroom body docs.AddClassroom true "classroom information"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/classroom/create [post]
func (cm *Classroom) Add(c *gin.Context) {

	provider, exist := c.Get("Provider")
	if exist == false {
		provider = ""
	}

	var req db.ClassRoomInfo

	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	classrromId := strings.Join([]string{consts.NS_prefix, uuid.New().String()}, "-")
	req.ID = classrromId

	//use transaction avoid partial update
	tx := cm.DB.Begin()

	if err := req.NewEntry(tx); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("Failed to create classroomInfo entry: %s", err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_INFO_FMT, req.Name)
		return
	}

	course := db.ClassRoomCourseRelation{
		ClassroomID: classrromId,
	}
	if err := course.NewEntry(tx, req.CourseList); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create course of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_COURSE_FMT, req.Name)
		return
	}

	schedule := db.ClassRoomScheduleRelation{
		ClassroomID: classrromId,
	}
	if err := schedule.NewEntry(tx, req.ScheduleTime.CronFormat); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create schedule of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_SCHEDULE_FMT, req.Name)
		return
	}

	teacher := db.ClassRoomTeacherRelation{
		ClassRoomUser: db.ClassRoomUser{
			ClassroomID: classrromId,
		},
	}
	if err := teacher.NewEntry(tx, req.TeacherList, provider.(string)); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create teacher of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_TEACHER_FMT, req.Name)
		return
	}

	student := db.ClassRoomStudentRelation{
		ClassRoomUser: db.ClassRoomUser{
			ClassroomID: classrromId,
		},
	}
	if err := student.NewEntry(tx, req.StudentList, provider.(string)); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create student of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_STUDENT_FMT, req.Name)
		return
	}

	calendar := db.ClassRoomCalendarRelation{
		ClassroomID: classrromId,
	}
	if err := calendar.NewEntry(tx, req.CalendarTime); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create calendar of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_CALENDAR_FMT, req.Name)
		return
	}

	opts := db.ClassRoomSelectedOptionRelation{
		ClassroomID: classrromId,
	}

	if err := opts.NewEntry(tx, req.ScheduleTime.SelectedOption); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create time info of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	_, err = cm.KClientSet.CoreV1().Namespaces().Create(
		context.Background(),
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   classrromId,
				Labels: map[string]string{consts.NamespaceLabelInstance: cm.Config.APIConfig.NamespacePrefix},
			},
		},
		metav1.CreateOptions{},
	)

	if err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create kubernetes namespace for classroom %s fail: %s", classrromId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_NS_FMT, req.Name)
		return
	}

	// create all dataset pv & pvc into new namespace
	err = cm.CreateDataSetPVC(classrromId)
	if err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create dataset PVC for classroom %s namespace fail: %s", classrromId, err.Error())
		log.Error(errStr)
		err2 := cm.KClientSet.CoreV1().Namespaces().Delete(context.Background(), classrromId, metav1.DeleteOptions{})
		if err2 != nil {
			log.Errorf("Rollback namespace {%s} creation fail: %s", classrromId, err2.Error())
		}

		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_DATASET_FMT, req.Name)
		return
	}

	// todo: update existing secret when aitrain-system secret is being update
	if err = cm.copySecretFromSystem(classrromId); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create secret for classroom %s namespace fail: %s", classrromId, err.Error())
		log.Error(errStr)
		err2 := cm.KClientSet.CoreV1().Namespaces().Delete(context.Background(), classrromId, metav1.DeleteOptions{})
		if err2 != nil {
			log.Errorf("Rollback namespace {%s} creation fail: %s", classrromId, err2.Error())
		}
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_SECRET_FMT, req.Name)
		return
	}

	// create role to use scc, this is must have if run on OCP. But it's fine to create on K8S also.
	if _, err = cm.KClientSet.RbacV1().Roles(classrromId).Create(
		context.Background(), newRoleForSCC(classrromId), metav1.CreateOptions{}); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create role for classroom %s namespace fail: %s", classrromId, err.Error())
		log.Error(errStr)
		err2 := cm.KClientSet.CoreV1().Namespaces().Delete(context.Background(), classrromId, metav1.DeleteOptions{})
		if err2 != nil {
			log.Errorf("Rollback namespace {%s} creation fail: %s", classrromId, err2.Error())
		}
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_ROLE_FMT, req.Name)
		return
	}

	if _, err = cm.KClientSet.RbacV1().RoleBindings(classrromId).Create(
		context.Background(), newRoleBinding(classrromId), metav1.CreateOptions{}); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("create role for classroom %s namespace fail: %s", classrromId, err.Error())
		log.Error(errStr)
		err2 := cm.KClientSet.CoreV1().Namespaces().Delete(context.Background(), classrromId, metav1.DeleteOptions{})
		if err2 != nil {
			log.Errorf("Rollback namespace {%s} creation fail: %s", classrromId, err2.Error())
		}
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_CREATE_ROLE_FMT, req.Name)
		return
	}

	tx.Commit()
	RespondWithOk(c, "Classroom %s created successfully", req.Name)
}

// @Summary Get one classroom information by id
// @Description Get one classroom information by id
// @Tags Classroom
// @Accept  json
// @Produce  json
// @Param id path string true "classroom uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GetClassroomResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/classroom/get/{id} [get]
func (cm *Classroom) Get(c *gin.Context) {
	classroomId := c.Param("id")

	if classroomId == "" {
		log.Errorf("Empty classroom id")
		RespondWithError(c, http.StatusBadRequest, "Empty classroom id")
		return
	}

	classroom := db.ClassRoomInfo{
		Model: db.Model{
			ID: classroomId,
		},
	}

	cmInfo, err := classroom.GetClassRoomDetail(cm.DB)
	if err != nil {
		errStr := fmt.Sprintf("Query detail of classroom {%s} fail: %s", classroomId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	count, err := classroom.GetStudentCount(cm.DB)
	if err != nil {
		errStr := fmt.Sprintf("Query student count of classroom {%s} fail: %s", classroomId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	tlist, err := classroom.GetTeacherList(cm.DB)
	if err != nil {
		errStr := fmt.Sprintf("Query teacher name of classroom {%s} fail: %s", classroomId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	slist, err := classroom.GetStudentList(cm.DB)
	if err != nil {
		errStr := fmt.Sprintf("Query student list of classroom {%s} fail: %s", classroomId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	schedule, err := cmInfo.GetSchedule(cm.DB)
	if err != nil {
		errStr := fmt.Sprintf("Query schedule of classroom {%s} fail: %s", classroomId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	calendar, err := cmInfo.GetCalendar(cm.DB)
	if err != nil {
		errStr := fmt.Sprintf("Query calendar of classroom {%s} fail: %s", classroomId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	cmInfo.StudentCount = util.Int32Ptr(count)
	cmInfo.TeacherList = tlist
	cmInfo.StudentList = slist
	cmInfo.ScheduleTime = schedule
	cmInfo.CalendarTime = calendar

	c.JSON(http.StatusOK, model.GetClassroomResponse{
		Error:     false,
		Classroom: *cmInfo,
	})
}

// @Summary update exist Classroom information
// @Description update existClassroom information
// @Tags Classroom
// @Accept  json
// @Produce json
// @Param classroom body docs.UpdateClassroom true "classroom information"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/classroom/update [put]
func (cm *Classroom) Update(c *gin.Context) {

	provider, exist := c.Get("Provider")
	if exist == false {
		provider = ""
	}

	var req db.ClassRoomInfo

	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	//use transaction avoid partial update
	tx := cm.DB.Begin()

	// 1. Update Classroom basic info
	if err := tx.Model(&req).Updates(
		db.ClassRoomInfo{
			Name:                req.Name,
			Description:         req.Description,
			ScheduleDescription: req.ScheduleTime.Description,
			SelectedType:        req.ScheduleTime.SelectedType,
			StartAt:             req.ScheduleTime.StartDate,
			EndAt:               req.ScheduleTime.EndDate,
		}).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update classroom {%s} fail: %s", req.ID, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_UPDATE_INFO_FMT, req.Name)
		return
	}

	// MySQL doesn't have bool type, this is workaround for update boolean type field
	if err := tx.Model(&req).
		UpdateColumn("is_public", db.Bool2Sqlbool(req.IsPublicBool)).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update classroom {%s} public field fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_UPDATE_INFO_FMT, req.Name)
		return
	}

	// 2. Update Classroom relationship info
	student := db.ClassRoomStudentRelation{
		ClassRoomUser: db.ClassRoomUser{
			ClassroomID: req.ID,
		},
	}
	if req.ID != consts.PUBLIC_CLASSROOM {
		if err := student.Update(tx, req.StudentList, provider.(string)); err != nil {
			tx.Rollback()
			errStr := fmt.Sprintf("update student of classroom {%s} fail: %s", req.ID, err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_UPDATE_STUDENT_FMT, req.Name)
			return
		}
	}

	teacher := db.ClassRoomTeacherRelation{
		ClassRoomUser: db.ClassRoomUser{
			ClassroomID: req.ID,
		},
	}
	if req.ID != consts.PUBLIC_CLASSROOM {
		if err := teacher.Update(tx, req.TeacherList, provider.(string)); err != nil {
			tx.Rollback()
			errStr := fmt.Sprintf("update teacher of classroom {%s} fail: %s", req.ID, err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_UPDATE_TEACHER_FMT, req.Name)
			return
		}
	}

	schedule := db.ClassRoomScheduleRelation{
		ClassroomID: req.ID,
	}
	if err := schedule.Update(tx, req.ScheduleTime.CronFormat); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update schedule of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_UPDATE_SCHEDULE_FMT, req.Name)
		return
	}

	course := db.ClassRoomCourseRelation{
		ClassroomID: req.ID,
	}
	if err := course.Update(tx, req.CourseList); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update course of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_UPDATE_COURSE_FMT, req.Name)
		return
	}

	calendar := db.ClassRoomCalendarRelation{
		ClassroomID: req.ID,
	}

	if req.CalendarTime != nil {
		if err := calendar.Update(tx, req.CalendarTime); err != nil {
			tx.Rollback()
			errStr := fmt.Sprintf("update calendar of classroom {%s} fail: %s", req.ID, err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_UPDATE_CALENDAR_FMT, req.Name)
			return
		}
	}

	opts := db.ClassRoomSelectedOptionRelation{
		ClassroomID: req.ID,
	}

	if err := opts.Update(tx, req.ScheduleTime.SelectedOption); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update time info of classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if err := cm.updateCourseCRD(req); err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update CRD schedule spec under classroom {%s} fail: %s", req.ID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	tx.Commit()
	RespondWithOk(c, "Classroom %s update successfully", req.ID)
}

// @Summary List all public and non-public classroom
// @Description List all public and non-public classroom
// @Tags Classroom
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.SimpleListClassroomResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/classroom/list [get]
func (cm *Classroom) ListAll(c *gin.Context) {

	var classroomResult []db.ClassRoomInfo
	var err error

	if classroomResult, err = db.GetAllClassroom(cm.DB); err != nil {
		log.Error(fmt.Sprintf("List all classrooms fail: %s", err.Error()))
		RespondWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	for i, cminfo := range classroomResult {

		tlist, err := cminfo.GetTeacherList(cm.DB)

		if err != nil {
			errStr := fmt.Sprintf("Query teacher name of classroom {%s} fail: %s", cminfo.ID, err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		calendar, err := cminfo.GetCalendar(cm.DB)

		if err != nil {
			errStr := fmt.Sprintf("Query calendar of classroom {%s} fail: %s", cminfo.ID, err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		classroomResult[i].TeacherList = tlist
		classroomResult[i].IsPublicBool = db.Sqlbool2Bool(cminfo.IsPublic)
		classroomResult[i].CalendarTime = calendar

	}

	c.JSON(http.StatusOK, model.ListClassroomResponse{
		Error:      false,
		Classrooms: classroomResult,
	})
}

// @Summary List someone's all public classroom
// @Description List someone's all public classroom
// @Tags Classroom
// @Accept  json
// @Produce  json
// @Param list_user body docs.OauthUser true "search user public classroom"
// @Success 200 {object} docs.ListClassroomResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/classroom/list [post]
func (cm *Classroom) List(c *gin.Context) {

	provider, exist := c.Get("Provider")
	if exist == false {
		provider = db.DEFAULT_PROVIDER
	}

	req := db.ClassRoomStudentRelation{}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}
	req.Provider = provider.(string)

	if req.User == "" {
		log.Errorf("Empty user name")
		RespondWithError(c, http.StatusBadRequest, "Empty user name")
		return
	}

	var classroomResult []db.ClassRoomInfo
	if classroomResult, err = db.GetUserClassroom(cm.DB, req.User, req.Provider); err != nil {
		log.Error(fmt.Sprintf("Query user {%s}'s classroom fail: %s", req.User, err.Error()))
		RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	for i, cminfo := range classroomResult {
		tlist, err := cminfo.GetTeacherList(cm.DB)

		if err != nil {
			errStr := fmt.Sprintf("Query teacher name of classroom {%s} fail: %s", cminfo.ID, err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		count, err := cminfo.GetStudentCount(cm.DB)

		if err != nil {
			errStr := fmt.Sprintf("Query student number of classroom {%s} fail: %s", cminfo.ID, err.Error())
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		//schedule, err := cminfo.GetSchedule(cm.DB)
		//if err != nil {
		//	errStr := fmt.Sprintf("Query schedule of classroom {%s} fail: %s", cminfo.ID, err.Error())
		//	log.Error(errStr)
		//	RespondWithError(c, http.StatusInternalServerError, errStr)
		//	return
		//}

		classroomResult[i].TeacherList = tlist
		classroomResult[i].StudentCount = util.Int32Ptr(count)
		//classroomResult[i].ScheduleTime = schedule
	}

	c.JSON(http.StatusOK, model.ListClassroomResponse{
		Error:      false,
		Classrooms: classroomResult,
	})
}

// @Summary Delete one classroom
// @Description Delete one classroom
// @Tags Classroom
// @Accept  json
// @Produce  json
// @Param id path string true "classroom uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/classroom/delete/{id} [delete]
func (cm *Classroom) Delete(c *gin.Context) {

	classroomID := c.Param("id")

	if classroomID == "" {
		RespondWithError(c, http.StatusBadRequest,
			"Classroom Id is not found")
		return
	}

	if classroomID == consts.PUBLIC_CLASSROOM {
		log.Warningf("default Classroom {%s} can NOT be delete", consts.PUBLIC_CLASSROOM)
		RespondWithError(c, http.StatusBadRequest, consts.ERROR_CLASSROOM_DELETE_DEFAULT_FMT, consts.PUBLIC_CLASSROOM)
		return
	}

	classroom := db.ClassRoomInfo{
		Model: db.Model{
			ID: classroomID,
		},
	}

	cmInfo, err := classroom.GetClassRoomDetail(cm.DB)
	if err != nil {
		errStr := fmt.Sprintf("Query detail of classroom {%s} fail: %s", classroomID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	// todo: resource created in rfstack should be also deleted.
	// rfstack course need to be found before ClassRoomInfo delete

	if err := cm.DB.Unscoped().Delete(&classroom).Error; err != nil {
		errStr := fmt.Sprintf("Delete Classroom {%s} from database fail: %s", classroomID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	deletePolicy := metav1.DeletePropagationForeground

	// delete namespace scope resource
	err = cm.KClientSet.CoreV1().Namespaces().Delete(
		context.Background(), classroomID, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
	if err != nil {
		errStr := fmt.Sprintf("Delete Classroom namespace {%s} fail: %s", classroomID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_DELETE_NS_FMT, cmInfo.Name)
		return
	}

	// delete non-namespace scope resource, i.e. PV
	lblSelector := labels.FormatLabels(map[string]string{"classroom": classroomID})
	pvs, err := cm.KClientSet.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{LabelSelector: lblSelector})

	if err != nil {
		errStr := fmt.Sprintf("List PV belong to Classroom {%s} fail: %s", classroomID, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_CLASSROOM_DELETE_DATASET_FMT, cmInfo.Name)
		return
	}

	for _, pv := range pvs.Items {
		err := cm.KClientSet.CoreV1().PersistentVolumes().Delete(
			context.Background(), pv.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
		if err != nil {
			errStr := fmt.Sprintf("Delete PV {%s} belong to Classroom {%s} fail. Skip: %s", pv.Name, classroomID, err.Error())
			log.Warning(errStr)
			continue
		}
	}

	RespondWithOk(c, "Classroom {%s} is deleted successfully", classroomID)
}

// private helper func
func (cm *Classroom) CreateDataSetPVC(namespace string) error {

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			consts.NamespaceLabelInstance: cm.Config.APIConfig.NamespacePrefix,
		},
	}
	pvcs, err := cm.KClientSet.CoreV1().
		PersistentVolumeClaims(metav1.NamespaceDefault).
		List(context.Background(),
			metav1.ListOptions{
				LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
			})
	if err != nil {
		return err
	}

	for _, inPVC := range pvcs.Items {

		// if pvs is not dataset, skip
		if !strings.HasPrefix(inPVC.Name, consts.DatasetPVCPrefix) {
			continue
		}

		// prepare annotation map
		annotation := map[string]string{
			"nchc.ai/link-data":         "true",
			"nchc.ai/src-pvc-namespace": metav1.NamespaceDefault,
			"nchc.ai/src-pvc-name":      inPVC.Name,
		}

		label := map[string]string{
			"type": "dataset",
		}

		// create outPVC use outPV in namespace
		outPVC := v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:        inPVC.Name,
				Namespace:   namespace,
				Annotations: annotation,
				Labels:      label,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadOnlyMany,
				},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Mi"),
					},
				},
				StorageClassName: util.StringPtr(cm.Config.K8SConfig.StorageClass),
			},
		}

		_, err = cm.KClientSet.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), outPVC.Name, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			_, err = cm.KClientSet.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), &outPVC, metav1.CreateOptions{})
			log.Infof("Create PVC {%s} in namespace {%s}", inPVC.Name, namespace)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (cm *Classroom) RemoveDataSetPVC(namespace string) error {
	targetLabelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			consts.NamespaceLabelInstance: cm.Config.APIConfig.NamespacePrefix,
		},
	}

	targetPVCs, err := cm.KClientSet.CoreV1().
		PersistentVolumeClaims(metav1.NamespaceDefault).
		List(context.Background(),
			metav1.ListOptions{
				LabelSelector: labels.Set(targetLabelSelector.MatchLabels).String(),
			})

	if err != nil {
		return err
	}

	existLabelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"type": "dataset",
		},
	}

	existingPVC, err := cm.KClientSet.CoreV1().PersistentVolumeClaims(namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: labels.Set(existLabelSelector.MatchLabels).String(),
		})

	if err != nil {
		return err
	}

	for _, existPVC := range existingPVC.Items {

		found := false
		for _, target := range targetPVCs.Items {
			if existPVC.Name == target.Name {
				found = true
				break
			}
		}

		if !found {
			err := cm.KClientSet.CoreV1().PersistentVolumeClaims(namespace).
				Delete(context.Background(), existPVC.Name, metav1.DeleteOptions{})
			log.Infof("Delete PVC {%s} in namespace {%s}", existPVC.Name, namespace)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (cm *Classroom) copySecretFromSystem(namespace string) error {
	// get from aitrain-system
	origSec, err := cm.KClientSet.CoreV1().Secrets(consts.AiTrainSystemNamespace).Get(
		context.Background(), consts.TlsSecretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	_, err = cm.KClientSet.CoreV1().Secrets(namespace).Get(context.Background(), consts.TlsSecretName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		// if secret is not found, create one
		newSec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      consts.TlsSecretName,
			},
			Data: make(map[string][]byte),
			Type: corev1.SecretTypeTLS,
		}

		for k, v := range origSec.Data {
			newSec.Data[k] = v
		}

		_, err := cm.KClientSet.CoreV1().Secrets(namespace).Create(context.Background(), newSec, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

func (cm *Classroom) updateCourseCRD(req db.ClassRoomInfo) error {
	namespace := req.ID
	cronStrings := req.ScheduleTime.CronFormat

	crds, err := cm.CourseCrdClient.NchcV1alpha1().Courses(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, c := range crds.Items {
		clone := c.DeepCopy()
		clone.Spec.Schedule = cronStrings
		_, err := cm.CourseCrdClient.NchcV1alpha1().Courses(namespace).Update(context.Background(), clone, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func newRoleForSCC(namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.SccRoleName,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"security.openshift.io"},
				ResourceNames: []string{"anyuid", "hostmount-anyuid"},
				Resources:     []string{"securitycontextconstraints"},
				Verbs:         []string{"use"},
			},
		},
	}
}

func newRoleBinding(namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.SccRoleBindingName,
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Name:     consts.SccRoleName,
			Kind:     "Role",
		},
	}
}
