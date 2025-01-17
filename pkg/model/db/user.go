package db

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
)

type User struct {
	User       string  `sql:"unique_index:idx_first_second;size:50;not null" json:"user,omitempty"`
	Provider   *string `sql:"unique_index:idx_first_second;size:30;not null;default:'default-provider'" json:"-"`
	Uid        uint64  `gorm:"primary_key"`
	Role       string  `gorm:"size:30;not null;default:'student'"`
	Repository string
}

func (User) TableName() string {
	return "user"
}

func (u *User) NewEntry(DB *gorm.DB) error {

	if err := DB.Create(u).Error; err != nil {
		return err
	}

	return nil
}

func (u *User) GetRole(db *gorm.DB) (string, error) {

	if u.User == "" {
		return "", errors.New("user is not defined")
	}

	result := User{}

	if err := db.Where(u).Find(&result).Error; err != nil {
		return "", err
	}

	// Role has default value student, should not occur
	if result.Role == "" {
		return "", errors.New(fmt.Sprintf("%s/%s Role is not defined", *u.Provider, u.User))
	}

	return result.Role, nil
}

func (u *User) GetRepositroy(db *gorm.DB) (string, error) {

	if u.User == "" {
		return "", errors.New("user is not defined")
	}

	result := User{}

	if err := db.Where(u).Find(&result).Error; err != nil {
		return "", err
	}

	return result.Repository, nil
}

func (u *User) GetRoleList(db *gorm.DB) ([]string, error) {

	result := []string{}
	userResult := []User{}
	if err := db.Where(u).Find(&userResult).Error; err != nil {
		return nil, err
	}

	for _, v := range userResult {
		result = append(result, v.User)
	}

	return result, nil
}

func (u *User) Update(db *gorm.DB, newU *User) error {

	tx := db.Begin()
	if err := tx.Model(&User{}).Where(u).Updates(
		User{
			User:       newU.User,
			Provider:   newU.Provider,
			Role:       newU.Role,
			Repository: newU.Repository,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (u *User) FindUser(db *gorm.DB) (*User, error) {
	result := User{}

	if err := db.Where(u).Find(&result).Error; err != nil {
		return nil, err
	}

	return &result, nil
}

func MaxUid(db *gorm.DB) (uint64, error) {

	result := User{}
	if err := db.Order("uid desc").First(&result).Error; err != nil {
		return 0, err
	}
	return result.Uid, nil
}
