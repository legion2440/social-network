package http

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

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
	presenceGeneration, err := h.hub.BeginPresenceSync(user.ID)
	if err != nil {
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
	if err := realtimews.Serve(
		w, r, h.hub, h.chats, user.ID, user.SessionToken, user.SessionExpiresAt,
		displayName, presenceGeneration, peers,
	); err != nil {
		h.logger.Printf("websocket: %v", err)
	}
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
