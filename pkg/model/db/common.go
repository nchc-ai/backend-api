package db

import "time"

type OauthUser struct {
	User     string `gorm:"size:50;not null" json:"user,omitempty"`
	Provider string `gorm:"size:30;not null;default:'default-provider'" json:"-"`
}

type Model struct {
	ID        string     `gorm:"primary_key;size:72" json:"id"`
	CreatedAt time.Time  `json:"createAt"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `sql:"index" json:"-"`
}

type Dataset struct {
	// foreign key
	CourseID    string `gorm:"primary_key;size:36"`
	DatasetName string `gorm:"primary_key"`
}

func (Dataset) TableName() string {
	return "containerDatasets"
}

type Port struct {
	Name string `gorm:"size:20;not null" json:"name"`
	Port uint   `gorm:"primary_key" sql:"type:SMALLINT UNSIGNED NOT NULL" json:"port"`
	// foreign key
	CourseID string `gorm:"primary_key;size:36" json:"-"`
}

func (Port) TableName() string {
	return "containerPorts"
}
