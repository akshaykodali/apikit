package apikit

import (
	"bytes"
	"context"
	"net/http"
)

type ctxKey string

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
	Date       string
	Time       string
}

type Payload interface {
	Validate(ctx context.Context) map[string]string
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
