package beta

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dghubble/sling"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/consts"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/common"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/db"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/util"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	rfstackmodel "github.com/nchc-ai/rfstack/model"
	"github.com/nitishm/go-rejson/v4"
)

type Course struct {
	DB              *gorm.DB
	Redis           *rejson.Handler
	CourseCrdClient *versioned.Clientset
	rfStackBase     *sling.Sling
}

// @Summary Add new course information
// @Description Add new course information into database
// @Tags Course
// @Accept  json
// @Produce  json
// @Param course body docs.AddCourseBeta true "course information"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/course/create [post]
func (co *Course) Add(c *gin.Context) {
	var req db.Course
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.User == "" {
		log.Errorf("user field in request cannot be empty")
		RespondWithError(c, http.StatusBadRequest, "user field in request cannot be empty")
		return
	}

	hasNilValue, errs := req.CheckNilField(consts.COURSE_CREATE_ERROR)
	if hasNilValue {
		log.Errorf(errs[0].Error())
		RespondWithError(c, http.StatusBadRequest, errs[1].Error())
		return
	}

	// add course information in DB
	courseID := uuid.New().String()

	provider, exist := c.Get("Provider")
	if !exist {
		provider = db.DEFAULT_PROVIDER
	}

	//use transaction avoid partial update
	tx := co.DB.Begin()

	newCourseId := db.CourseID{
		Model: db.Model{
			ID: courseID,
		},
	}
	err = tx.Create(&newCourseId).Error

	if err != nil {
		tx.Rollback()
		log.Errorf("Failed to register new course id: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_CREATE_INFO_FMT, req.Name)
		return
	}

	newCourse := db.Course{
		Model: db.Model{
			ID: courseID,
		},
		OauthUser: db.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
		Introduction: req.Introduction,
		Name:         req.Name,
		Image:        req.ImageLV.Value,
		Level:        req.Level,
		Gpu:          req.GpuLV.Value,
		WritablePath: req.WritablePath,
		AccessType:   req.AccessType,
	}

	err = tx.Create(&newCourse).Error

	if err != nil {
		tx.Rollback()
		log.Errorf("Failed to create course information: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_CREATE_INFO_FMT, req.Name)
		return
	}

	// add dataset required by course in DB
	datasets := *req.Datasets
	for _, data := range datasets {
		newDataset := db.Dataset{
			CourseID:    courseID,
			DatasetName: data.Value,
		}
		err = tx.Create(&newDataset).Error
		if err != nil {
			tx.Rollback()
			log.Errorf("Failed to create course-dataset information in DB: %s", err.Error())
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_CREATE_DATASET_FMT, req.Name)
			return
		}
	}

	// add course required port number in DB
	ports := *req.Ports

	for _, port := range ports {

		if port.Name == "" {
			tx.Rollback()
			log.Errorf("Empty Port name is not allowed")
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_CREATE_PORT_EMPTY_FMT, req.Name)
			return
		}

		if e := isDNS1035Label(port.Name); e != nil {
			tx.Rollback()
			errMsg := fmt.Sprintf("%s is invalid: %s", port.Name, e.Error())
			log.Errorf(errMsg)
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_CREATE_PORT_INVLID_FMT, req.Name)
			return
		}

		newPort := db.Port{
			CourseID: courseID,
			Name:     strings.TrimSpace(port.Name),
			Port:     port.Port,
		}

		err = tx.Create(&newPort).Error
		if err != nil {
			tx.Rollback()
			log.Errorf("Failed to create course-port information in DB: %s", err.Error())
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_CREATE_PORT_FMT, req.Name)
			return
		}
	}

	tx.Commit()
	RespondWithOk(c, "Course %s created successfully", req.Name)
}

// @Summary Update course information
// @Description Update course information
// @Tags Course
// @Accept  json
// @Produce  json
// @Param course body docs.UpdateCourseBeta true "new course information"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/course/update [put]
func (co *Course) Update(c *gin.Context) {
	var req db.Course

	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.ID == "" {
		log.Errorf("Course id is empty")
		RespondWithError(c, http.StatusBadRequest, "Course id is empty")
		return
	}

	hasNilValue, errs := req.CheckNilField(consts.COURSE_UPDATE_ERROR)
	if hasNilValue {
		log.Errorf(errs[0].Error())
		RespondWithError(c, http.StatusBadRequest, errs[1].Error())
		return
	}

	findCourse := db.Course{
		Model: db.Model{
			ID: req.ID,
		},
	}

	if err = co.DB.First(&findCourse).Error; err != nil {
		errStr := fmt.Sprintf("find course {%s} fail: %s", req.ID, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	tx := co.DB.Begin()

	// update Course DB
	if err := tx.Model(&findCourse).Updates(
		db.Course{
			Introduction: req.Introduction,
			Name:         req.Name,
			Image:        req.ImageLV.Value,
			Level:        req.Level,
			Gpu:          req.GpuLV.Value,
			AccessType:   req.AccessType,
			WritablePath: req.WritablePath,
		}).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("update course {%s} information fail: %s", req.ID, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_UPDATE_INFO_FMT, req.Name)
		return
	}

	// update dataset required by course in DB
	// Step 1: delete dataset used by course
	if err = tx.Where("course_id = ?", req.ID).Delete(db.Dataset{}).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("Failed to delete course {%s} dataset information in DB: %s", req.ID, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_UPDATE_DATASET_FMT, req.Name)
	}

	// Step 2: create new datasets
	for _, data := range *req.Datasets {
		newDataset := db.Dataset{
			CourseID:    req.ID,
			DatasetName: data.Value,
		}
		if err = tx.Create(&newDataset).Error; err != nil {
			tx.Rollback()
			log.Errorf("Failed to create course-dataset information in DB: %s", err.Error())
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_UPDATE_DATASET_FMT, req.Name)
			return
		}
	}

	// update ports required by course
	// Step 1: delete ports used by course
	if err = tx.Where("course_id = ?", req.ID).Delete(db.Port{}).Error; err != nil {
		tx.Rollback()
		errStr := fmt.Sprintf("Failed to delete course {%s} port information in DB: %s", req.ID, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_UPDATE_PORT_FMT, req.Name)
	}
	// Step 2: create new ports
	for _, port := range *req.Ports {

		if port.Name == "" {
			tx.Rollback()
			log.Errorf("Empty Port name is not allowed")
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_UPDATE_PORT_EMPTY_FMT, req.Name)
			return
		}

		if e := isDNS1035Label(port.Name); e != nil {
			tx.Rollback()
			errMsg := fmt.Sprintf("%s is invalid: %s", port.Name, e.Error())
			log.Errorf(errMsg)
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_UPDATE_PORT_INVLID_FMT, req.Name)
			return
		}

		newPort := db.Port{
			CourseID: req.ID,
			Name:     strings.TrimSpace(port.Name),
			Port:     port.Port,
		}
		if err = tx.Create(&newPort).Error; err != nil {
			tx.Rollback()
			log.Errorf("Failed to create course-port information in DB: %s", err.Error())
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_UPDATE_PORT_FMT, req.Name)
			return
		}
	}

	tx.Commit()
	RespondWithOk(c, "Course {%s} update successfully", req.ID)
}

// @Summary Delete course information
// @Description All associated job, Deployment and svc in kubernetes are also deleted.
// @Tags Course
// @Accept  json
// @Produce  json
// @Param id path string true "course uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/course/delete/{id} [delete]
func (co *Course) Delete(c *gin.Context) {

	courseId := c.Param("id")

	if courseId == "" {
		RespondWithError(c, http.StatusBadRequest,
			"Course Id is not found")
		return
	}

	course, err := db.GetCourse(co.DB, courseId)
	if err != nil {
		log.Errorf("Fail to get course {%s} infomation: %s", courseId, err.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_DELETE_JOB_FMT, courseId)
		return
	}

	jobs := []db.Job{}
	// Step 1: Find all associated Deployment/Service
	if err := co.DB.Model(course).Related(&jobs).Error; err != nil {
		log.Errorf("Failed to find jobs belong to course {%s} information : %s", courseId, err.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_DELETE_JOB_FMT, course.Name)
		return
	}

	// Step 2: Stop deployment and service in kubernetes
	// 	 Step 2-1: delete jobs in DB.
	for _, j := range jobs {
		if errStr, err := j.DeleteCourseCRD(co.DB, co.Redis, co.CourseCrdClient, *j.ClassroomID); err != nil {
			log.Error(errStr)
			RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_DELETE_JOB_FMT, course.Name)
			return
		}
	}

	// Step 3: Delete course in DB.
	// Step 4: delete required dataset in DB. (With foreign key, this should be done automatically)
	courseid := db.CourseID{
		Model: db.Model{
			ID: courseId,
		},
	}

	err = co.DB.Unscoped().Delete(&courseid).Error
	if err != nil {
		log.Errorf("Failed to delete course {%s} information : %s", courseId, err.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_COURSE_DELETE_INFO_FMT, course.Name)
		return
	}

	RespondWithOk(c, "Course %s is deleted successfully, associated jobs are also deleted", courseId)
}

// @Summary Get one courses information by course id
// @Description Get one courses information by course id
// @Tags Course
// @Accept  json
// @Produce  json
// @Param id path string true "course uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GetCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/course/get/{id} [get]
func (co *Course) Get(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		log.Errorf("Empty course id")
		RespondWithError(c, http.StatusBadRequest, "Empty course id")
		return
	}

	course := db.Course{
		Model: db.Model{
			ID: id,
		},
	}

	result := db.Course{}
	if err := co.DB.Where(&course).First(&result).Error; err != nil {
		log.Errorf("Query courses {%s} fail: %s", id, err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Query courses {%s} fail: %s", id, err.Error())
		return
	}

	// query dataset table
	dataset := db.Dataset{
		CourseID: id,
	}
	datasetResult := []db.Dataset{}
	if err := co.DB.Where(&dataset).Find(&datasetResult).Error; err != nil {
		log.Errorf("Query course {%s} datasets fail: %s", id, err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Query course {%s} datasets fail: %s", id, err.Error())
		return
	}

	result.GpuLV = &common.LabelIntValue{
		Label: fmt.Sprintf("%d", *result.Gpu),
		Value: result.Gpu,
	}

	result.ImageLV = &common.LabelValue{
		Label: result.Image,
		Value: result.Image,
	}

	courseDataset := []common.LabelValue{}
	for _, s := range datasetResult {
		// https://gitlab.com/nchc-ai/AI-Eduational-Platform/issues/18
		// should remove "dataset-" prefix in dataset name (i.e. PVC name)
		dataset_name := strings.SplitN(s.DatasetName, "-", 2)
		if len(dataset_name) != 2 || dataset_name[0] != "dataset" {
			log.Warning(fmt.Sprintf("%s doesn't start with 'dataset-', NOT valided dataset name, skip", s.DatasetName))
			continue
		}
		courseDataset = append(courseDataset, common.LabelValue{
			Label: dataset_name[1],
			Value: s.DatasetName,
		})
	}
	result.Datasets = &courseDataset

	// query port table
	port := db.Port{
		CourseID: id,
	}
	portResult := []db.Port{}
	if err := co.DB.Where(&port).Find(&portResult).Error; err != nil {
		log.Errorf("Query course {%s} ports fail: %s", id, err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Query course {%s} ports fail: %s", id, err.Error())
		return
	}
	result.Ports = &portResult

	c.JSON(http.StatusOK, model.GetCourseResponse{
		Error:  false,
		Course: result,
	})
}

// @Summary course is executed by vm or container
// @Description course is executed by vm or container
// @Tags Course
// @Accept  json
// @Produce  json
// @Param id path string true "course uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.CourseTypeResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/course/type/{id} [get]
func (co *Course) CourseType(c *gin.Context) {
	id := c.Param("id")

	if id == "" {
		log.Errorf("Empty course id")
		RespondWithError(c, http.StatusBadRequest, "Empty course id")
		return
	}

	course := db.Course{
		Model: db.Model{
			ID: id,
		},
	}

	courseType, err := course.Type(co.DB)

	if err != nil {
		errStr := fmt.Sprintf("Query course {%s} type fail: %s", id, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	switch courseType {
	case db.CONTAINER:
		c.JSON(http.StatusOK, model.CourseTypeResponse{
			Error: false,
			Type:  db.CONTAINER,
		})
		return
	case db.VM:
		c.JSON(http.StatusOK, model.CourseTypeResponse{
			Error: false,
			Type:  db.VM,
		})
		return
	}
}

// @Summary List someone's all courses information
// @Description List someone's all courses information
// @Tags Course
// @Accept  json
// @Produce  json
// @Param list_user body docs.OauthUser true "search user course"
// @Success 200 {object} docs.ListCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/course/list [post]
func (co *Course) ListUserCourse(c *gin.Context) {
	provider, exist := c.Get("Provider")
	if exist == false {
		provider = db.DEFAULT_PROVIDER
	}

	req := db.Course{}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.User == "" {
		log.Errorf("Empty user name")
		RespondWithError(c, http.StatusBadRequest, "Empty user name")
		return
	}

	course := db.Course{
		OauthUser: db.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
	}

	results, err := db.QueryCourse(co.DB, course)

	if err != nil {
		errStr := fmt.Sprintf("query user {%s} course fail: %s", req.User, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if co.rfStackBase != nil {
		rfstackReq := rfstackmodel.Course{
			OauthUser: rfstackmodel.OauthUser{
				User: req.User,
			},
		}

		token := c.GetHeader("Authorization")
		err := invokeRfStack(co.rfStackBase.Set("Authorization", token).
			BodyJSON(&rfstackReq).Post("/v1/course/list"), &results)

		if err != nil {
			errStr := fmt.Sprintf("fetch vm course fail: %s", err.Error())
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}
	}

	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: results,
	})

}

// @Summary List basic or advance courses information
// @Description List basic or advance courses information
// @Tags Course
// @Accept  json
// @Produce  json
// @Param level path string true "basic or advance"
// @Success 200 {object} docs.ListCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/course/level/{level} [get]
func (co *Course) ListLevelCourse(c *gin.Context) {
	level := c.Param("level")

	if level == "" {
		log.Errorf("empty level string")
		RespondWithError(c, http.StatusBadRequest, "empty level string")
		return
	}

	provider, exist := c.Get("Provider")
	if exist == false {
		provider = db.DEFAULT_PROVIDER
	}

	course := db.Course{
		OauthUser: db.OauthUser{
			Provider: provider.(string),
		},
		Level: level,
	}

	results, err := db.QueryCourse(co.DB, course)

	if err != nil {
		errStr := fmt.Sprintf("query %s level course fail: %s", level, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if co.rfStackBase != nil {
		err := invokeRfStack(co.rfStackBase.Get("/v1/course/level/"+level), &results)
		if err != nil {
			errStr := fmt.Sprintf("fetch vm course fail: %s", err.Error())
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}
	}

	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: results,
	})
}

// @Summary List all course information
// @Description get all course information
// @Tags Course
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.ListCourseResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/course/list [get]
func (co *Course) ListAllCourse(c *gin.Context) {
	provider, exist := c.Get("Provider")
	if exist == false {
		provider = db.DEFAULT_PROVIDER
	}

	course := db.Course{
		OauthUser: db.OauthUser{
			Provider: provider.(string),
		},
	}

	results, err := db.QueryCourse(co.DB, course)

	if err != nil {
		errStr := fmt.Sprintf("query all course fail: %s", err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if co.rfStackBase != nil {

		err := invokeRfStack(co.rfStackBase.Get("/v1/course/list"), &results)
		if err != nil {
			errStr := fmt.Sprintf("fetch vm course fail: %s", err.Error())
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

	}

	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: results,
	})
}

// @Summary Search course name
// @Description Search course name
// @Tags Course
// @Accept  json
// @Produce  json
// @Param search body docs.Search true "search keyword"
// @Success 200 {object} docs.ListCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/course/search [post]
func (co *Course) SearchCourse(c *gin.Context) {
	req := model.Search{}
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if req.Query == "" {
		log.Errorf("Empty query condition")
		RespondWithError(c, http.StatusBadRequest, "Empty query condition")
		return
	}
	results, err := db.QueryCourse(co.DB, "name LIKE ?", "%"+req.Query+"%")

	if err != nil {
		errStr := fmt.Sprintf("search course on condition Name like %% %s %% fail: %s", req.Query, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if co.rfStackBase != nil {
		rfstackReq := rfstackmodel.Search{
			Query: req.Query,
		}

		token := c.GetHeader("Authorization")
		err := invokeRfStack(co.rfStackBase.Set("Authorization", token).
			BodyJSON(&rfstackReq).Post("/v1/course/search"), &results)

		if err != nil {
			errStr := fmt.Sprintf("fetch vm course fail: %s", err.Error())
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}
	}

	c.JSON(http.StatusOK, model.ListCourseResponse{
		Error:   false,
		Courses: results,
	})
}

// @Summary Get all courses name, including vm and container
// @Description Get all courses name, including vm and container
// @Tags Course
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.CourseNameListResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/course/namelist [get]
func (co *Course) CourseNameList(c *gin.Context) {

	courseList := []common.LabelValue{}

	provider, exist := c.Get("Provider")
	if exist == false {
		provider = db.DEFAULT_PROVIDER
	}

	course := db.Course{
		OauthUser: db.OauthUser{
			Provider: provider.(string),
		},
	}

	// query container course name & id
	results, err := db.QueryCourse(co.DB, course)
	if err != nil {
		errStr := fmt.Sprintf("Query container course fail: %s", err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	for _, course := range results {
		lbval := common.LabelValue{
			Label: course.Name,
			Value: course.ID,
		}
		courseList = append(courseList, lbval)
	}

	// fetch vm course name & id
	if co.rfStackBase != nil {
		vmCourseList := new(rfstackmodel.ListCourseResponse)
		errResp := new(rfstackmodel.GenericResponse)

		// todo: should consider course is created by user in different provider
		_, err = co.rfStackBase.Get("/v1/course/list").Receive(vmCourseList, errResp)

		if err != nil {
			errStr := fmt.Sprintf("fetch vm course fail: %s", err.Error())
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		if errResp.Error == true {
			errStr := fmt.Sprintf("rfstack get vm list fail: %s", errResp.Message)
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		for _, course := range vmCourseList.Courses {
			lbval := common.LabelValue{
				Label: course.Name,
				Value: course.ID,
			}
			courseList = append(courseList, lbval)
		}
	}

	c.JSON(http.StatusOK, model.CourseNameListResponse{
		Error:   false,
		Courses: courseList,
	})

}

// private helper func
func invokeRfStack(rfstack *sling.Sling, results *[]db.Course) error {

	vmCourseList := new(rfstackmodel.ListCourseResponse)
	errResp := new(rfstackmodel.GenericResponse)

	_, err := rfstack.Receive(vmCourseList, errResp)

	if err != nil {
		return err
	}

	if errResp.Error == true {
		errStr := fmt.Sprintf("rfstack get vm list fail: %s", errResp.Message)
		return errors.New(errStr)
	}

	for _, course := range vmCourseList.Courses {
		vmCourse := db.Course{
			Model: db.Model{
				ID:        course.ID,
				CreatedAt: course.CreatedAt,
			},
			Name:       course.Name,
			Level:      course.Level,
			CourseType: util.StringPtr(db.VM),
		}
		*results = append(*results, vmCourse)
	}

	return nil
}
