package beta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/dghubble/sling"
	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/consts"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/common"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/config"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/db"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/util"
	"github.com/nchc-ai/course-crd/pkg/apis/coursecontroller/v1alpha1"
	"github.com/nchc-ai/course-crd/pkg/client/clientset/versioned"
	"github.com/nchc-ai/course-cron/pkg/cron"
	rfstackmodel "github.com/nchc-ai/rfstack/model"
	"github.com/nitishm/go-rejson/v4"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	JobStatusCreated = "Created"
	JobStatusPending = "Pending"
	JobStatueReady   = "Ready"
)

type Job struct {
	CourseCrdClient *versioned.Clientset
	DB              *gorm.DB
	redis           *rejson.Handler
	config          *config.Config
	rfStackBase     *sling.Sling
	StopChanMap     map[string](chan string)
}

// @Summary List all running course deployment for a user
// @Description List all running course deployment for a user
// @Tags Job
// @Accept  json
// @Produce  json
// @Param list_user body docs.OauthUser true "search user's job"
// @Success 200 {object} docs.JobListResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/job/list [post]
func (j *Job) List(c *gin.Context) {
	provider, exist := c.Get("Provider")
	if exist == false {
		provider = db.DEFAULT_PROVIDER
	}

	req := db.Job{}
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

	job := db.Job{
		OauthUser: db.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
	}

	redisKey := fmt.Sprintf("%s:%s", job.Provider, job.User)
	// get from redis if available
	redisResult, err := redis.Bytes(j.redis.JSONGet(redisKey, "."))
	result := model.JobListResponse{}
	if err == nil && json.Unmarshal(redisResult, &result) == nil {
		c.JSON(http.StatusOK, result)
		return
	}

	resultJobs, err := job.GetJobOwnByUser(j.DB)
	if err != nil {
		strErr := fmt.Sprintf("Query Job table for user {%s} fail: %s", req.User, err.Error())
		log.Errorf(strErr)
		RespondWithError(c, http.StatusInternalServerError, strErr)
		return
	}

	//namespace := j.namespace
	jobList := []model.JobInfo{}
	for _, result := range resultJobs {
		// find course information
		courseInfo, err := result.GetCourse(j.DB)
		if err != nil {
			errStr := fmt.Sprintf("Query Course info for job {%s} fail: %s", result.ID, err.Error())
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		// find access URL information
		accessURL, err := getCRDPort(j.CourseCrdClient, result, j.config.K8SConfig, *result.ClassroomID)
		if err != nil {
			errStr := fmt.Sprintf("Parse Service info for job {%s} fail: %s", result.ID, err.Error())
			log.Errorf(errStr)
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}

		snapshot := false
		snapshot, _ = courseInfo.IsOwner(j.DB, req.User, provider.(string))

		jobInfo := model.JobInfo{
			Id:           result.ID,
			CourseID:     courseInfo.ID,
			StartAt:      result.CreatedAt,
			Status:       result.Status,
			Name:         courseInfo.Name,
			Introduction: *courseInfo.Introduction,
			Image:        courseInfo.Image,
			Level:        courseInfo.Level,
			GPU:          *courseInfo.Gpu,
			CanSnapshot:  snapshot,
			//Dataset:      courseInfo.Datasets,
			Service: accessURL,
		}

		jobList = append(jobList, jobInfo)
	}

	result.Error = false
	result.Jobs = jobList

	// add result to redis
	_, err = j.redis.JSONSet(redisKey, ".", result)
	if err != nil {
		log.Warningf("Failed to JSONSet")
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Create a course CRD in kubernetes
// @Description Create a course CRD in kubernetes
// @Tags Job
// @Accept  json
// @Produce  json
// @Param launch_course body docs.LaunchCourseRequest true "course want to launch"
// @Success 200 {object} docs.LaunchCourseResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/job/launch [post]
func (j *Job) Launch(c *gin.Context) {

	var req model.LaunchCourseRequest
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	course := db.Course{
		Model: db.Model{
			ID: req.CourseId,
		},
	}

	isVerified, errs := j.preCheckJob(c, &req)
	if isVerified != true {
		errStr := fmt.Sprintf("Pre-check launch course {%s} job fail: %s", req.CourseId, errs[0].Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errs[1].Error())
		return
	}

	// if classroom_id is empty, and pass preCheckJob(), means own by some teacher, lauch in default namespace.
	if req.ClassroomId == "" {
		req.ClassroomId = consts.TEACHER_CLASSROOM
	}

	courseType, err := course.Type(j.DB)

	if err != nil {
		errStr := fmt.Sprintf("Query course {%s} type fail: %s", req.CourseId, err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	switch courseType {
	case db.CONTAINER:
		// course type is container, launch container job
		j.launchContainerJob(c, &req)
		return
	case db.VM:
		// course type is VM, use rfstack api launch VM job
		j.launchVMJob(c, &req)
		return
	}
}

// @Summary Delete a course CRD in user namespace
// @Description Delete a running job deployment in user namespace
// @Tags Job
// @Accept  json
// @Produce  json
// @Param id path string true "course CRD uuid, eg: 131ba8a9-b60b-44f9-83b5-46590f756f41"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/job/delete/{id} [delete]
func (j *Job) Delete(c *gin.Context) {
	jobId := c.Param("id")

	if jobId == "" {
		RespondWithError(c, http.StatusBadRequest,
			"Job Id is empty")
		return
	}

	job := db.Job{
		Model: db.Model{
			ID: jobId,
		},
	}

	audit := db.Audit{
		Model: db.Model{
			ID: jobId,
		},
		//DeletedBy: util.StringPtr("UI"),
	}

	// default type is CONTAINER, if job id not found in containerJob table, set type to VM
	// rfstack will return job not found when job id is not valid.
	courseType := db.CONTAINER
	if err := j.DB.Find(&job).Error; err != nil {
		courseType = db.VM
	}

	switch courseType {
	case db.CONTAINER:
		j.DB.Model(&audit).Update("deleted_by", "UI")
		if err := j.DB.Delete(audit).Error; err != nil {
			log.Warningf(fmt.Sprintf("Failed to mark job {%s} deletion audit information : %s", audit.ID, err.Error()))
		}

		// course type is container, delete container job
		if errStr, err := job.DeleteCourseCRD(j.DB, j.redis,
			j.CourseCrdClient, *job.ClassroomID); err != nil {
			RespondWithError(c, http.StatusInternalServerError, errStr)
			return
		}
		j.StopChanMap[jobId] <- "STOP"
		RespondWithOk(c, "Job {%s} is deleted successfully", jobId)
	case db.VM:
		// course type is VM, use rfstack api delete VM job
		j.deleteVMJob(c, jobId)
		return
	}
}

// PUBLIC utility function
func CheckCourseCRDStatus(DB *gorm.DB, redis *rejson.Handler, crdClient *versioned.Clientset, ns string, CRDId string, stop chan string) {

	jobObj := db.Job{
		Model: db.Model{
			ID: CRDId,
		},
	}
	if err := DB.First(&jobObj).Error; err != nil {
		return
	}
	redisKey := fmt.Sprintf("%s:%s", jobObj.Provider, jobObj.User)

	// binary backoff retry check if CRD accessible

	operation := func() error {

		v, _ := <-stop
		if v != "STOP" {
			stop <- ""
		} else {
			log.Info("Course CRD checker is stopped.")
			return nil
		}

		course, err := crdClient.NchcV1alpha1().Courses(ns).Get(context.Background(), CRDId, metav1.GetOptions{})
		if err != nil {
			log.Warningf("Get Course CRD {%s} fail: %s", CRDId, err.Error())
			return err
		}
		if !course.Status.Accessible {
			if err := DB.Model(&jobObj).Update("status", JobStatusPending).Error; err != nil {
				log.Errorf("update job {%s} status to %s fail: %s", CRDId, JobStatusPending, err.Error())
				return err
			}

			// delete redis cache
			_, err = redis.JSONDel(redisKey, ".")
			if err != nil {
				log.Errorf("Delete cache key {%s} fail for update job status to %s: %s", redisKey, JobStatusPending, err.Error())
				return err
			}

			log.Warningf(fmt.Sprintf("Course CRD {%s} is not Accessible", CRDId))
			return errors.New(fmt.Sprintf("Course CRD {%s} is not Accessible", CRDId))
		}
		return nil
	}

	err := backoff.Retry(operation, backoff.NewExponentialBackOff())
	if err != nil {
		log.Warningf("check CRD {%s} Accessible Retry timeout: %s", CRDId, err.Error())
		return
	}

	if err := DB.Model(&jobObj).Update("status", JobStatueReady).Error; err != nil {
		log.Errorf("update job {%s} status to %s fail: %s", CRDId, JobStatueReady, err.Error())
		return
	}
	// delete redis cache
	_, err = redis.JSONDel(redisKey, ".")
	if err != nil {
		log.Errorf("Delete cache key {%s} fail for update job status to %s: %s", redisKey, JobStatueReady, err.Error())
		return
	}

}

// PRIVATE function
// func buildCourseCRD(DB *gorm.DB, classroomID, courseID, userId string, config *config.K8SConfig) (*v1alpha1.Course, []error) {
func buildCourseCRD(DB *gorm.DB, classroomID, courseID string, user *db.User, config *config.Config) (*v1alpha1.Course, []error) {

	// verify classroom has course
	cmInfo := db.ClassRoomInfo{
		Model: db.Model{
			ID: classroomID,
		},
	}
	cm, err := cmInfo.GetClassRoomDetail(DB)
	if err != nil {
		return nil, []error{err, err}
	}

	//Step 1: retrive required information
	course, err := db.GetCourse(DB, courseID)
	if err != nil {
		return nil, []error{err, err}
	}

	// todo: if classroom is default, course is ONLY allowed run for one hour
	schedule, err := cm.GetSchedule(DB)
	if err != nil {
		return nil, []error{err, err}
	}

	// Step 2: handle required field
	crdDef := &v1alpha1.Course{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"user": user.User,
			},
			Name:      uuid.New().String(),
			Namespace: classroomID,
		},
		Spec: v1alpha1.CourseSpec{
			AccessType: course.AccessType,
			Image:      course.Image,
			Gpu:        *course.Gpu,
			Schedule:   schedule.CronFormat,
		},
	}

	// Step 3: handle option field
	// 	Step 3-1: find dataset required by course
	datasets, err := course.GetDataset(DB)
	if err != nil {
		log.Error(fmt.Sprintf("Query course {%s} required dataset fail", courseID))
		return nil, []error{
			err,
			errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_BUILDCRD_FMT, course.Name)),
		}
	}
	if len(datasets) > 0 {
		crdDef.Spec.Dataset = datasets
	}

	// 	Step 3-2: find port required by course
	ports, err := course.GetPort(DB)
	if err != nil {
		log.Error(fmt.Sprintf("Query course {%s} required port fail", courseID))
		return nil, []error{
			err,
			errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_BUILDCRD_FMT, course.Name)),
		}
	}

	if len(ports) == 0 {
		return nil, []error{
			errors.New(fmt.Sprintf("Ports is not defined in course {%s}", courseID)),
			errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_PORT_FMT, course.Name, course.User)),
		}
	}
	portMap := make(map[string]int32)
	for _, p := range ports {
		portMap[p.Name] = int32(p.Port)
	}
	crdDef.Spec.Port = portMap

	// 	Step 3-3: if need writable volume

	// if uidRange start from 0, use root user forever, otherwise, set UID
	uidStart := strings.Split(config.APIConfig.UidRange, "/")[0]
	var uid int64
	if uidStart == "0" {
		uid = 0
	} else {
		uid = int64(user.Uid)
	}

	if *course.WritablePath != "" {
		crdDef.Spec.WritableVolume = &v1alpha1.WritableVolume{
			Owner:        user.User,
			Uid:          uid,
			StorageClass: config.K8SConfig.StorageClass,
			MountPoint:   *course.WritablePath,
		}
	}

	return crdDef, nil
}

func getCRDPort(crdclient *versioned.Clientset, job db.Job, config *config.K8SConfig, namespace string) ([]common.LabelValue, error) {

	result := []common.LabelValue{}

	crd, err := crdclient.NchcV1alpha1().Courses(namespace).Get(context.Background(), job.ID, metav1.GetOptions{})

	if err != nil {
		log.Errorf("Get Course CRD {%s} fail: %s", job.ID, err.Error())
		return nil, err
	}

	switch accessType := crd.Spec.AccessType; accessType {
	case v1alpha1.AccessTypeIngress:
		for portName, path := range crd.Status.SubPath {
			lv := common.LabelValue{
				Label: portName,
				// <SUBPATH>
				Value: fmt.Sprintf("http://%s", path),
			}
			result = append(result, lv)
		}
	case v1alpha1.AccessTypeNodePort:
		for portName, portValue := range crd.Status.NodePort {
			lv := common.LabelValue{
				Label: portName,
				// <NodePort_DNS>:<NODE_PORT>
				Value: fmt.Sprintf("%s:%d", config.NodePortDNS, portValue),
			}
			result = append(result, lv)
		}
	}

	return result, nil

}

func (j *Job) preCheckJob(c *gin.Context, req *model.LaunchCourseRequest) (bool, []error) {

	user := req.User
	if user == "" {
		return false, []error{
			errors.New("user field in request cannot be empty"),
			errors.New("user field in request cannot be empty"),
		}
	}

	provider, exist := c.Get("Provider")
	if !exist {
		provider = db.DEFAULT_PROVIDER
	}

	if req.ClassroomId == "" {
		//without classroom id
		// check user is superuser -> check count
		// check user is onwer -> check count
		return j.precheckWithoutClassroom(req, provider.(string))
	} else {
		//with classroom id
		// check classroom is pubic
		// check classroom schedule is valid
		// check classroom has course
		// check user is superuser -> check count
		// check user is teacher or student -> check count
		return j.precheckWithClassroom(req, provider.(string))
	}

	return true, nil
}

func (j *Job) precheckWithClassroom(req *model.LaunchCourseRequest, provider string) (bool, []error) {

	newJob := db.Job{
		OauthUser: db.OauthUser{
			User:     req.User,
			Provider: provider,
		},
		CourseID:    req.CourseId,
		ClassroomID: &req.ClassroomId,
		Status:      JobStatusCreated,
	}

	cmInfo := db.ClassRoomInfo{
		Model: db.Model{
			ID: req.ClassroomId,
		},
	}

	cm, err := cmInfo.GetClassRoomDetail(j.DB)
	if err != nil {
		return false, []error{err, err}
	}

	// check classroom is pubic
	if cm.IsPublic == db.FALSE {
		return false, []error{
			errors.New(fmt.Sprintf("Classroom {%s} is not public", req.ClassroomId)),
			errors.New(fmt.Sprintf("Classroom {%s} is not public", req.ClassroomId)),
		}
	}

	// check classroom schedule is valid
	schedules, err := cmInfo.GetSchedule(j.DB)
	if err != nil {
		return false, []error{err, err}
	}

	isSchedulable := false
	for _, schedule := range schedules.CronFormat {
		isSchedulable = isSchedulable || cron.IsNowMatchCronExpression(schedule, "Asia/Taipei")
	}

	if !isSchedulable {
		return false, []error{
			errors.New(fmt.Sprintf("Classroom {%s} is allowed in {%s}", req.ClassroomId, cm.ScheduleDescription)),
			errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_TIME_FMT, cm.Name, cm.ScheduleDescription)),
		}
	}

	// check classroom has course
	ok, err := cm.HasCourse(j.DB, req.CourseId)
	if ok == false {
		return false, []error{
			errors.New(fmt.Sprintf("course {%s} is not in classroom {%s}", req.CourseId, req.ClassroomId)),
			errors.New(fmt.Sprintf("course {%s} is not in classroom {%s}", req.CourseId, req.ClassroomId)),
		}
	}

	// check user is superuser -> check count
	isSuperuser := j.isSuperuser(req.User, provider)
	if isSuperuser {
		return j.precheckCount(req, newJob)
	}

	// check user is teacher or student -> check count
	if req.ClassroomId != consts.PUBLIC_CLASSROOM {
		ok1, _ := cm.HasStudent(j.DB, req.User, provider)
		ok2, _ := cm.HasTeacher(j.DB, req.User, provider)

		if !ok1 && !ok2 {
			return false, []error{
				errors.New(fmt.Sprintf("{%s}:{%s} is neither student nor teacher of classroom {%s}", req.User, provider, req.ClassroomId)),
				errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_MEMBER_FMT, req.User, cm.Name)),
			}
		}
	}

	return j.precheckCount(req, newJob)
}

func (j *Job) precheckWithoutClassroom(req *model.LaunchCourseRequest, provider string) (bool, []error) {

	newJob := db.Job{
		OauthUser: db.OauthUser{
			User:     req.User,
			Provider: provider,
		},
		CourseID:    req.CourseId,
		ClassroomID: &req.ClassroomId,
		Status:      JobStatusCreated,
	}

	rfJob := rfstackmodel.Job{
		OauthUser: rfstackmodel.OauthUser{
			User:     req.User,
			Provider: provider,
		},
		CourseID: req.CourseId,
		Status:   JobStatusCreated,
	}

	// check user is superuser -> check count
	isSuperuser := j.isSuperuser(req.User, provider)
	if isSuperuser {
		return j.precheckCount(req, newJob)
	}

	// check user is onwer -> check count
	// container course
	isOwnContainer, _ := containerJobOwnByUser(&newJob, j.DB, req.User, provider)
	// vm course
	isOwnVM, _ := vmJobOwnByUser(&rfJob, j.DB, req.User, provider)

	if isOwnContainer || isOwnVM {
		return j.precheckCount(req, newJob)
	} else {
		return false, []error{
			errors.New(fmt.Sprintf("course {%s} isn't owned by user {%s:%s}", req.CourseId, req.User, provider)),
			errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_OWNER_FMT, req.User)),
		}
	}
}

func (j *Job) precheckCount(req *model.LaunchCourseRequest, newJob db.Job) (bool, []error) {
	// user usage constraint: one user can start ONLY one job at one time
	c_count, err := newJob.UserJobCount(j.DB)
	if err != nil {
		return false, []error{
			errors.New(fmt.Sprintf("Query user container job count fail: %s", err.Error())),
			errors.New(fmt.Sprintf("Query user container job count fail: %s", err.Error())),
		}
	}
	if c_count > 0 {
		return false, []error{
			errors.New(fmt.Sprintf("user {%s} already lauch %d container job", req.User, c_count)),
			errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_QUOTA_FMT, req.User, c_count)),
		}
	}

	vm_count, err := vmJobCount(j.DB, req.User)
	if err != nil {
		return false, []error{
			errors.New(fmt.Sprintf("Query user vm job count fail: %s", err.Error())),
			errors.New(fmt.Sprintf("Query user vm job count fail: %s", err.Error())),
		}
	}
	if vm_count > 0 {
		return false, []error{
			errors.New(fmt.Sprintf("user {%s} already lauch %d vm job", req.User, vm_count)),
			errors.New(fmt.Sprintf(consts.ERROR_JOB_LAUNCH_QUOTA_FMT, req.User, vm_count)),
		}
	}

	return true, nil
}

func (j *Job) isSuperuser(user, provider string) bool {

	u := db.User{
		Provider: util.StringPtr(provider),
		Role:     "superuser",
	}

	superuserList, err := u.GetRoleList(j.DB)

	if err != nil {
		return false
	}

	for _, u := range superuserList {
		if u == user {
			return true
		}
	}

	return false
}

func (j *Job) launchContainerJob(c *gin.Context, req *model.LaunchCourseRequest) {

	provider, exist := c.Get("Provider")
	if !exist {
		log.Warning("Provider is not found in request context, set empty")
		provider = db.DEFAULT_PROVIDER
	}

	newJob := db.Job{
		OauthUser: db.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
		CourseID:    req.CourseId,
		ClassroomID: &req.ClassroomId,
		Status:      JobStatusCreated,
	}

	u := db.User{
		User:     req.User,
		Provider: util.StringPtr(provider.(string)),
	}

	user, err := u.FindUser(j.DB)
	if err != nil {
		log.Errorf("Query user info fail: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_JOB_LAUNCH_BUILDCRD_FMT, req.CourseId)
		return
	}

	course, err := db.GetCourse(j.DB, req.CourseId)
	if err != nil {
		log.Errorf("Query course info fail: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_JOB_LAUNCH_BUILDCRD_FMT, req.CourseId)
		return
	}

	CRDDef, errs := buildCourseCRD(j.DB, req.ClassroomId, req.CourseId, user, j.config)

	if errs != nil {
		log.Errorf(" user {%s} build Course {%s} CRD in Classroom {%s} fail: %s",
			req.User, req.CourseId, req.ClassroomId, errs[0].Error())
		RespondWithError(c, http.StatusInternalServerError, errs[1].Error())
		return
	}
	CRDId := CRDDef.Name

	courseCRD, createErr := j.CourseCrdClient.NchcV1alpha1().Courses(req.ClassroomId).Create(
		context.Background(), CRDDef, metav1.CreateOptions{})

	if createErr != nil {
		log.Errorf("create CRD fail: %s", createErr.Error())
		RespondWithError(c, http.StatusInternalServerError, consts.ERROR_JOB_LAUNCH_RUNCRD_FMT, course.Name)
		return
	}

	// Step 3: update Job Table
	newJob.ID = courseCRD.Name
	if err := newJob.NewEntry(j.DB); err != nil {
		//update job table fail, we need delete created CRD.
		deletePolicy := metav1.DeletePropagationForeground
		if delErr := j.CourseCrdClient.NchcV1alpha1().Courses(req.ClassroomId).
			Delete(context.Background(), CRDId, metav1.DeleteOptions{PropagationPolicy: &deletePolicy}); delErr != nil {
			errStrt2 := fmt.Sprintf("Delete CRD when insert new job fail: %s", delErr.Error())
			log.Error(errStrt2)
			RespondWithError(c, http.StatusInternalServerError, errStrt2)
			return
		}
		errStrt := fmt.Sprintf("Insert new job {id = %s} in Job table fail: %s, and CRD {%s} is also deleted",
			courseCRD.Name, err.Error(), CRDId)
		log.Errorf(errStrt)
		RespondWithError(c, http.StatusInternalServerError, errStrt)
		return
	}

	// delete redis cache
	redisKey := fmt.Sprintf("%s:%s", newJob.Provider, newJob.User)
	_, err = j.redis.JSONDel(redisKey, ".")
	if err != nil {
		log.Warningf("Failed to JSONDel")
	}

	newAudit := db.Audit{
		Model: db.Model{
			ID: courseCRD.Name,
		},
		OauthUser: db.OauthUser{
			User:     req.User,
			Provider: provider.(string),
		},
		CourseID:    req.CourseId,
		ClassroomID: &req.ClassroomId,
	}
	if err := newAudit.NewEntry(j.DB); err != nil {
		log.Warningf(fmt.Sprintf("Insert new job {id = %s} in Audit table fail: %s",
			courseCRD.Name, err.Error()))
	}

	// this goroutine should be also stop  when job is deleted (#71)
	// ref: https://stackoverflow.com/questions/6807590/how-to-stop-a-goroutine
	j.StopChanMap[newJob.ID] = make(chan string, 5)
	j.StopChanMap[newJob.ID] <- ""
	go CheckCourseCRDStatus(j.DB, j.redis, j.CourseCrdClient, req.ClassroomId, courseCRD.Name, j.StopChanMap[newJob.ID])

	c.JSON(http.StatusOK, model.LaunchCourseResponse{
		Error: false,
		Job: model.JobStatus{
			JobId:  newJob.ID,
			Ready:  false,
			Status: "Created",
		},
	})
}

func (j *Job) launchVMJob(c *gin.Context, req *model.LaunchCourseRequest) {
	token := c.GetHeader("Authorization")

	// todo: use api LaunchCourseRequest, instead rfstackmodel.LaunchCourseRequest. There is no classroom_id
	rfsReq := rfstackmodel.LaunchCourseRequest{
		User:     req.User,
		CourseId: req.CourseId,
	}

	launchVMResp := new(rfstackmodel.LaunchCourseResponse)
	ErrResp := new(model.GenericResponse)

	// todo: rfstack will take some time wait for vm become active. Should we define http client timeout ?
	_, err := j.rfStackBase.
		Set("Authorization", token).
		BodyJSON(&rfsReq).
		Post("/v1/job/launch").Receive(launchVMResp, ErrResp)

	if err != nil {
		errStr := fmt.Sprintf("Connect to rfstack fail: %s", err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if ErrResp.Error == true {
		errStr := fmt.Sprintf("rfstack create vm fail: %s", ErrResp.Message)
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, model.LaunchCourseResponse{
		Error: false,
		Job: model.JobStatus{
			JobId:  launchVMResp.Job.JobId,
			Ready:  launchVMResp.Job.Ready,
			Status: launchVMResp.Job.Status,
		},
	})
}

func (j *Job) deleteVMJob(c *gin.Context, jobid string) {
	token := c.GetHeader("Authorization")

	launchVMResp := new(rfstackmodel.GenericResponse)
	ErrResp := new(model.GenericResponse)

	_, err := j.rfStackBase.
		Set("Authorization", token).
		Delete(fmt.Sprintf("/v1/job/delete/%s", jobid)).Receive(launchVMResp, ErrResp)

	if err != nil {
		errStr := fmt.Sprintf("Connect to rfstack fail: %s", err.Error())
		log.Error(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	if ErrResp.Error == true {
		errStr := fmt.Sprintf("rfstack delete vm fail: %s", ErrResp.Message)
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}
	RespondWithOk(c, "Job {%s} is deleted successfully", jobid)
}

func vmJobCount(db *gorm.DB, user string) (int, error) {
	count := 0
	if err := db.Model(&rfstackmodel.Job{}).Where("user = ?", user).Count(&count).Error; err != nil {
		return count, err
	}
	return count, nil
}

func containerJobOwnByUser(job *db.Job, db *gorm.DB, user string, provider string) (bool, error) {
	course, err := job.GetCourse(db)
	if err != nil {
		return false, errors.New(fmt.Sprintf("course {%s} isn't owned by user {%s:%s}: %s",
			job.CourseID, user, provider, err.Error()))
	}

	isOwnbyUser, err := course.IsOwner(db, user, provider)
	if err != nil {
		return false, errors.New(fmt.Sprintf("container course {%s} isn't owned by user {%s:%s}: %s",
			job.CourseID, user, provider, err.Error()))
	}

	return isOwnbyUser, nil
}

func vmJobOwnByUser(job *rfstackmodel.Job, db *gorm.DB, user string, provider string) (bool, error) {

	c := rfstackmodel.Course{
		Model: rfstackmodel.Model{
			ID: job.CourseID,
		},
		OauthUser: rfstackmodel.OauthUser{
			User:     user,
			Provider: provider,
		},
	}

	if result := db.First(&c); result.Error != nil {
		if result.RecordNotFound() {
			log.Errorf(fmt.Sprintf("vm course {%s} isn't owned by user {%s:%s}", c.ID, user, provider))
		}
		return false, errors.New(fmt.Sprintf("vm course {%s} isn't owned by user {%s:%s}: %s",
			c.ID, user, provider, result.Error.Error()))
	}

	return true, nil
}
