package service

import "errors"

var (
	ErrInvalidInput                  = errors.New("invalid input")
	ErrUnauthorized                  = errors.New("unauthorized")
	ErrForbidden                     = errors.New("forbidden")
	ErrNotFound                      = errors.New("not found")
	ErrConflict                      = errors.New("conflict")
	ErrEmailTaken                    = errors.New("email already exists")
	ErrInvalidCredentials            = errors.New("invalid credentials")
	ErrInvalidMediaType              = errors.New("invalid media type")
	ErrMediaTooBig                   = errors.New("media is too big")
	ErrOAuthProviderUnavailable      = errors.New("oauth_provider_unavailable")
	ErrOAuthProviderError            = errors.New("oauth_provider_error")
	ErrOAuthStateInvalid             = errors.New("oauth_state_invalid")
	ErrOAuthCodeMissing              = errors.New("oauth_code_missing")
	ErrOAuthTokenExchangeFailed      = errors.New("oauth_token_exchange_failed")
	ErrOAuthIdentityFetchFailed      = errors.New("oauth_identity_fetch_failed")
	ErrOAuthVerifiedEmailUnavailable = errors.New("oauth_verified_email_unavailable")
	ErrOAuthEmailAlreadyRegistered   = errors.New("oauth_email_already_registered")
	ErrOAuthFlowExpired              = errors.New("oauth_flow_expired")
	ErrOAuthIdentityConflict         = errors.New("oauth_identity_conflict")
)
