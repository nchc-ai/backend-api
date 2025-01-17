package model

import (
	"time"

	"github.com/nchc-ai/backend-api/pkg/model/common"
	"github.com/nchc-ai/backend-api/pkg/model/db"
	v1 "k8s.io/api/core/v1"
)

type HealthDatabaseResponse struct {
	GenericResponse
	Tables []string `json:"tables"`
}

type GenericResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

type Node struct {
	Name   string               `json:"name"`
	Status v1.NodeConditionType `json:"status"`
}

type HealthKubernetesResponse struct {
	Error   bool   `json:"error"`
	Message []Node `json:"message"`
}

type GenericRequest struct {
	Message string `json:"message"`
}

type ListCourseResponse struct {
	Error   bool        `json:"error"`
	Courses []db.Course `json:"courses"`
}

type GetCourseResponse struct {
	Error  bool      `json:"error"`
	Course db.Course `json:"course"`
}

type DatasetsListResponse struct {
	Error    bool                `json:"error"`
	Datasets []common.LabelValue `json:"datasets"`
}

type ImagesListResponse struct {
	Error  bool                `json:"error"`
	Images []common.LabelValue `json:"images"`
}

type LaunchCourseRequest struct {
	User        string `json:"user"`
	CourseId    string `json:"course_id"`
	ClassroomId string `json:"classroom_id"`
}

type LaunchCourseResponse struct {
	Error bool      `json:"error"`
	Job   JobStatus `json:"job"`
}

type JobStatus struct {
	JobId  string `json:"job_id"`
	Ready  bool   `json:"ready"`
	Status string `json:"status"`
}

type JobListResponse struct {
	Error bool      `json:"error"`
	Jobs  []JobInfo `json:"jobs"`
}

type JobInfo struct {
	Id           string              `json:"id"`
	CourseID     string              `json:"course_id"`
	StartAt      time.Time           `json:"startAt"`
	Status       string              `json:"status"`
	Name         string              `json:"name"`
	Introduction string              `json:"introduction"`
	Image        string              `json:"image"`
	GPU          int32               `json:"gpu"`
	Level        string              `json:"level"`
	CanSnapshot  bool                `json:"canSnapshot"`
	Service      []common.LabelValue `json:"service"`
}

type Search struct {
	Query string `json:"query"`
}

type ListClassroomResponse struct {
	Error      bool               `json:"error"`
	Classrooms []db.ClassRoomInfo `json:"classrooms"`
}

type GetClassroomResponse struct {
	Error     bool             `json:"error"`
	Classroom db.ClassRoomInfo `json:"classroom"`
}

type UploaduserResponse struct {
	Error bool             `json:"error"`
	Users []UserLabelValue `json:"users"`
}

type CourseTypeResponse struct {
	Error bool   `json:"error"`
	Type  string `json:"type"`
}

type CourseNameListResponse struct {
	Error   bool                `json:"error"`
	Courses []common.LabelValue `json:"courses"`
}

type CommitReq struct {
	JobID     string `json:"id"`
	ImageName string `json:"name"`
}

type UserLabelValue struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
