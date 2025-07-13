package apikit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

const TraceIdKey ctxKey = "traceId"

func decode[P Payload](r *http.Request) (P, map[string]string, error) {
	var p P
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return p, nil, ErrorBadRequest
	}

	if problems := p.Validate(); len(problems) > 0 {
		return p, problems, ErrorUnprocessableEntity
	}

	return p, nil, nil
}

func encode[T any](r *http.Request, w http.ResponseWriter, data T, problems map[string]string, err error) {
	w.Header().Set("Content-Type", "application/json")

	var status int
	switch err {
	case nil:
		switch r.Method {
		case "POST":
			status = http.StatusCreated
		case "DELETE":
			status = http.StatusNoContent
		default:
			status = http.StatusOK
		}
	case ErrorBadRequest:
		status = http.StatusBadRequest
	case ErrorUnauthorized:
		status = http.StatusUnauthorized
	case ErrorForbidden:
		status = http.StatusForbidden
	case ErrorConflict:
		status = http.StatusConflict
	case ErrorNotFound:
		status = http.StatusNotFound
	case ErrorUnprocessableEntity:
		status = http.StatusUnprocessableEntity
	default:
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)

	if status == http.StatusNoContent {
		return
	}

	res := Response[T]{
		Success: err == nil,
		Message: strings.ToLower(http.StatusText(status)),
		Data:    data,
		Errors:  problems,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error("error encoding response", "err", err, "traceId", GetTraceId(r.Context()))
	}
}

func GetTraceId(ctx context.Context) string {
	if traceId, ok := ctx.Value(TraceIdKey).(string); ok {
		return traceId
	}

	return ""
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, o := range allowedOrigins {
		if o == "*" {
			return true
		}

		if o == origin {
			return true
		}
	}

	return false
}
