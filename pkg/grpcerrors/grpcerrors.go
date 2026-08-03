// Package grpcerrors translates the framework-agnostic apperrors.AppError
// into gRPC status errors, mirroring pkg/response's HTTP-status mapping so
// every module's gRPC presentation layer maps domain error codes the same way.
package grpcerrors

import (
	apperrors "github.com/dbklik/dbklik-kompetitor-service/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ToStatus translates an application error (or any error) into a gRPC status
// error, mapping domain error codes to gRPC codes at the presentation boundary.
func ToStatus(err error) error {
	if err == nil {
		return nil
	}

	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		return status.Error(codes.Internal, err.Error())
	}

	code := codes.Internal
	switch appErr.Code {
	case apperrors.CodeNotFound:
		code = codes.NotFound
	case apperrors.CodeInvalidInput:
		code = codes.InvalidArgument
	case apperrors.CodeConflict:
		code = codes.AlreadyExists
	case apperrors.CodeUnauthorized:
		code = codes.Unauthenticated
	case apperrors.CodeForbidden:
		code = codes.PermissionDenied
	case apperrors.CodeUnavailable:
		code = codes.Unavailable
	}

	return status.Error(code, appErr.Message)
}
