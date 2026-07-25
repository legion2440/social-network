package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxProviderResponseBytes int64 = 1 << 20

type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GitHubProvider struct {
	client       *http.Client
	clientID     string
	clientSecret string
	redirectURL  string
	authorizeURL string
	tokenURL     string
	userURL      string
	emailsURL    string
}

func NewGitHubProvider(config GitHubConfig, client *http.Client) (*GitHubProvider, error) {
	config.ClientID = strings.TrimSpace(config.ClientID)
	config.ClientSecret = strings.TrimSpace(config.ClientSecret)
	config.RedirectURL = strings.TrimSpace(config.RedirectURL)
	if config.ClientID == "" || config.ClientSecret == "" || config.RedirectURL == "" {
		return nil, errors.New("complete GitHub OAuth configuration is required")
	}
	redirect, err := url.Parse(config.RedirectURL)
	if err != nil ||
		(redirect.Scheme != "http" && redirect.Scheme != "https") ||
		redirect.Host == "" {
		return nil, errors.New("GitHub OAuth redirect URL must be an absolute HTTP(S) URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &GitHubProvider{
		client:       client,
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
		redirectURL:  config.RedirectURL,
		authorizeURL: "https://github.com/login/oauth/authorize",
		tokenURL:     "https://github.com/login/oauth/access_token",
		userURL:      "https://api.github.com/user",
		emailsURL:    "https://api.github.com/user/emails",
	}, nil
}

func (p *GitHubProvider) Name() string {
	return ProviderGitHub
}

func (p *GitHubProvider) Label() string {
	return "GitHub"
}

func (p *GitHubProvider) AuthorizationURL(state string) string {
	query := url.Values{}
	query.Set("client_id", p.clientID)
	query.Set("redirect_uri", p.redirectURL)
	query.Set("scope", "read:user user:email")
	query.Set("state", state)
	return p.authorizeURL + "?" + query.Encode()
}

func (p *GitHubProvider) Exchange(ctx context.Context, code string) (string, error) {
	if p == nil || p.client == nil {
		return "", errors.New("GitHub OAuth provider is not configured")
	}
	values := url.Values{}
	values.Set("client_id", p.clientID)
	values.Set("client_secret", p.clientSecret)
	values.Set("code", strings.TrimSpace(code))
	values.Set("redirect_uri", p.redirectURL)
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.tokenURL,
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := p.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("GitHub token endpoint returned status %d", response.StatusCode)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := decodeProviderJSON(response.Body, &payload); err != nil {
		return "", err
	}
	if payload.Error != "" || strings.TrimSpace(payload.AccessToken) == "" {
		return "", errors.New("GitHub token response did not contain an access token")
	}
	return strings.TrimSpace(payload.AccessToken), nil
}

func (p *GitHubProvider) Identity(ctx context.Context, accessToken string) (*Identity, error) {
	if p == nil || p.client == nil {
		return nil, errors.New("GitHub OAuth provider is not configured")
	}
	var user struct {
		ID    json.Number `json:"id"`
		Login string      `json:"login"`
		Name  string      `json:"name"`
	}
	if err := p.getJSON(ctx, p.userURL, accessToken, &user); err != nil {
		return nil, err
	}
	providerUserID := strings.TrimSpace(user.ID.String())
	if providerUserID == "" {
		return nil, errors.New("GitHub user response did not contain an id")
	}
	if _, err := strconv.ParseInt(providerUserID, 10, 64); err != nil {
		return nil, errors.New("GitHub user id is invalid")
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := p.getJSON(ctx, p.emailsURL, accessToken, &emails); err != nil {
		return nil, err
	}
	verifiedEmail := ""
	for _, email := range emails {
		value := strings.TrimSpace(email.Email)
		if email.Primary && email.Verified && value != "" {
			verifiedEmail = value
			break
		}
	}
	if verifiedEmail == "" {
		for _, email := range emails {
			value := strings.TrimSpace(email.Email)
			if email.Verified && value != "" {
				verifiedEmail = value
				break
			}
		}
	}
	if verifiedEmail == "" {
		return nil, ErrVerifiedEmailUnavailable
	}

	return &Identity{
		ProviderUserID: providerUserID,
		Email:          verifiedEmail,
		EmailVerified:  true,
		Username:       strings.TrimSpace(user.Login),
		DisplayName:    strings.TrimSpace(user.Name),
	}, nil
}

func (p *GitHubProvider) getJSON(
	ctx context.Context,
	endpoint string,
	accessToken string,
	target any,
) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	response, err := p.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("GitHub API returned status %d", response.StatusCode)
	}
	return decodeProviderJSON(response.Body, target)
}

func decodeProviderJSON(reader io.Reader, target any) error {
	limited := io.LimitReader(reader, maxProviderResponseBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return err
	}
	if int64(len(payload)) > maxProviderResponseBytes {
		return errors.New("OAuth provider response is too large")
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("OAuth provider response contains multiple JSON values")
		}
		return err
	}
	return nil
}

var ErrVerifiedEmailUnavailable = errors.New("verified email unavailable")
