package apikit

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func AuthMiddleware(role string, next http.Handler) http.Handler {
	// tbd: implement rbac
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func CorsMiddleware(config CorsConfig, next http.Handler) http.Handler {
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}

	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{"Authorization", "Content-Type"}
	}

	if config.MaxAge == 0 {
		config.MaxAge = 300
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if isOriginAllowed(origin, config.AllowedOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))

		if config.AllowCredentials {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func JsonMiddleware[P Payload, Result any](service Service[P, Result]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var result Result
		var e error

		switch r.Method {
		case "GET", "DELETE":
			result, e = service.Process(r)
		case "POST", "PUT":
			payload, problems, err := decode[P](r)
			if len(problems) > 0 || err != nil {
				encode(r, w, result, problems, err)
				return
			}

			result, e = service.ProcessWithPayload(r, payload)
		default:
			e = ErrorMethodNotImplemented
		}

		encode(r, w, result, nil, e)
	})
}

func LoggingMiddleware(next http.Handler, logCh chan Log) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		var requestBody bytes.Buffer
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				slog.Error("failed to read request body", "err", err)
			} else {
				requestBody.Write(body)
				r.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}

		rec := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next.ServeHTTP(rec, r)

		traceId := rec.Header().Get("X-Trace-Id")
		duration := time.Since(start)
		logCh <- Log{
			TraceId:    traceId,
			RemoteAddr: r.RemoteAddr,
			Method:     r.Method,
			Path:       r.URL.Path,
			Params:     r.URL.Query().Encode(),
			Payload:    requestBody.String(),
			Status:     rec.status,
			Response:   rec.body.String(),
			Duration:   duration.Milliseconds(),
			Timestamp:  start,
		}
	})
}

func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceId := uuid.New().String()

		w.Header().Set("X-Trace-Id", traceId)

		ctx := context.WithValue(r.Context(), TraceIdKey, traceId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
