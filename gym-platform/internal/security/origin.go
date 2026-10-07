package security

import (
	"net/http"
	"net/url"
)

// OriginCheck rejects state-changing requests whose Origin header points at a
// different site. extraHosts lists additional allowed hosts (used in dev, where
// templ's proxy serves the page on a different port than the app).
func OriginCheck(extraHosts []string, next http.Handler) http.Handler {
	allowed := map[string]bool{}
	for _, h := range extraHosts {
		allowed[h] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		u, err := url.Parse(origin)
		if err != nil || (u.Host != r.Host && !allowed[u.Host]) {
			http.Error(w, "cross-origin request blocked", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
