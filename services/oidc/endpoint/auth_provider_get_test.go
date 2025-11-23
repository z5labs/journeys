// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package endpoint

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/z5labs/humus"
)

func TestValidateProvider(t *testing.T) {
	oauth := OAuthConfig{
		Google:   ProviderConfig{ClientID: "google-client"},
		Facebook: ProviderConfig{ClientID: "fb-client"},
		Apple:    ProviderConfig{ClientID: "apple-client"},
	}
	
	h := &authProviderHandler{oauth: oauth}
	
	tests := []struct {
		name      string
		provider  string
		wantError bool
	}{
		{
			name:      "valid google provider",
			provider:  "google",
			wantError: false,
		},
		{
			name:      "valid facebook provider",
			provider:  "facebook",
			wantError: false,
		},
		{
			name:      "valid apple provider",
			provider:  "apple",
			wantError: false,
		},
		{
			name:      "valid google provider uppercase",
			provider:  "GOOGLE",
			wantError: false,
		},
		{
			name:      "valid facebook provider mixed case",
			provider:  "FaceBook",
			wantError: false,
		},
		{
			name:      "invalid provider github",
			provider:  "github",
			wantError: true,
		},
		{
			name:      "invalid provider empty",
			provider:  "",
			wantError: true,
		},
		{
			name:      "invalid provider microsoft",
			provider:  "microsoft",
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.validateProvider(tt.provider)
			if (err != nil) != tt.wantError {
				t.Errorf("validateProvider() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestGenerateState(t *testing.T) {
	state1, err := generateState()
	if err != nil {
		t.Fatalf("generateState() error = %v", err)
	}
	
	if state1 == "" {
		t.Error("generateState() returned empty string")
	}
	
	state2, err := generateState()
	if err != nil {
		t.Fatalf("generateState() second call error = %v", err)
	}
	
	if state1 == state2 {
		t.Error("generateState() returned same state twice, should be random")
	}
	
	if strings.Contains(state1, "+") || strings.Contains(state1, "/") || strings.Contains(state1, "=") {
		t.Errorf("generateState() returned non-URL-safe string: %s", state1)
	}
}

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		wantError   bool
	}{
		{
			name:        "valid https URL",
			redirectURI: "https://app.example.com/callback",
			wantError:   false,
		},
		{
			name:        "valid http URL",
			redirectURI: "http://localhost:3000/callback",
			wantError:   false,
		},
		{
			name:        "valid custom scheme",
			redirectURI: "myapp://callback",
			wantError:   false,
		},
		{
			name:        "empty redirect URI",
			redirectURI: "",
			wantError:   true,
		},
		{
			name:        "missing scheme",
			redirectURI: "app.example.com/callback",
			wantError:   true,
		},
		{
			name:        "missing host",
			redirectURI: "https:///callback",
			wantError:   true,
		},
		{
			name:        "invalid URL format",
			redirectURI: "ht!tp://invalid",
			wantError:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRedirectURI(tt.redirectURI)
			if (err != nil) != tt.wantError {
				t.Errorf("validateRedirectURI() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestBuildAuthorizationURL(t *testing.T) {
	tests := []struct {
		name        string
		cfg         ProviderConfig
		redirectURI string
		state       string
		wantErr     bool
		checkURL    func(t *testing.T, authURL string)
	}{
		{
			name: "google provider",
			cfg: ProviderConfig{
				ClientID: "google-client-id",
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				Scopes:   []string{"openid", "profile", "email"},
			},
			redirectURI: "https://app.example.com/callback",
			state:       "test-state-123",
			wantErr:     false,
			checkURL: func(t *testing.T, authURL string) {
				parsedURL, err := url.Parse(authURL)
				if err != nil {
					t.Fatalf("failed to parse URL: %v", err)
				}
				
				if parsedURL.Scheme != "https" || parsedURL.Host != "accounts.google.com" {
					t.Errorf("unexpected base URL: %s", authURL)
				}
				
				query := parsedURL.Query()
				if query.Get("client_id") != "google-client-id" {
					t.Errorf("client_id = %s, want google-client-id", query.Get("client_id"))
				}
				if query.Get("redirect_uri") != "https://app.example.com/callback" {
					t.Errorf("redirect_uri = %s, want https://app.example.com/callback", query.Get("redirect_uri"))
				}
				if query.Get("response_type") != "code" {
					t.Errorf("response_type = %s, want code", query.Get("response_type"))
				}
				if query.Get("scope") != "openid profile email" {
					t.Errorf("scope = %s, want 'openid profile email'", query.Get("scope"))
				}
				if query.Get("state") != "test-state-123" {
					t.Errorf("state = %s, want test-state-123", query.Get("state"))
				}
			},
		},
		{
			name: "facebook provider",
			cfg: ProviderConfig{
				ClientID: "fb-app-id",
				AuthURL:  "https://www.facebook.com/v12.0/dialog/oauth",
				Scopes:   []string{"public_profile", "email"},
			},
			redirectURI: "https://app.example.com/callback",
			state:       "fb-state",
			wantErr:     false,
			checkURL: func(t *testing.T, authURL string) {
				parsedURL, err := url.Parse(authURL)
				if err != nil {
					t.Fatalf("failed to parse URL: %v", err)
				}
				
				query := parsedURL.Query()
				if query.Get("scope") != "public_profile email" {
					t.Errorf("scope = %s, want 'public_profile email'", query.Get("scope"))
				}
			},
		},
		{
			name: "apple provider",
			cfg: ProviderConfig{
				ClientID: "com.example.app",
				AuthURL:  "https://appleid.apple.com/auth/authorize",
				Scopes:   []string{"name", "email"},
			},
			redirectURI: "https://app.example.com/callback",
			state:       "apple-state",
			wantErr:     false,
			checkURL: func(t *testing.T, authURL string) {
				parsedURL, err := url.Parse(authURL)
				if err != nil {
					t.Fatalf("failed to parse URL: %v", err)
				}
				
				query := parsedURL.Query()
				if query.Get("scope") != "name email" {
					t.Errorf("scope = %s, want 'name email'", query.Get("scope"))
				}
			},
		},
		{
			name: "missing auth URL",
			cfg: ProviderConfig{
				ClientID: "client-id",
				AuthURL:  "",
				Scopes:   []string{"openid"},
			},
			redirectURI: "https://app.example.com/callback",
			state:       "state",
			wantErr:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authURL, err := buildAuthorizationURL(tt.cfg, tt.redirectURI, tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildAuthorizationURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.checkURL != nil {
				tt.checkURL(t, authURL)
			}
		})
	}
}

func TestAuthProviderHandler_handle(t *testing.T) {
	oauth := OAuthConfig{
		Google: ProviderConfig{
			ClientID:     "google-client-id",
			ClientSecret: "google-secret",
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes:       []string{"openid", "profile", "email"},
		},
		Facebook: ProviderConfig{
			ClientID:     "fb-app-id",
			ClientSecret: "fb-secret",
			AuthURL:      "https://www.facebook.com/v12.0/dialog/oauth",
			TokenURL:     "https://graph.facebook.com/v12.0/oauth/access_token",
			Scopes:       []string{"public_profile", "email"},
		},
		Apple: ProviderConfig{
			ClientID:     "com.example.app",
			ClientSecret: "apple-secret",
			AuthURL:      "https://appleid.apple.com/auth/authorize",
			TokenURL:     "https://appleid.apple.com/auth/token",
			Scopes:       []string{"name", "email"},
		},
	}
	
	tests := []struct {
		name        string
		provider    string
		redirectURI string
		state       string
		wantErr     bool
		checkResult func(t *testing.T, resp *AuthProviderResponse)
	}{
		{
			name:        "successful google request with state",
			provider:    "google",
			redirectURI: "https://app.example.com/callback",
			state:       "user-provided-state",
			wantErr:     false,
			checkResult: func(t *testing.T, resp *AuthProviderResponse) {
				if resp.Provider != "google" {
					t.Errorf("provider = %s, want google", resp.Provider)
				}
				if resp.ClientID != "google-client-id" {
					t.Errorf("clientID = %s, want google-client-id", resp.ClientID)
				}
				if resp.ResponseType != "code" {
					t.Errorf("responseType = %s, want code", resp.ResponseType)
				}
				if resp.State != "user-provided-state" {
					t.Errorf("state = %s, want user-provided-state", resp.State)
				}
				if len(resp.Scopes) != 3 {
					t.Errorf("scopes length = %d, want 3", len(resp.Scopes))
				}
				if !strings.Contains(resp.AuthorizationURL, "accounts.google.com") {
					t.Errorf("authorizationURL doesn't contain google domain: %s", resp.AuthorizationURL)
				}
			},
		},
		{
			name:        "successful facebook request without state",
			provider:    "facebook",
			redirectURI: "https://app.example.com/callback",
			state:       "",
			wantErr:     false,
			checkResult: func(t *testing.T, resp *AuthProviderResponse) {
				if resp.Provider != "facebook" {
					t.Errorf("provider = %s, want facebook", resp.Provider)
				}
				if resp.State == "" {
					t.Error("state should be generated when not provided")
				}
				if len(resp.Scopes) != 2 {
					t.Errorf("scopes length = %d, want 2", len(resp.Scopes))
				}
			},
		},
		{
			name:        "successful apple request",
			provider:    "apple",
			redirectURI: "https://app.example.com/callback",
			state:       "apple-state",
			wantErr:     false,
			checkResult: func(t *testing.T, resp *AuthProviderResponse) {
				if resp.Provider != "apple" {
					t.Errorf("provider = %s, want apple", resp.Provider)
				}
				if resp.ClientID != "com.example.app" {
					t.Errorf("clientID = %s, want com.example.app", resp.ClientID)
				}
			},
		},
		{
			name:        "missing redirect_uri",
			provider:    "google",
			redirectURI: "",
			state:       "some-state",
			wantErr:     true,
		},
		{
			name:        "invalid redirect_uri",
			provider:    "google",
			redirectURI: "not-a-valid-url",
			state:       "",
			wantErr:     true,
		},
		{
			name:        "invalid provider",
			provider:    "github",
			redirectURI: "https://app.example.com/callback",
			state:       "",
			wantErr:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &authProviderHandler{
				oauth: oauth,
				log:   humus.Logger("test"),
			}
			
			ctx := context.Background()
			
			resp, err := h.handle(ctx, tt.provider, tt.redirectURI, tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, resp)
			}
		})
	}
}
