package proxy

import (
	"github.com/nchc-ai/backend-api/pkg/model/common"
)

type TokenReq struct {
	Code string `json:"code"`
}

type IntrospectionReq struct {
	Token string `json:"token"`
}

type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token"`
}

type TokenResp struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

type ErrResp struct {
	Error string `json:"error"`
}

type RoleListResponse struct {
	Error bool                `json:"error"`
	Users []common.LabelValue `json:"users"`
}
