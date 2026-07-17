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
	profile      *service.ProfileService
	follows      *service.FollowService
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
	profile *service.ProfileService,
	follows *service.FollowService,
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
		profile:      profile,
		follows:      follows,
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
	protected := func(handler http.HandlerFunc) http.Handler {
		return h.authMiddleware(h.requireAuth(handler))
	}

	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/auth/register", h.handleRegister)
	mux.HandleFunc("/api/auth/login", h.handleLogin)
	mux.HandleFunc("/api/auth/logout", h.handleLogout)
	mux.Handle("/api/auth/me", protected(h.handleMe))
	mux.Handle("/api/profile", protected(h.handleProfile))
	mux.Handle("/api/profile/avatar", protected(h.handleProfileAvatar))
	mux.Handle("/api/profile/", protected(h.handleNotImplemented))
	mux.Handle("/api/users/{id}/follow", protected(h.handleFollow))
	mux.Handle("/api/users/{id}/followers", protected(h.handleFollowers))
	mux.Handle("/api/users/{id}/following", protected(h.handleFollowing))
	mux.Handle("/api/follow-requests", protected(h.handleFollowRequests))
	mux.Handle("/api/follow-requests/{id}/accept", protected(h.handleFollowRequestAccept))
	mux.Handle("/api/follow-requests/{id}", protected(h.handleFollowRequestReject))
	mux.Handle("/api/follow-requests/", protected(h.handleNotImplemented))
	mux.Handle("/ws", protected(h.handleWS))
	mux.Handle("/api/media", protected(h.handleMediaUpload))
	mux.Handle("/uploads/", protected(h.handleMediaDownload))
	mux.Handle("/static/avatars/", http.FileServer(http.FS(avatarPlaceholderFiles)))

	for _, group := range []string{
		"/api/auth",
		"/api/users",
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

	return h.recoverMiddleware(mux)
}
