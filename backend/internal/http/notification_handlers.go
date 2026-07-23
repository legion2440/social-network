package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"social-network/backend/internal/domain"
	realtimews "social-network/backend/internal/realtime/ws"
	"social-network/backend/internal/service"
)

func (h *Handler) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	cursor, limit, err := readNotificationPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.notifications.List(r.Context(), current.ID, cursor, limit)
	if h.handleNotificationServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newNotificationPageResponse(page))
}

func (h *Handler) handleNotificationRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(r.URL.Query()) != 0 {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if !requireEmptyRequestBody(w, r) {
		return
	}
	notificationID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	result, err := h.notifications.MarkRead(r.Context(), current.ID, notificationID)
	if h.handleNotificationServiceError(w, err) {
		return
	}
	h.publishNotificationUpsert(result.Notification, result.UnreadCount, result.Revision)
	writeJSON(w, http.StatusOK, notificationReadResponse{
		Notification: newNotificationResponse(result.Notification), UnreadCount: result.UnreadCount, Revision: result.Revision,
	})
}

func (h *Handler) handleNotificationsReadAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(r.URL.Query()) != 0 {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if !requireEmptyRequestBody(w, r) {
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	result, err := h.notifications.MarkAllRead(r.Context(), current.ID)
	if h.handleNotificationServiceError(w, err) {
		return
	}
	if result.Changed {
		h.publishNotificationsReadAll(current.ID, result.ReadAt, result.UnreadCount, result.Revision)
	}
	writeJSON(w, http.StatusOK, notificationReadAllResponse{
		ReadAt: result.ReadAt, UnreadCount: result.UnreadCount, Revision: result.Revision,
	})
}

func (h *Handler) handleNotificationAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(r.URL.Query()) != 0 {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if !requireJSONContentType(w, r) {
		return
	}
	notificationID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	values, err := readStrictGroupJSONObject(w, r, map[string]bool{"action": true})
	if handleGroupJSONReadError(w, err) {
		return
	}
	action := service.NotificationAction(values["action"].(string))
	if !action.Valid() {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	result, err := h.notifications.Action(r.Context(), current.ID, notificationID, action)
	if h.handleNotificationServiceError(w, err) {
		return
	}
	if result.SourceTransitionApplied {
		h.applyNotificationActionRealtime(result, action)
	}
	h.publishNotificationEffects(result.NotificationEffects)
	writeJSON(w, http.StatusOK, notificationActionResponse{
		Notification: newNotificationResponse(result.Notification),
		UnreadCount:  result.UnreadCount, Revision: result.Revision,
		Source: newNotificationActionSourceResponse(result.Source),
	})
}

func (h *Handler) applyNotificationActionRealtime(result *service.NotificationActionResult, action service.NotificationAction) {
	if h == nil || result == nil || result.Notification == nil {
		return
	}
	notification := result.Notification
	switch notification.Type {
	case domain.NotificationFollowRequest:
		h.refreshRealtimeRelationship(notification.RecipientUserID, notification.ActorUserID)
	case domain.NotificationGroupInvitation:
		if action == service.NotificationActionAccept && notification.GroupID != nil {
			h.changeRealtimeGroupAccess(*notification.GroupID, notification.RecipientUserID, true)
		}
	case domain.NotificationGroupJoinRequest:
		if action == service.NotificationActionAccept && notification.GroupID != nil {
			h.changeRealtimeGroupAccess(*notification.GroupID, notification.ActorUserID, true)
		}
	}
}

func (h *Handler) publishNotificationEffects(effects *service.NotificationEffects) {
	if h == nil || h.hub == nil || effects == nil {
		return
	}
	for _, notification := range effects.Upserts {
		state, ok := effects.StatesByUser[notification.RecipientUserID]
		if !ok {
			continue
		}
		h.publishNotificationUpsert(notification, state.UnreadCount, state.Revision)
	}
}

func (h *Handler) publishNotificationUpsert(notification *domain.Notification, unreadCount, revision int64) {
	if h == nil || h.hub == nil || notification == nil {
		return
	}
	payload, err := json.Marshal(notificationUpsertEnvelope{
		Type: "notification:upsert", Notification: newNotificationResponse(notification),
		UnreadCount: unreadCount, Revision: revision,
	})
	if err != nil {
		h.logger.Printf("notification realtime marshal: %v", err)
		return
	}
	if err := h.hub.PublishUsers(map[int64][]byte{notification.RecipientUserID: payload}); err != nil && !errors.Is(err, realtimews.ErrHubStopped) {
		h.logger.Printf("notification realtime publish: %v", err)
	}
}

func (h *Handler) publishNotificationsReadAll(userID int64, readAt time.Time, unreadCount, revision int64) {
	if h == nil || h.hub == nil || userID <= 0 {
		return
	}
	payload, err := json.Marshal(notificationReadAllEnvelope{
		Type: "notifications:read-all", ReadAt: readAt, UnreadCount: unreadCount, Revision: revision,
	})
	if err != nil {
		h.logger.Printf("notification read-all realtime marshal: %v", err)
		return
	}
	if err := h.hub.PublishUsers(map[int64][]byte{userID: payload}); err != nil && !errors.Is(err, realtimews.ErrHubStopped) {
		h.logger.Printf("notification read-all realtime publish: %v", err)
	}
}

func readNotificationPageQuery(r *http.Request) (*domain.NotificationCursor, int, error) {
	values := r.URL.Query()
	for name := range values {
		if name != "cursor" && name != "limit" {
			return nil, 0, service.ErrInvalidInput
		}
	}
	limit := service.DefaultNotificationPageLimit
	if raw, exists := values["limit"]; exists {
		if len(raw) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed < 1 || parsed > service.MaxNotificationPageLimit {
			return nil, 0, service.ErrInvalidInput
		}
		limit = parsed
	}
	var cursor *domain.NotificationCursor
	if raw, exists := values["cursor"]; exists {
		if len(raw) != 1 || raw[0] == "" {
			return nil, 0, service.ErrInvalidInput
		}
		var err error
		cursor, err = service.DecodeNotificationCursor(raw[0])
		if err != nil {
			return nil, 0, err
		}
	}
	return cursor, limit, nil
}

func requireEmptyRequestBody(w http.ResponseWriter, r *http.Request) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1))
	if err != nil || len(body) != 0 {
		writeError(w, http.StatusBadRequest, "invalid input")
		return false
	}
	return true
}

func (h *Handler) handleNotificationServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, "conflict")
	default:
		h.logger.Printf("notification request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

type notificationGroupSummaryResponse struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type notificationEventSummaryResponse struct {
	ID       int64     `json:"id"`
	Title    string    `json:"title"`
	StartsAt time.Time `json:"starts_at"`
}

type notificationResponse struct {
	ID         int64                             `json:"id"`
	Type       domain.NotificationType           `json:"type"`
	Actor      userSummaryResponse               `json:"actor"`
	FollowID   *int64                            `json:"follow_id"`
	Group      *notificationGroupSummaryResponse `json:"group"`
	Event      *notificationEventSummaryResponse `json:"event"`
	Resolution *domain.NotificationResolution    `json:"resolution"`
	ResolvedAt *time.Time                        `json:"resolved_at"`
	ReadAt     *time.Time                        `json:"read_at"`
	CreatedAt  time.Time                         `json:"created_at"`
}

func newNotificationResponse(notification *domain.Notification) *notificationResponse {
	if notification == nil {
		return nil
	}
	response := &notificationResponse{
		ID: notification.ID, Type: notification.Type, Actor: newUserSummaryResponse(notification.Actor),
		FollowID: notification.FollowID, Resolution: notification.Resolution,
		ResolvedAt: notification.ResolvedAt, ReadAt: notification.ReadAt, CreatedAt: notification.CreatedAt,
	}
	if notification.GroupID != nil && notification.GroupTitle != nil {
		response.Group = &notificationGroupSummaryResponse{ID: *notification.GroupID, Title: *notification.GroupTitle}
	}
	if notification.EventID != nil && notification.EventTitle != nil && notification.EventStartsAt != nil {
		response.Event = &notificationEventSummaryResponse{
			ID: *notification.EventID, Title: *notification.EventTitle, StartsAt: *notification.EventStartsAt,
		}
	}
	return response
}

type notificationPageResponse struct {
	Notifications []*notificationResponse `json:"notifications"`
	NextCursor    *string                 `json:"next_cursor"`
	UnreadCount   int64                   `json:"unread_count"`
	Revision      int64                   `json:"revision"`
}

func newNotificationPageResponse(page *service.NotificationPage) notificationPageResponse {
	response := notificationPageResponse{Notifications: []*notificationResponse{}}
	if page == nil {
		return response
	}
	response.NextCursor, response.UnreadCount, response.Revision = page.NextCursor, page.UnreadCount, page.Revision
	for _, notification := range page.Notifications {
		response.Notifications = append(response.Notifications, newNotificationResponse(notification))
	}
	return response
}

type notificationReadResponse struct {
	Notification *notificationResponse `json:"notification"`
	UnreadCount  int64                 `json:"unread_count"`
	Revision     int64                 `json:"revision"`
}

type notificationReadAllResponse struct {
	ReadAt      time.Time `json:"read_at"`
	UnreadCount int64     `json:"unread_count"`
	Revision    int64     `json:"revision"`
}

type notificationActionSourceResponse struct {
	Kind         service.NotificationActionSourceKind `json:"kind"`
	UserID       *int64                               `json:"user_id,omitempty"`
	Relationship *relationshipResponse                `json:"relationship,omitempty"`
	Group        *groupResponse                       `json:"group,omitempty"`
}

func newNotificationActionSourceResponse(source *service.NotificationActionSource) *notificationActionSourceResponse {
	if source == nil {
		return nil
	}
	response := &notificationActionSourceResponse{Kind: source.Kind}
	if source.Kind == service.NotificationSourceRelationship && source.Relationship != nil {
		userID := source.UserID
		response.UserID = &userID
		response.Relationship = &relationshipResponse{Status: source.Relationship.Status, FollowsMe: source.Relationship.FollowsMe}
	}
	if source.Kind == service.NotificationSourceGroup {
		response.Group = newGroupResponse(source.Group)
	}
	return response
}

type notificationActionResponse struct {
	Notification *notificationResponse             `json:"notification"`
	UnreadCount  int64                             `json:"unread_count"`
	Revision     int64                             `json:"revision"`
	Source       *notificationActionSourceResponse `json:"source"`
}

type notificationUpsertEnvelope struct {
	Type         string                `json:"type"`
	Notification *notificationResponse `json:"notification"`
	UnreadCount  int64                 `json:"unread_count"`
	Revision     int64                 `json:"revision"`
}

type notificationReadAllEnvelope struct {
	Type        string    `json:"type"`
	ReadAt      time.Time `json:"read_at"`
	UnreadCount int64     `json:"unread_count"`
	Revision    int64     `json:"revision"`
}
