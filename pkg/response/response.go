// Package response provides transport-agnostic JSON response builders
// shared by every module's presentation layer.
package response

import (
	"net/http"

	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

func OK(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data})
}

func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Message: message, Data: data})
}

func Paginated(c *gin.Context, message string, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data, Meta: meta})
}

// Error translates an application error (or any error) into an HTTP response,
// mapping domain error codes to HTTP status codes at the presentation boundary.
func Error(c *gin.Context, err error) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		c.JSON(http.StatusInternalServerError, Envelope{Success: false, Error: err.Error()})
		return
	}

	status := http.StatusInternalServerError
	switch appErr.Code {
	case apperrors.CodeNotFound:
		status = http.StatusNotFound
	case apperrors.CodeInvalidInput:
		status = http.StatusBadRequest
	case apperrors.CodeConflict:
		status = http.StatusConflict
	case apperrors.CodeUnauthorized:
		status = http.StatusUnauthorized
	case apperrors.CodeForbidden:
		status = http.StatusForbidden
	case apperrors.CodeUnavailable:
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, Envelope{Success: false, Error: appErr.Message})
}
