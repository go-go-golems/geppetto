// Package oauth provides profile-agnostic OAuth 2.0 protocol operations for
// hosts that use Geppetto credentials. It owns neither profile configuration,
// persistent storage, nor browser callback handling.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-go-golems/geppetto/pkg/steps/ai/credentials"
	oauth2 "golang.org/x/oauth2"
)

// Config identifies one OAuth client and its standard authorization and token
// endpoints. Hosts own where this configuration comes from and must not expose
// ClientSecret in diagnostics.
type Config struct {
	AuthorizationURL string
	TokenURL         string
	ClientID         string
	ClientSecret     string
	RedirectURL      string
	Scopes           []string
}

// RefreshTokenPolicy controls handling when a successful refresh response does
// not include a replacement refresh token.
type RefreshTokenPolicy int

const (
	// PreservePreviousRefreshToken retains the prior token. This is the standard
	// OAuth behavior for providers that omit refresh_token when it did not rotate.
	PreservePreviousRefreshToken RefreshTokenPolicy = iota
	// RequireReplacementRefreshToken rejects an incomplete refresh response. Use
	// it for providers whose refresh tokens rotate on every successful refresh.
	RequireReplacementRefreshToken
)

// Option configures a Client.
type Option func(*Client)

// WithRefreshTokenPolicy chooses how Refresh handles an omitted refresh token.
func WithRefreshTokenPolicy(policy RefreshTokenPolicy) Option {
	return func(client *Client) {
		client.refreshTokenPolicy = policy
	}
}

// Client executes standard OAuth protocol requests using explicit config. It is
// safe to share between goroutines; it retains no tokens or mutable request
// state.
type Client struct {
	config             oauth2.Config
	refreshTokenPolicy RefreshTokenPolicy
}

// PKCE contains a verifier that must be retained only until the authorization
// code is exchanged and its matching S256 challenge for the authorization URL.
type PKCE struct {
	Verifier  string
	Challenge string
}

// NewClient validates config and constructs a reusable protocol client.
func NewClient(config Config, options ...Option) (*Client, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	client := &Client{
		config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  config.AuthorizationURL,
				TokenURL: config.TokenURL,
			},
			RedirectURL: config.RedirectURL,
			Scopes:      append([]string(nil), config.Scopes...),
		},
		refreshTokenPolicy: PreservePreviousRefreshToken,
	}
	for _, option := range options {
		if option != nil {
			option(client)
		}
	}
	if client.refreshTokenPolicy != PreservePreviousRefreshToken && client.refreshTokenPolicy != RequireReplacementRefreshToken {
		return nil, errors.New("invalid OAuth refresh token policy")
	}
	return client, nil
}

// NewPKCE creates a cryptographically random RFC 7636 verifier and its S256
// challenge. The caller stores the verifier only for its pending browser flow.
func NewPKCE() PKCE {
	verifier := oauth2.GenerateVerifier()
	return PKCE{
		Verifier:  verifier,
		Challenge: oauth2.S256ChallengeFromVerifier(verifier),
	}
}

// AuthorizationURL builds an authorization request with PKCE S256 and offline
// access requested. Hosts must generate and validate state and must own the
// browser/loopback callback lifecycle.
func (c *Client) AuthorizationURL(state string, pkce PKCE, options ...oauth2.AuthCodeOption) (string, error) {
	if c == nil {
		return "", errors.New("nil OAuth client")
	}
	if strings.TrimSpace(state) == "" {
		return "", errors.New("OAuth authorization state is required")
	}
	if strings.TrimSpace(pkce.Verifier) == "" || strings.TrimSpace(pkce.Challenge) == "" {
		return "", errors.New("OAuth PKCE verifier and challenge are required")
	}
	if oauth2.S256ChallengeFromVerifier(pkce.Verifier) != pkce.Challenge {
		return "", errors.New("OAuth PKCE challenge does not match verifier")
	}
	allOptions := make([]oauth2.AuthCodeOption, 0, len(options)+2)
	allOptions = append(allOptions, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(pkce.Verifier))
	allOptions = append(allOptions, options...)
	return c.config.AuthCodeURL(state, allOptions...), nil
}

// ExchangeAuthorizationCode exchanges one authorization code with its PKCE
// verifier. It returns a normalized credential and hides token endpoint bodies
// and token values from errors.
func (c *Client) ExchangeAuthorizationCode(ctx context.Context, code string, pkce PKCE) (credentials.Credential, error) {
	if c == nil {
		return credentials.Credential{}, errors.New("nil OAuth client")
	}
	if strings.TrimSpace(code) == "" {
		return credentials.Credential{}, errors.New("OAuth authorization code is required")
	}
	if strings.TrimSpace(pkce.Verifier) == "" {
		return credentials.Credential{}, errors.New("OAuth PKCE verifier is required")
	}
	token, err := c.config.Exchange(ctx, code, oauth2.VerifierOption(pkce.Verifier))
	if err != nil {
		return credentials.Credential{}, errors.New("OAuth authorization code exchange failed")
	}
	return credentialFromToken(token, "", RequireReplacementRefreshToken)
}

// Refresh forces a refresh-token grant even when previous.ExpiresAt is in the
// future. This is required after a provider has rejected an otherwise usable
// bearer token with HTTP 401. It never returns endpoint response details.
func (c *Client) Refresh(ctx context.Context, previous credentials.Credential) (credentials.Credential, error) {
	if c == nil {
		return credentials.Credential{}, errors.New("nil OAuth client")
	}
	if strings.TrimSpace(previous.RefreshToken) == "" {
		return credentials.Credential{}, errors.New("OAuth refresh token is required")
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {previous.RefreshToken},
	}
	if c.config.ClientSecret == "" {
		form.Set("client_id", c.config.ClientID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Endpoint.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return credentials.Credential{}, errors.New("OAuth refresh grant request creation failed")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if c.config.ClientSecret != "" {
		req.SetBasicAuth(c.config.ClientID, c.config.ClientSecret)
	}

	response, err := oauthHTTPClient(ctx).Do(req)
	if err != nil {
		return credentials.Credential{}, errors.New("OAuth refresh grant failed")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return credentials.Credential{}, errors.New("OAuth refresh grant failed")
	}

	var payload tokenPayload
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return credentials.Credential{}, errors.New("OAuth refresh token response is invalid")
	}
	token, err := payload.token(time.Now())
	if err != nil {
		return credentials.Credential{}, err
	}
	return credentialFromToken(token, previous.RefreshToken, c.refreshTokenPolicy)
}

type tokenPayload struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    json.Number `json:"expires_in"`
}

func (p tokenPayload) token(now time.Time) (*oauth2.Token, error) {
	if strings.TrimSpace(p.AccessToken) == "" {
		return nil, errors.New("OAuth token response has no access token")
	}
	token := &oauth2.Token{AccessToken: p.AccessToken, RefreshToken: p.RefreshToken}
	if p.ExpiresIn == "" {
		return token, nil
	}
	seconds, err := strconv.ParseFloat(string(p.ExpiresIn), 64)
	if err != nil || seconds < 0 {
		return nil, errors.New("OAuth token response has invalid expiry")
	}
	token.Expiry = now.Add(time.Duration(seconds * float64(time.Second)))
	return token, nil
}

func oauthHTTPClient(ctx context.Context) *http.Client {
	if client, ok := ctx.Value(oauth2.HTTPClient).(*http.Client); ok && client != nil {
		return client
	}
	return http.DefaultClient
}

func credentialFromToken(token *oauth2.Token, previousRefreshToken string, policy RefreshTokenPolicy) (credentials.Credential, error) {
	if token == nil || strings.TrimSpace(token.AccessToken) == "" {
		return credentials.Credential{}, errors.New("OAuth token response has no access token")
	}
	refreshToken := strings.TrimSpace(token.RefreshToken)
	if refreshToken == "" {
		switch policy {
		case PreservePreviousRefreshToken:
			refreshToken = previousRefreshToken
		case RequireReplacementRefreshToken:
			return credentials.Credential{}, errors.New("OAuth token response has no refresh token")
		default:
			return credentials.Credential{}, errors.New("invalid OAuth refresh token policy")
		}
	}
	return credentials.Credential{
		AccessToken:  token.AccessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    token.Expiry,
	}, nil
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.ClientID) == "" {
		return errors.New("OAuth client ID is required")
	}
	if err := validateAbsoluteURL(config.AuthorizationURL, "authorization"); err != nil {
		return err
	}
	if err := validateAbsoluteURL(config.TokenURL, "token"); err != nil {
		return err
	}
	if err := validateAbsoluteURL(config.RedirectURL, "redirect"); err != nil {
		return err
	}
	return nil
}

func validateAbsoluteURL(raw, kind string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("OAuth " + kind + " URL must be absolute HTTP(S)")
	}
	return nil
}
