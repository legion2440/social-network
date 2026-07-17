package http

import (
	"database/sql"
	"io"
	"log"
	"net/http"

	"social-network/backend/internal/config"
	"social-network/backend/internal/service"
)

type Handler struct {
	db           *sql.DB
	sessions     *service.SessionService
	media        *service.MediaService
	auth         *service.AuthService
	sessionToken SessionTokenExtractor
	cookieSecure bool
	frontend     http.Handler
	logger       *log.Logger
}

func NewHandler(
	db *sql.DB,
	sessions *service.SessionService,
	media *service.MediaService,
	auth *service.AuthService,
	sessionToken SessionTokenExtractor,
	cookieSecure bool,
	frontendDir string,
	logger *log.Logger,
) *Handler {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	if sessionToken == nil {
		sessionToken = NewCookieSessionTokenExtractor(config.SessionCookieName)
	}
	return &Handler{
		db:           db,
		sessions:     sessions,
		media:        media,
		auth:         auth,
		sessionToken: sessionToken,
		cookieSecure: cookieSecure,
		frontend:     newFrontendHandler(frontendDir),
		logger:       logger,
	}
}

func newFrontendHandler(frontendDir string) http.Handler {
	if frontendDir == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, http.StatusNotFound, "not found")
		})
	}
	return http.FileServer(http.Dir(frontendDir))
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/auth/register", h.handleRegister)
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/logout", h.handleLogout)
	mux.HandleFunc("/api/auth/me", h.requireAuth(h.handleMe))
	mux.HandleFunc("/ws", h.requireAuth(h.handleWS))
	mux.HandleFunc("/api/media", h.requireAuth(h.handleMediaUpload))
	mux.HandleFunc("/uploads/", h.requireAuth(h.handleMediaDownload))
	mux.Handle("/static/avatars/", http.FileServer(http.FS(avatarPlaceholderFiles)))

	for _, group := range []string{
		"/api/auth",
		"/api/users",
		"/api/profile",
		"/api/follow",
		"/api/posts",
		"/api/groups",
		"/api/events",
		"/api/chats",
		"/api/notifications",
	} {
		mux.HandleFunc(group, h.handleNotImplemented)
		mux.HandleFunc(group+"/", h.handleNotImplemented)
	}

	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.Handle("/", h.frontend)

	return h.recoverMiddleware(h.authMiddleware(mux))
}
