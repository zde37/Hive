package handler

import (
	"net/http"
)

// ErrorStatus represents an error with an associated HTTP status code and error level.
// The error level indicates the severity of the error:
//
//	0 -> HTTP status codes >= 400
//	1 -> HTTP status codes >= 500
type ErrorStatus struct {
	error
	statusCode int
	level      int
}

// ErrorResponse is the structure for JSON error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// Unwrap returns the underlying error.
func (e ErrorStatus) Unwrap() error { return e.error }

// ErrorInfo returns an ErrorResponse, the HTTP status code, and the error level for the given error.
func ErrorInfo(err error) (ErrorResponse, int, int) {
	if errStatus, ok := err.(ErrorStatus); ok {
		return ErrorResponse{Error: errStatus.error.Error()}, errStatus.statusCode, errStatus.level
	}
	return ErrorResponse{Error: "an error occurred"}, http.StatusInternalServerError, 1
}

// NewErrorStatus creates a new ErrorStatus with the given error, status code and error level.
func NewErrorStatus(err error, code, errLevel int) error {
	return ErrorStatus{
		error:      err,
		statusCode: code,
		level:      errLevel,
	}
}
