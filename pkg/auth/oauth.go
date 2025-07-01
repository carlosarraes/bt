package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// OAuthAuth implements OAuth 2.0 authentication flow for Bitbucket
type OAuthAuth struct {
	config      *Config
	storage     CredentialStorage
	credentials *StoredCredentials
	httpClient  *http.Client
}

// OAuthTokenResponse represents the OAuth token response from Bitbucket
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// NewOAuthAuth creates a new OAuth authenticator
func NewOAuthAuth(config *Config, storage CredentialStorage) (Authenticator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage cannot be nil")
	}
	
	if config.OAuthClientID == "" {
		return nil, fmt.Errorf("OAuth client ID is required")
	}
	
	auth := &OAuthAuth{
		config:  config,
		storage: storage,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
	
	// Try to load existing credentials
	var creds StoredCredentials
	if storage.Exists("auth") {
		if err := storage.Retrieve("auth", &creds); err == nil && creds.Method == AuthMethodOAuth {
			auth.credentials = &creds
		}
	}
	
	return auth, nil
}

func (a *OAuthAuth) Authenticate(ctx context.Context) error {
	// Check if we have valid stored credentials first
	if a.credentials != nil && a.credentials.AccessToken != "" {
		// Check if token is still valid
		if valid, _ := a.IsValid(ctx); valid {
			return nil
		}
		
		// Try to refresh the token
		if a.credentials.RefreshToken != "" {
			if err := a.Refresh(ctx); err == nil {
				return nil
			}
		}
	}
	
	// No valid credentials, need to do full OAuth flow
	return a.performOAuthFlow(ctx)
}

func (a *OAuthAuth) SetHTTPHeaders(req *http.Request) error {
	if a.credentials == nil || a.credentials.AccessToken == "" {
		return fmt.Errorf("no OAuth access token available")
	}
	
	req.Header.Set("Authorization", "Bearer "+a.credentials.AccessToken)
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	return nil
}

func (a *OAuthAuth) IsValid(ctx context.Context) (bool, error) {
	if a.credentials == nil || a.credentials.AccessToken == "" {
		return false, nil
	}
	
	// Check if token has expired
	if !a.credentials.TokenExpiry.IsZero() && time.Now().After(a.credentials.TokenExpiry) {
		return false, nil
	}
	
	// Validate by making a test API call
	err := a.validateToken(ctx, a.credentials.AccessToken)
	return err == nil, nil
}

func (a *OAuthAuth) Refresh(ctx context.Context) error {
	if a.credentials == nil || a.credentials.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}
	
	// Prepare refresh token request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", a.credentials.RefreshToken)
	
	req, err := http.NewRequestWithContext(ctx, "POST", "https://bitbucket.org/site/oauth2/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	// Set basic auth with client ID (for public clients, client secret is empty)
	req.SetBasicAuth(a.config.OAuthClientID, "")
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status %d", resp.StatusCode)
	}
	
	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// Update credentials
	a.credentials.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		a.credentials.RefreshToken = tokenResp.RefreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		a.credentials.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	a.credentials.UpdatedAt = time.Now()
	
	// Store updated credentials
	if err := a.storage.Store("auth", a.credentials); err != nil {
		return fmt.Errorf("failed to store refreshed token: %w", err)
	}
	
	return nil
}

func (a *OAuthAuth) GetUser(ctx context.Context) (*User, error) {
	if a.credentials == nil || a.credentials.AccessToken == "" {
		return nil, fmt.Errorf("no OAuth access token available")
	}
	
	return a.fetchUser(ctx, a.credentials.AccessToken)
}

func (a *OAuthAuth) Clear() error {
	if err := a.storage.Delete("auth"); err != nil {
		return fmt.Errorf("failed to clear stored OAuth credentials: %w", err)
	}
	
	a.credentials = nil
	return nil
}

// performOAuthFlow executes the complete OAuth 2.0 authorization code flow
func (a *OAuthAuth) performOAuthFlow(ctx context.Context) error {
	// Generate state parameter for security
	state, err := a.generateState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}
	
	// Build authorization URL
	authURL := a.buildAuthorizationURL(state)
	
	fmt.Printf("Opening browser for Bitbucket authentication...\n")
	fmt.Printf("If the browser doesn't open automatically, visit: %s\n", authURL)
	
	// Open browser
	if err := a.openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Printf("Please manually open the URL above.\n")
	}
	
	// Start local server to receive the callback
	code, err := a.receiveAuthorizationCode(state)
	if err != nil {
		return fmt.Errorf("failed to receive authorization code: %w", err)
	}
	
	// Exchange authorization code for access token
	if err := a.exchangeCodeForToken(ctx, code); err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}
	
	return nil
}

// generateState generates a random state parameter for OAuth security
func (a *OAuthAuth) generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// buildAuthorizationURL constructs the OAuth authorization URL
func (a *OAuthAuth) buildAuthorizationURL(state string) string {
	params := url.Values{}
	params.Set("client_id", a.config.OAuthClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", "http://localhost:8080/callback")
	params.Set("state", state)
	params.Set("scope", "account repositories")
	
	return "https://bitbucket.org/site/oauth2/authorize?" + params.Encode()
}

// openBrowser opens the authorization URL in the default browser
func (a *OAuthAuth) openBrowser(url string) error {
	var cmd string
	var args []string
	
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// receiveAuthorizationCode starts a local server to receive the OAuth callback
func (a *OAuthAuth) receiveAuthorizationCode(expectedState string) (string, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Check state parameter
		if state := r.URL.Query().Get("state"); state != expectedState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("invalid state parameter")
			return
		}
		
		// Get authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			errCh <- fmt.Errorf("no authorization code received")
			return
		}
		
		// Send success response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Authentication successful!</h1><p>You can close this window and return to the terminal.</p></body></html>`)
		
		codeCh <- code
	})
	
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	
	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("failed to start callback server: %w", err)
		}
	}()
	
	// Wait for code or error
	select {
	case code := <-codeCh:
		server.Close()
		return code, nil
	case err := <-errCh:
		server.Close()
		return "", err
	case <-time.After(5 * time.Minute):
		server.Close()
		return "", fmt.Errorf("timeout waiting for authorization")
	}
}

// exchangeCodeForToken exchanges the authorization code for an access token
func (a *OAuthAuth) exchangeCodeForToken(ctx context.Context, code string) error {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", "http://localhost:8080/callback")
	
	req, err := http.NewRequestWithContext(ctx, "POST", "https://bitbucket.org/site/oauth2/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	// Set basic auth with client ID (for public clients, client secret is empty)
	req.SetBasicAuth(a.config.OAuthClientID, "")
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}
	
	var tokenResp OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}
	
	// Store credentials
	a.credentials = &StoredCredentials{
		Method:       AuthMethodOAuth,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	if tokenResp.ExpiresIn > 0 {
		a.credentials.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	
	if err := a.storage.Store("auth", a.credentials); err != nil {
		return fmt.Errorf("failed to store OAuth credentials: %w", err)
	}
	
	return nil
}

// validateToken makes a test API call to validate the token
func (a *OAuthAuth) validateToken(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()
	
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid or expired OAuth token")
	case http.StatusForbidden:
		return fmt.Errorf("OAuth token does not have required permissions")
	case http.StatusOK:
		return nil
	default:
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}
}

// fetchUser retrieves user information from the Bitbucket API
func (a *OAuthAuth) fetchUser(ctx context.Context, token string) (*User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", a.config.BaseURL+"/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "bt-cli/1.0")
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}
	
	var apiResponse struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		AccountID   string `json:"account_id"`
		UUID        string `json:"uuid"`
		Email       string `json:"email"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %w", err)
	}
	
	return &User{
		Username:    apiResponse.Username,
		DisplayName: apiResponse.DisplayName,
		AccountID:   apiResponse.AccountID,
		UUID:        strings.Trim(apiResponse.UUID, "{}"),
		Email:       apiResponse.Email,
	}, nil
}