package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// corsMiddleware is a middleware function that sets the appropriate CORS headers for an HTTP request. 
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// errorMiddleware is a middleware function that wraps a handler function and handles any errors that occur.
// The wrapped handler function should return an error, which this middleware will handle.
func errorMiddleware(f func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		r = r.WithContext(ctx)

		startTime := time.Now()
		err := f(w, r)
		duration := time.Since(startTime)

		if err != nil {
			errRes, statusCode, errLevel := ErrorInfo(err)

			if errLevel == 1 { // log only critical errors
				log.Printf("Log => status: failed, error: %s, status_code: %d, method: %s, path: %s, duration: %s", errRes, statusCode, r.Method, r.URL.Path, duration)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(errRes)
			return
		}
		// log.Printf("Log => status: success, method: %s, path: %s, duration: %s", r.Method, r.URL.Path, duration)
	}
}
