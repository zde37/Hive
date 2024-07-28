package handler

import (
	"net/http"
)

// ErrorStatus represents an error with an associated HTTP status code.
type ErrorStatus struct {
	error
	statusCode int

	// error levels :
	// 0 -> http status codes >= 400
	// 1 -> http status codes >= 500
	level int
}

// ErrorResponse is the structure for JSON error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Unwrap returns the underlying error.
func (e ErrorStatus) Unwrap() error { return e.error }

func ErrorInfo(err error) (ErrorResponse, int, int) {
	if errStatus, ok := err.(ErrorStatus); ok {
		return ErrorResponse{Error: errStatus.error.Error()}, errStatus.statusCode, errStatus.level
	}
	return ErrorResponse{Error: "unknown error occurred"}, http.StatusInternalServerError, 1
}

// NewErrorStatus creates a new ErrorStatus with the given error, status code and error level.
func NewErrorStatus(err error, code, errLevel int) error {
	return ErrorStatus{
		error:      err,
		statusCode: code,
		level:      errLevel,
	}
}
