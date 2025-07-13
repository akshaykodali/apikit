package apikit

import (
	"errors"
)

var (
	ErrorBadRequest           = errors.New("bad request")
	ErrorUnauthorized         = errors.New("unauthorized")
	ErrorForbidden            = errors.New("forbidden")
	ErrorNotFound             = errors.New("not found")
	ErrorMethodNotImplemented = errors.New("method not implemented")
	ErrorConflict             = errors.New("conflict")
	ErrorUnprocessableEntity  = errors.New("unprocessable entity")
	ErrorInternalServerError  = errors.New("internal server error")
)
