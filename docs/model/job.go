package docs

type LaunchCourseRequest struct {
	User        string `json:"user" example:"user@gamil.com"`
	CourseId    string `json:"course_id" example:"5ab02011-9ab7-40c3-b691-d335f93a12ee"`
	ClassroomId string `json:"classroom_id" example:"5ab02011-9ab7-40c3-b691-d335f93a12ee"`
}

type LaunchCourseResponse struct {
	Error bool      `json:"error" example:"false" format:"bool"`
	Job   JobStatus `json:"job"`
}

type JobStatus struct {
	JobId  string `json:"job_id" example:"5ab02011-9ab7-40c3-b691-d335f93a12ee"`
	Ready  bool   `json:"ready" example:"false" format:"bool"`
	Status string `json:"status" example:"Created"`
}

type JobListResponse struct {
	Error bool      `json:"error" example:"false" format:"bool"`
	Jobs  []JobInfo `json:"jobs"`
}

type JobInfo struct {
	Id           string          `json:"id" example:"49a31009-7d1b-4ff2-badd-e8c717e2256c"`
	CourseID     string          `json:"course_id" example:"b86b2893-b876-45c2-a3f6-5e099c15d638"`
	StartAt      string          `json:"startAt" example:"2018-06-25T09:24:38Z"`
	Status       string          `json:"status" example:"Ready"`
	Name         string          `json:"name" example:"mage process"`
	Introduction string          `json:"introduction" example:"markdown text with escape"`
	Image        string          `json:"image" example:"nvidia/caffe:latest"`
	GPU          uint8           `json:"gpu" example:"1" format:"int64"`
	Level        string          `json:"level" example:"basic"`
	CanSnapshot  bool            `json:"canSnapshot" example:"true" format:"bool"`
	Dataset      []string        `json:"dataset" example:"cifar-10,mnist"`
	Service      []SVCLabelValue `json:"service"`
}

type SVCLabelValue struct {
	Label string `json:"label" example:"jupyter"`
	Value string `json:"value" example:"http://140.110.5.22:30010"`
}
