package apikit

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Log struct {
	TraceId    string
	RemoteAddr string
	Method     string
	Path       string
	Params     string
	Payload    string
	Status     int
	Response   string
	Duration   int64
	Timestamp  time.Time
}

type Flusher interface {
	Interval() time.Duration
	Flush([]Log)
}

var buffer = [1000]Log{}
var size = 0
var logWg sync.WaitGroup
var logMutex sync.Mutex
var logFlusher Flusher

func CreateLogger(ctx context.Context, wg *sync.WaitGroup, flusher Flusher) {
	wg.Add(1)
	logFlusher = flusher

	go func() {
		defer wg.Done()

		var interval time.Duration
		if flusher.Interval() == 0 {
			interval = 5 * time.Second
		} else {
			interval = flusher.Interval()
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if size > 0 {
					if logMutex.TryLock() {
						flusher.Flush(buffer[0:size])
						size = 0
						logMutex.Unlock()
					}
				}
			case <-ctx.Done():
				logWg.Wait()
				logMutex.Lock()
				flusher.Flush(buffer[0:size])
				size = 0
				logMutex.Unlock()
				return
			}
		}
	}()
}

type ctxKey string

const TraceIdKey ctxKey = "traceId"

func GetTraceId(ctx context.Context) string {
	if traceId, ok := ctx.Value(TraceIdKey).(string); ok {
		return traceId
	}

	return ""
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func LoggingMiddleware(blacklist []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logWg.Add(1)

		traceId := uuid.New().String()
		w.Header().Set("X-Trace-Id", traceId)
		ctx := context.WithValue(r.Context(), TraceIdKey, traceId)

		var requestBody bytes.Buffer
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				slog.Error("failed to read request body", "err", err)
				encode[*string](r, w, nil, nil, ErrInternalServerError)
				return
			} else {
				requestBody.Write(body)
				r.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}

		rec := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next.ServeHTTP(rec, r.WithContext(ctx))

		duration := time.Since(start)
		go func() {
			defer logWg.Done()

			log := Log{
				TraceId:    traceId,
				RemoteAddr: getRemoteAddr(r),
				Method:     r.Method,
				Path:       r.URL.Path,
				Params:     r.URL.RawQuery,
				Payload:    requestBody.String(),
				Status:     rec.status,
				Response:   rec.body.String(),
				Duration:   duration.Milliseconds(),
				Timestamp:  start,
			}
			if slices.Contains(blacklist, r.Pattern) {
				log.Payload = "redacted"
				log.Response = "redacted"
			}

			for {
				logMutex.Lock()
				if size == len(buffer) {
					logFlusher.Flush(buffer[0:size])
					buffer[0] = log
					size = 1
					logMutex.Unlock()
				} else {
					buffer[size] = log
					size++
					logMutex.Unlock()
					break
				}
			}
		}()
	})
}
