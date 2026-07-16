package gcal

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"focus-cli/internal/model"

	"google.golang.org/api/calendar/v3"
)

var (
	reDurSquare = regexp.MustCompile(`^\[(\d+)\/(\d+)\]\s*(.+)$`)
	reDurParen  = regexp.MustCompile(`^\((\d+)\/(\d+)\)\s*(.+)$`)
)

// parseGCalTitle mengurai judul event Google Calendar untuk mengekstrak target sesi dan durasi kustom
func parseGCalTitle(summary string) (cleanTitle string, targetSessions int, focusDuration int, breakDuration int, skip bool) {
	summary = strings.TrimSpace(summary)

	// Abaikan event riwayat fokus selesai atau event yang sudah bertanda done
	if strings.HasPrefix(summary, "Focus:") || strings.HasPrefix(summary, "Focus :") ||
		strings.HasPrefix(strings.ToLower(summary), "[done]") || strings.HasPrefix(strings.ToLower(summary), "[selesai]") ||
		strings.HasPrefix(strings.ToLower(summary), "(done)") || strings.HasPrefix(strings.ToLower(summary), "(selesai)") {
		return "", 0, 0, 0, true
	}

	// Pola 1: [Focus/Break] Judul (misal: "[50/10] Implementasi GCal")
	if matches := reDurSquare.FindStringSubmatch(summary); len(matches) == 4 {
		focus, _ := strconv.Atoi(matches[1])
		brk, _ := strconv.Atoi(matches[2])
		title := strings.TrimSpace(matches[3])
		return title, 0, focus, brk, false
	}

	// Pola 2: (Focus/Break) Judul (misal: "(50/10) Implementasi GCal")
	if matches := reDurParen.FindStringSubmatch(summary); len(matches) == 4 {
		focus, _ := strconv.Atoi(matches[1])
		brk, _ := strconv.Atoi(matches[2])
		title := strings.TrimSpace(matches[3])
		return title, 0, focus, brk, false
	}

	return "", 0, 0, 0, true
}

// SyncSessionEventWithService menyinkronkan sesi fokus ke Google Calendar menggunakan service yang diberikan
func (c *Client) SyncSessionEventWithService(ctx context.Context, srv *calendar.Service, title string, startTime, endTime time.Time, calendarName string) (string, error) {
	// 1. Temukan kalender berdasarkan nama
	var calendarID string
	
	listCall := srv.CalendarList.List()
	list, err := listCall.Do()
	if err == nil {
		for _, entry := range list.Items {
			if entry.Summary == calendarName {
				calendarID = entry.Id
				break
			}
		}
	}

	// 2. Jika kalender belum ada, buat kalender baru
	if calendarID == "" {
		newCal := &calendar.Calendar{
			Summary: calendarName,
		}
		createdCal, err := srv.Calendars.Insert(newCal).Do()
		if err != nil {
			// Fallback ke kalender utama jika gagal membuat kalender baru
			calendarID = "primary"
		} else {
			calendarID = createdCal.Id
		}
	}

	// 3. Masukkan event baru ke kalender
	event := &calendar.Event{
		Summary:     "Focus: " + title,
		Description: "Sesi fokus pomodoro yang diselesaikan menggunakan focus-cli",
		Start: &calendar.EventDateTime{
			DateTime: startTime.Format(time.RFC3339),
		},
		End: &calendar.EventDateTime{
			DateTime: endTime.Format(time.RFC3339),
		},
	}

	createdEvent, err := srv.Events.Insert(calendarID, event).Do()
	if err != nil {
		return "", fmt.Errorf("membuat event kalender gagal: %w", err)
	}

	// Simpan Calendar ID ke konfigurasi jika berhasil
	cfg, err := c.store.LoadConfig()
	if err == nil && cfg.GCalCalendarID != calendarID {
		cfg.GCalCalendarID = calendarID
		_ = c.store.SaveConfig(cfg)
	}

	return createdEvent.Id, nil
}

// SyncSessionEvent menyinkronkan sesi fokus ke Google Calendar
func (c *Client) SyncSessionEvent(ctx context.Context, title string, startTime, endTime time.Time, calendarName string) (string, error) {
	srv, err := c.GetCalendarService(ctx)
	if err != nil {
		return "", err
	}

	return c.SyncSessionEventWithService(ctx, srv, title, startTime, endTime, calendarName)
}

// ImportTasksWithService mengimpor tugas dari Google Calendar menggunakan service yang diberikan
func (c *Client) ImportTasksWithService(ctx context.Context, srv *calendar.Service, calendarName string) ([]model.Task, error) {
	// 1. Temukan kalender berdasarkan nama
	var calendarID string
	
	listCall := srv.CalendarList.List()
	list, err := listCall.Do()
	if err != nil {
		return nil, fmt.Errorf("mengambil list kalender gagal: %w", err)
	}

	for _, entry := range list.Items {
		if entry.Summary == calendarName {
			calendarID = entry.Id
			break
		}
	}

	// Jika kalender tidak ditemukan, kembalikan kosong (tidak ada task)
	if calendarID == "" {
		return nil, nil
	}

	// 2. Ambil event dari kalender
	// Membatasi pengambilan event dari 7 hari yang lalu sampai 7 hari ke depan agar tidak overload
	timeMin := time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339)
	eventsCall := srv.Events.List(calendarID).SingleEvents(true).TimeMin(timeMin)
	events, err := eventsCall.Do()
	if err != nil {
		return nil, fmt.Errorf("mengambil event kalender gagal: %w", err)
	}

	// Load task store to check DeletedGCalEventIDs
	var deletedIDs []string
	if ts, err := c.store.LoadTasks(); err == nil {
		deletedIDs = ts.DeletedGCalEventIDs
	}

	var tasks []model.Task
	for _, item := range events.Items {
		if item.Summary == "" {
			continue
		}

		// Saring tugas yang sudah pernah dihapus lokal
		isDeleted := false
		for _, delID := range deletedIDs {
			if delID == item.Id {
				isDeleted = true
				break
			}
		}
		if isDeleted {
			continue
		}

		// Urai template judul tugas
		cleanTitle, targetSessions, focusDur, breakDur, skip := parseGCalTitle(item.Summary)
		if skip {
			continue
		}

		// Hitung durasi event untuk kalkulasi target sesi
		var eventDuration time.Duration
		if item.Start != nil && item.End != nil {
			var start, end time.Time
			var errStart, errEnd error
			if item.Start.DateTime != "" {
				start, errStart = time.Parse(time.RFC3339, item.Start.DateTime)
			}
			if item.End.DateTime != "" {
				end, errEnd = time.Parse(time.RFC3339, item.End.DateTime)
			}
			if errStart == nil && errEnd == nil && !start.IsZero() && !end.IsZero() {
				eventDuration = end.Sub(start)
			}
		}

		// Jika TargetSessions belum diisi lewat template [N], kalkulasi dari durasi
		if targetSessions <= 0 {
			cfg, err := c.store.LoadConfig()
			focusMin := 25
			if err == nil && cfg.FocusMinutes > 0 {
				focusMin = cfg.FocusMinutes
			}

			if focusDur > 0 {
				cycleMin := focusDur + breakDur
				if cycleMin > 0 {
					targetSessions = int(eventDuration.Minutes() / float64(cycleMin))
				}
			} else {
				targetSessions = int(eventDuration.Minutes() / float64(focusMin))
			}

			if targetSessions < 1 {
				targetSessions = 1
			}
		}

		createdAt := time.Now()
		if item.Created != "" {
			if t, err := time.Parse(time.RFC3339, item.Created); err == nil {
				createdAt = t
			}
		}

		updatedAt := time.Now()
		if item.Updated != "" {
			if t, err := time.Parse(time.RFC3339, item.Updated); err == nil {
				updatedAt = t
			}
		}

		task := model.Task{
			Title:          cleanTitle,
			Description:    item.Description,
			Done:           false,
			TargetSessions: targetSessions,
			FocusDuration:  focusDur,
			BreakDuration:  breakDur,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			GCalEventID:    item.Id,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ImportTasks mengimpor tugas dari Google Calendar
func (c *Client) ImportTasks(ctx context.Context, calendarName string) ([]model.Task, error) {
	srv, err := c.GetCalendarService(ctx)
	if err != nil {
		return nil, err
	}
	return c.ImportTasksWithService(ctx, srv, calendarName)
}

// UpdateEventTitle memperbarui judul event Google Calendar secara asinkron
func (c *Client) UpdateEventTitle(ctx context.Context, eventID, newTitle string, calendarName string) error {
	srv, err := c.GetCalendarService(ctx)
	if err != nil {
		return fmt.Errorf("mendapatkan service GCal gagal: %w", err)
	}

	// 1. Temukan kalender berdasarkan nama
	var calendarID string
	listCall := srv.CalendarList.List()
	list, err := listCall.Do()
	if err == nil {
		for _, entry := range list.Items {
			if entry.Summary == calendarName {
				calendarID = entry.Id
				break
			}
		}
	}
	if calendarID == "" {
		calendarID = "primary"
	}

	// 2. Patch judul event
	event := &calendar.Event{
		Summary: newTitle,
	}
	_, err = srv.Events.Patch(calendarID, eventID, event).Do()
	if err != nil {
		return fmt.Errorf("update event GCal gagal: %w", err)
	}

	return nil
}

// MarkEventAsDone menandai event Google Calendar selesai dengan menambahkan awalan [Done]
func (c *Client) MarkEventAsDone(ctx context.Context, eventID string, calendarName string) error {
	srv, err := c.GetCalendarService(ctx)
	if err != nil {
		return fmt.Errorf("mendapatkan service GCal gagal: %w", err)
	}
	return c.MarkEventAsDoneWithService(ctx, srv, eventID, calendarName)
}

// MarkEventAsDoneWithService menandai event Google Calendar selesai dengan menambahkan awalan [Done] menggunakan service yang diberikan
func (c *Client) MarkEventAsDoneWithService(ctx context.Context, srv *calendar.Service, eventID string, calendarName string) error {
	// 1. Temukan kalender berdasarkan nama
	var calendarID string
	listCall := srv.CalendarList.List()
	list, err := listCall.Do()
	if err == nil {
		for _, entry := range list.Items {
			if entry.Summary == calendarName {
				calendarID = entry.Id
				break
			}
		}
	}
	if calendarID == "" {
		calendarID = "primary"
	}

	// 2. Dapatkan event saat ini
	event, err := srv.Events.Get(calendarID, eventID).Do()
	if err != nil {
		return fmt.Errorf("mengambil event GCal gagal: %w", err)
	}

	// 3. Tambahkan awalan [Done] jika belum ada
	if !strings.HasPrefix(strings.ToLower(event.Summary), "[done]") && !strings.HasPrefix(strings.ToLower(event.Summary), "[selesai]") {
		event.Summary = "[Done] " + event.Summary
		_, err = srv.Events.Update(calendarID, eventID, event).Do()
		if err != nil {
			return fmt.Errorf("memperbarui event GCal gagal: %w", err)
		}
	}

	return nil
}

