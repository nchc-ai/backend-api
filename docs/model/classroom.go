package docs

type ListClassroomResponse struct {
	Error      bool            `json:"error" example:"false" format:"bool"`
	Classrooms []ClassRoomInfo `json:"classrooms"`
}

type SimpleListClassroomResponse struct {
	Error      bool                  `json:"error" example:"false" format:"bool"`
	Classrooms []SimpleClassRoomInfo `json:"classrooms"`
}

type BasicClassRoomInfo struct {
	ID          string `json:"id" example:"default" format:"string"`
	Name        string `json:"name" example:"Public Course" format:"string"`
	Public      bool   `json:"public" example:"true" format:"bool"`
	Description string `json:"description" example:"description" format:"string"`
	CreatedAt   string `json:"createAt" example:"2018-06-25T09:24:38Z"`
}

type ClassRoomInfo struct {
	BasicClassRoomInfo
	StudentCount int               `json:"studentCount" example:"18" format:"int"`
	Teachers     []UserLabelValue  `json:"teachers"`
	Course       []ClassroomCourse `json:"courseInfo"`
}

type ClassRoomInfoWithStudent struct {
	BasicClassRoomInfo
	StudentCount int               `json:"studentCount" example:"18" format:"int"`
	Teachers     []UserLabelValue  `json:"teachers"`
	Students     []UserLabelValue  `json:"students"`
	Course       []ClassroomCourse `json:"courseInfo"`
	Schedules    Schedule          `json:"schedule"`
	CalendarTime []CalendarTime    `json:"calendar"`
}

type SimpleClassRoomInfo struct {
	BasicClassRoomInfo
	Teachers     []UserLabelValue `json:"teachers"`
	CalendarTime []CalendarTime   `json:"calendar"`
}

type AddClassroom struct {
	Name         string             `json:"name" example:"國衛院教室" format:"string"`
	Description  string             `json:"description" example:"國衛院教室說明" format:"string"`
	IsPublic     bool               `json:"public" example:"true" format:"bool"`
	Schedule     Schedule           `json:"schedule"`
	Teachers     []UserLabelValue   `json:"teachers"`
	Students     []UserLabelValue   `json:"students"`
	Courses      []CourseLabelValue `json:"courses"`
	CalendarTime []CalendarTime     `json:"calendar"`
}

type UpdateClassroom struct {
	AddClassroom
	Id string `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
}

type UploadUserResponse struct {
	Error bool                `json:"error" example:"false" format:"bool"`
	users []AccountLabelValue `json:"users"`
}

type GetClassroomResponse struct {
	Error     bool                     `json:"error"`
	Classroom ClassRoomInfoWithStudent `json:"classroom"`
}

type ClassroomCourseWithType struct {
	Id        string `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	CreatedAt string `json:"createAt" example:"2018-06-25T09:24:38Z"`
	Name      string `json:"name" example:"jimmy的課" format:"string"`
	Level     string `json:"level" example:"basic" format:"string"`
	Type      string `json:"type" example:"CONTAINER" format:"string"`
}

type ClassroomCourse struct {
	ClassroomCourseWithType
	ClassroomID string `json:"roomId" example:"d385a235-e1a1-49de-8b65-0b0d6a5783e5" format:"string"`
}

type AccountLabelValue struct {
	Name  string `json:"name" example:"莊小明"`
	Email string `json:"email" example:"student1@gmail.com"`
}

type CalendarTime struct {
	StartMonth uint   `json:"startMonth" example:"2" format:"int"`
	Length     uint   `json:"length" example:"3" format:"int"`
	StartDate  string `json:"startDate" example:"2019-01-22" format:"string"`
	EndDate    string `json:"endDate" example:"2019-01-25" format:"string"`
}

type Schedule struct {
	Description    string          `json:"description" example:"2019/2/11 至 2019/6/15 每周一、三" format:"string"`
	CronFormat     []string        `json:"cronFormat" example:"* * * * * *" format:"string"`
	StartDate      string          `json:"startDate" example:"2019-01-25" format:"string"`
	EndDate        string          `json:"endDate" example:"2019-01-26" format:"string"`
	SelectedType   int             `json:"selectedType" example:"1" format:"int"`
	SelectedOption []OptLabelValue `json:"selectedOption"`
}

type OptLabelValue struct {
	Label string `json:"label" example:"平日"`
	Value string `json:"value" example:"1-5"`
}
