package gcal

import (
	"context"
	"fmt"
	"time"

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
