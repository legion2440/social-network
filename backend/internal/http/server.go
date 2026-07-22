package http

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"

	"social-network/backend/internal/config"
	realtimews "social-network/backend/internal/realtime/ws"
	"social-network/backend/internal/service"
)

type Handler struct {
	db           *sql.DB
	sessions     *service.SessionService
	media        *service.MediaService
	auth         *service.AuthService
	profile      *service.ProfileService
	follows      *service.FollowService
	users        *service.UserService
	avatars      *service.AvatarDeliveryService
	posts        *service.PostService
	postMedia    *service.PostMediaDeliveryService
	comments     *service.CommentService
	groups       *service.GroupService
	chats        *service.ChatService
	hub          *realtimews.Hub
	admission    atomic.Bool
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
	users *service.UserService,
	avatars *service.AvatarDeliveryService,
	posts *service.PostService,
	postMedia *service.PostMediaDeliveryService,
	comments *service.CommentService,
	groups *service.GroupService,
	chats *service.ChatService,
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
	handler := &Handler{
		db:           db,
		sessions:     sessions,
		media:        media,
		auth:         auth,
		profile:      profile,
		follows:      follows,
		users:        users,
		avatars:      avatars,
		posts:        posts,
		postMedia:    postMedia,
		comments:     comments,
		groups:       groups,
		chats:        chats,
		sessionToken: sessionToken,
		cookieSecure: cookieSecure,
		frontend:     newFrontendHandler(frontendDir),
		logger:       logger,
	}
	handler.admission.Store(true)
	return handler
}

func (h *Handler) SetRealtimeHub(hub *realtimews.Hub) {
	if h != nil {
		h.hub = hub
	}
}

func (h *Handler) CloseAdmission() {
	if h != nil {
		h.admission.Store(false)
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
	mux.Handle("/api/users", protected(h.handleUsers))
	mux.Handle("/api/users/{id}", protected(h.handleUserProfile))
	mux.Handle("/api/users/{id}/follow", protected(h.handleFollow))
	mux.Handle("/api/users/{id}/followers", protected(h.handleFollowers))
	mux.Handle("/api/users/{id}/following", protected(h.handleFollowing))
	mux.Handle("/api/users/{id}/avatar", protected(h.handleUserAvatar))
	mux.Handle("/api/users/{id}/posts", protected(h.handleUserPosts))
	mux.Handle("/api/follow-requests", protected(h.handleFollowRequests))
	mux.Handle("/api/follow-requests/{id}/accept", protected(h.handleFollowRequestAccept))
	mux.Handle("/api/follow-requests/{id}", protected(h.handleFollowRequestReject))
	mux.Handle("/api/follow-requests/", protected(h.handleNotImplemented))
	mux.Handle("/ws", protected(h.handleWS))
	mux.Handle("/api/media", protected(h.handleMediaUpload))
	mux.Handle("/api/posts", protected(h.handlePosts))
	mux.Handle("/api/posts/feed", protected(h.handlePostFeed))
	mux.Handle("/api/posts/{id}/media", protected(h.handlePostMedia))
	mux.Handle("/api/posts/{id}/comments", protected(h.handlePostComments))
	mux.Handle("/api/groups", protected(h.handleGroups))
	mux.Handle("/api/groups/{id}", protected(h.handleGroupDetail))
	mux.Handle("/api/groups/{id}/members", protected(h.handleGroupMembers))
	mux.Handle("/api/groups/{id}/join-request", protected(h.handleGroupJoinRequest))
	mux.Handle("/api/groups/{id}/join-requests", protected(h.handleGroupJoinRequests))
	mux.Handle("/api/groups/{id}/join-requests/{user_id}/accept", protected(h.handleGroupJoinRequestAccept))
	mux.Handle("/api/groups/{id}/join-requests/{user_id}", protected(h.handleGroupJoinRequestReject))
	mux.Handle("/api/groups/{id}/invitations", protected(h.handleGroupInvitations))
	mux.Handle("/api/groups/{id}/invitation/accept", protected(h.handleGroupInvitationAccept))
	mux.Handle("/api/groups/{id}/invitation", protected(h.handleGroupInvitationDecline))
	mux.Handle("/api/groups/{id}/membership", protected(h.handleGroupMembership))
	mux.Handle("/api/group-invitations", protected(h.handleGroupInvitationInbox))
	mux.Handle("/api/chats", protected(h.handleChats))
	mux.Handle("/api/chats/direct/{user_id}/messages", protected(h.handleDirectChatMessages))
	mux.Handle("/api/groups/{id}/chat/messages", protected(h.handleGroupChatMessages))
	mux.HandleFunc("/api/posts/", h.handleNotImplemented)
	mux.Handle("/uploads/", protected(h.handleMediaDownload))
	mux.Handle("/static/avatars/", http.FileServer(http.FS(avatarPlaceholderFiles)))

	for _, group := range []string{
		"/api/auth",
		"/api/follow",
		"/api/events",
		"/api/notifications",
	} {
		mux.HandleFunc(group, h.handleNotImplemented)
		mux.HandleFunc(group+"/", h.handleNotImplemented)
	}

	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.Handle("/", h.frontend)

	return h.recoverMiddleware(h.admissionMiddleware(mux))
}

func (h *Handler) admissionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.admission.Load() && (r.URL.Path == "/ws" || (strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions)) {
			writeError(w, http.StatusServiceUnavailable, "server is shutting down")
			return
		}
		next.ServeHTTP(w, r)
	})
}
