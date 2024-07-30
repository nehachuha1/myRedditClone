package middleware

import (
	"myredditclone/pkg/session"
	"net/http"
)

func Auth(sm *session.SessionsManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := sm.Check(w, r)
			if err == nil || sess != nil {
				ctx := session.ContextWithSession(r.Context(), sess)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
