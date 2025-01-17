package beta

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/backend-api/pkg/consts"
	"github.com/nchc-ai/backend-api/pkg/model"
	"github.com/nchc-ai/backend-api/pkg/model/common"
	"github.com/nchc-ai/backend-api/pkg/model/db"
	img "github.com/nchc-ai/backend-api/pkg/model/image"
	"github.com/nchc-ai/backend-api/pkg/util"
	"github.com/nchc-ai/oauth-provider/pkg/provider"
)

type Image struct {
	provider provider.Provider
	db       *gorm.DB
}

// @Summary List images in nchcai/train dockerhub repo
// @Description List images in nchcai/train dockerhub repo
// @Tags Image
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.ImagesListResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/images [get]
func (i *Image) List(c *gin.Context) {

	providerName, exist := c.Get("Provider")
	if exist == false {
		providerName = ""
	}

	userRepo := ""
	u, err := getUserInfoFromToken(i.provider, c)
	if err != nil {
		log.Warningf("Something wrong when query user from token: %s", err.Error())
		log.Warningf("Only list nchcai/train dockerhub image")
	} else {
		uu := db.User{
			User:     u.Username,
			Provider: util.StringPtr(providerName.(string)),
		}
		repo, err1 := uu.GetRepositroy(i.db)
		if err1 != nil {
			log.Warningf("Something wrong when query user's repository: %s", err1.Error())
			log.Warningf("Only list nchcai/train dockerhub image")
		} else {
			log.Infof("List images from nchcai/train and %s", repo)
			userRepo = repo
		}
	}
	// get userRepo name from user db
	imgs, err := listhubimage(userRepo)
	if err != nil {
		log.Errorf("Failed to get images information from Dockerhub: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Failed to get images information from Dockerhub: %s", err.Error())
		return
	}

	imageList := []common.LabelValue{}

	for _, n := range imgs {
		lbval := common.LabelValue{
			Label: n,
			Value: n,
		}
		imageList = append(imageList, lbval)
	}

	c.JSON(http.StatusOK, model.ImagesListResponse{
		Error:  false,
		Images: imageList,
	})
}

// @Summary Commit current container into new image
// @Description Commit current container into new image
// @Tags Image
// @Accept  json
// @Produce  json
// @Param commit body docs.CommitImage true "course job id and new image name:tag"
// @Success 200 {object} docs.GenericOKResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/images/commit [post]
func (i *Image) Commit(c *gin.Context) {
	// todo: use docker api to commit image (#66)
	// here is a mockup api implementation

	var req model.CommitReq

	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	s := strings.Split(req.ImageName, ":")

	tag := ""
	if len(s) == 2 {
		tag = s[1]
	} else {
		tag = "latest"
	}

	RespondWithOk(c, "Commit Job {%s} into new image {%s}, use tag {%s}", req.JobID, s[0], tag)
}

func listhubimage(userRepo string) ([]string, error) {

	// 1. query system default repo (nchcai/train:<all-tag>)

	var wg sync.WaitGroup
	wg.Add(2)

	var err1 error
	var r1 []string
	go func() {
		defer wg.Done()
		r, err := _find(consts.AiTrainUser)
		if err != nil {
			err1 = err
			return
		}
		r1 = r
	}()

	// 2. query user defined repo (<user-repo>/<trainXXX>:<all-tag>)
	//	  Only image name start with train is feasible.
	var err2 error
	var r2 []string
	go func() {
		defer wg.Done()
		r, err := _find(userRepo)
		if err != nil {
			err2 = err
			return
		}
		r2 = r
	}()
	wg.Wait()

	if err1 != nil {
		return nil, err1
	}

	if err2 != nil {
		return nil, err2
	}

	result := append([]string{}, append(r1, r2...)...)
	return result, nil
}

func findRepoImagesName(userRepo string) ([]string, error) {

	result := []string{}
	nextURL := consts.BaseDockerHubUrl + userRepo

	// golang style do-while loop
	//https://yourbasic.org/golang/do-while-loop/
	for {
		req, err := http.NewRequest("GET", nextURL, nil)

		if err != nil {
			return nil, err
		}

		req.Header.Add("Cache-Control", "no-cache")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		err = res.Body.Close()
		if err != nil {
			return nil, err
		}

		imgResult := img.ImageResult{}
		err = json.Unmarshal(data, &imgResult)
		if err != nil {
			return nil, err
		}

		for _, aa := range imgResult.Results {
			if strings.HasPrefix(aa.Name, consts.AiTrainImagePrefix) {
				result = append(result, aa.Name)
			}
		}

		if imgResult.Next == nil {
			break
		} else {
			nextURL = *imgResult.Next
		}
	}
	return result, nil
}

func findImageTag(userRepo, image string) ([]string, error) {
	result := []string{}
	nextURL := consts.BaseDockerHubUrl + strings.Join([]string{userRepo, image, "tags"}, "/")

	for {
		req, err := http.NewRequest("GET", nextURL, nil)

		if err != nil {
			return nil, err
		}

		req.Header.Add("Cache-Control", "no-cache")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		err = res.Body.Close()
		if err != nil {
			return nil, err
		}

		tagResult := img.TagResult{}
		err = json.Unmarshal(data, &tagResult)
		if err != nil {
			return nil, err
		}

		for _, aa := range tagResult.Results {
			result = append(result, aa.Name)
		}

		if tagResult.Next == nil {
			break
		} else {
			nextURL = *tagResult.Next
		}
	}

	return result, nil
}

func _find(user string) ([]string, error) {

	result := []string{}

	if user == "" {
		return result, nil
	}

	nchcaiImg, err := findRepoImagesName(user)
	if err != nil {
		return nil, err
	}

	for _, imgName := range nchcaiImg {
		imgTags, err := findImageTag(user, imgName)
		if err != nil {
			return nil, err
		}

		for _, tag := range imgTags {
			result = append(result, fmt.Sprintf("%s/%s:%s", user, imgName, tag))
		}
	}

	return result, nil
}

// todo: refact. same with proxy.QueryUser()
func getUserInfoFromToken(p provider.Provider, c *gin.Context) (*provider.UserInfo, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return nil, errors.New("Can not find token in Authorization header")
	}

	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		return nil, errors.New(fmt.Sprintf("Can not find token in Authorization header: %s", authHeader))
	}

	token := bearerToken[1]

	result, err := p.QueryUser(token)
	if err != nil {
		return nil, err
	}

	return result, nil
}
