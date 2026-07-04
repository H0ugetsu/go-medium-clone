package validator

import (
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/h0ugetsu/realworld-api/internal/httputil/httperror"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		return validationErrors(err)
	}
	return nil
}

func NewValidator() *CustomValidator {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "-" {
			return ""
		}
		return name
	})

	return &CustomValidator{validator: v}
}

func validationErrors(err error) *httperror.Error {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return httperror.New(http.StatusUnprocessableEntity, map[string]any{
			"errors": map[string][]string{"body": {"is invalid"}},
		})
	}
	result := map[string][]string{}
	for _, fe := range ve {
		result[fe.Field()] = append(result[fe.Field()], messageForTag(fe.Tag()))
	}
	return httperror.New(http.StatusUnprocessableEntity, map[string]any{"errors": result})
}

func messageForTag(tag string) string {
	switch tag {
	case "required":
		return "can't be blank"
	case "email":
		return "is invalid"
	default:
		return "is invalid"
	}
}
