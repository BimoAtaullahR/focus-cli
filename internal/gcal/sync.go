package gcal

import (
	"context"
	"fmt"
	"time"

	"focus-cli/internal/model"

	"google.golang.org/api/calendar/v3"
)

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

	var tasks []model.Task
	for _, item := range events.Items {
		if item.Summary == "" {
			continue
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
			Title:          item.Summary,
			Description:    item.Description,
			Done:           false,
			TargetSessions: 1,
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

