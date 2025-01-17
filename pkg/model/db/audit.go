package db

import "github.com/jinzhu/gorm"

type Audit struct {
	Model
	OauthUser
	CourseID    string  `gorm:"size:36"`
	ClassroomID *string `gorm:"size:72"`
	DeletedBy   *string `gorm:"size:20"`
}

func (Audit) TableName() string {
	return "jobAudit"
}

func (a *Audit) NewEntry(DB *gorm.DB) error {

	if err := DB.Create(a).Error; err != nil {
		return err
	}

	return nil
}
