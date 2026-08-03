// Package validator wraps go-playground/validator so request structs
// across every module can validate themselves the same way.
package validator

import (
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	instance *validator.Validate
	once     sync.Once
)

func get() *validator.Validate {
	once.Do(func() {
		instance = validator.New()
	})
	return instance
}

// Struct validates a struct's `validate` tags and returns a human-readable
// error message joining every failed field, or nil if valid.
func Struct(s interface{}) error {
	if err := get().Struct(s); err != nil {
		var msgs []string
		for _, fe := range err.(validator.ValidationErrors) {
			msgs = append(msgs, fe.Field()+" failed on '"+fe.Tag()+"'")
		}
		return &Error{Message: strings.Join(msgs, "; ")}
	}
	return nil
}

type Error struct {
	Message string
}

func (e *Error) Error() string { return e.Message }
