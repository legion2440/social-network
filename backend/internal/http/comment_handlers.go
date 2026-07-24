package http

import (
	"errors"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

const commentMultipartMemory = 1 << 20

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
		if err != nil || mediaType != "multipart/form-data" {
			writeError(w, http.StatusUnsupportedMediaType, "content type must be multipart/form-data")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, service.MaxMediaBodyBytes)
		input, mediaFile, err := readCreateCommentInput(r)
		if mediaFile != nil {
			defer mediaFile.Close()
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		if err != nil {
			if isMultipartTooLarge(err) {
				writeError(w, http.StatusRequestEntityTooLarge, "media is too big (max 20MB)")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		comment, err := h.comments.Create(r.Context(), current.ID, postID, input)
		if h.handleCommentServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, newCommentResponse(comment))
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func readCreateCommentInput(r *http.Request) (service.CreateCommentInput, multipart.File, error) {
	if err := r.ParseMultipartForm(commentMultipartMemory); err != nil {
		return service.CreateCommentInput{}, nil, err
	}
	form := r.MultipartForm
	if form == nil {
		return service.CreateCommentInput{}, nil, service.ErrInvalidInput
	}
	for name := range form.Value {
		if name != "text" {
			return service.CreateCommentInput{}, nil, service.ErrInvalidInput
		}
	}
	for name := range form.File {
		if name != "media" {
			return service.CreateCommentInput{}, nil, service.ErrInvalidInput
		}
	}
	textValues, exists := form.Value["text"]
	if !exists || len(textValues) != 1 {
		return service.CreateCommentInput{}, nil, service.ErrInvalidInput
	}
	input := service.CreateCommentInput{Text: textValues[0]}
	mediaHeaders := form.File["media"]
	if len(mediaHeaders) > 1 {
		return service.CreateCommentInput{}, nil, service.ErrInvalidInput
	}
	if len(mediaHeaders) == 0 {
		return input, nil, nil
	}
	mediaFile, err := mediaHeaders[0].Open()
	if err != nil {
		return service.CreateCommentInput{}, nil, err
	}
	input.Media = &service.MediaUpload{
		OriginalName: mediaHeaders[0].Filename,
		Reader:       mediaFile,
	}
	return input, mediaFile, nil
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
	case errors.Is(err, service.ErrInvalidMediaType):
		writeError(w, http.StatusBadRequest, "media must be JPEG, PNG, GIF or WebP")
	case errors.Is(err, service.ErrMediaTooBig):
		writeError(w, http.StatusRequestEntityTooLarge, "media is too big (max 20MB)")
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
	MediaURL  *string             `json:"media_url"`
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
		MediaURL:  domain.CommentMediaURL(comment),
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
