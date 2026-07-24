package http

import (
	"errors"
	"net/http"
	"strconv"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

func (h *Handler) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.users == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	cursor, limit, err := readUserPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.users.Directory(r.Context(), current.ID, cursor, limit)
	if h.handleUserServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newUserDirectoryResponse(page))
}

func (h *Handler) handleUserProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.users == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	targetUserID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	profile, err := h.users.Profile(r.Context(), current.ID, targetUserID)
	if h.handleUserServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newUserProfileResponse(profile))
}

func readUserPageQuery(r *http.Request) (*domain.UserCursor, int, error) {
	values := r.URL.Query()
	for name := range values {
		if name != "cursor" && name != "limit" {
			return nil, 0, service.ErrInvalidInput
		}
	}
	limit := service.DefaultUserPageLimit
	if rawLimit, exists := values["limit"]; exists {
		if len(rawLimit) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		parsed, err := strconv.Atoi(rawLimit[0])
		if err != nil || parsed < 1 || parsed > service.MaxUserPageLimit {
			return nil, 0, service.ErrInvalidInput
		}
		limit = parsed
	}
	var cursor *domain.UserCursor
	if rawCursor, exists := values["cursor"]; exists {
		if len(rawCursor) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		var err error
		cursor, err = service.DecodeUserCursor(rawCursor[0])
		if err != nil {
			return nil, 0, err
		}
	}
	return cursor, limit, nil
}

func (h *Handler) handleUserServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		h.logger.Printf("user request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

type userProfileResponse struct {
	ID             int64           `json:"id"`
	FirstName      string          `json:"first_name"`
	LastName       string          `json:"last_name"`
	Nickname       *string         `json:"nickname"`
	AvatarURL      string          `json:"avatar_url"`
	IsPrivate      bool            `json:"is_private"`
	CanViewProfile bool            `json:"can_view_profile"`
	Email          *string         `json:"email,omitempty"`
	DateOfBirth    *string         `json:"date_of_birth,omitempty"`
	Gender         **domain.Gender `json:"gender,omitempty"`
	AboutMe        **string        `json:"about_me,omitempty"`
	PostsCount     *int64          `json:"posts_count,omitempty"`
	FollowersCount *int64          `json:"followers_count,omitempty"`
	FollowingCount *int64          `json:"following_count,omitempty"`
}

func newUserProfileResponse(profile *service.UserProfile) *userProfileResponse {
	if profile == nil || profile.User == nil {
		return nil
	}
	user := profile.User
	response := &userProfileResponse{
		ID:             user.ID,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		Nickname:       user.Nickname,
		AvatarURL:      domain.UserAvatarURL(user),
		IsPrivate:      user.IsPrivate,
		CanViewProfile: profile.CanView,
	}
	if !profile.CanView {
		return response
	}
	response.Email = &user.Email
	response.DateOfBirth = &user.DateOfBirth
	response.Gender = &user.Gender
	response.AboutMe = &user.AboutMe
	if profile.Statistics != nil {
		response.PostsCount = &profile.Statistics.Posts
		response.FollowersCount = &profile.Statistics.Followers
		response.FollowingCount = &profile.Statistics.Following
	}
	return response
}

type userDirectoryResponse struct {
	Users      []relatedUserSummaryResponse `json:"users"`
	NextCursor *string                      `json:"next_cursor"`
}

func newUserDirectoryResponse(page *service.UserPage) userDirectoryResponse {
	response := userDirectoryResponse{Users: make([]relatedUserSummaryResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, user := range page.Users {
		response.Users = append(response.Users, newRelatedUserSummaryResponse(user))
	}
	return response
}
