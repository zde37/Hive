package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

func TestErrorMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		handlerFunc    func(http.ResponseWriter, *http.Request) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success case",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name: "Error case - Critical error",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				return errors.New("critical error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"an error occurred"}`,
		},
		{
			name: "Error case - Non-critical error",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				return ErrorStatus{error: fmt.Errorf("non-critical error"), statusCode: http.StatusBadRequest, level: 0}
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"non-critical error"}`,
		},
		// {
		// 	name: "Context timeout",
		// 	handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
		// 		time.Sleep(31 * time.Second)
		// 		return nil
		// 	},
		// 	expectedStatus: http.StatusGatewayTimeout,
		// 	expectedBody:   `{"error":"Request timeout"}`,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := errorMiddleware(tt.handlerFunc)

			handler.ServeHTTP(rr, req)
			require.Equal(t, rr.Code, tt.expectedStatus)
			require.Equal(t, strings.TrimSpace(rr.Body.String()), tt.expectedBody)

			if tt.expectedStatus != http.StatusOK {
				contentType := rr.Header().Get("Content-Type")
				require.Equal(t, contentType, "application/json")
			}
		})
	}
}

func TestErrorMiddlewareWithConcurrency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		handlerFunc    func(http.ResponseWriter, *http.Request) error
		expectedStatus int
		expectedBody   string
		concurrency    int
	}{
		{
			name: "Multiple concurrent requests - Success",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
			concurrency:    10,
		},
		{
			name: "Multiple concurrent requests - Mixed results",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				time.Sleep(100 * time.Millisecond)
				if rand.Intn(2) == 0 {
					return errors.New("random error")
				}
				return nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"an error occurred"}`,
			concurrency:    10,
		},
		{
			name: "Long-running request - Just under timeout",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) error {
				time.Sleep(29 * time.Second)
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
			concurrency:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			results := make(chan *httptest.ResponseRecorder, tt.concurrency)

			for i := 0; i < tt.concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					req, err := http.NewRequest("GET", "/test", nil)
					require.NoError(t, err)

					rr := httptest.NewRecorder()
					handler := errorMiddleware(tt.handlerFunc)

					handler.ServeHTTP(rr, req)
					results <- rr
				}()
			}

			wg.Wait()
			close(results)

			for rr := range results {
				if tt.expectedStatus == http.StatusOK {
					require.Equal(t, tt.expectedStatus, rr.Code)
					require.Equal(t, tt.expectedBody, strings.TrimSpace(rr.Body.String()))
				} else {
					require.True(t, rr.Code == http.StatusOK || rr.Code == tt.expectedStatus)
					if rr.Code != http.StatusOK {
						require.Equal(t, tt.expectedBody, strings.TrimSpace(rr.Body.String()))
						require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
					}
				}
			}
		})
	}
}

func TestCorsMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedHeaders map[string]string
	}{
		{
			name:           "GET request",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS, PATCH",
				"Access-Control-Allow-Headers": "*",
			},
		},
		{
			name:           "OPTIONS request",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS, PATCH",
				"Access-Control-Allow-Headers": "*",
			},
		},
		{
			name:           "POST request",
			method:         "POST",
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS, PATCH",
				"Access-Control-Allow-Headers": "*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/test", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rr, req)

			require.Equal(t, tt.expectedStatus, rr.Code)

			for key, value := range tt.expectedHeaders {
				require.Equal(t, value, rr.Header().Get(key))
			}

			if tt.method == "OPTIONS" {
				require.Empty(t, rr.Body.String())
			}
		})
	}
}

func TestCorsMiddlewareWithCustomHandler(t *testing.T) {
	t.Parallel()

	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Custom response"))
	})

	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := corsMiddleware(customHandler)

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "Custom response", rr.Body.String())
	require.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "GET, POST, PUT, DELETE, OPTIONS, PATCH", rr.Header().Get("Access-Control-Allow-Methods"))
	require.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Headers"))
}
