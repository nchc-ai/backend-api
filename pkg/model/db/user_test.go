package db

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/nchc-ai/backend-api/pkg/util"
	"github.com/stretchr/testify/assert"
)

var Sqlite *gorm.DB

const (
	GO_OAUTH    = "go-oauth"
	RoleTeacher = "teacher"
	RoleAdmin   = "superuser"
	RoleStudent = "student"
	REPO        = "some_repo"
)

func TestMain(m *testing.M) {

	// Setup testing
	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		return
	}
	Sqlite = db
	Sqlite.AutoMigrate(&User{})

	// Start Testing
	m.Run()

	// Teardown testing
	defer Sqlite.Close()
	os.Remove("test.db")
}

func TestMaxUid_Before(t *testing.T) {
	maxUID, err := MaxUid(Sqlite)

	assert.EqualError(t, err, "record not found")
	assert.Equal(t, 0, int(maxUID))

}

func TestUser_NewEntry(t *testing.T) {
	u_teacher := User{
		User:       "user@teacher",
		Provider:   util.StringPtr(GO_OAUTH + ":" + "test-provider"),
		Role:       RoleTeacher,
		Repository: REPO,
	}

	u_teacher1 := User{
		User:       "user1@teacher",
		Provider:   util.StringPtr(GO_OAUTH + ":" + "test-provider"),
		Role:       RoleTeacher,
		Repository: REPO,
	}

	u_student := User{
		User:       "user@student",
		Provider:   util.StringPtr(GO_OAUTH + ":" + "test-provider"),
		Role:       RoleStudent,
		Repository: "",
	}

	u_admin := User{
		User:       "user@admin",
		Provider:   util.StringPtr(GO_OAUTH + ":" + "test-provider"),
		Role:       RoleAdmin,
		Repository: "",
	}

	// add 4 user, 2 teachers, 1 student, 1 admin
	err := u_teacher.NewEntry(Sqlite)
	assert.NoError(t, err)

	err = u_teacher1.NewEntry(Sqlite)
	assert.NoError(t, err)

	err = u_admin.NewEntry(Sqlite)
	assert.NoError(t, err)

	err = u_student.NewEntry(Sqlite)
	assert.NoError(t, err)

	count := 0
	Sqlite.Table(u_admin.TableName()).Count(&count)

	// should return 4 user
	assert.Equal(t, count, 4)
}

func TestUser_GetRole(t *testing.T) {

	u := User{
		User:     "user@teacher",
		Provider: util.StringPtr(GO_OAUTH + ":" + "test-provider"),
	}
	role, err := u.GetRole(Sqlite)
	assert.NoError(t, err)

	// should get teacher role when query teacher id
	assert.Equal(t, "teacher", role)
}

func TestUser_NewEntryWithoutRole(t *testing.T) {
	u := User{
		User:       "user2@teacher",
		Provider:   util.StringPtr(GO_OAUTH + ":" + "test-provider"),
		Role:       "",
		Repository: REPO,
	}
	err := u.NewEntry(Sqlite)
	assert.NoError(t, err)

	r, err := u.GetRole(Sqlite)
	assert.NoError(t, err)
	// if role is not specified, student as default
	assert.Equal(t, RoleStudent, r)
}

func TestUser_GetRepositroy(t *testing.T) {

	u := User{
		User:     "user@teacher",
		Provider: util.StringPtr(GO_OAUTH + ":" + "test-provider"),
	}

	repo, err := u.GetRepositroy(Sqlite)
	assert.NoError(t, err)

	// should get correct repo name
	assert.Equal(t, REPO, repo)
}

func TestUser_GetRepositroyWithoutProvider(t *testing.T) {

	u := User{
		User:     "user@teacher",
		Provider: util.StringPtr(""),
	}
	repo, err := u.GetRepositroy(Sqlite)

	// if provider is not specified, return no result
	assert.EqualError(t, err, "record not found")
	assert.Equal(t, "", repo)
}

func TestUser_GetRoleList(t *testing.T) {
	u := User{
		Provider: util.StringPtr(GO_OAUTH + ":" + "test-provider"),
		Role:     RoleTeacher,
	}

	r, _ := u.GetRoleList(Sqlite)

	// should get 2 teacher
	assert.Equal(t, len(r), 2)
}

func TestUser_Update(t *testing.T) {

	u := User{
		User:     "user@teacher",
		Provider: util.StringPtr(GO_OAUTH + ":" + "test-provider"),
	}

	// Original role is teacher, repo is REPO
	repo, _ := u.GetRepositroy(Sqlite)
	assert.Equal(t, REPO, repo)
	role, _ := u.GetRole(Sqlite)
	assert.Equal(t, RoleTeacher, role)

	// Execute update
	err := u.Update(Sqlite, &User{
		User:       "user@teacher",
		Provider:   util.StringPtr(GO_OAUTH + ":" + "test-provider"),
		Role:       RoleStudent,
		Repository: "xyz",
	})
	assert.NoError(t, err)

	// After update, new role is student, new repo is xyz
	newRepo, _ := u.GetRepositroy(Sqlite)
	assert.Equal(t, "xyz", newRepo)
	newRole, _ := u.GetRole(Sqlite)
	assert.Equal(t, RoleStudent, newRole)
}

func TestMaxUid_After(t *testing.T) {
	maxUID, err := MaxUid(Sqlite)
	assert.NoError(t, err)
	assert.Equal(t, 5, int(maxUID))
}

//func TestUser_FindUser(t *testing.T) {
//
//}
