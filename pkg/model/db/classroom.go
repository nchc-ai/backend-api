package db

import (
	"errors"
	"fmt"

	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/backend-api/pkg/consts"
	"github.com/nchc-ai/backend-api/pkg/model/common"
	"github.com/nchc-ai/backend-api/pkg/util"
	rfstackmodl "github.com/nchc-ai/rfstack/model"
)

type Sqlbool uint8

const TRUE = 1
const FALSE = 0

func Bool2Sqlbool(input bool) Sqlbool {
	if input == true {
		return TRUE
	} else {
		return FALSE
	}
}

func Sqlbool2Bool(input Sqlbool) bool {

	if input == TRUE {
		return true
	} else {
		return false
	}

}

// work around for MySQL don't have bool type, and use tinyint for bool.
// json public field is bool type, mysql is_public is tinyint type
// util.Bool2Sqlbool() convert bool to uint8
type ClassRoomInfo struct {
	//ScheduleTime        []string                    `gorm:"-" json:"schedules,omitempty"`

	Model
	Name                string                      `gorm:"size:50;not null" json:"name"`
	Description         string                      `gorm:"size:200" json:"description"`
	ScheduleDescription string                      `gorm:"size:200" json:"-"`
	IsPublic            Sqlbool                     `gorm:"not null;type:tinyint" json:"-"`
	SelectedType        *int32                      `gorm:"selectedType" json:"-"`
	StartAt             string                      `gorm:"startAt" json:"-"`
	EndAt               string                      `gorm:"endAt" json:"-"`
	IsPublicBool        bool                        `gorm:"-" json:"public"`
	StudentCount        *int32                      `gorm:"-" json:"studentCount,omitempty"`
	ScheduleTime        *Schedule                   `gorm:"-" json:"schedule,omitempty"`
	TeacherList         *[]common.LabelValue        `gorm:"-" json:"teachers,omitempty"`
	StudentList         *[]common.LabelValue        `gorm:"-" json:"students,omitempty"`
	CourseList          []common.LabelValue         `gorm:"-" json:"courses,omitempty"`
	Course              []Course                    `gorm:"-" json:"courseInfo,omitempty"`
	CalendarTime        *[]CalendarTime             `gorm:"-" json:"calendar,omitempty"`
	Schedule            []ClassRoomScheduleRelation `json:"-"`
	Teachers            []ClassRoomTeacherRelation  `json:"-"`
	Students            []ClassRoomStudentRelation  `json:"-"`
}

type Schedule struct {
	CronFormat     []string            `gorm:"-" json:"cronFormat"`
	Description    string              `gorm:"-" json:"description"`
	StartDate      string              `gorm:"-" json:"startDate"`
	EndDate        string              `gorm:"-" json:"endDate"`
	SelectedType   *int32              `gorm:"-" json:"selectedType"`
	SelectedOption []common.LabelValue `gorm:"-" json:"selectedOption"`
}

func (ClassRoomInfo) TableName() string {
	return "classroomInfo"
}

func (classroom *ClassRoomInfo) HasCourse(db *gorm.DB, course_id string) (bool, error) {

	classroomCourse := ClassRoomCourseRelation{
		ClassroomID: classroom.ID,
		CourseID:    course_id,
	}

	if result := db.First(&classroomCourse); result.Error != nil {
		if result.RecordNotFound() {
			log.Errorf(fmt.Sprintf("classroom {%s} does not include course {%s}", classroom.ID, course_id))
		}
		return false, result.Error
	}
	return true, nil
}

func (classroom *ClassRoomInfo) HasStudent(db *gorm.DB, user_id string, provider string) (bool, error) {

	classroomStduent := ClassRoomStudentRelation{
		ClassRoomUser: ClassRoomUser{
			ClassroomID: classroom.ID,
			User:        user_id,
			Provider:    provider,
		},
	}

	if result := db.First(&classroomStduent); result.Error != nil {
		if result.RecordNotFound() {
			log.Errorf(fmt.Sprintf("classroom {%s} does not include student {%s}", classroom.ID, user_id))
		}
		return false, result.Error
	}
	return true, nil
}

func (classroom *ClassRoomInfo) HasTeacher(db *gorm.DB, user_id string, provider string) (bool, error) {
	classroomTeacher := ClassRoomTeacherRelation{
		ClassRoomUser: ClassRoomUser{
			ClassroomID: classroom.ID,
			User:        user_id,
			Provider:    provider,
		},
	}

	if result := db.First(&classroomTeacher); result.Error != nil {
		if result.RecordNotFound() {
			log.Errorf(fmt.Sprintf("classroom {%s} does not include teacher {%s}", classroom.ID, user_id))
		}
		return false, result.Error
	}
	return true, nil
}

func (classroom *ClassRoomInfo) GetSchedule(db *gorm.DB) (*Schedule, error) {

	cm := ClassRoomScheduleRelation{
		ClassroomID: classroom.ID,
	}
	result := []ClassRoomScheduleRelation{}
	if err := db.Where(&cm).Find(&result).Error; err != nil {
		return nil, err
	}
	cronResult := []string{}
	for _, s := range result {
		cronResult = append(cronResult, s.Schedule)
	}

	csr := ClassRoomSelectedOptionRelation{
		ClassroomID: classroom.ID,
	}
	csrResult := []ClassRoomSelectedOptionRelation{}
	if err := db.Where(&csr).Find(&csrResult).Error; err != nil {
		return nil, err
	}
	optsResult := []common.LabelValue{}
	for _, s := range csrResult {
		optsResult = append(optsResult, common.LabelValue{
			Label: s.Label,
			Value: s.Value,
		})
	}

	return &Schedule{
		CronFormat:     cronResult,
		StartDate:      classroom.StartAt,
		EndDate:        classroom.EndAt,
		Description:    classroom.ScheduleDescription,
		SelectedType:   classroom.SelectedType,
		SelectedOption: optsResult,
	}, nil
}

func (classroom *ClassRoomInfo) GetCourseID(db *gorm.DB) ([]string, error) {
	classroomInfo := ClassRoomCourseRelation{
		ClassroomID: classroom.ID,
	}

	results := []ClassRoomCourseRelation{}

	if err := db.Where(&classroomInfo).Find(&results).Error; err != nil {
		log.Errorf("Query courses table fail: %s", err.Error())
		return nil, err
	}

	final := []string{}
	for _, r := range results {
		final = append(final, r.CourseID)
	}

	return final, nil
}

func (classroom *ClassRoomInfo) GetTeacherList(db *gorm.DB) (*[]common.LabelValue, error) {

	cm := ClassRoomTeacherRelation{
		ClassRoomUser: ClassRoomUser{
			ClassroomID: classroom.ID,
		},
	}

	result := []ClassRoomTeacherRelation{}
	if err := db.Where(&cm).Find(&result).Error; err != nil {
		return nil, err
	}

	finalResult := []common.LabelValue{}

	for _, s := range result {
		finalResult = append(finalResult, common.LabelValue{
			Label: s.Name,
			Value: s.User,
		})
	}

	return &finalResult, nil

}

func (classroom *ClassRoomInfo) GetStudentList(db *gorm.DB) (*[]common.LabelValue, error) {
	cm := ClassRoomStudentRelation{
		ClassRoomUser: ClassRoomUser{
			ClassroomID: classroom.ID,
		},
	}

	result := []ClassRoomStudentRelation{}
	if err := db.Where(&cm).Find(&result).Error; err != nil {
		return nil, err
	}

	finalResult := []common.LabelValue{}

	for _, s := range result {
		finalResult = append(finalResult, common.LabelValue{
			Label: s.Name,
			Value: s.User,
		})
	}

	return &finalResult, nil
}

func (classroom *ClassRoomInfo) GetCalendar(db *gorm.DB) (*[]CalendarTime, error) {

	cm := ClassRoomCalendarRelation{
		ClassroomID: classroom.ID,
	}

	result := []ClassRoomCalendarRelation{}
	if err := db.Where(&cm).Find(&result).Error; err != nil {
		return nil, err
	}

	finalResult := []CalendarTime{}

	for _, s := range result {
		finalResult = append(finalResult, CalendarTime{
			StartMonth: s.StartMonth,
			StartDate:  s.StartDate,
			EndDate:    s.EndDate,
			Length:     s.Length,
		})
	}

	return &finalResult, nil

}

func (classroom *ClassRoomInfo) NewEntry(db *gorm.DB) error {

	if classroom.ScheduleTime == nil {
		return errors.New("schedule is not defined")
	}

	classroom.IsPublic = Bool2Sqlbool(classroom.IsPublicBool)
	classroom.SelectedType = classroom.ScheduleTime.SelectedType
	classroom.StartAt = classroom.ScheduleTime.StartDate
	classroom.EndAt = classroom.ScheduleTime.EndDate
	classroom.ScheduleDescription = classroom.ScheduleTime.Description

	if err := db.Create(classroom).Error; err != nil {
		return err
	}
	return nil
}

func (classroom *ClassRoomInfo) GetClassRoomDetail(db *gorm.DB) (*ClassRoomInfo, error) {

	classroomInfo := ClassRoomInfo{
		Model: Model{
			ID: classroom.ID,
		},
	}

	if err := db.Find(&classroomInfo).Error; err != nil {
		return nil, err
	}

	classroomInfo.IsPublicBool = Sqlbool2Bool(classroomInfo.IsPublic)

	courseId, err := classroomInfo.GetCourseID(db)

	if err != nil {
		return nil, err
	}

	var courses []Course

	for _, id := range courseId {

		course := Course{
			Model: Model{
				ID: id,
			},
		}

		courseType, err := course.Type(db)

		if err != nil {
			return nil, err
		}

		switch courseType {
		case CONTAINER:
			c, err := GetCourse(db, id)
			if err != nil {
				return nil, err
			}
			course.Name = c.Name
			course.Level = c.Level
			course.CreatedAt = c.CreatedAt
			course.ClasroomID = util.StringPtr(classroom.ID)
			course.CourseType = util.StringPtr(CONTAINER)
		case VM:
			c, err := getVMCourse(db, id)
			if err != nil {
				return nil, err
			}
			course.Name = c.Name
			course.Level = c.Level
			course.CreatedAt = c.CreatedAt
			course.ClasroomID = util.StringPtr(classroom.ID)
			course.CourseType = util.StringPtr(VM)
		}
		courses = append(courses, course)
	}

	classroomInfo.Course = courses

	return &classroomInfo, nil
}

func (classroom *ClassRoomInfo) GetStudentCount(db *gorm.DB) (int32, error) {

	count := int32(0)
	if err := db.Model(&ClassRoomStudentRelation{}).Where("classroom_id = ?", classroom.ID).Count(&count).Error; err != nil {
		return count, err
	}
	return count, nil
}

func GetAllClassroom(db *gorm.DB) ([]ClassRoomInfo, error) {
	results := []ClassRoomInfo{}

	// aitrain-teacher is a dummy classroom, never seen or edit by anyone, even superuser
	if err := db.Where("id != ?", consts.TEACHER_CLASSROOM).Find(&results).Error; err != nil {
		log.Errorf("Query courses table fail: %s", err.Error())
		return nil, err
	}

	return results, nil
}

func GetUserClassroom(db *gorm.DB, user_id string, provider string) ([]ClassRoomInfo, error) {
	finalresult := []ClassRoomInfo{}
	studentResult := []ClassRoomStudentRelation{}
	teacherResult := []ClassRoomTeacherRelation{}

	studentConditon := ClassRoomStudentRelation{
		ClassRoomUser: ClassRoomUser{
			Provider: provider,
			User:     user_id,
		},
	}

	if first := db.Where(&studentConditon).Find(&studentResult); first.Error != nil {
		// we always has default classroom, DO NOT return if not found any record
		if !first.RecordNotFound() {
			return nil, first.Error
		}
	}

	teacherCondition := ClassRoomTeacherRelation{
		ClassRoomUser: ClassRoomUser{
			Provider: provider,
			User:     user_id,
		},
	}

	if first := db.Where(&teacherCondition).Find(&teacherResult); first.Error != nil {
		// we always has default classroom, DO NOT return if not found any record
		if !first.RecordNotFound() {
			return nil, first.Error
		}
	}

	// distinct union student and teacher classroom
	result := unionUserClassroom(teacherResult, studentResult)

	result = append(result, ClassRoomUser{
		ClassroomID: consts.PUBLIC_CLASSROOM,
		Provider:    provider,
		User:        user_id,
	})

	for _, s := range result {
		cmInfo := ClassRoomInfo{
			Model: Model{
				ID: s.ClassroomID,
			},
		}
		rr, err := cmInfo.GetClassRoomDetail(db)

		if err != nil {
			return nil, err
		}

		if rr.IsPublic == TRUE {
			finalresult = append(finalresult, *rr)
		}
	}

	return finalresult, nil
}

func unionUserClassroom(teacherClassroom []ClassRoomTeacherRelation,
	studentClassroom []ClassRoomStudentRelation) []ClassRoomUser {

	mark := make(map[string]bool)

	result := []ClassRoomUser{}

	for _, t := range teacherClassroom {
		result = append(result, ClassRoomUser{
			ClassroomID: t.ClassroomID,
			User:        t.User,
			Provider:    t.Provider,
		})
		mark[fmt.Sprintf("%s-%s-%s", t.ClassroomID, t.User, t.Provider)] = true
	}

	for _, s := range studentClassroom {
		if _, ok := mark[fmt.Sprintf("%s-%s-%s", s.ClassroomID, s.User, s.Provider)]; !ok {
			result = append(result, ClassRoomUser{
				ClassroomID: s.ClassroomID,
				User:        s.User,
				Provider:    s.Provider,
			})
		}
	}

	return result

}

func getVMCourse(db *gorm.DB, id string) (*rfstackmodl.Course, error) {
	course := rfstackmodl.Course{
		Model: rfstackmodl.Model{
			ID: id,
		},
	}
	err := db.First(&course).Error

	if err != nil {
		return nil, err
	}
	return &course, nil
}
