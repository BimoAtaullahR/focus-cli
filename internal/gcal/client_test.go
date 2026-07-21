package gcal

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"focus-cli/internal/storage"

	"golang.org/x/oauth2"
)

func TestNewClientNoCredentials(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)

	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Without credentials file, NewClient should fail or return a specific error
	_, err = NewClient(store)
	if err == nil {
		t.Errorf("expected error when credentials file is missing")
	}
}

func TestNewClientWithCredentials(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)

	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Write mock credentials
	credsJSON := `{
		"installed": {
			"client_id": "mock-client-id",
			"client_secret": "mock-client-secret",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"redirect_uris": ["http://localhost:8080/callback"]
		}
	}`
	credsPath := filepath.Join(cfgHome, "focus-cli", "gcal_credentials.json")
	err = os.WriteFile(credsPath, []byte(credsJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write credentials: %v", err)
	}

	client, err := NewClient(store)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("expected client to not be nil")
	}

	if client.oauthConfig.ClientID != "mock-client-id" {
		t.Errorf("expected ClientID to be 'mock-client-id', got '%s'", client.oauthConfig.ClientID)
	}
}

func TestGetHTTPClient(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)

	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	credsJSON := `{
		"installed": {
			"client_id": "mock-client-id",
			"client_secret": "mock-client-secret",
			"auth_uri": "https://accounts.google.com/o/oauth2/auth",
			"token_uri": "https://oauth2.googleapis.com/token",
			"redirect_uris": ["http://localhost:8080/callback"]
		}
	}`
	credsPath := filepath.Join(cfgHome, "focus-cli", "gcal_credentials.json")
	err = os.WriteFile(credsPath, []byte(credsJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write credentials: %v", err)
	}

	client, err := NewClient(store)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// 1. Without token, GetHTTPClient should fail
	ctx := context.Background()
	_, err = client.GetHTTPClient(ctx)
	if err == nil {
		t.Errorf("expected error when token is missing")
	}

	// 2. With valid token, GetHTTPClient should succeed
	tokenJSON := `{"access_token":"mock-access-token","token_type":"Bearer","refresh_token":"mock-refresh-token","expiry":"2026-07-13T21:18:14Z"}`
	err = store.SaveGCalToken([]byte(tokenJSON))
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	httpClient, err := client.GetHTTPClient(ctx)
	if err != nil {
		t.Fatalf("GetHTTPClient() error = %v", err)
	}
	if httpClient == nil {
		t.Errorf("expected httpClient to not be nil")
	}
}

type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, m.err
}

func TestPersistingTokenSource(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)

	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	newToken := &oauth2.Token{
		AccessToken:  "refreshed-access-token",
		TokenType:    "Bearer",
		RefreshToken: "refreshed-refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}

	mockSrc := &mockTokenSource{token: newToken}
	pts := &persistingTokenSource{
		src:   mockSrc,
		store: store,
	}

	gotToken, err := pts.Token()
	if err != nil {
		t.Fatalf("pts.Token() error = %v", err)
	}
	if gotToken.AccessToken != "refreshed-access-token" {
		t.Errorf("expected access token 'refreshed-access-token', got '%s'", gotToken.AccessToken)
	}

	// Verify that the new token was saved to store
	savedBytes, err := store.LoadGCalToken()
	if err != nil {
		t.Fatalf("failed to load saved token: %v", err)
	}

	var savedToken oauth2.Token
	if err := json.Unmarshal(savedBytes, &savedToken); err != nil {
		t.Fatalf("failed to unmarshal saved token: %v", err)
	}

	if savedToken.AccessToken != "refreshed-access-token" {
		t.Errorf("expected saved token AccessToken to be 'refreshed-access-token', got '%s'", savedToken.AccessToken)
	}
}


