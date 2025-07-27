package apikit

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

func decode[P Payload](r *http.Request) (P, map[string]string, error) {
	var payload P

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if e, ok := err.(*json.UnmarshalTypeError); ok {
			problems := make(map[string]string)
			problems[e.Field] = fmt.Sprintf("must be %s type", e.Type.String())

			return payload, problems, ErrBadRequest
		}

		return payload, nil, ErrBadRequest
	}

	if problems := payload.Validate(); len(problems) > 0 {
		return payload, problems, ErrUnprocessableEntity
	}

	return payload, nil, nil
}

func encode[T any](r *http.Request, w http.ResponseWriter, data T, meta *Meta, err error) {
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
	case ErrBadRequest:
		status = http.StatusBadRequest
	case ErrUnauthorized:
		status = http.StatusUnauthorized
	case ErrForbidden:
		status = http.StatusForbidden
	case ErrConflict:
		status = http.StatusConflict
	case ErrNotFound:
		status = http.StatusNotFound
	case ErrMethodNotAllowed:
		status = http.StatusMethodNotAllowed
	case ErrUnprocessableEntity:
		status = http.StatusUnprocessableEntity
	default:
		slog.Error(
			"unhandled error",
			"err", err,
			"traceId", GetTraceId(r.Context()),
		)
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
		Meta:    meta,
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		slog.Error(
			"error encoding response",
			"err", err,
			"traceId", GetTraceId(r.Context()),
		)
	}
}

func getRemoteAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "invalid remote address"
	}

	return host
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, o := range allowedOrigins {
		if o == "*" || o == origin {
			return true
		}
	}

	return false
}
