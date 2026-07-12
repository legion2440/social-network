package http

import (
	"errors"
	"net/http"
	"time"

	"social-network/backend/internal/config"
	"social-network/backend/internal/service"
)

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(config.SessionCookieName)
		if err == nil && cookie.Value != "" && h.sessions != nil {
			session, sessionErr := h.sessions.Get(r.Context(), cookie.Value)
			switch {
			case sessionErr == nil:
				r = r.WithContext(withCurrentUser(r.Context(), CurrentUser{
					ID:           session.UserID,
					SessionToken: session.Token,
				}))
			case errors.Is(sessionErr, service.ErrUnauthorized):
				ClearSessionCookie(w, h.cookieSecure)
			default:
				h.logger.Printf("session lookup: %v", sessionErr)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := CurrentUserFromContext(r.Context()); !ok {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	}
}

func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.SessionCookieName,
		Value:    token,
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     config.SessionCookieName,
		Value:    "",
		Path:     "/",
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *Handler) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				h.logger.Printf("panic serving %s %s: %v", r.Method, r.URL.Path, recovered)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
