package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

func (h *Handler) handleFollow(w http.ResponseWriter, r *http.Request) {
	if h.follows == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	targetUserID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())

	switch r.Method {
	case http.MethodPut:
		follow, err := h.follows.Follow(r.Context(), current.ID, targetUserID)
		if h.handleFollowServiceError(w, err) {
			return
		}
		h.refreshRealtimeRelationship(current.ID, targetUserID)
		writeJSON(w, http.StatusOK, followStatusResponse{Status: relationshipStatusResponse(follow.Status)})
	case http.MethodGet:
		relationship, err := h.follows.Relationship(r.Context(), current.ID, targetUserID)
		if h.handleFollowServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, relationshipResponse{
			Status:    relationship.Status,
			FollowsMe: relationship.FollowsMe,
		})
	case http.MethodDelete:
		if h.handleFollowServiceError(w, h.follows.Unfollow(r.Context(), current.ID, targetUserID)) {
			return
		}
		h.refreshRealtimeRelationship(current.ID, targetUserID)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPut+", "+http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleFollowers(w http.ResponseWriter, r *http.Request) {
	h.handleFollowUserList(w, r, true)
}

func (h *Handler) handleFollowing(w http.ResponseWriter, r *http.Request) {
	h.handleFollowUserList(w, r, false)
}

func (h *Handler) handleFollowUserList(w http.ResponseWriter, r *http.Request, followers bool) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.follows == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	userID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())

	var (
		users []*domain.RelatedUser
		err   error
	)
	if followers {
		users, err = h.follows.ListFollowers(r.Context(), current.ID, userID)
	} else {
		users, err = h.follows.ListFollowing(r.Context(), current.ID, userID)
	}
	if h.handleFollowServiceError(w, err) {
		return
	}
	items := make([]relatedUserSummaryResponse, 0, len(users))
	for _, user := range users {
		items = append(items, newRelatedUserSummaryResponse(user))
	}
	writeJSON(w, http.StatusOK, userListResponse{Users: items})
}

func (h *Handler) handleFollowRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.follows == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	requests, err := h.follows.ListPendingRequests(r.Context(), current.ID)
	if h.handleFollowServiceError(w, err) {
		return
	}
	items := make([]followRequestResponse, 0, len(requests))
	for _, request := range requests {
		items = append(items, followRequestResponse{
			ID:        request.Follow.ID,
			User:      newUserSummaryResponse(request.User),
			CreatedAt: request.Follow.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, followRequestListResponse{Requests: items})
}

func (h *Handler) handleFollowRequestAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.follows == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	requestID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	follow, err := h.follows.AcceptRequest(r.Context(), current.ID, requestID)
	if h.handleFollowServiceError(w, err) {
		return
	}
	if h.hub != nil {
		h.hub.RelationshipChanged(current.ID, follow.FollowerUserID, true)
	}
	writeJSON(w, http.StatusOK, followStatusResponse{Status: relationshipStatusResponse(follow.Status)})
}

func (h *Handler) refreshRealtimeRelationship(firstUserID, secondUserID int64) {
	if h.hub == nil || h.follows == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	relationship, err := h.follows.Relationship(ctx, firstUserID, secondUserID)
	if err != nil {
		h.logger.Printf("realtime relationship refresh: %v", err)
		return
	}
	eligible := relationship.Status == service.RelationshipAccepted || relationship.FollowsMe
	h.hub.RelationshipChanged(firstUserID, secondUserID, eligible)
}

func (h *Handler) handleFollowRequestReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.follows == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	requestID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	if h.handleFollowServiceError(w, h.follows.RejectRequest(r.Context(), current.ID, requestID)) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func positivePathID(r *http.Request) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return value, err == nil && value > 0
}

func (h *Handler) handleFollowServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	default:
		h.logger.Printf("follow request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

func relationshipStatusResponse(status domain.FollowStatus) service.RelationshipStatus {
	if status == domain.FollowAccepted {
		return service.RelationshipAccepted
	}
	return service.RelationshipPending
}

type followStatusResponse struct {
	Status service.RelationshipStatus `json:"status"`
}

type relationshipResponse struct {
	Status    service.RelationshipStatus `json:"status"`
	FollowsMe bool                       `json:"follows_me"`
}

type userSummaryResponse struct {
	ID        int64   `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Nickname  *string `json:"nickname"`
	AvatarURL string  `json:"avatar_url"`
	IsPrivate bool    `json:"is_private"`
}

type relatedUserSummaryResponse struct {
	userSummaryResponse
	Relationship relationshipResponse `json:"relationship"`
}

func newUserSummaryResponse(user *domain.User) userSummaryResponse {
	if user == nil {
		return userSummaryResponse{}
	}
	return userSummaryResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Nickname:  user.Nickname,
		AvatarURL: domain.UserAvatarURL(user),
		IsPrivate: user.IsPrivate,
	}
}

func newRelatedUserSummaryResponse(user *domain.RelatedUser) relatedUserSummaryResponse {
	if user == nil {
		return relatedUserSummaryResponse{Relationship: relationshipResponse{Status: service.RelationshipNone}}
	}
	status := service.RelationshipNone
	if user.Status != nil {
		status = relationshipStatusResponse(*user.Status)
	}
	return relatedUserSummaryResponse{
		userSummaryResponse: newUserSummaryResponse(user.User),
		Relationship: relationshipResponse{
			Status:    status,
			FollowsMe: user.FollowsMe,
		},
	}
}

type userListResponse struct {
	Users []relatedUserSummaryResponse `json:"users"`
}

type followRequestResponse struct {
	ID        int64               `json:"id"`
	User      userSummaryResponse `json:"user"`
	CreatedAt time.Time           `json:"created_at"`
}

type followRequestListResponse struct {
	Requests []followRequestResponse `json:"requests"`
}
