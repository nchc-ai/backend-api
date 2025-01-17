package beta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const TEST_USER = "some_github_user"

func TestListImage(t *testing.T) {

	r, err := findRepoImagesName("nchcai")
	// should be no error
	assert.NoError(t, err)

	// only onw image exist in nchcai repository
	assert.Equal(t, 1, len(r))

	// the only one image name is train
	assert.Equal(t, "train", r[0])
}

func TestListImageTag(t *testing.T) {
	// There is only one train-test:latest
	r, err := findImageTag(TEST_USER, "train-test")

	assert.NoError(t, err)
	// should only 1 tag of image train-test
	assert.Equal(t, 1, len(r))
	// the only one tag is latest
	assert.Equal(t, "latest", r[0])
}

func TestList(t *testing.T) {
	nchcaiResult, _ := listhubimage("")
	r, _ := listhubimage(TEST_USER)

	// listhubimage(TEST_USER) will loockup nchcai repo & one more TEST_USER repo
	// TEST_USER repo has only one train-test image
	assert.Equal(t, len(nchcaiResult)+1, len(r))
}

func TestListNonExistRepo(t *testing.T) {
	// an non-existing docker huh
	r, err := _find("ogreaaa")

	// error is not occur when lookup non-existing dockerhub
	assert.NoError(t, err)

	// but should return zero result
	assert.Equal(t, 0, len(r))
}
