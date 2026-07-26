package http

import (
	"errors"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"social-network/backend/internal/oauth"
	"social-network/backend/internal/service"
)

func (h *Handler) handleOAuthProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeJSON(w, http.StatusOK, map[string]any{"providers": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": h.auth.OAuthProviders()})
}

func (h *Handler) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !h.oauthStartLimiter.Allow(requestClientIP(r, h.trustProxy), time.Now()) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusNotFound, service.ErrOAuthProviderUnavailable.Error())
		return
	}
	browserNonce, err := oauth.NewBrowserNonce()
	if err != nil {
		h.logger.Printf("OAuth browser nonce: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	target, err := h.auth.StartOAuth(
		r.Context(),
		oauth.ProviderGitHub,
		r.URL.Query().Get("next"),
		oauth.HashBrowserNonce(browserNonce),
	)
	if err != nil {
		h.handleOAuthAPIError(w, err)
		return
	}
	setOAuthNonceCookie(w, browserNonce, h.cookieSecure)
	http.Redirect(w, r, target, http.StatusFound)
}

func (h *Handler) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		redirectOAuthError(w, r, service.ErrOAuthProviderUnavailable)
		return
	}
	registrationBrowserNonce, err := oauth.NewBrowserNonce()
	if err != nil {
		h.logger.Printf("OAuth registration browser nonce: %v", err)
		redirectOAuthError(w, r, service.ErrOAuthProviderError)
		return
	}
	result, err := h.auth.HandleOAuthCallback(r.Context(), service.OAuthCallbackInput{
		Provider:                     oauth.ProviderGitHub,
		State:                        r.URL.Query().Get("state"),
		Code:                         r.URL.Query().Get("code"),
		ProviderError:                r.URL.Query().Get("error"),
		BrowserNonce:                 readOAuthNonceCookie(r),
		RegistrationBrowserNonceHash: oauth.HashBrowserNonce(registrationBrowserNonce),
	})
	if err != nil {
		if result != nil && result.BrowserBindingVerified {
			clearOAuthNonceCookie(w, h.cookieSecure)
		}
		if !isKnownOAuthError(err) {
			h.logger.Printf("OAuth callback: %v", err)
			err = service.ErrOAuthProviderError
		}
		redirectOAuthError(w, r, err)
		return
	}
	if result == nil {
		h.logger.Printf("OAuth callback returned an empty result")
		clearOAuthNonceCookie(w, h.cookieSecure)
		redirectOAuthError(w, r, service.ErrOAuthProviderError)
		return
	}
	if result.Auth != nil {
		clearOAuthNonceCookie(w, h.cookieSecure)
		SetSessionCookie(
			w,
			result.Auth.Session.Token,
			result.Auth.Session.ExpiresAt,
			h.cookieSecure,
		)
		http.Redirect(w, r, result.Next, http.StatusFound)
		return
	}
	if result.RegistrationToken == "" {
		h.logger.Printf("OAuth callback returned no authentication or registration result")
		clearOAuthNonceCookie(w, h.cookieSecure)
		redirectOAuthError(w, r, service.ErrOAuthProviderError)
		return
	}
	setOAuthNonceCookie(w, registrationBrowserNonce, h.cookieSecure)
	target := "/oauth/complete?flow=" + url.QueryEscape(result.RegistrationToken)
	http.Redirect(w, r, target, http.StatusFound)
}

func (h *Handler) handleOAuthRegistrationFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusNotFound, service.ErrOAuthProviderUnavailable.Error())
		return
	}
	preview, err := h.auth.OAuthRegistrationPreview(
		r.Context(),
		r.PathValue("token"),
		readOAuthNonceCookie(r),
	)
	if err != nil {
		h.handleOAuthAPIError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *Handler) handleOAuthRegistrationComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !validOAuthCompletionRequest(r, h.oauthExpectedOrigin) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if h.auth == nil {
		writeError(w, http.StatusNotFound, service.ErrOAuthProviderUnavailable.Error())
		return
	}
	if !h.oauthCompleteLimiter.Allow(requestClientIP(r, h.trustProxy), time.Now()) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}
	token := r.PathValue("token")
	browserNonce := readOAuthNonceCookie(r)
	if _, err := h.auth.OAuthRegistrationPreview(r.Context(), token, browserNonce); err != nil {
		h.handleOAuthAPIError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, service.MaxAvatarBodyBytes)
	input, avatarFile, err := readOAuthRegistrationInput(r)
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
	result, err := h.auth.CompleteOAuthRegistration(r.Context(), token, input, browserNonce)
	if err != nil {
		h.handleOAuthAPIError(w, err)
		return
	}
	if result == nil || result.Auth == nil || result.Auth.User == nil || result.Auth.Session == nil {
		h.logger.Printf("OAuth completion returned an empty result")
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	clearOAuthNonceCookie(w, h.cookieSecure)
	SetSessionCookie(w, result.Auth.Session.Token, result.Auth.Session.ExpiresAt, h.cookieSecure)
	writeJSON(w, http.StatusCreated, map[string]any{
		"user": newAuthUserResponse(result.Auth.User),
		"next": result.Next,
	})
}

func readOAuthRegistrationInput(
	r *http.Request,
) (service.CompleteOAuthRegistrationInput, multipart.File, error) {
	if err := r.ParseMultipartForm(registrationMultipartMemory); err != nil {
		return service.CompleteOAuthRegistrationInput{}, nil, err
	}
	form := r.MultipartForm
	if form == nil {
		return service.CompleteOAuthRegistrationInput{}, nil, service.ErrInvalidInput
	}
	allowedValues := map[string]bool{
		"first_name": true, "last_name": true, "date_of_birth": true,
		"nickname": true, "about_me": true,
	}
	for name := range form.Value {
		if !allowedValues[name] {
			return service.CompleteOAuthRegistrationInput{}, nil, service.ErrInvalidInput
		}
	}
	for name := range form.File {
		if name != "avatar" {
			return service.CompleteOAuthRegistrationInput{}, nil, service.ErrInvalidInput
		}
	}
	required := func(name string) (string, error) {
		values, exists := form.Value[name]
		if !exists || len(values) != 1 {
			return "", service.ErrInvalidInput
		}
		return values[0], nil
	}
	firstName, err := required("first_name")
	if err != nil {
		return service.CompleteOAuthRegistrationInput{}, nil, err
	}
	lastName, err := required("last_name")
	if err != nil {
		return service.CompleteOAuthRegistrationInput{}, nil, err
	}
	dateOfBirth, err := required("date_of_birth")
	if err != nil {
		return service.CompleteOAuthRegistrationInput{}, nil, err
	}
	nickname, err := optionalFormValue(form, "nickname")
	if err != nil {
		return service.CompleteOAuthRegistrationInput{}, nil, err
	}
	aboutMe, err := optionalFormValue(form, "about_me")
	if err != nil {
		return service.CompleteOAuthRegistrationInput{}, nil, err
	}
	input := service.CompleteOAuthRegistrationInput{
		FirstName: firstName, LastName: lastName, DateOfBirth: dateOfBirth,
		Nickname: nickname, AboutMe: aboutMe,
	}
	avatarHeaders := form.File["avatar"]
	if len(avatarHeaders) > 1 {
		return service.CompleteOAuthRegistrationInput{}, nil, service.ErrInvalidInput
	}
	if len(avatarHeaders) == 0 {
		return input, nil, nil
	}
	avatarFile, err := avatarHeaders[0].Open()
	if err != nil {
		return service.CompleteOAuthRegistrationInput{}, nil, err
	}
	input.Avatar = &service.MediaUpload{
		OriginalName: avatarHeaders[0].Filename,
		Reader:       avatarFile,
	}
	return input, avatarFile, nil
}

func (h *Handler) handleOAuthAPIError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid input")
	case errors.Is(err, service.ErrOAuthProviderUnavailable):
		writeError(w, http.StatusNotFound, service.ErrOAuthProviderUnavailable.Error())
	case errors.Is(err, service.ErrOAuthFlowExpired):
		writeError(w, http.StatusGone, service.ErrOAuthFlowExpired.Error())
	case errors.Is(err, service.ErrOAuthEmailAlreadyRegistered):
		writeError(w, http.StatusConflict, service.ErrOAuthEmailAlreadyRegistered.Error())
	case errors.Is(err, service.ErrOAuthIdentityConflict):
		writeError(w, http.StatusConflict, service.ErrOAuthIdentityConflict.Error())
	case errors.Is(err, service.ErrInvalidMediaType):
		writeError(w, http.StatusBadRequest, "avatar must be JPEG, PNG, GIF or WebP")
	case errors.Is(err, service.ErrMediaTooBig):
		writeError(w, http.StatusRequestEntityTooLarge, "avatar is too big (max 20MB)")
	case isKnownOAuthError(err):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		h.logger.Printf("OAuth request: %v", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func redirectOAuthError(w http.ResponseWriter, r *http.Request, err error) {
	code := service.ErrOAuthProviderError.Error()
	if isKnownOAuthError(err) {
		code = err.Error()
	}
	http.Redirect(w, r, "/login?oauth_error="+url.QueryEscape(code), http.StatusFound)
}

func isKnownOAuthError(err error) bool {
	for _, known := range []error{
		service.ErrOAuthProviderUnavailable,
		service.ErrOAuthProviderError,
		service.ErrOAuthStateInvalid,
		service.ErrOAuthCodeMissing,
		service.ErrOAuthTokenExchangeFailed,
		service.ErrOAuthIdentityFetchFailed,
		service.ErrOAuthVerifiedEmailUnavailable,
		service.ErrOAuthEmailAlreadyRegistered,
		service.ErrOAuthFlowExpired,
		service.ErrOAuthIdentityConflict,
	} {
		if errors.Is(err, known) {
			return true
		}
	}
	return false
}
