package apikit

import "net/http"

func authMiddleware(role string, next http.Handler) http.Handler {
	// tbd: implement rbac
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
