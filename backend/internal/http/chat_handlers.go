package http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

func (h *Handler) handleChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.chats == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	cursor, limit, err := readChatListPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.chats.List(r.Context(), current.ID, cursor, limit)
	if h.handleChatServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newChatPageResponse(page))
}

func (h *Handler) handleDirectChatMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	targetUserID, ok := positiveNamedPathID(r, "user_id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	h.handleChatHistory(w, r, domain.ChatRef{Kind: domain.ChatDirect, TargetID: targetUserID})
}

func (h *Handler) handleGroupChatMessages(w http.ResponseWriter, r *http.Request) {
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
	h.handleChatHistory(w, r, domain.ChatRef{Kind: domain.ChatGroup, TargetID: groupID})
}

func (h *Handler) handleChatHistory(w http.ResponseWriter, r *http.Request, chat domain.ChatRef) {
	if h.chats == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	cursor, limit, err := readChatMessagePageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	var page *service.ChatMessagePage
	if chat.Kind == domain.ChatDirect {
		page, err = h.chats.DirectHistory(r.Context(), current.ID, chat.TargetID, cursor, limit)
	} else {
		page, err = h.chats.GroupHistory(r.Context(), current.ID, chat.TargetID, cursor, limit)
	}
	if h.handleChatServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newChatMessagePageResponse(page))
}

func readChatListPageQuery(r *http.Request) (*domain.ChatListCursor, int, error) {
	limit, rawCursor, err := readChatPageValues(r)
	if err != nil || rawCursor == "" {
		return nil, limit, err
	}
	cursor, err := service.DecodeChatListCursor(rawCursor)
	return cursor, limit, err
}

func readChatMessagePageQuery(r *http.Request) (*domain.ChatMessageCursor, int, error) {
	limit, rawCursor, err := readChatPageValues(r)
	if err != nil || rawCursor == "" {
		return nil, limit, err
	}
	cursor, err := service.DecodeChatMessageCursor(rawCursor)
	return cursor, limit, err
}

func readChatPageValues(r *http.Request) (int, string, error) {
	values := r.URL.Query()
	for name := range values {
		if name != "cursor" && name != "limit" {
			return 0, "", service.ErrInvalidInput
		}
	}
	limit := service.DefaultChatPageLimit
	if raw, exists := values["limit"]; exists {
		if len(raw) != 1 {
			return 0, "", service.ErrInvalidInput
		}
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed < 1 || parsed > service.MaxChatPageLimit {
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

func (h *Handler) handleChatServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, "conflict")
	default:
		h.logger.Printf("chat request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

type chatMessageResponse struct {
	ID              int64               `json:"id"`
	ClientMessageID string              `json:"client_message_id"`
	Chat            domain.ChatRef      `json:"chat"`
	Sender          userSummaryResponse `json:"sender"`
	Body            string              `json:"body"`
	CreatedAt       time.Time           `json:"created_at"`
}

func newChatMessageResponse(message *domain.ChatMessage) *chatMessageResponse {
	if message == nil {
		return nil
	}
	return &chatMessageResponse{
		ID: message.ID, ClientMessageID: message.ClientMessageID, Chat: message.Chat,
		Sender: newUserSummaryResponse(message.Sender), Body: message.Body, CreatedAt: message.CreatedAt,
	}
}

type chatMessagePageResponse struct {
	Messages   []*chatMessageResponse `json:"messages"`
	NextCursor *string                `json:"next_cursor"`
}

func newChatMessagePageResponse(page *service.ChatMessagePage) chatMessagePageResponse {
	response := chatMessagePageResponse{Messages: make([]*chatMessageResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, message := range page.Messages {
		response.Messages = append(response.Messages, newChatMessageResponse(message))
	}
	return response
}

type chatSummaryResponse struct {
	Kind        domain.ChatKind      `json:"kind"`
	TargetID    int64                `json:"target_id"`
	User        *userSummaryResponse `json:"user,omitempty"`
	Group       *groupResponse       `json:"group,omitempty"`
	LastMessage *chatMessageResponse `json:"last_message"`
	ActivityAt  time.Time            `json:"activity_at"`
}

type chatPageResponse struct {
	Chats      []chatSummaryResponse `json:"chats"`
	NextCursor *string               `json:"next_cursor"`
}

func newChatPageResponse(page *service.ChatPage) chatPageResponse {
	response := chatPageResponse{Chats: make([]chatSummaryResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, chat := range page.Chats {
		item := chatSummaryResponse{
			Kind: chat.Kind, TargetID: chat.TargetID, LastMessage: newChatMessageResponse(chat.LastMessage), ActivityAt: chat.ActivityAt,
		}
		if chat.User != nil {
			user := newUserSummaryResponse(chat.User)
			item.User = &user
		}
		if chat.Group != nil {
			item.Group = newGroupResponse(chat.Group)
		}
		response.Chats = append(response.Chats, item)
	}
	return response
}
