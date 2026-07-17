package http

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"social-network/backend/internal/service"
)

func (h *Handler) handleUserAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.avatars == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	targetUserID, ok := positivePathID(r)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	delivery, err := h.avatars.Open(r.Context(), current.ID, targetUserID)
	if h.handleAvatarDeliveryError(w, err) {
		return
	}

	file, err := os.Open(delivery.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			h.logger.Printf("avatar user %d media %d: file not found", targetUserID, delivery.MediaID)
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		h.logger.Printf("open avatar user %d media %d: %v", targetUserID, delivery.MediaID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		h.logger.Printf("stat avatar user %d media %d: %v", targetUserID, delivery.MediaID, err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if !info.Mode().IsRegular() {
		h.logger.Printf("avatar user %d media %d: storage path is not a regular file", targetUserID, delivery.MediaID)
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	w.Header().Set("Content-Type", delivery.MIME)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	if _, err := io.Copy(w, file); err != nil {
		h.logger.Printf("write avatar user %d media %d: %v", targetUserID, delivery.MediaID, err)
	}
}

func (h *Handler) handleAvatarDeliveryError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	default:
		h.logger.Printf("avatar delivery: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}
