package cors

import "net/http"

func Middleware(origin string) func(http.Handler) http.Handler {
	if origin == "" {
		origin = "*"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setOrigin := func() {
				if origin == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					reqOrigin := r.Header.Get("Origin")
					if reqOrigin == origin {
						w.Header().Set("Access-Control-Allow-Origin", reqOrigin)
						w.Header().Set("Vary", "Origin")
					}
				}
			}

			if r.Method == http.MethodOptions {
				reqMethod := r.Header.Get("Access-Control-Request-Method")
				reqHeaders := r.Header.Get("Access-Control-Request-Headers")
				if reqMethod != "" || reqHeaders != "" {
					setOrigin()
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					w.Header().Set("Access-Control-Max-Age", "86400")
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}

			setOrigin()
			next.ServeHTTP(w, r)
		})
	}
}
