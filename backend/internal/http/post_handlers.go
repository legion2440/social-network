package http

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"
)

const postMultipartMemory = 1 << 20

func (h *Handler) handlePosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.posts == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, service.MaxMediaBodyBytes)
	input, mediaFile, err := readCreatePostInput(r)
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

	current, _ := CurrentUserFromContext(r.Context())
	post, err := h.posts.Create(r.Context(), current.ID, input)
	if h.handlePostServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, newPostResponse(post))
}

func (h *Handler) handlePostFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.posts == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	cursor, limit, err := readPostPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.posts.Feed(r.Context(), current.ID, cursor, limit)
	if h.handlePostServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newPostPageResponse(page))
}

func (h *Handler) handleUserPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.posts == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	targetUserID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	cursor, limit, err := readPostPageQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	page, err := h.posts.UserPosts(r.Context(), current.ID, targetUserID, cursor, limit)
	if h.handlePostServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newPostPageResponse(page))
}

func (h *Handler) handleGroupPosts(w http.ResponseWriter, r *http.Request) {
	if h.posts == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	groupID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())

	switch r.Method {
	case http.MethodGet:
		cursor, limit, err := readPostPageQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		page, err := h.posts.GroupPosts(r.Context(), current.ID, groupID, cursor, limit)
		if h.handlePostServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, newPostPageResponse(page))
	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, service.MaxMediaBodyBytes)
		input, mediaFile, err := readCreateGroupPostInput(r)
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
		post, err := h.posts.CreateGroupPost(r.Context(), current.ID, groupID, input)
		if h.handlePostServiceError(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, newPostResponse(post))
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handlePostMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.postMedia == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	postID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	delivery, err := h.postMedia.Open(r.Context(), current.ID, postID)
	if h.handlePostMediaError(w, err) {
		return
	}

	file, err := os.Open(delivery.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			h.logger.Printf("post %d media %d: file not found", postID, delivery.MediaID)
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.logger.Printf("open post %d media %d: %v", postID, delivery.MediaID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		h.logger.Printf("stat post %d media %d: %v", postID, delivery.MediaID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !info.Mode().IsRegular() {
		h.logger.Printf("post %d media %d: storage path is not a regular file", postID, delivery.MediaID)
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	w.Header().Set("Content-Type", delivery.MIME)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	if _, err := io.Copy(w, file); err != nil {
		h.logger.Printf("write post %d media %d: %v", postID, delivery.MediaID, err)
	}
}

func readCreatePostInput(r *http.Request) (service.CreatePostInput, multipart.File, error) {
	if err := r.ParseMultipartForm(postMultipartMemory); err != nil {
		return service.CreatePostInput{}, nil, err
	}
	form := r.MultipartForm
	if form == nil {
		return service.CreatePostInput{}, nil, service.ErrInvalidInput
	}

	for name := range form.Value {
		if name != "text" && name != "privacy" && name != "selected_user_id" {
			return service.CreatePostInput{}, nil, service.ErrInvalidInput
		}
	}
	for name := range form.File {
		if name != "media" {
			return service.CreatePostInput{}, nil, service.ErrInvalidInput
		}
	}
	if len(form.Value["text"]) != 1 || len(form.Value["privacy"]) != 1 {
		return service.CreatePostInput{}, nil, service.ErrInvalidInput
	}

	input := service.CreatePostInput{
		Text:    form.Value["text"][0],
		Privacy: domain.PostPrivacy(form.Value["privacy"][0]),
	}
	for _, rawID := range form.Value["selected_user_id"] {
		id, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || id <= 0 {
			return service.CreatePostInput{}, nil, service.ErrInvalidInput
		}
		input.SelectedUserIDs = append(input.SelectedUserIDs, id)
	}

	mediaHeaders := form.File["media"]
	if len(mediaHeaders) > 1 {
		return service.CreatePostInput{}, nil, service.ErrInvalidInput
	}
	if len(mediaHeaders) == 0 {
		return input, nil, nil
	}
	if strings.TrimSpace(mediaHeaders[0].Filename) == "" {
		return service.CreatePostInput{}, nil, service.ErrInvalidInput
	}
	mediaFile, err := mediaHeaders[0].Open()
	if err != nil {
		return service.CreatePostInput{}, nil, err
	}
	input.Media = &service.MediaUpload{
		OriginalName: mediaHeaders[0].Filename,
		Reader:       mediaFile,
	}
	return input, mediaFile, nil
}

func readCreateGroupPostInput(r *http.Request) (service.CreateGroupPostInput, multipart.File, error) {
	if err := r.ParseMultipartForm(postMultipartMemory); err != nil {
		return service.CreateGroupPostInput{}, nil, err
	}
	form := r.MultipartForm
	if form == nil {
		return service.CreateGroupPostInput{}, nil, service.ErrInvalidInput
	}
	for name := range form.Value {
		if name != "text" {
			return service.CreateGroupPostInput{}, nil, service.ErrInvalidInput
		}
	}
	for name := range form.File {
		if name != "media" {
			return service.CreateGroupPostInput{}, nil, service.ErrInvalidInput
		}
	}
	if len(form.Value["text"]) != 1 {
		return service.CreateGroupPostInput{}, nil, service.ErrInvalidInput
	}

	input := service.CreateGroupPostInput{Text: form.Value["text"][0]}
	mediaHeaders := form.File["media"]
	if len(mediaHeaders) > 1 {
		return service.CreateGroupPostInput{}, nil, service.ErrInvalidInput
	}
	if len(mediaHeaders) == 0 {
		return input, nil, nil
	}
	if strings.TrimSpace(mediaHeaders[0].Filename) == "" {
		return service.CreateGroupPostInput{}, nil, service.ErrInvalidInput
	}
	mediaFile, err := mediaHeaders[0].Open()
	if err != nil {
		return service.CreateGroupPostInput{}, nil, err
	}
	input.Media = &service.MediaUpload{OriginalName: mediaHeaders[0].Filename, Reader: mediaFile}
	return input, mediaFile, nil
}

func readPostPageQuery(r *http.Request) (*domain.PostCursor, int, error) {
	values := r.URL.Query()
	for name := range values {
		if name != "cursor" && name != "limit" {
			return nil, 0, service.ErrInvalidInput
		}
	}
	limit := service.DefaultPostPageLimit
	if rawLimit, exists := values["limit"]; exists {
		if len(rawLimit) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		parsed, err := strconv.Atoi(rawLimit[0])
		if err != nil || parsed < 1 || parsed > service.MaxPostPageLimit {
			return nil, 0, service.ErrInvalidInput
		}
		limit = parsed
	}
	var cursor *domain.PostCursor
	if rawCursor, exists := values["cursor"]; exists {
		if len(rawCursor) != 1 {
			return nil, 0, service.ErrInvalidInput
		}
		var err error
		cursor, err = service.DecodePostCursor(rawCursor[0])
		if err != nil {
			return nil, 0, err
		}
	}
	return cursor, limit, nil
}

func (h *Handler) handlePostServiceError(w http.ResponseWriter, err error) bool {
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
	case errors.Is(err, service.ErrMediaTooBig), isMultipartTooLarge(err):
		writeError(w, http.StatusRequestEntityTooLarge, "media is too big (max 20MB)")
	default:
		h.logger.Printf("post request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

func (h *Handler) handlePostMediaError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		h.logger.Printf("post media delivery: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

type postResponse struct {
	ID            int64               `json:"id"`
	Author        userSummaryResponse `json:"author"`
	GroupID       *int64              `json:"group_id,omitempty"`
	Text          string              `json:"text"`
	Privacy       *domain.PostPrivacy `json:"privacy,omitempty"`
	MediaURL      *string             `json:"media_url"`
	CommentsCount int64               `json:"comments_count"`
	CreatedAt     time.Time           `json:"created_at"`
}

func newPostResponse(post *domain.Post) *postResponse {
	if post == nil {
		return nil
	}
	return &postResponse{
		ID:            post.ID,
		Author:        newUserSummaryResponse(post.Author),
		GroupID:       post.GroupID,
		Text:          post.Text,
		Privacy:       post.Privacy,
		MediaURL:      domain.PostMediaURL(post),
		CommentsCount: post.CommentsCount,
		CreatedAt:     post.CreatedAt,
	}
}

type postPageResponse struct {
	Posts      []*postResponse `json:"posts"`
	NextCursor *string         `json:"next_cursor"`
}

func newPostPageResponse(page *service.PostPage) postPageResponse {
	response := postPageResponse{Posts: make([]*postResponse, 0), NextCursor: nil}
	if page == nil {
		return response
	}
	response.NextCursor = page.NextCursor
	for _, post := range page.Posts {
		response.Posts = append(response.Posts, newPostResponse(post))
	}
	return response
}
