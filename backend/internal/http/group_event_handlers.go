package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

func (h *Handler) handleGroupEvents(w http.ResponseWriter, r *http.Request) {
	if h.groupEvents == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	groupID, ok := positiveNamedPathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		cursor, limit, err := readGroupEventPageQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		page, err := h.groupEvents.List(r.Context(), current.ID, groupID, cursor, limit)
		if h.handleGroupServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, newGroupEventPageResponse(page))
	case http.MethodPost:
		if !requireJSONContentType(w, r) {
			return
		}
		input, err := readCreateGroupEventJSON(w, r)
		if handleGroupJSONReadError(w, err) {
			return
		}
		result, err := h.groupEvents.CreateWithEffects(r.Context(), current.ID, groupID, input)
		if h.handleGroupServiceError(w, err) {
			return
		}
		h.publishNotificationEffects(result.NotificationEffects)
		writeJSON(w, http.StatusCreated, newGroupEventResponse(result.Event))
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleGroupEventResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	groupID, groupOK := positiveNamedPathID(r, "id")
	eventID, eventOK := positiveNamedPathID(r, "event_id")
	if !groupOK || !eventOK {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if !requireJSONContentType(w, r) {
		return
	}
	response, err := readGroupEventResponseJSON(w, r)
	if handleGroupJSONReadError(w, err) {
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	event, err := h.groupEvents.Respond(r.Context(), current.ID, groupID, eventID, response)
	if h.handleGroupServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newGroupEventResponse(event))
}

func readCreateGroupEventJSON(w http.ResponseWriter, r *http.Request) (service.CreateGroupEventInput, error) {
	values, err := readStrictGroupEventStrings(w, r, []string{"title", "description", "starts_at"})
	if err != nil {
		return service.CreateGroupEventInput{}, err
	}
	startsAt, err := time.Parse(time.RFC3339, values["starts_at"])
	if err != nil {
		return service.CreateGroupEventInput{}, service.ErrInvalidInput
	}
	return service.CreateGroupEventInput{
		Title: values["title"], Description: values["description"], StartsAt: startsAt.UTC(),
	}, nil
}

func readGroupEventResponseJSON(w http.ResponseWriter, r *http.Request) (domain.GroupEventResponse, error) {
	values, err := readStrictGroupEventStrings(w, r, []string{"response"})
	if err != nil {
		return "", err
	}
	response := domain.GroupEventResponse(values["response"])
	if !response.Valid() {
		return "", service.ErrInvalidInput
	}
	return response, nil
}

func readStrictGroupEventStrings(w http.ResponseWriter, r *http.Request, required []string) (map[string]string, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxGroupJSONBytes))
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(body) {
		return nil, service.ErrInvalidInput
	}
	allowed := make(map[string]bool, len(required))
	for _, name := range required {
		allowed[name] = true
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, service.ErrInvalidInput
	}
	values := make(map[string]string, len(required))
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
		var value string
		if err := decoder.Decode(&value); err != nil {
			return nil, service.ErrInvalidInput
		}
		values[name] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || len(values) != len(required) {
		return nil, service.ErrInvalidInput
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, service.ErrInvalidInput
	}
	return values, nil
}

func readGroupEventPageQuery(r *http.Request) (*domain.GroupEventCursor, int, error) {
	values := r.URL.Query()
	for name := range values {
		if name != "cursor" && name != "limit" {
			return nil, 0, service.ErrInvalidInput
		}
	}
	limit := service.DefaultGroupEventPageLimit
	if raw, exists := values["limit"]; exists {
		if len(raw) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed < 1 || parsed > service.MaxGroupEventPageLimit {
			return nil, 0, service.ErrInvalidInput
		}
		limit = parsed
	}
	var cursor *domain.GroupEventCursor
	if raw, exists := values["cursor"]; exists {
		if len(raw) != 1 || raw[0] == "" {
			return nil, 0, service.ErrInvalidInput
		}
		var err error
		cursor, err = service.DecodeGroupEventCursor(raw[0])
		if err != nil {
			return nil, 0, err
		}
	}
	return cursor, limit, nil
}

type groupEventResponse struct {
	ID             int64                      `json:"id"`
	GroupID        int64                      `json:"group_id"`
	Creator        userSummaryResponse        `json:"creator"`
	Title          string                     `json:"title"`
	Description    string                     `json:"description"`
	StartsAt       time.Time                  `json:"starts_at"`
	CreatedAt      time.Time                  `json:"created_at"`
	GoingCount     int64                      `json:"going_count"`
	NotGoingCount  int64                      `json:"not_going_count"`
	ViewerResponse *domain.GroupEventResponse `json:"viewer_response"`
}

func newGroupEventResponse(event *domain.GroupEvent) *groupEventResponse {
	if event == nil {
		return nil
	}
	return &groupEventResponse{
		ID: event.ID, GroupID: event.GroupID, Creator: newUserSummaryResponse(event.Creator),
		Title: event.Title, Description: event.Description, StartsAt: event.StartsAt, CreatedAt: event.CreatedAt,
		GoingCount: event.GoingCount, NotGoingCount: event.NotGoingCount, ViewerResponse: event.ViewerResponse,
	}
}

type groupEventPageResponse struct {
	Events     []*groupEventResponse `json:"events"`
	NextCursor *string               `json:"next_cursor"`
}

func newGroupEventPageResponse(page *service.GroupEventPage) groupEventPageResponse {
	response := groupEventPageResponse{Events: make([]*groupEventResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, event := range page.Events {
		response.Events = append(response.Events, newGroupEventResponse(event))
	}
	return response
}
