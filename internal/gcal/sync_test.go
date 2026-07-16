package gcal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"focus-cli/internal/model"
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
	now := time.Now().Truncate(time.Second)
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
						Created:     now.Format(time.RFC3339),
						Updated:     now.Format(time.RFC3339),
						Start: &calendar.EventDateTime{
							DateTime: now.Format(time.RFC3339),
						},
						End: &calendar.EventDateTime{
							DateTime: now.Add(60 * time.Minute).Format(time.RFC3339),
						},
					},
					{
						Id:          "event-2",
						Summary:     "[4] Tugas Sesi",
						Description: "Explicit target sessions",
						Created:     now.Format(time.RFC3339),
						Updated:     now.Format(time.RFC3339),
						Start: &calendar.EventDateTime{
							DateTime: now.Format(time.RFC3339),
						},
						End: &calendar.EventDateTime{
							DateTime: now.Add(30 * time.Minute).Format(time.RFC3339),
						},
					},
					{
						Id:          "event-3",
						Summary:     "[50/10] Tugas Kustom",
						Description: "Custom durations",
						Created:     now.Format(time.RFC3339),
						Updated:     now.Format(time.RFC3339),
						Start: &calendar.EventDateTime{
							DateTime: now.Format(time.RFC3339),
						},
						End: &calendar.EventDateTime{
							DateTime: now.Add(120 * time.Minute).Format(time.RFC3339),
						},
					},
					{
						Id:          "event-4",
						Summary:     "(25/5) Tugas Parentheses",
						Description: "Parentheses durations",
						Created:     now.Format(time.RFC3339),
						Updated:     now.Format(time.RFC3339),
						Start: &calendar.EventDateTime{
							DateTime: now.Format(time.RFC3339),
						},
						End: &calendar.EventDateTime{
							DateTime: now.Add(60 * time.Minute).Format(time.RFC3339),
						},
					},
					{
						Id:          "event-5",
						Summary:     "[Done] [50/10] Tugas Selesai",
						Description: "Should be skipped",
					},
					{
						Id:          "event-6",
						Summary:     "Focus: Sesi Selesai",
						Description: "Should be skipped",
					},
					{
						Id:          "event-deleted",
						Summary:     "(25/5) Tugas Dihapus Lokal",
						Description: "Should be skipped",
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

	// Save task store with "event-deleted" in DeletedGCalEventIDs to mock local deletion
	ts := model.TaskStore{
		NextID:              1,
		Tasks:               []model.Task{},
		DeletedGCalEventIDs: []string{"event-deleted"},
	}
	_ = store.SaveTasks(ts)

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

	// We expect exactly 2 tasks to be imported (event-3, event-4)
	// event-1 is skipped because it has no custom duration format
	// event-2 is skipped because it has [N] format (no longer supported)
	// event-5 is skipped because it starts with [Done]
	// event-6 is skipped because it starts with Focus:
	// event-deleted is skipped because its ID is in DeletedGCalEventIDs
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks imported, got %d", len(tasks))
	}

	// Task 1: "[50/10] Tugas Kustom" -> FocusDuration=50, BreakDuration=10, TargetSessions: 120 / 60 = 2 sessions
	if tasks[0].Title != "Tugas Kustom" || tasks[0].TargetSessions != 2 || tasks[0].FocusDuration != 50 || tasks[0].BreakDuration != 10 {
		t.Errorf("unexpected task 0: %+v", tasks[0])
	}

	// Task 2: "(25/5) Tugas Parentheses" -> FocusDuration=25, BreakDuration=5, TargetSessions: 60 / 30 = 2 sessions
	if tasks[1].Title != "Tugas Parentheses" || tasks[1].TargetSessions != 2 || tasks[1].FocusDuration != 25 || tasks[1].BreakDuration != 5 {
		t.Errorf("unexpected task 1: %+v", tasks[1])
	}
}

func TestMarkEventAsDone(t *testing.T) {
	var eventFetched bool
	var eventUpdated bool
	var newSummary string

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

		// Mock GET event
		if r.Method == "GET" && r.URL.Path == "/calendars/focus-sessions-cal-id/events/mock-event-id" {
			eventFetched = true
			event := calendar.Event{
				Id:      "mock-event-id",
				Summary: "[50/10] Tugas Pertama",
			}
			json.NewEncoder(w).Encode(event)
			return
		}

		// Mock PUT update event
		if r.Method == "PUT" && r.URL.Path == "/calendars/focus-sessions-cal-id/events/mock-event-id" {
			eventUpdated = true
			var event calendar.Event
			json.NewDecoder(r.Body).Decode(&event)
			newSummary = event.Summary
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
	srv, err := calendar.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatalf("failed to create mock calendar service: %v", err)
	}

	err = client.MarkEventAsDoneWithService(ctx, srv, "mock-event-id", "Focus Sessions")
	if err != nil {
		t.Fatalf("MarkEventAsDoneWithService() error = %v", err)
	}

	if !eventFetched {
		t.Errorf("expected event fetch request to be made")
	}

	if !eventUpdated {
		t.Errorf("expected event update request to be made")
	}

	if newSummary != "[Done] [50/10] Tugas Pertama" {
		t.Errorf("expected summary '[Done] [50/10] Tugas Pertama', got '%s'", newSummary)
	}
}

