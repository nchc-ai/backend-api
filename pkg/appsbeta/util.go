package beta

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/nchc-ai/AI-Eduational-Platform/backend/pkg/model"
)

func RespondWithError(c *gin.Context, code int, format string, args ...interface{}) {
	resp := genericResponse(true, format, args...)
	c.JSON(code, resp)
	c.Abort()
}

func RespondWithOk(c *gin.Context, format string, args ...interface{}) {
	resp := genericResponse(false, format, args...)
	c.JSON(http.StatusOK, resp)
	c.Abort()
}

func genericResponse(isError bool, format string, args ...interface{}) model.GenericResponse {
	resp := model.GenericResponse{
		Error:   isError,
		Message: fmt.Sprintf(format, args...),
	}
	return resp
}

const dns1035LabelFmt string = "[a-z]([-a-z0-9]*[a-z0-9])?"

var dns1035LabelRegexp = regexp.MustCompile("^" + dns1035LabelFmt + "$")

func isDNS1035Label(value string) error {
	if !dns1035LabelRegexp.MatchString(value) {
		return errors.New("must consist of lower case alphanumeric characters or '-', start with an alphabetic character, and end with an alphanumeric character")
	}
	return nil
}
