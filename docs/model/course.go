package docs

type ListCourseResponse struct {
	Error   bool                      `json:"error" example:"false" format:"bool"`
	Courses []ClassroomCourseWithType `json:"courses"`
}

type GetCourseResponse struct {
	Error  bool      `json:"error" example:"false" format:"bool"`
	Course GetCourse `json:"course"`
}

type Search struct {
	Query string `json:"query" example:"course keyword"`
}

type AddCourseBeta struct {
	OauthUser
	Name         string              `json:"name" example:"jimmy的課" format:"string"`
	Introduction string              `json:"introduction" example:"課程說明" format:"string"`
	Image        ImageLabelValue     `json:"image"`
	Gpu          GPULabelValue       `json:"gpu"`
	Level        string              `json:"level" example:"basic" format:"string"`
	AccessType   string              `json:"accessType" example:"Ingress" format:"string"`
	Datasets     []DatasetLabelValue `json:"datasets"`
	Ports        []PortLabelValue    `json:"ports"`
	WritablePath string              `json:"writablePath" example:"/tmp/work"`
}

type UpdateCourseBeta struct {
	Id           string              `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	Name         string              `json:"name" example:"jimmy的課" format:"string"`
	Introduction string              `json:"introduction" example:"課程說明" format:"string"`
	Image        ImageLabelValue     `json:"image"`
	Gpu          GPULabelValue       `json:"gpu"`
	Level        string              `json:"level" example:"basic" format:"string"`
	Datasets     []DatasetLabelValue `json:"datasets"`
	Ports        []PortLabelValue    `json:"ports"`
	WritablePath string              `json:"writablePath" example:"/tmp/work"`
}

type GetCourse struct {
	Id           string              `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	CreatedAt    string              `json:"createAt" example:"2018-06-25T09:24:38Z"`
	User         string              `json:"user" example:"jimmy191@teacher"`
	Name         string              `json:"name" example:"jimmy的課" format:"string"`
	Introduction string              `json:"introduction" example:"課程說明" format:"string"`
	Image        ImageLabelValue     `json:"image"`
	Gpu          GPULabelValue       `json:"gpu"`
	Level        string              `json:"level" example:"basic" format:"string"`
	Datasets     []DatasetLabelValue `json:"datasets"`
	Ports        []PortLabelValue    `json:"ports"`
	WritablePath string              `json:"writablePath" example:"/tmp/work"`
	AccessType   string              `json:"accessType" example:"NodePort"`
}

type PortLabelValue struct {
	Name string `json:"name" example:"jupyter"`
	Port uint   `json:"port" example:"8080" format:"int64"`
}

type CourseTypeResponse struct {
	Error bool   `json:"error" example:"false" format:"bool"`
	Type  string `json:"type" example:"container"`
}

type CourseNameListResponse struct {
	Error   bool               `json:"error" example:"false" format:"bool"`
	Courses []CourseLabelValue `json:"courses"`
}

type CourseLabelValue struct {
	Label string `json:"label" example:"tensorflow introduction"`
	Value string `json:"value" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
}

type GPULabelValue struct {
	Label string `json:"label" example:"0"`
	Value uint   `json:"value" example:"0" format:"int64"`
}
