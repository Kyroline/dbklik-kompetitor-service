// Package utils holds small standalone utility functions
// (string/slice/id helpers) shared across modules.
package utils

import "github.com/google/uuid"

// NewID generates a new UUID v4 string, used as the default ID strategy
// for entities across modules.
func NewID() string {
	return uuid.NewString()
}

// Paginate computes offset/limit from a 1-based page number and page size.
func Paginate(page, pageSize int) (offset, limit int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return (page - 1) * pageSize, pageSize
}
