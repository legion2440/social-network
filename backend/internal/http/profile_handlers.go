package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"social-network/backend/internal/service"
)

const maxProfileJSONBytes = 1 << 20

var profileJSONFields = map[string]func(*service.UpdateProfileInput) *service.ProfileField{
	"first_name":    func(input *service.UpdateProfileInput) *service.ProfileField { return &input.FirstName },
	"last_name":     func(input *service.UpdateProfileInput) *service.ProfileField { return &input.LastName },
	"date_of_birth": func(input *service.UpdateProfileInput) *service.ProfileField { return &input.DateOfBirth },
	"gender":        func(input *service.UpdateProfileInput) *service.ProfileField { return &input.Gender },
	"nickname":      func(input *service.UpdateProfileInput) *service.ProfileField { return &input.Nickname },
	"about_me":      func(input *service.UpdateProfileInput) *service.ProfileField { return &input.AboutMe },
}

func (h *Handler) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", http.MethodPatch)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.profile == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	input, err := readUpdateProfileInput(w, r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	user, err := h.profile.Update(r.Context(), current.ID, input)
	if h.handleProfileServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newAuthUserResponse(user))
}

func (h *Handler) handleProfileAvatar(w http.ResponseWriter, r *http.Request) {
	if h.profile == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())

	var (
		userErr error
		user    *authUserResponse
	)
	switch r.Method {
	case http.MethodPut:
		r.Body = http.MaxBytesReader(w, r.Body, service.MaxAvatarBodyBytes)
		file, originalName, err := readMultipartUploadFile(r, "avatar")
		if err != nil {
			if isMultipartTooLarge(err) {
				writeError(w, http.StatusRequestEntityTooLarge, "avatar is too big (max 20MB)")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid input")
			return
		}
		defer file.Close()
		updated, err := h.profile.ReplaceAvatar(r.Context(), current.ID, service.MediaUpload{
			OriginalName: originalName,
			Reader:       file,
		})
		userErr = err
		user = newAuthUserResponse(updated)
	case http.MethodDelete:
		updated, err := h.profile.DeleteAvatar(r.Context(), current.ID)
		userErr = err
		user = newAuthUserResponse(updated)
	default:
		w.Header().Set("Allow", http.MethodPut+", "+http.MethodDelete)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.handleProfileServiceError(w, userErr) {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func readUpdateProfileInput(w http.ResponseWriter, r *http.Request) (service.UpdateProfileInput, error) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxProfileJSONBytes))
	var raw map[string]json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return service.UpdateProfileInput{}, err
	}
	if err := ensureJSONEOF(decoder); err != nil || len(raw) == 0 {
		return service.UpdateProfileInput{}, service.ErrInvalidInput
	}

	var input service.UpdateProfileInput
	for name, value := range raw {
		fieldForName, ok := profileJSONFields[name]
		if !ok {
			return service.UpdateProfileInput{}, service.ErrInvalidInput
		}
		field := fieldForName(&input)
		field.Present = true
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			continue
		}
		var stringValue string
		if err := json.Unmarshal(value, &stringValue); err != nil {
			return service.UpdateProfileInput{}, service.ErrInvalidInput
		}
		field.Value = &stringValue
	}
	return input, nil
}

func (h *Handler) handleProfileServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, service.ErrInvalidMediaType):
		writeError(w, http.StatusBadRequest, "avatar must be JPEG, PNG, GIF or WebP")
	case errors.Is(err, service.ErrMediaTooBig), isMultipartTooLarge(err):
		writeError(w, http.StatusRequestEntityTooLarge, "avatar is too big (max 20MB)")
	default:
		h.logger.Printf("profile request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}
