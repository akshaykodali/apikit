package apikit

import "net/http"

type Payload interface {
	Validate() map[string]string
}

type Meta struct {
	CurrentPage  *int              `json:"currentPage,omitempty"`
	TotalPages   *int              `json:"totalPages,omitempty"`
	TotalRecords *int              `json:"totalRecords,omitempty"`
	Errors       map[string]string `json:"errors,omitempty,omitzero"`
}

type Response[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
}

func JsonMiddleware[P Payload](process func(*http.Request, P) (any, *Meta, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, problems, err := decode[P](r)
		if len(problems) > 0 || err != nil {
			encode[*string](r, w, nil, &Meta{Errors: problems}, err)
			return
		}

		result, meta, err := process(r, payload)
		encode(r, w, result, meta, err)
	})
}

func JsonResponseMiddleware(process func(*http.Request) (any, *Meta, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result, meta, err := process(r)
		encode(r, w, result, meta, err)
	})
}
