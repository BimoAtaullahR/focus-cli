package gcal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"focus-cli/internal/storage"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func TestSyncSessionEvent(t *testing.T) {
	// Setup mock Google Calendar API server
	var calendarCreated bool
	var eventCreated bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("Mock server request: %s %s", r.Method, r.URL.Path)
		
		// Mock GET calendar list
		if r.Method == "GET" && r.URL.Path == "/users/me/calendarList" {
			list := calendar.CalendarList{
				Items: []*calendar.CalendarListEntry{
					{
						Id:      "primary",
						Summary: "Primary Calendar",
					},
				},
			}
			json.NewEncoder(w).Encode(list)
			return
		}

		// Mock POST insert calendar
		if r.Method == "POST" && r.URL.Path == "/calendars" {
			calendarCreated = true
			cal := calendar.Calendar{
				Id:      "focus-sessions-cal-id",
				Summary: "Focus Sessions",
			}
			json.NewEncoder(w).Encode(cal)
			return
		}

		// Mock POST insert event
		if r.Method == "POST" && r.URL.Path == "/calendars/focus-sessions-cal-id/events" {
			eventCreated = true
			event := calendar.Event{
				Id:      "mock-event-id",
				Summary: "Focus: Belajar Go",
			}
			json.NewEncoder(w).Encode(event)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Initialize Storage
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Initialize Client with custom mock service
	client := &Client{
		store: store,
		oauthConfig: &oauth2.Config{
			ClientID:     "mock-client",
			ClientSecret: "mock-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  server.URL + "/auth",
				TokenURL: server.URL + "/token",
			},
		},
	}

	ctx := context.Background()
	// Create calendar service pointing to our mock server
	srv, err := calendar.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatalf("failed to create mock calendar service: %v", err)
	}

	// Run SyncSessionEvent using the service
	start := time.Now().Add(-25 * time.Minute)
	end := time.Now()
	
	eventID, err := client.SyncSessionEventWithService(ctx, srv, "Belajar Go", start, end, "Focus Sessions")
	if err != nil {
		t.Fatalf("SyncSessionEventWithService() error = %v", err)
	}

	if eventID != "mock-event-id" {
		t.Errorf("expected eventID 'mock-event-id', got '%s'", eventID)
	}

	if !calendarCreated {
		t.Errorf("expected calendar creation request to be made")
	}

	if !eventCreated {
		t.Errorf("expected event creation request to be made")
	}
}

func TestImportTasks(t *testing.T) {
	// Setup mock Google Calendar API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Mock GET calendar list
		if r.Method == "GET" && r.URL.Path == "/users/me/calendarList" {
			list := calendar.CalendarList{
				Items: []*calendar.CalendarListEntry{
					{
						Id:      "focus-sessions-cal-id",
						Summary: "Focus Sessions",
					},
				},
			}
			json.NewEncoder(w).Encode(list)
			return
		}

		// Mock GET list events
		if r.Method == "GET" && r.URL.Path == "/calendars/focus-sessions-cal-id/events" {
			events := calendar.Events{
				Items: []*calendar.Event{
					{
						Id:          "event-1",
						Summary:     "Tugas Pertama",
						Description: "Deskripsi Tugas Pertama",
						Created:     time.Now().Format(time.RFC3339),
						Updated:     time.Now().Format(time.RFC3339),
					},
					{
						Id:          "event-2",
						Summary:     "Tugas Kedua",
						Description: "",
						Created:     time.Now().Format(time.RFC3339),
						Updated:     time.Now().Format(time.RFC3339),
					},
				},
			}
			json.NewEncoder(w).Encode(events)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Initialize Storage
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	store, err := storage.NewStore()
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Initialize Client with custom mock service
	client := &Client{
		store: store,
		oauthConfig: &oauth2.Config{
			ClientID:     "mock-client",
			ClientSecret: "mock-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  server.URL + "/auth",
				TokenURL: server.URL + "/token",
			},
		},
	}

	ctx := context.Background()
	srv, err := calendar.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatalf("failed to create mock calendar service: %v", err)
	}

	tasks, err := client.ImportTasksWithService(ctx, srv, "Focus Sessions")
	if err != nil {
		t.Fatalf("ImportTasksWithService() error = %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks imported, got %d", len(tasks))
	}

	if tasks[0].Title != "Tugas Pertama" || tasks[0].GCalEventID != "event-1" || tasks[0].Description != "Deskripsi Tugas Pertama" {
		t.Errorf("unexpected task 0: %+v", tasks[0])
	}

	if tasks[1].Title != "Tugas Kedua" || tasks[1].GCalEventID != "event-2" || tasks[1].Description != "" {
		t.Errorf("unexpected task 1: %+v", tasks[1])
	}
}

