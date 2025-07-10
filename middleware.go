package apikit

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func AuthMiddleware(role string, next http.Handler) http.Handler {
	// tbd: implement rbac
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func CorsMiddleware(next http.Handler) http.Handler {
	// tbd: allow based on domain
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func JsonMiddleware[P Payload, R any](db *pgxpool.Pool, process func(*http.Request, *pgxpool.Pool, P) (R, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" && r.Method != "PUT" {
			var p P
			result, err := process(r, db, p)
			encode(r, w, result, nil, err)
			return
		}

		payload, problems, err := decode[P](r)
		if len(problems) > 0 || err != nil {
			encode[*R](r, w, nil, problems, err)
			return
		}

		result, err := process(r, db, payload)
		encode(r, w, result, nil, err)
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
			Date:       start.Format("2006-01-02"),
			Time:       start.Format("15:04:05"),
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
