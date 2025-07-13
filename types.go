package apikit

import (
	"bytes"
	"net/http"
	"time"
)

type CorsConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type ctxKey string

type Flusher interface {
	Interval() time.Duration
	Flush(*[]*Log)
}

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

type Payload interface {
	Validate() map[string]string
}

type Response[T any] struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    T                 `json:"data,omitempty"`
	Errors  map[string]string `json:"errors,omitempty,omitzero"`
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

type Service[P Payload, Result any] interface {
	Process(*http.Request) (Result, error)
	ProcessWithPayload(*http.Request, P) (Result, error)
}
