package http

import (
	"database/sql"
	"io"
	"log"
	"net/http"

	"social-network/backend/internal/service"
)

type Handler struct {
	db           *sql.DB
	sessions     *service.SessionService
	media        *service.MediaService
	cookieSecure bool
	logger       *log.Logger
}

func NewHandler(db *sql.DB, sessions *service.SessionService, media *service.MediaService, cookieSecure bool, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &Handler{
		db:           db,
		sessions:     sessions,
		media:        media,
		cookieSecure: cookieSecure,
		logger:       logger,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/ws", h.requireAuth(h.handleWS))
	mux.HandleFunc("/api/media", h.requireAuth(h.handleMediaUpload))
	mux.HandleFunc("/uploads/", h.requireAuth(h.handleMediaDownload))

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
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})

	return h.recoverMiddleware(h.authMiddleware(mux))
}
