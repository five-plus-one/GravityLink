package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

func Error(c *gin.Context, status int, code int, message string) {
	c.JSON(status, Body{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
