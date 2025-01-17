package db

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model/common"
	"github.com/nchc-ai/course-cron/pkg/cron"
)

type ClassRoomUser struct {
	ClassroomID string `gorm:"size:72;primary_key"`
	User        string `gorm:"size:50;primary_key"`
	Provider    string `gorm:"size:30;primary_key;default:'default-provider'"`
	Name        string `gorm:"size:50"`
}

type ClassRoomStudentRelation struct {
	ClassRoomUser
}

func (ClassRoomStudentRelation) TableName() string {
	return "classroomStudent"
}

func (students *ClassRoomStudentRelation) Update(DB *gorm.DB, list *[]common.LabelValue, provider string) error {

	// Delete all previous info
	if err := DB.Where(ClassRoomStudentRelation{
		ClassRoomUser: ClassRoomUser{
			ClassroomID: students.ClassroomID,
		},
	}).
		Delete(ClassRoomStudentRelation{}).Error; err != nil {
		return err
	}
	// add all new info
	return students.NewEntry(DB, list, provider)
}

func (students *ClassRoomStudentRelation) NewEntry(DB *gorm.DB, list *[]common.LabelValue, provider string) error {

	if list == nil {
		log.Warningf("student list is not defined")
		return nil
	}

	clist := []ClassRoomStudentRelation{}
	for _, student := range *list {
		clist = append(clist, ClassRoomStudentRelation{
			ClassRoomUser: ClassRoomUser{
				ClassroomID: students.ClassroomID,
				Name:        student.Label,
				User:        student.Value,
				Provider:    provider,
			},
		})
	}
	if err := batchInsert(DB, clist); err != nil {
		return err
	}
	return nil
}

type ClassRoomTeacherRelation struct {
	ClassRoomUser
}

func (ClassRoomTeacherRelation) TableName() string {
	return "classroomTeacher"
}

func (teachers *ClassRoomTeacherRelation) Update(DB *gorm.DB, list *[]common.LabelValue, provider string) error {
	// Delete all previous info
	if err := DB.Where(ClassRoomTeacherRelation{
		ClassRoomUser: ClassRoomUser{
			ClassroomID: teachers.ClassroomID,
		},
	}).
		Delete(ClassRoomTeacherRelation{}).Error; err != nil {
		return err
	}

	// add all new info
	return teachers.NewEntry(DB, list, provider)
}

func (teachers *ClassRoomTeacherRelation) NewEntry(DB *gorm.DB, list *[]common.LabelValue, provider string) error {

	if list == nil {
		log.Warningf("teacher list is not found")
		return nil
	}

	clist := []ClassRoomTeacherRelation{}
	for _, teacher := range *list {
		clist = append(clist, ClassRoomTeacherRelation{
			ClassRoomUser: ClassRoomUser{
				ClassroomID: teachers.ClassroomID,
				Name:        teacher.Label,
				User:        teacher.Value,
				Provider:    provider,
			},
		})
	}
	if err := batchInsert(DB, clist); err != nil {
		return err
	}

	return nil
}

type ClassRoomScheduleRelation struct {
	// foreign key
	ClassroomID string `gorm:"size:72;primary_key"`
	Schedule    string `gorm:"size:180;primary_key"`
}

func (ClassRoomScheduleRelation) TableName() string {
	return "classroomSchedule"
}

func (schedule *ClassRoomScheduleRelation) Update(DB *gorm.DB, list []string) error {
	// Delete all previous info
	if err := DB.Where(ClassRoomScheduleRelation{ClassroomID: schedule.ClassroomID}).
		Delete(ClassRoomScheduleRelation{}).Error; err != nil {
		return err
	}

	// add all new info
	return schedule.NewEntry(DB, list)
}

func (schedule *ClassRoomScheduleRelation) NewEntry(DB *gorm.DB, list []string) error {
	clist := []ClassRoomScheduleRelation{}
	for _, s := range list {

		if cron.IsValid(s) == false {
			return errors.New(fmt.Sprintf("cron format {%s} is not valid", s))
		}

		clist = append(clist, ClassRoomScheduleRelation{
			ClassroomID: schedule.ClassroomID,
			Schedule:    s,
		})
	}
	if err := batchInsert(DB, clist); err != nil {
		return err
	}

	return nil
}

type ClassRoomCourseRelation struct {
	// foreign key
	ClassroomID string `gorm:"size:72;primary_key"`
	// foreign key
	CourseID string `gorm:"size:36;primary_key"`
}

func (ClassRoomCourseRelation) TableName() string {
	return "classroomCourse"
}

func (course *ClassRoomCourseRelation) Update(DB *gorm.DB, list []common.LabelValue) error {
	// Delete all previous info
	if err := DB.Where(ClassRoomCourseRelation{ClassroomID: course.ClassroomID}).
		Delete(ClassRoomCourseRelation{}).Error; err != nil {
		return err
	}

	// add all new info
	return course.NewEntry(DB, list)
}

func (course *ClassRoomCourseRelation) NewEntry(DB *gorm.DB, list []common.LabelValue) error {

	clist := []ClassRoomCourseRelation{}
	for _, c := range list {
		clist = append(clist, ClassRoomCourseRelation{
			ClassroomID: course.ClassroomID,
			CourseID:    c.Value,
		})
	}
	if err := batchInsert(DB, clist); err != nil {
		return err
	}
	return nil
}

type CalendarTime struct {
	StartMonth uint   `gorm:"not null" sql:"type:TINYINT UNSIGNED" json:"startMonth,omitempty"`
	Length     uint   `gorm:"not null" sql:"type:TINYINT UNSIGNED" json:"length,omitempty"`
	StartDate  string `gorm:"size:10;not null" json:"startDate,omitempty"`
	EndDate    string `gorm:"size:10;not null" json:"endDate,omitempty"`
}

type ClassRoomCalendarRelation struct {
	CalendarTime
	ClassroomID string `gorm:"size:72;primary_key"`
	Seq         uint   `gorm:"primary_key" sql:"type:TINYINT UNSIGNED"`
}

func (ClassRoomCalendarRelation) TableName() string {
	return "classroomCalendar"
}

func (cal *ClassRoomCalendarRelation) NewEntry(DB *gorm.DB, calendars *[]CalendarTime) error {

	if calendars == nil {
		log.Warningf("calendar list is not defined")
		return nil
	}

	clist := []ClassRoomCalendarRelation{}

	for index, c := range *calendars {
		clist = append(clist, ClassRoomCalendarRelation{
			ClassroomID: cal.ClassroomID,
			// batchInsert() will ignore primary_key with blank value(0 for int), so we start from 1, instead of 0
			Seq: uint(index + 1),
			CalendarTime: CalendarTime{
				StartMonth: c.StartMonth,
				StartDate:  c.StartDate,
				EndDate:    c.EndDate,
				Length:     c.Length,
			},
		})
	}
	if err := batchInsert(DB, clist); err != nil {
		return err
	}
	return nil
}

func (cal *ClassRoomCalendarRelation) Update(DB *gorm.DB, calendars *[]CalendarTime) error {
	// Delete all previous info
	if err := DB.Where(ClassRoomCalendarRelation{ClassroomID: cal.ClassroomID}).
		Delete(ClassRoomCalendarRelation{}).Error; err != nil {
		return err
	}

	// add all new info
	return cal.NewEntry(DB, calendars)
}

type ClassRoomSelectedOptionRelation struct {
	Label       string `gorm:"size:20;not null"`
	Value       string `gorm:"size:20;not null"`
	ClassroomID string `gorm:"size:72"`
}

func (ClassRoomSelectedOptionRelation) TableName() string {
	return "classroomSelectedOption"
}

func (opt *ClassRoomSelectedOptionRelation) NewEntry(DB *gorm.DB, options []common.LabelValue) error {
	clist := []ClassRoomSelectedOptionRelation{}
	for _, s := range options {

		clist = append(clist, ClassRoomSelectedOptionRelation{
			ClassroomID: opt.ClassroomID,
			Label:       s.Label,
			Value:       s.Value,
		})
	}
	if err := batchInsert(DB, clist); err != nil {
		return err
	}

	return nil
}

func (opt *ClassRoomSelectedOptionRelation) Update(DB *gorm.DB, options []common.LabelValue) error {
	// Delete all previous info
	if err := DB.Where(ClassRoomSelectedOptionRelation{ClassroomID: opt.ClassroomID}).
		Delete(ClassRoomSelectedOptionRelation{}).Error; err != nil {
		return err
	}

	// add all new info
	return opt.NewEntry(DB, options)
}

func batchInsert(DB *gorm.DB, slice interface{}) error {

	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return errors.New("InterfaceSlice() given a non-slice type")
	}

	objArr := make([]interface{}, s.Len())

	ss := strings.Split(s.Type().String(), ".")
	switch ss[len(ss)-1] {
	case "ClassRoomCourseRelation":
		for i := 0; i < s.Len(); i++ {
			objArr[i] = s.Index(i).Interface().(ClassRoomCourseRelation)
		}
	case "ClassRoomScheduleRelation":
		for i := 0; i < s.Len(); i++ {
			objArr[i] = s.Index(i).Interface().(ClassRoomScheduleRelation)
		}
	case "ClassRoomTeacherRelation":
		for i := 0; i < s.Len(); i++ {
			objArr[i] = s.Index(i).Interface().(ClassRoomTeacherRelation)
		}
	case "ClassRoomStudentRelation":
		for i := 0; i < s.Len(); i++ {
			objArr[i] = s.Index(i).Interface().(ClassRoomStudentRelation)
		}
	case "ClassRoomCalendarRelation":
		for i := 0; i < s.Len(); i++ {
			objArr[i] = s.Index(i).Interface().(ClassRoomCalendarRelation)
		}
	case "ClassRoomSelectedOptionRelation":
		for i := 0; i < s.Len(); i++ {
			objArr[i] = s.Index(i).Interface().(ClassRoomSelectedOptionRelation)
		}
	}

	if len(objArr) == 0 {
		return nil
	}

	mainObj := objArr[0]
	mainScope := DB.NewScope(mainObj)
	mainFields := mainScope.Fields()
	quoted := make([]string, 0, len(mainFields))
	for i := range mainFields {
		// If primary key has blank value (0 for int, "" for string, nil for interface ...), skip it.
		// If field is ignore field, skip it.
		if (mainFields[i].IsPrimaryKey && mainFields[i].IsBlank) || (mainFields[i].IsIgnored) {
			continue
		}
		quoted = append(quoted, mainScope.Quote(mainFields[i].DBName))
	}

	placeholdersArr := make([]string, 0, len(objArr))

	for _, obj := range objArr {
		scope := DB.NewScope(obj)
		fields := scope.Fields()
		placeholders := make([]string, 0, len(fields))
		for i := range fields {
			if (fields[i].IsPrimaryKey && fields[i].IsBlank) || (fields[i].IsIgnored) {
				continue
			}
			placeholders = append(placeholders, scope.AddToVars(fields[i].Field.Interface()))
		}
		placeholdersStr := "(" + strings.Join(placeholders, ", ") + ")"
		placeholdersArr = append(placeholdersArr, placeholdersStr)
		// add real variables for the replacement of placeholders' '?' letter later.
		mainScope.SQLVars = append(mainScope.SQLVars, scope.SQLVars...)
	}

	mainScope.Raw(fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		mainScope.QuotedTableName(),
		strings.Join(quoted, ", "),
		strings.Join(placeholdersArr, ", "),
	))

	if _, err := mainScope.SQLDB().Exec(mainScope.SQL, mainScope.SQLVars...); err != nil {
		return err
	}
	return nil

}
