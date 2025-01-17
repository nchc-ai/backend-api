package beta

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/nchc-ai/backend-api/pkg/consts"
	"github.com/nchc-ai/backend-api/pkg/model/config"
	"github.com/nchc-ai/backend-api/pkg/model/db"
	"github.com/nchc-ai/backend-api/pkg/model/proxy"
	"github.com/nchc-ai/backend-api/pkg/util"
	provider_err "github.com/nchc-ai/oauth-provider/pkg/errors"
	"github.com/nchc-ai/oauth-provider/pkg/provider"
)

type Proxy struct {
	provider provider.Provider
	db       *gorm.DB
	config   *config.Config
}

// @Summary Exchange token from Provider
// @Description Exchange token from Provider
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Param token_request body docs.TokenReq true "token request"
// @Success 200 {object} docs.TokenResp
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/proxy/token [post]
func (p *Proxy) GetToken(c *gin.Context) {

	var req proxy.TokenReq
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	token, err := p.provider.GetToken(req.Code)

	if err != nil {
		log.Errorf("Exchange Token fail: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Exchange Token fail: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK,
		proxy.TokenResp{
			Token:        token.AccessToken,
			RefreshToken: token.RefreshToken,
		},
	)
}

// @Summary Refresh token with provider
// @Description Refresh token with provider
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Param refresh_token body docs.RefreshTokenReq true "refresh token"
// @Success 200 {object} docs.TokenResp
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/proxy/refresh [post]
func (p *Proxy) RefreshToken(c *gin.Context) {
	var req proxy.RefreshTokenReq
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	newToken, err := p.provider.RefreshToken(req.RefreshToken)

	if err != nil {
		log.Errorf("Refresh Token fail: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Refresh Token fail: %s", err.Error())
		return
	}

	c.JSON(http.StatusOK,
		proxy.TokenResp{
			Token:        newToken.AccessToken,
			RefreshToken: newToken.RefreshToken,
		},
	)
}

// @Summary Get token meta information from provider
// @Description Get token meta information from provider
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Param Introspection_token body docs.IntrospectionReq true "Introspection token"
// @Success 200 {object} docs.IntrospectResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/proxy/introspection [post]
func (p *Proxy) Introspection(c *gin.Context) {

	var req proxy.IntrospectionReq
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	introspectionResult, err := p.provider.Introspection(req.Token)

	if err != nil {
		errStr := fmt.Sprintf("Introspection Token {%s} fail: %s", req.Token, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	newU := db.User{
		User:     introspectionResult.Username,
		Provider: util.StringPtr(fmt.Sprintf("%s:%s", p.provider.Type(), p.provider.Name())),
	}

	role, err := newU.GetRole(p.db)
	if err != nil {
		errStr := fmt.Sprintf("Introspection Token {%s} fail: %s", req.Token, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError,
			fmt.Sprintf(consts.ERROR_LOGIN_ROLE_NOT_FOUND, introspectionResult.Username))
		return
	}
	introspectionResult.Role = role

	c.JSON(http.StatusOK, introspectionResult)
}

// @Summary Revoke all tokens of a user
// @Description Revoke all tokens of a user
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Param user_token body docs.IntrospectionReq true "user token"
// @Success 200 {object} docs.PlainResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/proxy/logout [post]
func (p *Proxy) Logout(c *gin.Context) {
	// Logout and Introspection use the same request format
	var req proxy.IntrospectionReq
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	logoutResp, err := p.provider.Logout(req.Token)

	if err != nil {
		errStr := fmt.Sprintf("Logout use Token {%s} fail: %s", req.Token, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, logoutResp)
}

// @Summary Register a new user
// @Description Register a new user
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Param new_user_info body docs.UserInfo true "user information"
// @Success 200 {object} docs.PlainResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Router /beta/proxy/register [post]
func (p *Proxy) RegisterUser(c *gin.Context) {

	// check uid range before go on
	uidStart, _ := strconv.Atoi(strings.Split(p.config.APIConfig.UidRange, "/")[0])
	uidCount, _ := strconv.Atoi(strings.Split(p.config.APIConfig.UidRange, "/")[1])
	maxUid, err := db.MaxUid(p.db)

	if err != nil && err.Error() != "record not found" {
		log.Errorf("Failed to find out maximum uid: %s", err.Error())
		RespondWithError(c, http.StatusInternalServerError, "Failed to find out maximum uid: %s", err.Error())
		return
	}

	if maxUid >= uint64(uidStart+uidCount-1) {
		log.Warningf("Registered User already reach maximum count {%d}", uidCount)
		RespondWithError(c, http.StatusInternalServerError, "Registered User already reach maximum count {%d}", uidCount)
		return
	}

	var req provider.UserInfo
	err = c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	// empty string will create student role
	if !(req.Role == "student" || req.Role == "teacher" || req.Role == "superuser" || req.Role == "") {
		log.Errorf("{%s} is not valid role string", req.Role)
		RespondWithError(c, http.StatusBadRequest, "{%s} is not valid role string", req.Role)
		return
	}

	registerResult, err := p.provider.RegisterUser(&req)

	if err != nil && !provider_err.IsNotSupport(err) {
		errStr := fmt.Sprintf("Regsiter new user {%s} fail: %s", req.Username, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	newU := db.User{
		User:       req.Username,
		Provider:   util.StringPtr(fmt.Sprintf("%s:%s", p.provider.Type(), p.provider.Name())),
		Role:       req.Role,
		Repository: req.Repository,
	}

	err = newU.NewEntry(p.db)
	if err != nil {
		errStr := fmt.Sprintf("Regsiter new user {%s} in local DB fail: %s", req.Username, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, registerResult)

}

// @Summary Update a existing user information
// @Description Update a existing user information
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Param user_info body docs.UpdatedUser true "user information"
// @Success 200 {object} docs.PlainResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/proxy/update [post]
func (p *Proxy) UpdateUserBasicInfo(c *gin.Context) {
	updateUser(p, c)
}

// @Summary Change user password
// @Description Change user password
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Param user_password body docs.PasswordInfo true "new password"
// @Success 200 {object} docs.PlainResponse
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/proxy/changePW [post]
func (p *Proxy) ChangeUserPassword(c *gin.Context) {
	updateUser(p, c)
}

// @Summary query a existing user information
// @Description query a existing user information
// @Tags Proxy
// @Accept  json
// @Produce  json
// @Success 200 {object} docs.UserInfo
// @Failure 400 {object} docs.GenericErrorResponse
// @Failure 401 {object} docs.GenericErrorResponse
// @Failure 403 {object} docs.GenericErrorResponse
// @Failure 500 {object} docs.GenericErrorResponse
// @Security ApiKeyAuth
// @Router /beta/proxy/query [get]
func (p *Proxy) QueryUser(c *gin.Context) {

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		log.Error("Can not find token in Authorization header")
		RespondWithError(c, http.StatusBadRequest, "Can not find token in Authorization header")
		return
	}

	bearerToken := strings.Split(authHeader, " ")

	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		log.Errorf("Can not find token in Authorization header: %s", authHeader)
		RespondWithError(c, http.StatusBadRequest, "Can not find token in Authorization header")
		return
	}

	token := bearerToken[1]

	result, err := p.provider.QueryUser(token)

	if err != nil {
		errStr := fmt.Sprintf("query user from token fail: %s", err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	u := db.User{
		User:     result.Username,
		Provider: util.StringPtr(fmt.Sprintf("%s:%s", p.provider.Type(), p.provider.Name())),
	}
	repo, err := u.GetRepositroy(p.db)
	if err != nil {
		errStr := fmt.Sprintf("query user defined repo fail: %s", err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}
	result.Repository = repo

	c.JSON(http.StatusOK, result)
}

// PRIVATE function
func updateUser(p *Proxy, c *gin.Context) {
	var req provider.UserInfo
	err := c.BindJSON(&req)
	if err != nil {
		log.Errorf("Failed to parse spec request request: %s", err.Error())
		RespondWithError(c, http.StatusBadRequest, "Failed to parse spec request request: %s", err.Error())
		return
	}

	if !(req.Role == "student" || req.Role == "teacher" || req.Role == "superuser" || req.Role == "") {
		log.Errorf("{%s} is not valid role string", req.Role)
		RespondWithError(c, http.StatusBadRequest, "{%s} is not valid role string", req.Role)
		return
	}

	result, err := p.provider.UpdateUser(&req)

	if err != nil && !provider_err.IsNotSupport(err) {
		errStr := fmt.Sprintf("update user {%s} fail: %s", req.Username, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	newU := db.User{
		User:     req.Username,
		Provider: util.StringPtr(fmt.Sprintf("%s:%s", p.provider.Type(), p.provider.Name())),
	}

	if err := newU.Update(p.db, &db.User{
		User:       req.Username,
		Provider:   util.StringPtr(fmt.Sprintf("%s:%s", p.provider.Type(), p.provider.Name())),
		Role:       req.Role,
		Repository: req.Repository,
	}); err != nil {
		errStr := fmt.Sprintf("update user {%s} in local DB fail: %s", req.Username, err.Error())
		log.Errorf(errStr)
		RespondWithError(c, http.StatusInternalServerError, errStr)
		return
	}

	c.JSON(http.StatusOK, result)
}
