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

const maxCommentJSONBytes = 64 << 10

func (h *Handler) handlePostComments(w http.ResponseWriter, r *http.Request) {
	if h.comments == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	postID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		cursor, limit, err := readCommentPageQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		page, err := h.comments.List(r.Context(), current.ID, postID, cursor, limit)
		if h.handleCommentServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, newCommentPageResponse(page))
	case http.MethodPost:
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
			return
		}
		text, err := readCreateCommentText(w, r)
		if err != nil {
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		comment, err := h.comments.Create(r.Context(), current.ID, postID, text)
		if h.handleCommentServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, newCommentResponse(comment))
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func readCreateCommentText(w http.ResponseWriter, r *http.Request) (string, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxCommentJSONBytes))
	if err != nil {
		return "", err
	}
	if !utf8.Valid(body) {
		return "", service.ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return "", service.ErrInvalidInput
	}

	seenText := false
	text := ""
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return "", service.ErrInvalidInput
		}
		name, ok := token.(string)
		if !ok || name != "text" || seenText {
			return "", service.ErrInvalidInput
		}
		seenText = true
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return "", service.ErrInvalidInput
		}
		if err := json.Unmarshal(raw, &text); err != nil {
			return "", service.ErrInvalidInput
		}
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') || !seenText {
		return "", service.ErrInvalidInput
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return "", service.ErrInvalidInput
	}
	return text, nil
}

func readCommentPageQuery(r *http.Request) (*domain.CommentCursor, int, error) {
	values := r.URL.Query()
	for name := range values {
		if name != "cursor" && name != "limit" {
			return nil, 0, service.ErrInvalidInput
		}
	}
	limit := service.DefaultCommentPageLimit
	if rawLimit, exists := values["limit"]; exists {
		if len(rawLimit) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		parsed, err := strconv.Atoi(rawLimit[0])
		if err != nil || parsed < 1 || parsed > service.MaxCommentPageLimit {
			return nil, 0, service.ErrInvalidInput
		}
		limit = parsed
	}
	var cursor *domain.CommentCursor
	if rawCursor, exists := values["cursor"]; exists {
		if len(rawCursor) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		var err error
		cursor, err = service.DecodeCommentCursor(rawCursor[0])
		if err != nil {
			return nil, 0, err
		}
	}
	return cursor, limit, nil
}

func (h *Handler) handleCommentServiceError(w http.ResponseWriter, err error) bool {
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
	default:
		h.logger.Printf("comment request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

type commentResponse struct {
	ID        int64               `json:"id"`
	PostID    int64               `json:"post_id"`
	Text      string              `json:"text"`
	CreatedAt time.Time           `json:"created_at"`
	Author    userSummaryResponse `json:"author"`
}

func newCommentResponse(comment *domain.Comment) *commentResponse {
	if comment == nil {
		return nil
	}
	return &commentResponse{
		ID:        comment.ID,
		PostID:    comment.PostID,
		Text:      comment.Text,
		CreatedAt: comment.CreatedAt,
		Author:    newUserSummaryResponse(comment.Author),
	}
}

type commentPageResponse struct {
	Comments   []*commentResponse `json:"comments"`
	NextCursor *string            `json:"next_cursor"`
}

func newCommentPageResponse(page *service.CommentPage) commentPageResponse {
	response := commentPageResponse{Comments: make([]*commentResponse, 0)}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, comment := range page.Comments {
		response.Comments = append(response.Comments, newCommentResponse(comment))
	}
	return response
}
