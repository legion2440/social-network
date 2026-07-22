package http

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	realtimews "social-network/backend/internal/realtime/ws"
	"social-network/backend/internal/service"
)

const registrationMultipartMemory = 1 << 20

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, service.MaxAvatarBodyBytes)
	input, avatarFile, err := readRegisterInput(r)
	if avatarFile != nil {
		defer avatarFile.Close()
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		if isMultipartTooLarge(err) {
			writeError(w, http.StatusRequestEntityTooLarge, "avatar is too big (max 20MB)")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}

	result, err := h.auth.Register(r.Context(), input)
	if h.handleAuthServiceError(w, err) {
		return
	}
	SetSessionCookie(w, result.Session.Token, result.Session.ExpiresAt, h.cookieSecure)
	writeJSON(w, http.StatusCreated, newAuthUserResponse(result.User))
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var input service.LoginInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}
	if err := ensureJSONEOF(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "invalid input")
		return
	}

	result, err := h.auth.Login(r.Context(), input)
	if h.handleAuthServiceError(w, err) {
		return
	}
	SetSessionCookie(w, result.Session.Token, result.Session.ExpiresAt, h.cookieSecure)
	writeJSON(w, http.StatusOK, newAuthUserResponse(result.User))
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	token, _ := h.sessionToken.Extract(r)
	var sessionKey realtimews.SessionKey
	if strings.TrimSpace(token) != "" {
		sessionKey = realtimews.HashSessionToken(token)
	}
	if err := h.auth.Logout(r.Context(), token); err != nil {
		h.logger.Printf("logout: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if strings.TrimSpace(token) != "" && h.hub != nil {
		if err := h.hub.RevokeSession(sessionKey); err != nil && !errors.Is(err, realtimews.ErrHubStopped) {
			h.logger.Printf("logout realtime revoke: %v", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}
	ClearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	current, _ := CurrentUserFromContext(r.Context())
	user, err := h.auth.Me(r.Context(), current.ID)
	if h.handleAuthServiceError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, newAuthUserResponse(user))
}

func readRegisterInput(r *http.Request) (service.RegisterInput, multipart.File, error) {
	if err := r.ParseMultipartForm(registrationMultipartMemory); err != nil {
		return service.RegisterInput{}, nil, err
	}
	form := r.MultipartForm
	if form == nil {
		return service.RegisterInput{}, nil, service.ErrInvalidInput
	}

	value := func(name string, required bool) (string, error) {
		values, exists := form.Value[name]
		if !exists {
			if required {
				return "", service.ErrInvalidInput
			}
			return "", nil
		}
		if len(values) != 1 {
			return "", service.ErrInvalidInput
		}
		return values[0], nil
	}

	email, err := value("email", true)
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	password, err := value("password", true)
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	firstName, err := value("first_name", true)
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	lastName, err := value("last_name", true)
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	dateOfBirth, err := value("date_of_birth", true)
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	genderValue, err := value("gender", false)
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	nickname, err := optionalFormValue(form, "nickname")
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	aboutMe, err := optionalFormValue(form, "about_me")
	if err != nil {
		return service.RegisterInput{}, nil, err
	}

	var gender *domain.Gender
	if _, exists := form.Value["gender"]; exists {
		parsed := domain.Gender(genderValue)
		if !parsed.Valid() {
			return service.RegisterInput{}, nil, service.ErrInvalidInput
		}
		gender = &parsed
	}

	input := service.RegisterInput{
		Email:       email,
		Password:    password,
		FirstName:   firstName,
		LastName:    lastName,
		DateOfBirth: dateOfBirth,
		Gender:      gender,
		Nickname:    nickname,
		AboutMe:     aboutMe,
	}

	avatarHeaders := form.File["avatar"]
	if len(avatarHeaders) > 1 {
		return service.RegisterInput{}, nil, service.ErrInvalidInput
	}
	if len(avatarHeaders) == 0 {
		return input, nil, nil
	}
	avatarFile, err := avatarHeaders[0].Open()
	if err != nil {
		return service.RegisterInput{}, nil, err
	}
	input.Avatar = &service.MediaUpload{
		OriginalName: avatarHeaders[0].Filename,
		Reader:       avatarFile,
	}
	return input, avatarFile, nil
}

func optionalFormValue(form *multipart.Form, name string) (*string, error) {
	values, exists := form.Value[name]
	if !exists {
		return nil, nil
	}
	if len(values) != 1 {
		return nil, service.ErrInvalidInput
	}
	return &values[0], nil
}

func (h *Handler) handleAuthServiceError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already exists")
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, service.ErrInvalidMediaType):
		writeError(w, http.StatusBadRequest, "avatar must be JPEG, PNG, GIF or WebP")
	case errors.Is(err, service.ErrMediaTooBig):
		writeError(w, http.StatusRequestEntityTooLarge, "avatar is too big (max 20MB)")
	default:
		h.logger.Printf("auth request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
	return true
}

type authUserResponse struct {
	ID          int64          `json:"id"`
	Email       string         `json:"email"`
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	DateOfBirth string         `json:"date_of_birth"`
	Gender      *domain.Gender `json:"gender"`
	Nickname    *string        `json:"nickname"`
	AboutMe     *string        `json:"about_me"`
	AvatarURL   string         `json:"avatar_url"`
	IsPrivate   bool           `json:"is_private"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func newAuthUserResponse(user *domain.User) *authUserResponse {
	if user == nil {
		return nil
	}
	return &authUserResponse{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		DateOfBirth: user.DateOfBirth,
		Gender:      user.Gender,
		Nickname:    user.Nickname,
		AboutMe:     user.AboutMe,
		AvatarURL:   domain.UserAvatarURL(user),
		IsPrivate:   user.IsPrivate,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return service.ErrInvalidInput
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
