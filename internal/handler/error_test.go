package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorInfo(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedResp   ErrorResponse
		expectedStatus int
		expectedLevel  int
	}{
		{
			name:           "Generic error",
			err:            errors.New("generic error"),
			expectedResp:   ErrorResponse{Error: "an error occurred"},
			expectedStatus: http.StatusInternalServerError,
			expectedLevel:  1,
		},
		{
			name:           "ErrorStatus error",
			err:            ErrorStatus{error: errors.New("custom error"), statusCode: http.StatusBadRequest, level: 0},
			expectedResp:   ErrorResponse{Error: "custom error"},
			expectedStatus: http.StatusBadRequest,
			expectedLevel:  0,
		},
		{
			name:           "Nil error",
			err:            nil,
			expectedResp:   ErrorResponse{Error: "an error occurred"},
			expectedStatus: http.StatusInternalServerError,
			expectedLevel:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, status, level := ErrorInfo(tt.err)
			require.Equal(t, tt.expectedResp, resp)
			require.Equal(t, tt.expectedStatus, status)
			require.Equal(t, tt.expectedLevel, level)
		})
	}
}

func TestNewErrorStatus(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		code           int
		errLevel       int
		expectedStatus ErrorStatus
	}{
		{
			name:     "New error with custom status code and level",
			err:      errors.New("custom error"),
			code:     http.StatusInternalServerError,
			errLevel: 1,
			expectedStatus: ErrorStatus{
				error:      errors.New("custom error"),
				statusCode: http.StatusInternalServerError,
				level:      1,
			},
		},
		{
			name:     "New error with OK status code and zero level",
			err:      errors.New("ok error"),
			code:     http.StatusBadRequest,
			errLevel: 0,
			expectedStatus: ErrorStatus{
				error:      errors.New("ok error"),
				statusCode: http.StatusBadRequest,
				level:      0,
			},
		},
		{
			name:     "New error with nil error",
			err:      nil,
			code:     http.StatusInternalServerError,
			errLevel: 1,
			expectedStatus: ErrorStatus{
				error:      nil,
				statusCode: http.StatusInternalServerError,
				level:      1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewErrorStatus(tt.err, tt.code, tt.errLevel)
			errorStatus, ok := result.(ErrorStatus)
			require.True(t, ok, "Result should be of type ErrorStatus")

			if tt.err != nil {
				require.Equal(t, tt.err.Error(), errorStatus.error.Error())
			} else {
				require.Nil(t, errorStatus.error)
			}
			require.Equal(t, tt.expectedStatus.statusCode, errorStatus.statusCode)
			require.Equal(t, tt.expectedStatus.level, errorStatus.level)
		})
	}
}
