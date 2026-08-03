// Package responses writes the kompetitor module's HTTP bodies.
//
// Unlike the framework's pkg/response envelope, these are the RAW shapes the
// Laravel KompetitorController used to return ({data: ...}, the DataTables
// {draw, recordsTotal, ...}, {message: ...}). The Laravel side proxies this
// service and passes the body through verbatim, so the existing Blade/JS
// front end keeps working untouched — wrapping it in an envelope would break
// DataTables, which reads `draw`/`recordsTotal` at the top level.
package responses

import (
	"net/http"

	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
	"github.com/gin-gonic/gin"
)

// OK writes a 200 with the payload as-is.
func OK(c *gin.Context, payload interface{}) {
	c.JSON(http.StatusOK, payload)
}

// Created writes a 201 with a {message: ...} body, matching store().
func Created(c *gin.Context, message string) {
	c.JSON(http.StatusCreated, gin.H{"message": message})
}

// Message writes a 200 with a {message: ...} body.
func Message(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{"message": message})
}

// Error maps a domain error onto the status codes the Laravel controller
// used, with the same {message: ...} body the front end reads.
//
// Note CodeInvalidInput maps to 422, not the framework default of 400:
// these errors stand in for Laravel's validation responses, and the panels'
// JS branches on 422.
func Error(c *gin.Context, err error) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	status := http.StatusInternalServerError
	switch appErr.Code {
	case apperrors.CodeInvalidInput:
		status = http.StatusUnprocessableEntity
	case apperrors.CodeNotFound:
		status = http.StatusNotFound
	case apperrors.CodeConflict:
		status = http.StatusConflict
	case apperrors.CodeUnauthorized:
		status = http.StatusUnauthorized
	case apperrors.CodeForbidden:
		status = http.StatusForbidden
	case apperrors.CodeUnavailable:
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{"message": appErr.Message})
}
