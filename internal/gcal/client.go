package gcal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"focus-cli/internal/storage"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// IsInvalidGrantError memeriksa apakah error disebabkan oleh token OAuth2 yang kadaluwarsa atau dicabut
func IsInvalidGrantError(err error) bool {
	if err == nil {
		return false
	}
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		if retrieveErr.ErrorCode == "invalid_grant" || strings.Contains(strings.ToLower(retrieveErr.ErrorDescription), "expired or revoked") {
			return true
		}
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid_grant") || strings.Contains(msg, "expired or revoked")
}

// FormatGCalError mengonversi error GCal mentah menjadi pesan yang ramah pengguna
func FormatGCalError(err error) error {
	if err == nil {
		return nil
	}
	if IsInvalidGrantError(err) {
		return errors.New("Token Google Calendar telah kadaluwarsa atau dicabut. Silakan jalankan 'focus gcal login' untuk otentikasi ulang akun Anda.")
	}
	return err
}

type Client struct {
	store       *storage.Store
	oauthConfig *oauth2.Config
}

// NewClient inisialisasi GCal client berdasarkan kredensial yang tersimpan
func NewClient(store *storage.Store) (*Client, error) {
	credsData, err := store.ReadGCalCredentials()
	if err != nil {
		return nil, fmt.Errorf("membaca kredensial GCal gagal: %w (pastikan file gcal_credentials.json ada di ~/.config/focus-cli/)", err)
	}

	// Scope yang dibutuhkan: membaca/menulis kalender dan mengelola event
	config, err := google.ConfigFromJSON(credsData, calendar.CalendarScope, calendar.CalendarEventsScope)
	if err != nil {
		return nil, fmt.Errorf("parsing kredensial GCal gagal: %w", err)
	}

	// Redirect URL harus diarahkan ke local server callback
	config.RedirectURL = "http://localhost:8080/callback"

	return &Client{
		store:       store,
		oauthConfig: config,
	}, nil
}

type persistingTokenSource struct {
	src   oauth2.TokenSource
	store *storage.Store
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := p.src.Token()
	if err != nil {
		return nil, err
	}
	if tok.Valid() {
		if data, jsonErr := json.Marshal(tok); jsonErr == nil {
			_ = p.store.SaveGCalToken(data)
		}
	}
	return tok, nil
}

// GetHTTPClient mengembalikan http.Client yang terotentikasi dan otomatis me-refresh token jika kadaluwarsa serta menyimpannya ke disk
func (c *Client) GetHTTPClient(ctx context.Context) (*http.Client, error) {
	tokenData, err := c.store.LoadGCalToken()
	if err != nil {
		return nil, fmt.Errorf("token GCal tidak ditemukan atau tidak valid, silakan lakukan login: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(tokenData, &token); err != nil {
		return nil, fmt.Errorf("parsing token GCal gagal: %w", err)
	}

	ts := c.oauthConfig.TokenSource(ctx, &token)
	pts := &persistingTokenSource{
		src:   ts,
		store: c.store,
	}
	reuseTs := oauth2.ReuseTokenSource(&token, pts)

	return oauth2.NewClient(ctx, reuseTs), nil
}

// GetCalendarService mengembalikan Google Calendar Service
func (c *Client) GetCalendarService(ctx context.Context) (*calendar.Service, error) {
	httpClient, err := c.GetHTTPClient(ctx)
	if err != nil {
		return nil, FormatGCalError(err)
	}

	srv, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, FormatGCalError(fmt.Errorf("membuat service GCal gagal: %w", err))
	}

	return srv, nil
}

// Login menjalankan flow OAuth2 otentikasi di browser dan callback local server
func (c *Client) Login(ctx context.Context) error {
	// Buat listener lokal di port 8080
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		return fmt.Errorf("tidak dapat mendengarkan di port 8080 (pastikan port tidak digunakan aplikasi lain): %w", err)
	}
	defer listener.Close()

	// Channel untuk menerima code atau error dari handler callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Buat server HTTP lokal untuk callback
	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
	}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- errors.New("callback tidak menyertakan authorization code")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "<html><body><h2>Otentikasi Gagal!</h2><p>Tidak ada kode otentikasi yang diterima.</p></body></html>")
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body><h2>Otentikasi Sukses!</h2><p>Otentikasi Google Calendar berhasil. Anda dapat menutup tab ini dan kembali ke terminal.</p></body></html>")
		codeChan <- code
	})

	// Jalankan server di goroutine background
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()
	defer func() {
		// Shutdown server secara aman setelah selesai
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Generate authorization URL
	authURL := c.oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("Membuka browser untuk otentikasi Google Calendar...")
	fmt.Println("Jika browser tidak terbuka secara otomatis, silakan buka URL berikut:")
	fmt.Printf("\n%s\n\n", authURL)

	// Coba buka browser secara otomatis sesuai OS
	_ = openBrowser(authURL)

	// Tunggu code, error, atau timeout/cancel dari context
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return fmt.Errorf("kesalahan pada server callback lokal: %w", err)
	case code := <-codeChan:
		// Tukarkan authorization code dengan token
		token, err := c.oauthConfig.Exchange(ctx, code)
		if err != nil {
			return fmt.Errorf("penukaran token gagal: %w", err)
		}

		// Simpan token
		tokenData, err := json.Marshal(token)
		if err != nil {
			return fmt.Errorf("serialisasi token gagal: %w", err)
		}

		if err := c.store.SaveGCalToken(tokenData); err != nil {
			return fmt.Errorf("menyimpan token gagal: %w", err)
		}

		fmt.Println("Otentikasi berhasil! Token disimpan.")
		return nil
	}
}

// openBrowser utilitas pembuka browser default sesuai dengan sistem operasi
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // linux, freebsd, netbsd, openbsd
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}
