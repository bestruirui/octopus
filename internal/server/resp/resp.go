package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ResponseStruct struct {
	Code    int         `json:"code" example:"200"`
	Message string      `json:"message" example:"success"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, ResponseStruct{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, err string) {
	c.AbortWithStatusJSON(code, ResponseStruct{
		Code:    code,
		Message: err,
	})
}

func InternalError(c *gin.Context) {
	Error(c, http.StatusInternalServerError, ErrInternalServer)
}

func BadGateway(c *gin.Context) {
	Error(c, http.StatusBadGateway, "upstream service unavailable")
}
