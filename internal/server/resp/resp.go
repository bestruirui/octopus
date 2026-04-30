package resp

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type ResponseStruct struct {
	Code       int            `json:"code" example:"200"`
	Message    string         `json:"message" example:"success"`
	MessageKey string         `json:"message_key,omitempty"`
	MessageArgs map[string]any `json:"message_args,omitempty"`
	Data       interface{}    `json:"data,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, ResponseStruct{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, err string) {
	ErrorWithKey(c, code, err, inferErrorMessageKey(err), nil)
}

func ErrorWithKey(c *gin.Context, code int, err string, messageKey string, messageArgs map[string]any) {
	c.AbortWithStatusJSON(code, ResponseStruct{
		Code:       code,
		Message:    err,
		MessageKey: messageKey,
		MessageArgs: messageArgs,
	})
}

func InternalError(c *gin.Context) {
	Error(c, http.StatusInternalServerError, ErrInternalServer)
}

func BadGateway(c *gin.Context) {
	Error(c, http.StatusBadGateway, "upstream service unavailable")
}

func inferErrorMessageKey(message string) string {
	switch strings.TrimSpace(strings.ToLower(message)) {
	case strings.ToLower(ErrBadRequest):
		return "errors.invalidRequestParameters"
	case strings.ToLower(ErrInvalidJSON):
		return "errors.invalidJsonFormat"
	case strings.ToLower(ErrInvalidParam):
		return "errors.invalidParameter"
	case strings.ToLower(ErrValidation):
		return "errors.inputValidationFailed"
	case strings.ToLower(ErrDuplicateResource):
		return "errors.resourceAlreadyExists"
	case strings.ToLower(ErrResourceNotFound):
		return "errors.resourceNotFound"
	case strings.ToLower(ErrInternalServer):
		return "errors.internalServer"
	case strings.ToLower(ErrDatabase):
		return "errors.database"
	case strings.ToLower(ErrUnauthorized):
		return "errors.authenticationFailed"
	case strings.ToLower(ErrTooManyRequests):
		return "errors.tooManyRequests"
	case "permission denied":
		return "errors.permissionDenied"
	case "channel not found":
		return "errors.channelNotFound"
	case "group not found":
		return "errors.groupNotFound"
	case "group test progress not found":
		return "errors.groupTestProgressNotFound"
	case "ai route progress not found":
		return "errors.aiRouteProgressNotFound"
	case "missing progress id":
		return "errors.missingProgressId"
	case "channel name already exists":
		return "errors.channelNameAlreadyExists"
	case "group name already exists":
		return "errors.groupNameAlreadyExists"
	case "group contains duplicate channel/model items":
		return "errors.groupDuplicateChannelModelItems"
	case "database schema is outdated":
		return "errors.databaseSchemaOutdated"
	case "database schema is outdated; restart the service to apply the latest migrations":
		return "errors.databaseSchemaOutdatedRestart"
	case "upstream service unavailable":
		return "errors.upstreamServiceUnavailable"
	default:
		return ""
	}
}
