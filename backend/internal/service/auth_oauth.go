package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/oauth"
	"social-network/backend/internal/repo"
)

const (
	oauthStateKind        = "oauth_state"
	oauthRegistrationKind = "oauth_registration"
	oauthStateTTL         = 10 * time.Minute
	oauthRegistrationTTL  = 30 * time.Minute
)

type OAuthCallbackInput struct {
	Provider      string
	State         string
	Code          string
	ProviderError string
}

type OAuthCallbackResult struct {
	Auth              *AuthResult
	RegistrationToken string
	Next              string
}

type OAuthRegistrationPreview struct {
	Provider           string `json:"provider"`
	Email              string `json:"email"`
	GitHubUsername     string `json:"github_username"`
	SuggestedFirstName string `json:"suggested_first_name"`
	SuggestedLastName  string `json:"suggested_last_name"`
	SuggestedNickname  string `json:"suggested_nickname"`
}

type CompleteOAuthRegistrationInput struct {
	FirstName   string
	LastName    string
	DateOfBirth string
	Nickname    *string
	AboutMe     *string
	Avatar      *MediaUpload
}

type oauthStatePayload struct {
	Next string `json:"next"`
}

type oauthRegistrationPayload struct {
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	EmailVerified  bool   `json:"email_verified"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	Next           string `json:"next"`
}

func (s *AuthService) OAuthProviders() []oauth.ProviderInfo {
	if s == nil || s.oauthRegistry == nil {
		return []oauth.ProviderInfo{}
	}
	return s.oauthRegistry.Available()
}

func (s *AuthService) StartOAuth(ctx context.Context, providerName, next string) (string, error) {
	if !s.oauthConfigured() {
		return "", ErrOAuthProviderUnavailable
	}
	provider, ok := s.oauthRegistry.Get(providerName)
	if !ok {
		return "", ErrOAuthProviderUnavailable
	}
	payload, err := json.Marshal(oauthStatePayload{Next: normalizeOAuthNext(next)})
	if err != nil {
		return "", err
	}
	now := s.clock.Now()
	token, err := s.createOAuthFlow(ctx, &domain.AuthFlow{
		Kind:      oauthStateKind,
		Provider:  provider.Name(),
		Payload:   string(payload),
		CreatedAt: now,
		ExpiresAt: now.Add(oauthStateTTL),
	})
	if err != nil {
		return "", err
	}
	return provider.AuthorizationURL(token), nil
}

func (s *AuthService) HandleOAuthCallback(
	ctx context.Context,
	input OAuthCallbackInput,
) (*OAuthCallbackResult, error) {
	if s == nil || s.transactions == nil || s.clock == nil {
		return nil, ErrOAuthProviderUnavailable
	}
	input.State = strings.TrimSpace(input.State)
	if input.State == "" {
		return nil, ErrOAuthStateInvalid
	}

	var stateFlow *domain.AuthFlow
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		var takeErr error
		stateFlow, takeErr = repositories.AuthFlows().TakeByToken(ctx, input.State)
		return takeErr
	})
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrOAuthStateInvalid
	}
	if err != nil {
		return nil, err
	}
	if stateFlow.Kind != oauthStateKind ||
		stateFlow.Provider != strings.TrimSpace(input.Provider) ||
		!s.clock.Now().Before(stateFlow.ExpiresAt) {
		return nil, ErrOAuthStateInvalid
	}
	var statePayload oauthStatePayload
	if err := json.Unmarshal([]byte(stateFlow.Payload), &statePayload); err != nil {
		return nil, ErrOAuthStateInvalid
	}
	next := normalizeOAuthNext(statePayload.Next)

	if strings.TrimSpace(input.ProviderError) != "" {
		return nil, ErrOAuthProviderError
	}
	input.Code = strings.TrimSpace(input.Code)
	if input.Code == "" {
		return nil, ErrOAuthCodeMissing
	}
	if s.oauthRegistry == nil {
		return nil, ErrOAuthProviderUnavailable
	}
	provider, ok := s.oauthRegistry.Get(stateFlow.Provider)
	if !ok {
		return nil, ErrOAuthProviderUnavailable
	}
	accessToken, err := provider.Exchange(ctx, input.Code)
	if err != nil {
		return nil, ErrOAuthTokenExchangeFailed
	}
	providerIdentity, err := provider.Identity(ctx, accessToken)
	if errors.Is(err, oauth.ErrVerifiedEmailUnavailable) {
		return nil, ErrOAuthVerifiedEmailUnavailable
	}
	if err != nil {
		return nil, ErrOAuthIdentityFetchFailed
	}
	if providerIdentity == nil ||
		strings.TrimSpace(providerIdentity.ProviderUserID) == "" ||
		!providerIdentity.EmailVerified ||
		!validEmail(strings.TrimSpace(providerIdentity.Email)) {
		return nil, ErrOAuthIdentityFetchFailed
	}

	storedIdentity, err := s.authIdentities.GetByProviderUserID(
		ctx,
		provider.Name(),
		providerIdentity.ProviderUserID,
	)
	if err == nil {
		auth, loginErr := s.loginExistingOAuthIdentity(
			ctx,
			storedIdentity,
			provider.Name(),
			providerIdentity,
		)
		if loginErr != nil {
			return nil, loginErr
		}
		return &OAuthCallbackResult{Auth: auth, Next: next}, nil
	}
	if !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}
	if _, err := s.users.GetByEmail(ctx, providerIdentity.Email); err == nil {
		return nil, ErrOAuthEmailAlreadyRegistered
	} else if !errors.Is(err, repo.ErrNotFound) {
		return nil, err
	}

	registrationPayload, err := json.Marshal(oauthRegistrationPayload{
		ProviderUserID: strings.TrimSpace(providerIdentity.ProviderUserID),
		Email:          strings.TrimSpace(providerIdentity.Email),
		EmailVerified:  providerIdentity.EmailVerified,
		Username:       strings.TrimSpace(providerIdentity.Username),
		DisplayName:    strings.TrimSpace(providerIdentity.DisplayName),
		Next:           next,
	})
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	registrationToken, err := s.createOAuthFlow(ctx, &domain.AuthFlow{
		Kind:      oauthRegistrationKind,
		Provider:  provider.Name(),
		Payload:   string(registrationPayload),
		CreatedAt: now,
		ExpiresAt: now.Add(oauthRegistrationTTL),
	})
	if err != nil {
		return nil, err
	}
	return &OAuthCallbackResult{
		RegistrationToken: registrationToken,
		Next:              next,
	}, nil
}

func (s *AuthService) OAuthRegistrationPreview(
	ctx context.Context,
	token string,
) (*OAuthRegistrationPreview, error) {
	if !s.oauthConfigured() {
		return nil, ErrOAuthProviderUnavailable
	}
	flow, err := s.authFlows.GetByToken(ctx, strings.TrimSpace(token))
	if errors.Is(err, repo.ErrNotFound) {
		return nil, ErrOAuthFlowExpired
	}
	if err != nil {
		return nil, err
	}
	payload, err := s.validateRegistrationFlow(flow)
	if err != nil {
		return nil, err
	}
	firstName, lastName := suggestedOAuthNames(payload.DisplayName)
	return &OAuthRegistrationPreview{
		Provider:           flow.Provider,
		Email:              payload.Email,
		GitHubUsername:     payload.Username,
		SuggestedFirstName: firstName,
		SuggestedLastName:  lastName,
		SuggestedNickname:  payload.Username,
	}, nil
}

func (s *AuthService) CompleteOAuthRegistration(
	ctx context.Context,
	token string,
	input CompleteOAuthRegistrationInput,
) (*AuthResult, error) {
	if !s.oauthConfigured() || s.sessions == nil || s.passwords == nil || s.avatars == nil {
		return nil, ErrOAuthProviderUnavailable
	}
	token = strings.TrimSpace(token)
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.Nickname = optionalTrimmed(input.Nickname)
	input.AboutMe = optionalTrimmed(input.AboutMe)
	if token == "" ||
		input.FirstName == "" ||
		input.LastName == "" ||
		!domain.ValidDateOfBirth(input.DateOfBirth) {
		return nil, ErrInvalidInput
	}

	generatedPassword, err := randomOAuthPassword()
	if err != nil {
		return nil, err
	}
	passwordHash, err := s.passwords.Hash(generatedPassword)
	if err != nil {
		return nil, err
	}

	var stagedAvatar *StagedMedia
	if input.Avatar != nil {
		stagedAvatar, err = s.avatars.Stage(*input.Avatar)
		if err != nil {
			return nil, err
		}
		defer stagedAvatar.Discard()
	}

	now := s.clock.Now()
	var result *AuthResult
	err = s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		flow, err := repositories.AuthFlows().TakeByToken(ctx, token)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrOAuthFlowExpired
		}
		if err != nil {
			return err
		}
		payload, err := s.validateRegistrationFlow(flow)
		if err != nil {
			return err
		}
		if _, err := repositories.AuthIdentities().GetByProviderUserID(
			ctx,
			flow.Provider,
			payload.ProviderUserID,
		); err == nil {
			return ErrOAuthIdentityConflict
		} else if !errors.Is(err, repo.ErrNotFound) {
			return err
		}
		if _, err := repositories.Users().GetByEmail(ctx, payload.Email); err == nil {
			return ErrOAuthEmailAlreadyRegistered
		} else if !errors.Is(err, repo.ErrNotFound) {
			return err
		}

		user := &domain.User{
			Email:        payload.Email,
			PasswordHash: passwordHash,
			FirstName:    input.FirstName,
			LastName:     input.LastName,
			DateOfBirth:  input.DateOfBirth,
			Nickname:     input.Nickname,
			AboutMe:      input.AboutMe,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.createUserRecords(ctx, repositories, user, stagedAvatar, now); err != nil {
			if errors.Is(err, errUserEmailConflict) {
				return ErrOAuthEmailAlreadyRegistered
			}
			return err
		}
		identity := &domain.AuthIdentity{
			UserID:                user.ID,
			Provider:              flow.Provider,
			ProviderUserID:        payload.ProviderUserID,
			ProviderEmail:         payload.Email,
			ProviderEmailVerified: true,
			ProviderUsername:      payload.Username,
			ProviderDisplayName:   payload.DisplayName,
			LinkedAt:              now,
			LastLoginAt:           now,
		}
		identityID, err := repositories.AuthIdentities().Create(ctx, identity)
		if errors.Is(err, repo.ErrConflict) {
			return ErrOAuthIdentityConflict
		}
		if err != nil {
			return err
		}
		identity.ID = identityID
		session, err := s.createSessionRecord(ctx, repositories, user.ID)
		if err != nil {
			return err
		}
		if stagedAvatar != nil {
			if err := stagedAvatar.Finalize(); err != nil {
				return err
			}
		}
		result = &AuthResult{User: user, Session: session}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if stagedAvatar != nil {
		stagedAvatar.Keep()
	}
	return result, nil
}

func (s *AuthService) loginExistingOAuthIdentity(
	ctx context.Context,
	storedIdentity *domain.AuthIdentity,
	providerName string,
	providerIdentity *oauth.Identity,
) (*AuthResult, error) {
	var result *AuthResult
	err := s.transactions.WithinTransaction(ctx, func(repositories repo.TransactionRepositories) error {
		identity, err := repositories.AuthIdentities().GetByProviderUserID(
			ctx,
			providerName,
			providerIdentity.ProviderUserID,
		)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrOAuthIdentityConflict
		}
		if err != nil {
			return err
		}
		if storedIdentity == nil || identity.ID != storedIdentity.ID || identity.UserID != storedIdentity.UserID {
			return ErrOAuthIdentityConflict
		}
		user, err := repositories.Users().GetByID(ctx, identity.UserID)
		if errors.Is(err, repo.ErrNotFound) {
			return ErrOAuthIdentityConflict
		}
		if err != nil {
			return err
		}
		identity.ProviderEmail = strings.TrimSpace(providerIdentity.Email)
		identity.ProviderEmailVerified = providerIdentity.EmailVerified
		identity.ProviderUsername = strings.TrimSpace(providerIdentity.Username)
		identity.ProviderDisplayName = strings.TrimSpace(providerIdentity.DisplayName)
		identity.LastLoginAt = s.clock.Now()
		if err := repositories.AuthIdentities().UpdateMetadata(ctx, identity); err != nil {
			return err
		}
		session, err := s.createSessionRecord(ctx, repositories, user.ID)
		if err != nil {
			return err
		}
		result = &AuthResult{User: user, Session: session}
		return nil
	})
	return result, err
}

func (s *AuthService) createOAuthFlow(
	ctx context.Context,
	flow *domain.AuthFlow,
) (string, error) {
	for attempt := 0; attempt < 3; attempt++ {
		token, err := s.ids.New()
		if err != nil {
			return "", err
		}
		flow.Token = token
		if err := s.authFlows.Create(ctx, flow); err == nil {
			return token, nil
		} else if !errors.Is(err, repo.ErrConflict) {
			return "", err
		}
	}
	return "", repo.ErrConflict
}

func (s *AuthService) validateRegistrationFlow(
	flow *domain.AuthFlow,
) (*oauthRegistrationPayload, error) {
	if flow == nil ||
		flow.Kind != oauthRegistrationKind ||
		!s.clock.Now().Before(flow.ExpiresAt) {
		return nil, ErrOAuthFlowExpired
	}
	if s.oauthRegistry == nil {
		return nil, ErrOAuthProviderUnavailable
	}
	if _, ok := s.oauthRegistry.Get(flow.Provider); !ok {
		return nil, ErrOAuthProviderUnavailable
	}
	payload := &oauthRegistrationPayload{}
	if err := json.Unmarshal([]byte(flow.Payload), payload); err != nil {
		return nil, ErrOAuthFlowExpired
	}
	payload.ProviderUserID = strings.TrimSpace(payload.ProviderUserID)
	payload.Email = strings.TrimSpace(payload.Email)
	payload.Username = strings.TrimSpace(payload.Username)
	payload.DisplayName = strings.TrimSpace(payload.DisplayName)
	payload.Next = normalizeOAuthNext(payload.Next)
	if payload.ProviderUserID == "" || !payload.EmailVerified || !validEmail(payload.Email) {
		return nil, ErrOAuthFlowExpired
	}
	return payload, nil
}

func (s *AuthService) oauthConfigured() bool {
	return s != nil &&
		s.oauthRegistry != nil &&
		s.authIdentities != nil &&
		s.authFlows != nil &&
		s.ids != nil &&
		s.transactions != nil &&
		s.users != nil &&
		s.clock != nil
}

func normalizeOAuthNext(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	if !strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "//") ||
		strings.Contains(value, "\\") ||
		strings.ContainsAny(value, "\r\n\x00") {
		return "/"
	}
	return value
}

func suggestedOAuthNames(displayName string) (string, string) {
	parts := strings.Fields(displayName)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func randomOAuthPassword() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
