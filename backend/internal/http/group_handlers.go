package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

const maxGroupJSONBytes = 64 << 10

func (h *Handler) handleGroups(w http.ResponseWriter, r *http.Request) {
	if h.groups == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		cursor, limit, err := readGroupPageQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		page, err := h.groups.Directory(r.Context(), current.ID, cursor, limit)
		if h.handleGroupServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, newGroupPageResponse(page))
	case http.MethodPost:
		if !requireJSONContentType(w, r) {
			return
		}
		title, description, err := readCreateGroupJSON(w, r)
		if handleGroupJSONReadError(w, err) {
			return
		}
		group, err := h.groups.Create(r.Context(), current.ID, title, description)
		if h.handleGroupServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, newGroupResponse(group))
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleGroupDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	group, err := h.groups.Detail(r.Context(), current.ID, groupID)
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupResponse(group))
}

func (h *Handler) handleGroupMembers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	cursor, limit, err := readGroupMemberPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.groups.Members(r.Context(), current.ID, groupID, cursor, limit)
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupMemberPageResponse(page))
}

func (h *Handler) handleGroupJoinRequest(w http.ResponseWriter, r *http.Request) {
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	var (
		group *domain.Group
		err   error
	)
	switch r.Method {
	case http.MethodPost:
		group, err = h.groups.RequestJoin(r.Context(), current.ID, groupID)
	case http.MethodDelete:
		group, err = h.groups.CancelJoinRequest(r.Context(), current.ID, groupID)
	default:
		w.Header().Set("Allow", http.MethodPost+", "+http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupResponse(group))
}

func (h *Handler) handleGroupJoinRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	cursor, limit, err := readGroupMembershipPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.groups.JoinRequests(r.Context(), current.ID, groupID, cursor, limit)
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupStatePageResponse("requests", page))
}

func (h *Handler) handleGroupJoinRequestAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.handleOwnerGroupTransition(w, r, true)
}

func (h *Handler) handleGroupJoinRequestReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.handleOwnerGroupTransition(w, r, false)
}

func (h *Handler) handleOwnerGroupTransition(w http.ResponseWriter, r *http.Request, accept bool) {
	groupID, groupOK := positiveNamedPathID(r, "id")
	userID, userOK := positiveNamedPathID(r, "user_id")
	if !groupOK || !userOK {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	var (
		group *domain.Group
		err   error
	)
	if accept {
		group, err = h.groups.AcceptJoinRequest(r.Context(), current.ID, groupID, userID)
	} else {
		group, err = h.groups.RejectJoinRequest(r.Context(), current.ID, groupID, userID)
	}
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupResponse(group))
}

func (h *Handler) handleGroupInvitations(w http.ResponseWriter, r *http.Request) {
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		cursor, limit, err := readGroupMembershipPageQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		page, err := h.groups.SentInvitations(r.Context(), current.ID, groupID, cursor, limit)
		if h.handleGroupServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, newGroupStatePageResponse("invitations", page))
	case http.MethodPost:
		if !requireJSONContentType(w, r) {
			return
		}
		userID, err := readInviteUserJSON(w, r)
		if handleGroupJSONReadError(w, err) {
			return
		}
		group, err := h.groups.Invite(r.Context(), current.ID, groupID, userID)
		if h.handleGroupServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, newGroupResponse(group))
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleGroupInvitationAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.handleOwnInvitationTransition(w, r, true)
}

func (h *Handler) handleGroupInvitationDecline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	h.handleOwnInvitationTransition(w, r, false)
}

func (h *Handler) handleOwnInvitationTransition(w http.ResponseWriter, r *http.Request, accept bool) {
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	var (
		group *domain.Group
		err   error
	)
	if accept {
		group, err = h.groups.AcceptInvitation(r.Context(), current.ID, groupID)
	} else {
		group, err = h.groups.DeclineInvitation(r.Context(), current.ID, groupID)
	}
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupResponse(group))
}

func (h *Handler) handleGroupMembership(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	group, err := h.groups.Leave(r.Context(), current.ID, groupID)
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupResponse(group))
}

func (h *Handler) handleGroupInvitationInbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cursor, limit, err := readGroupInvitationPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.groups.InvitationInbox(r.Context(), current.ID, cursor, limit)
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupInvitationInboxResponse(page))
}

func requireJSONContentType(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return false
	}
	return true
}

func readCreateGroupJSON(w http.ResponseWriter, r *http.Request) (string, string, error) {
	values, err := readStrictGroupJSONObject(w, r, map[string]bool{"title": true, "description": true})
	if err != nil {
		return "", "", err
	}
	return values["title"].(string), values["description"].(string), nil
}

func readInviteUserJSON(w http.ResponseWriter, r *http.Request) (int64, error) {
	values, err := readStrictGroupJSONObject(w, r, map[string]bool{"user_id": true})
	if err != nil {
		return 0, err
	}
	value, ok := values["user_id"].(json.Number)
	if !ok {
		return 0, service.ErrInvalidInput
	}
	id, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, service.ErrInvalidInput
	}
	return id, nil
}

func readStrictGroupJSONObject(w http.ResponseWriter, r *http.Request, allowed map[string]bool) (map[string]any, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxGroupJSONBytes))
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(body) {
		return nil, service.ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, service.ErrInvalidInput
	}
	values := make(map[string]any, len(allowed))
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, service.ErrInvalidInput
		}
		name, ok := token.(string)
		if !ok || !allowed[name] {
			return nil, service.ErrInvalidInput
		}
		if _, duplicate := values[name]; duplicate {
			return nil, service.ErrInvalidInput
		}
		var value any
		if err := decoder.Decode(&value); err != nil || value == nil {
			return nil, service.ErrInvalidInput
		}
		if name != "user_id" {
			if _, ok := value.(string); !ok {
				return nil, service.ErrInvalidInput
			}
		}
		values[name] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || len(values) != len(allowed) {
		return nil, service.ErrInvalidInput
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, service.ErrInvalidInput
	}
	return values, nil
}

func handleGroupJSONReadError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
	} else {
		writeError(w, http.StatusBadRequest, "invalid input")
	}
	return true
}

func readGroupPageQuery(r *http.Request) (*domain.GroupCursor, int, error) {
	var cursor *domain.GroupCursor
	limit, rawCursor, err := readGroupPageValues(r)
	if err == nil && rawCursor != "" {
		cursor, err = service.DecodeGroupCursor(rawCursor)
	}
	return cursor, limit, err
}

func readGroupMemberPageQuery(r *http.Request) (*domain.GroupMemberCursor, int, error) {
	var cursor *domain.GroupMemberCursor
	limit, rawCursor, err := readGroupPageValues(r)
	if err == nil && rawCursor != "" {
		cursor, err = service.DecodeGroupMemberCursor(rawCursor)
	}
	return cursor, limit, err
}

func readGroupMembershipPageQuery(r *http.Request) (*domain.GroupMembershipCursor, int, error) {
	var cursor *domain.GroupMembershipCursor
	limit, rawCursor, err := readGroupPageValues(r)
	if err == nil && rawCursor != "" {
		cursor, err = service.DecodeGroupMembershipCursor(rawCursor)
	}
	return cursor, limit, err
}

func readGroupInvitationPageQuery(r *http.Request) (*domain.GroupInvitationCursor, int, error) {
	var cursor *domain.GroupInvitationCursor
	limit, rawCursor, err := readGroupPageValues(r)
	if err == nil && rawCursor != "" {
		cursor, err = service.DecodeGroupInvitationCursor(rawCursor)
	}
	return cursor, limit, err
}

func readGroupPageValues(r *http.Request) (int, string, error) {
	values := r.URL.Query()
	for name := range values {
		if name != "cursor" && name != "limit" {
			return 0, "", service.ErrInvalidInput
		}
	}
	limit := service.DefaultGroupPageLimit
	if raw, exists := values["limit"]; exists {
		if len(raw) != 1 {
			return 0, "", service.ErrInvalidInput
		}
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed < 1 || parsed > service.MaxGroupPageLimit {
			return 0, "", service.ErrInvalidInput
		}
		limit = parsed
	}
	rawCursor := ""
	if raw, exists := values["cursor"]; exists {
		if len(raw) != 1 || raw[0] == "" {
			return 0, "", service.ErrInvalidInput
		}
		rawCursor = raw[0]
	}
	return limit, rawCursor, nil
}

func positiveNamedPathID(r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return value, err == nil && value > 0
}

func (h *Handler) handleGroupServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, "conflict")
	default:
		h.logger.Printf("group request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

type groupResponse struct {
	ID           int64               `json:"id"`
	Title        string              `json:"title"`
	Description  string              `json:"description"`
	CreatedAt    time.Time           `json:"created_at"`
	MembersCount int64               `json:"members_count"`
	ViewerStatus string              `json:"viewer_status"`
	Owner        userSummaryResponse `json:"owner"`
}

func newGroupResponse(group *domain.Group) *groupResponse {
	if group == nil {
		return nil
	}
	status := "none"
	if group.ViewerStatus != nil {
		status = string(*group.ViewerStatus)
	}
	return &groupResponse{
		ID: group.ID, Title: group.Title, Description: group.Description, CreatedAt: group.CreatedAt,
		MembersCount: group.MembersCount, ViewerStatus: status, Owner: newUserSummaryResponse(group.Owner),
	}
}

type groupPageResponse struct {
	Groups     []*groupResponse `json:"groups"`
	NextCursor *string          `json:"next_cursor"`
}

func newGroupPageResponse(page *service.GroupPage) groupPageResponse {
	response := groupPageResponse{Groups: make([]*groupResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, group := range page.Groups {
		response.Groups = append(response.Groups, newGroupResponse(group))
	}
	return response
}

type groupMemberResponse struct {
	User      userSummaryResponse          `json:"user"`
	Status    domain.GroupMembershipStatus `json:"status"`
	CreatedAt time.Time                    `json:"created_at"`
}

type groupMemberPageResponse struct {
	Members    []groupMemberResponse `json:"members"`
	NextCursor *string               `json:"next_cursor"`
}

func newGroupMemberPageResponse(page *service.GroupMemberPage) groupMemberPageResponse {
	response := groupMemberPageResponse{Members: make([]groupMemberResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, member := range page.Members {
		response.Members = append(response.Members, groupMemberResponse{User: newUserSummaryResponse(member.User), Status: member.Status, CreatedAt: member.CreatedAt})
	}
	return response
}

type groupStateItemResponse struct {
	User      userSummaryResponse `json:"user"`
	CreatedAt time.Time           `json:"created_at"`
}

type groupStatePageResponse struct {
	Requests    []groupStateItemResponse `json:"requests,omitempty"`
	Invitations []groupStateItemResponse `json:"invitations,omitempty"`
	NextCursor  *string                  `json:"next_cursor"`
}

func newGroupStatePageResponse(kind string, page *service.GroupMembershipPage) groupStatePageResponse {
	response := groupStatePageResponse{}
	items := make([]groupStateItemResponse, 0)
	if page != nil {
		response.NextCursor = page.NextCursor
		for _, membership := range page.Memberships {
			items = append(items, groupStateItemResponse{User: newUserSummaryResponse(membership.User), CreatedAt: membership.CreatedAt})
		}
	}
	if kind == "requests" {
		response.Requests = items
	} else {
		response.Invitations = items
	}
	return response
}

type groupInvitationInboxItemResponse struct {
	Group     *groupResponse `json:"group"`
	CreatedAt time.Time      `json:"created_at"`
}

type groupInvitationInboxResponse struct {
	Invitations []groupInvitationInboxItemResponse `json:"invitations"`
	NextCursor  *string                            `json:"next_cursor"`
}

func newGroupInvitationInboxResponse(page *service.GroupInvitationPage) groupInvitationInboxResponse {
	response := groupInvitationInboxResponse{Invitations: make([]groupInvitationInboxItemResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, invitation := range page.Invitations {
		response.Invitations = append(response.Invitations, groupInvitationInboxItemResponse{Group: newGroupResponse(invitation.Group), CreatedAt: invitation.CreatedAt})
	}
	return response
}
