package http

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	realtimews "social-network/backend/internal/realtime/ws"
	"social-network/backend/internal/service"
)

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if h.db == nil {
		h.logger.Print("health check: database is not configured")
		writeJSON(w, http.StatusServiceUnavailable, map[string]bool{"ok": false})
		return
	}
	if err := h.db.PingContext(ctx); err != nil {
		h.logger.Printf("health check: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]bool{"ok": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) handleWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	user, _ := CurrentUserFromContext(r.Context())
	if h.hub == nil || h.chats == nil || h.auth == nil {
		writeError(w, http.StatusServiceUnavailable, "realtime unavailable")
		return
	}
	peers, err := h.chats.DirectPeerIDs(r.Context(), user.ID)
	if err != nil {
		h.logger.Printf("websocket peers: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	profile, err := h.auth.Me(r.Context(), user.ID)
	if err != nil {
		h.logger.Printf("websocket user: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	displayName := strings.TrimSpace(profile.FirstName + " " + profile.LastName)
	if profile.Nickname != nil && strings.TrimSpace(*profile.Nickname) != "" {
		displayName = strings.TrimSpace(*profile.Nickname)
	}
	if err := realtimews.Serve(w, r, h.hub, h.chats, user.ID, user.SessionToken, user.SessionExpiresAt, displayName, peers); err != nil {
		h.logger.Printf("websocket: %v", err)
	}
}

func (h *Handler) handleMediaUpload(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/media" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, _ := CurrentUserFromContext(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxMediaBodyBytes)
	file, originalName, err := readMediaUploadFile(r)
	if err != nil {
		if isMultipartTooLarge(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "media is too big (max 20MB)")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	defer file.Close()

	media, err := h.media.Upload(r.Context(), user.ID, service.MediaUpload{
		OriginalName: originalName,
		Reader:       file,
	})
	if h.handleMediaServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, newMediaResponse(media))
}

func (h *Handler) handleMediaDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	rawID := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if rawID == "" || strings.Contains(rawID, "/") {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	mediaID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || mediaID <= 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	user, _ := CurrentUserFromContext(r.Context())
	media, filePath, err := h.media.OpenOwned(r.Context(), mediaID, user.ID)
	if h.handleMediaServiceError(w, err) {
		return
	}
	if _, err := os.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.logger.Printf("stat media %d: %v", mediaID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", media.MIME)
	w.Header().Set("Content-Length", strconv.FormatInt(media.Size, 10))
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filePath)
}

func (h *Handler) handleNotImplemented(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func readMediaUploadFile(r *http.Request) (*multipart.Part, string, error) {
	return readMultipartUploadFile(r, "file")
}

func readMultipartUploadFile(r *http.Request, fieldName string) (*multipart.Part, string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", err
	}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return nil, "", service.ErrInvalidInput
		}
		if err != nil {
			return nil, "", err
		}
		if part.FormName() != fieldName {
			_ = part.Close()
			continue
		}
		filename := strings.TrimSpace(part.FileName())
		if filename == "" {
			_ = part.Close()
			return nil, "", service.ErrInvalidInput
		}
		return part, filename, nil
	}
}

func (h *Handler) handleMediaServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrInvalidMediaType):
		writeError(w, http.StatusBadRequest, "only JPEG, PNG, GIF and WebP are allowed")
	case errors.Is(err, service.ErrMediaTooBig), isMultipartTooLarge(err):
		writeError(w, http.StatusRequestEntityTooLarge, "media is too big (max 20MB)")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		h.logger.Printf("media request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

func isMultipartTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "request body too large") ||
		strings.Contains(message, "http: post too large") ||
		strings.Contains(message, "multipart: message too large")
}

type mediaResponse struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	MIME string `json:"mime"`
	Size int64  `json:"size"`
}

func newMediaResponse(media *domain.Media) *mediaResponse {
	if media == nil || media.ID <= 0 {
		return nil
	}
	return &mediaResponse{
		ID:   strconv.FormatInt(media.ID, 10),
		URL:  media.URL,
		MIME: media.MIME,
		Size: media.Size,
	}
}
