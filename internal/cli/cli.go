package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"focus-cli/internal/model"
	"focus-cli/internal/notify"
	"focus-cli/internal/pomodoro"
	"focus-cli/internal/storage"
	"focus-cli/internal/tui"
	"focus-cli/internal/gcal"
)

func Run(args []string) error {
	if len(args) == 0 {
		if !isInteractive() {
			printHelp()
			return nil
		}
		store, err := storage.NewStore()
		if err != nil {
			return err
		}
		return tui.Run(store)
	}

	store, err := storage.NewStore()
	if err != nil {
		return err
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp()
		return nil
	case "ui":
		if !isInteractive() {
			return errors.New("interactive UI requires a terminal")
		}
		store, err := storage.NewStore()
		if err != nil {
			return err
		}
		return tui.Run(store)
	case "task", "tasks":
		return runTask(store, args[1:])
	case "a":
		return runTask(store, append([]string{"add"}, args[1:]...))
	case "ls":
		return runTask(store, []string{"list"})
	case "e":
		return runTask(store, append([]string{"edit"}, args[1:]...))
	case "d", "rm":
		return runTask(store, append([]string{"delete"}, args[1:]...))
	case "done":
		return runTask(store, append([]string{"done"}, args[1:]...))
	case "config":
		return runConfig(store, args[1:])
	case "cfg":
		return runConfig(store, []string{"show"})
	case "set":
		return runQuickSet(store, args[1:])
	case "run":
		return runPomodoro(store, args[1:])
	case "focus", "f":
		return runQuickFocus(store, args[1:])
	case "break", "b":
		return runQuickBreak(store, args[1:])
	case "timer":
		return runSingleTimer(args[1:])
	case "t":
		return runQuickTimer(args[1:])
	case "stats":
		return runStats(store)
	case "gcal":
		return runGCal(store, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func printHelp() {
	fmt.Println("focus-cli - Pomodoro CLI")
	fmt.Println("")
	fmt.Println("Interactive mode:")
	fmt.Println("  focus")
	fmt.Println("  focus ui")
	fmt.Println("  ctrl+j / ctrl+k in dashboard to reorder tasks")
	fmt.Println("  p, x, n, q while a cycle is running")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  task add <title> [--target N] [--desc text]")
	fmt.Println("  task list")
	fmt.Println("  task edit <id> [--title text] [--target N] [--completed N] [--desc text]")
	fmt.Println("  task delete <id>")
	fmt.Println("  task done <id> [true|false]")
	fmt.Println("  config show")
	fmt.Println("  config set [--focus N] [--short N] [--long N] [--long-every N] [--theme name]")
	fmt.Println("             [--notifications on|off] [--notify-warning-before N]")
	fmt.Println("             [--notify-desktop on|off] [--notify-sound on|off]")
	fmt.Println("             [--notify-log on|off] [--notify-log-path path]")
	fmt.Println("  config notifications show")
	fmt.Println("  config notifications set [--enabled on|off] [--warning-before N]")
	fmt.Println("                           [--desktop on|off] [--sound on|off]")
	fmt.Println("                           [--log on|off] [--log-path path]")
	fmt.Println("  config key show")
	fmt.Println("  config key set <action> <key>")
	fmt.Println("  run [--task ID] [--sessions N] [--notifications on|off]")
	fmt.Println("      [--notify-warning-before N] [--notify-desktop on|off]")
	fmt.Println("      [--notify-sound on|off] [--notify-log on|off] [--notify-log-path path]")
	fmt.Println("  timer [--minutes N] [--label text]")
	fmt.Println("  stats")
	fmt.Println("  gcal <login|logout|status|sync>")
	fmt.Println("")
	fmt.Println("Shortcuts:")
	fmt.Println("  focus [N] [--task ID]")
	fmt.Println("  break [short|long|N]")
	fmt.Println("  t [N] [label words]")
	fmt.Println("  a <title> [--target N] [--desc text]")
	fmt.Println("  ls")
	fmt.Println("  e <id> [--title text] [--target N] [--desc text]")
	fmt.Println("  d <id>")
	fmt.Println("  done <id> [true|false]")
	fmt.Println("  cfg")
	fmt.Println("  set <focus|short|long|long-every> <N>")
}

func isInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func runQuickSet(store *storage.Store, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: set <focus|short|long|long-every> <value>")
	}
	value, err := strconv.Atoi(args[1])
	if err != nil || value < 1 {
		return errors.New("value must be an integer >= 1")
	}
	flagName := ""
	switch args[0] {
	case "focus":
		flagName = "--focus"
	case "short":
		flagName = "--short"
	case "long":
		flagName = "--long"
	case "long-every":
		flagName = "--long-every"
	default:
		return errors.New("key must be one of: focus, short, long, long-every")
	}
	return runConfig(store, []string{"set", flagName, strconv.Itoa(value)})
}

func runQuickFocus(store *storage.Store, args []string) error {
	if len(args) == 0 {
		return runPomodoro(store, []string{"--sessions", "1"})
	}

	if strings.HasPrefix(args[0], "-") {
		return runPomodoro(store, args)
	}

	sessions, err := strconv.Atoi(args[0])
	if err != nil || sessions < 1 {
		return errors.New("usage: focus [sessions>=1] [--task ID]")
	}
	runArgs := []string{"--sessions", strconv.Itoa(sessions)}
	runArgs = append(runArgs, args[1:]...)
	return runPomodoro(store, runArgs)
}

func runQuickBreak(store *storage.Store, args []string) error {
	cfg, err := store.LoadConfig()
	if err != nil {
		return err
	}
	if len(args) == 0 || args[0] == "short" {
		return runSingleTimer([]string{"--minutes", strconv.Itoa(cfg.ShortBreakMinutes), "--label", "SHORT BREAK"})
	}
	if args[0] == "long" {
		return runSingleTimer([]string{"--minutes", strconv.Itoa(cfg.LongBreakMinutes), "--label", "LONG BREAK"})
	}
	minutes, err := strconv.Atoi(args[0])
	if err != nil || minutes < 1 {
		return errors.New("usage: break [short|long|minutes>=1]")
	}
	return runSingleTimer([]string{"--minutes", strconv.Itoa(minutes), "--label", "BREAK"})
}

func runQuickTimer(args []string) error {
	if len(args) == 0 {
		return runSingleTimer(nil)
	}
	minutes, err := strconv.Atoi(args[0])
	if err != nil || minutes < 1 {
		return errors.New("usage: t [minutes>=1] [label words]")
	}
	runArgs := []string{"--minutes", strconv.Itoa(minutes)}
	if len(args) > 1 {
		runArgs = append(runArgs, "--label", strings.Join(args[1:], " "))
	}
	return runSingleTimer(runArgs)
}

func runTask(store *storage.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("task subcommand required")
	}

	ts, err := store.LoadTasks()
	if err != nil {
		return err
	}

	now := time.Now()
	switch args[0] {
	case "add":
		if len(args) < 2 {
			return errors.New("usage: task add <title> [--target N] [--desc text]")
		}
		fs := flag.NewFlagSet("task add", flag.ContinueOnError)
		target := fs.Int("target", 1, "target sessions")
		desc := fs.String("desc", "", "task description")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		if *target < 1 {
			return errors.New("target must be >= 1")
		}
		task := model.Task{
			ID:             ts.NextID,
			Title:          args[1],
			Description:    *desc,
			TargetSessions: *target,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		ts.NextID++
		ts.Tasks = append(ts.Tasks, task)
		if err := store.SaveTasks(ts); err != nil {
			return err
		}
		fmt.Printf("task created: #%d %s\n", task.ID, task.Title)
		return nil
	case "list":
		if len(ts.Tasks) == 0 {
			fmt.Println("no tasks yet")
			return nil
		}
		for _, t := range ts.Tasks {
			status := "todo"
			if t.Done {
				status = "done"
			}
			fmt.Printf("#%d [%s] %s | progress %d/%d\n", t.ID, status, t.Title, t.CompletedPomodoros, t.TargetSessions)
		}
		return nil
	case "edit":
		if len(args) < 2 {
			return errors.New("usage: task edit <id> [--title text] [--target N] [--completed N] [--desc text]")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		fs := flag.NewFlagSet("task edit", flag.ContinueOnError)
		title := fs.String("title", "", "new title")
		target := fs.Int("target", 0, "new target")
		completed := fs.Int("completed", -1, "new completed pomodoro sessions")
		desc := fs.String("desc", "", "new description")
		if err := fs.Parse(args[2:]); err != nil {
			return err
		}
		idx := -1
		for i := range ts.Tasks {
			if ts.Tasks[i].ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("task #%d not found", id)
		}
		if *title != "" {
			ts.Tasks[idx].Title = *title
		}
		if *target > 0 {
			ts.Tasks[idx].TargetSessions = *target
		}
		if *completed >= 0 {
			ts.Tasks[idx].CompletedPomodoros = *completed
		}
		if *desc != "" {
			ts.Tasks[idx].Description = *desc
		}
		ts.Tasks[idx].Done = ts.Tasks[idx].CompletedPomodoros >= ts.Tasks[idx].TargetSessions
		if ts.Tasks[idx].Done {
			ts.Tasks[idx].TimerPhase = ""
			ts.Tasks[idx].TimerRemainingSec = 0
			ts.Tasks[idx].TimerSessionIndex = 0
			ts.Tasks[idx].TimerTotalSessions = 0
		}
		ts.Tasks[idx].UpdatedAt = now
		if err := store.SaveTasks(ts); err != nil {
			return err
		}
		fmt.Printf("task updated: #%d\n", id)
		return nil
	case "delete":
		if len(args) < 2 {
			return errors.New("usage: task delete <id>")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		out := ts.Tasks[:0]
		deleted := false
		for _, t := range ts.Tasks {
			if t.ID == id {
				deleted = true
				if t.GCalEventID != "" {
					ts.DeletedGCalEventIDs = append(ts.DeletedGCalEventIDs, t.GCalEventID)
				}
				continue
			}
			out = append(out, t)
		}
		if !deleted {
			return fmt.Errorf("task #%d not found", id)
		}
		ts.Tasks = out
		if err := store.SaveTasks(ts); err != nil {
			return err
		}
		fmt.Printf("task deleted: #%d\n", id)
		return nil
	case "done":
		if len(args) < 2 {
			return errors.New("usage: task done <id> [true|false]")
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		value := true
		if len(args) >= 3 {
			v := strings.TrimSpace(strings.ToLower(args[2]))
			value = v == "true" || v == "1" || v == "yes" || v == "y"
		}
		for i := range ts.Tasks {
			if ts.Tasks[i].ID == id {
				ts.Tasks[i].Done = value
				if value {
					ts.Tasks[i].TimerPhase = ""
					ts.Tasks[i].TimerRemainingSec = 0
					ts.Tasks[i].TimerSessionIndex = 0
					ts.Tasks[i].TimerTotalSessions = 0

					// Update GCal event title asynchronously if GCal is enabled
					cfg, errCfg := store.LoadConfig()
					if errCfg == nil && cfg.GCalEnabled && ts.Tasks[i].GCalEventID != "" {
						go func(eventID, calendarName string) {
							client, err := gcal.NewClient(store)
							if err != nil {
								return
							}
							ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
							defer cancel()
							_ = client.MarkEventAsDone(ctx, eventID, calendarName)
						}(ts.Tasks[i].GCalEventID, cfg.GCalCalendarName)
					}
				}
				ts.Tasks[i].UpdatedAt = now
				if err := store.SaveTasks(ts); err != nil {
					return err
				}
				fmt.Printf("task #%d done=%v\n", id, value)
				return nil
			}
		}
		return fmt.Errorf("task #%d not found", id)
	default:
		return fmt.Errorf("unknown task command: %s", args[0])
	}
}

func runConfig(store *storage.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("config subcommand required")
	}
	cfg, err := store.LoadConfig()
	if err != nil {
		return err
	}

	switch args[0] {
	case "show":
		fmt.Printf("focus=%d short=%d long=%d long-every=%d theme=%s gcal-enabled=%v gcal-calendar-name=%q\n", cfg.FocusMinutes, cfg.ShortBreakMinutes, cfg.LongBreakMinutes, cfg.LongBreakEvery, cfg.Theme, cfg.GCalEnabled, cfg.GCalCalendarName)
		printNotifications(cfg)
		return nil
	case "set":
		fs := flag.NewFlagSet("config set", flag.ContinueOnError)
		focus := fs.Int("focus", 0, "focus minutes")
		short := fs.Int("short", 0, "short break minutes")
		long := fs.Int("long", 0, "long break minutes")
		longEvery := fs.Int("long-every", 0, "long break every n focus sessions")
		theme := fs.String("theme", "", "theme preset: sunrise|forest|mono")
		notifEnabled := fs.String("notifications", "", "notifications on|off")
		notifyWarningBefore := fs.Int("notify-warning-before", 0, "warning minutes before end")
		notifyDesktop := fs.String("notify-desktop", "", "desktop notification on|off")
		notifySound := fs.String("notify-sound", "", "sound notification on|off")
		notifyLog := fs.String("notify-log", "", "log notification on|off")
		notifyLogPath := fs.String("notify-log-path", "", "notification log file path")
		gcalEnabled := fs.String("gcal-enabled", "", "gcal integration on|off")
		gcalCalendarName := fs.String("gcal-calendar-name", "", "gcal calendar name for focus sessions")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if cfg.Notifications == nil {
			cfg.Notifications = model.DefaultNotificationConfig()
		}
		if *focus > 0 {
			cfg.FocusMinutes = *focus
		}
		if *short > 0 {
			cfg.ShortBreakMinutes = *short
		}
		if *long > 0 {
			cfg.LongBreakMinutes = *long
		}
		if *longEvery > 0 {
			cfg.LongBreakEvery = *longEvery
		}
		if *theme != "" {
			cfg.Theme = strings.ToLower(strings.TrimSpace(*theme))
			if cfg.Theme != "sunrise" && cfg.Theme != "forest" && cfg.Theme != "mono" {
				return errors.New("theme must be one of: sunrise, forest, mono")
			}
		}
		if *notifyWarningBefore > 0 {
			cfg.Notifications.WarningMinutesBefore = *notifyWarningBefore
		}
		if *notifyLogPath != "" {
			if cfg.Notifications.LogFile == nil {
				cfg.Notifications.LogFile = model.NewLogFileNotifConfig()
			}
			cfg.Notifications.LogFile.Path = strings.TrimSpace(*notifyLogPath)
		}
		if *notifEnabled != "" {
			v, err := parseOnOff(*notifEnabled)
			if err != nil {
				return fmt.Errorf("--notifications: %w", err)
			}
			cfg.Notifications.Enabled = v
		}
		if *notifyDesktop != "" {
			v, err := parseOnOff(*notifyDesktop)
			if err != nil {
				return fmt.Errorf("--notify-desktop: %w", err)
			}
			if cfg.Notifications.Desktop == nil {
				cfg.Notifications.Desktop = model.NewDesktopNotifConfig()
			}
			cfg.Notifications.Desktop.Enabled = v
		}
		if *notifySound != "" {
			v, err := parseOnOff(*notifySound)
			if err != nil {
				return fmt.Errorf("--notify-sound: %w", err)
			}
			if cfg.Notifications.Sound == nil {
				cfg.Notifications.Sound = model.NewSoundNotifConfig()
			}
			cfg.Notifications.Sound.Enabled = v
		}
		if *notifyLog != "" {
			v, err := parseOnOff(*notifyLog)
			if err != nil {
				return fmt.Errorf("--notify-log: %w", err)
			}
			if cfg.Notifications.LogFile == nil {
				cfg.Notifications.LogFile = model.NewLogFileNotifConfig()
			}
			cfg.Notifications.LogFile.Enabled = v
		}
		if *gcalEnabled != "" {
			v, err := parseOnOff(*gcalEnabled)
			if err != nil {
				return fmt.Errorf("--gcal-enabled: %w", err)
			}
			cfg.GCalEnabled = v
		}
		if *gcalCalendarName != "" {
			cfg.GCalCalendarName = strings.TrimSpace(*gcalCalendarName)
		}

		if cfg.FocusMinutes < 1 || cfg.ShortBreakMinutes < 1 || cfg.LongBreakMinutes < 1 || cfg.LongBreakEvery < 1 {
			return errors.New("all config values must be >= 1")
		}
		if err := store.SaveConfig(cfg); err != nil {
			return err
		}
		fmt.Println("config saved")
		return nil
	case "key":
		if len(args) < 2 {
			return errors.New("usage: config key <show|set>")
		}
		switch args[1] {
		case "show":
			fmt.Printf("nav_up=%s nav_down=%s alt_nav_up=%s alt_nav_down=%s\n", cfg.Keys.NavUp, cfg.Keys.NavDown, cfg.Keys.AltNavUp, cfg.Keys.AltNavDown)
			fmt.Printf("reorder_up=%s reorder_down=%s start_cycle=%s\n", cfg.Keys.ReorderUp, cfg.Keys.ReorderDown, cfg.Keys.StartCycle)
			fmt.Printf("add=%s edit=%s delete=%s toggle_done=%s\n", cfg.Keys.AddTask, cfg.Keys.EditTask, cfg.Keys.DeleteTask, cfg.Keys.ToggleDone)
			fmt.Printf("settings=%s refresh=%s quit=%s pause=%s end_phase=%s next_phase=%s\n", cfg.Keys.Settings, cfg.Keys.Refresh, cfg.Keys.Quit, cfg.Keys.Pause, cfg.Keys.EndPhase, cfg.Keys.NextPhase)
			return nil
		case "set":
			if len(args) < 4 {
				return errors.New("usage: config key set <action> <key>")
			}
			action := strings.ToLower(strings.TrimSpace(args[2]))
			newKey := strings.ToLower(strings.TrimSpace(args[3]))
			if err := setKeybinding(&cfg, action, newKey); err != nil {
				return err
			}
			if err := store.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Printf("keybinding updated: %s=%s\n", action, newKey)
			return nil
		default:
			return errors.New("usage: config key <show|set>")
		}
	case "notifications", "notif":
		if len(args) < 2 {
			return errors.New("usage: config notifications <show|set>")
		}
		ensureNotifications(&cfg)
		switch args[1] {
		case "show":
			printNotifications(cfg)
			return nil
		case "set":
			fs := flag.NewFlagSet("config notifications set", flag.ContinueOnError)
			enabled := fs.String("enabled", "", "notifications on|off")
			warning := fs.Int("warning-before", 0, "warning minutes before end")
			desktop := fs.String("desktop", "", "desktop notification on|off")
			sound := fs.String("sound", "", "sound notification on|off")
			logEnabled := fs.String("log", "", "log notification on|off")
			logPath := fs.String("log-path", "", "notification log file path")
			if err := fs.Parse(args[2:]); err != nil {
				return err
			}
			if *enabled != "" {
				v, err := parseOnOff(*enabled)
				if err != nil {
					return fmt.Errorf("--enabled: %w", err)
				}
				cfg.Notifications.Enabled = v
			}
			if *warning > 0 {
				cfg.Notifications.WarningMinutesBefore = *warning
			}
			if *desktop != "" {
				v, err := parseOnOff(*desktop)
				if err != nil {
					return fmt.Errorf("--desktop: %w", err)
				}
				cfg.Notifications.Desktop.Enabled = v
			}
			if *sound != "" {
				v, err := parseOnOff(*sound)
				if err != nil {
					return fmt.Errorf("--sound: %w", err)
				}
				cfg.Notifications.Sound.Enabled = v
			}
			if *logEnabled != "" {
				v, err := parseOnOff(*logEnabled)
				if err != nil {
					return fmt.Errorf("--log: %w", err)
				}
				cfg.Notifications.LogFile.Enabled = v
			}
			if *logPath != "" {
				cfg.Notifications.LogFile.Path = strings.TrimSpace(*logPath)
			}
			if err := store.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Println("notification config saved")
			printNotifications(cfg)
			return nil
		default:
			return errors.New("usage: config notifications <show|set>")
		}
	default:
		return fmt.Errorf("unknown config command: %s", args[0])
	}
}

func ensureNotifications(cfg *model.Config) {
	if cfg.Notifications == nil {
		cfg.Notifications = model.DefaultNotificationConfig()
	}
	if cfg.Notifications.Desktop == nil {
		cfg.Notifications.Desktop = model.NewDesktopNotifConfig()
	}
	if cfg.Notifications.Sound == nil {
		cfg.Notifications.Sound = model.NewSoundNotifConfig()
	}
	if cfg.Notifications.LogFile == nil {
		cfg.Notifications.LogFile = model.NewLogFileNotifConfig()
	}
}

func printNotifications(cfg model.Config) {
	if cfg.Notifications == nil {
		cfg.Notifications = model.DefaultNotificationConfig()
	}
	fmt.Printf("notifications=%v warning-before=%d desktop=%v sound=%v log=%v log-path=%s\n",
		cfg.Notifications.Enabled,
		cfg.Notifications.WarningMinutesBefore,
		cfg.Notifications.Desktop != nil && cfg.Notifications.Desktop.Enabled,
		cfg.Notifications.Sound != nil && cfg.Notifications.Sound.Enabled,
		cfg.Notifications.LogFile != nil && cfg.Notifications.LogFile.Enabled,
		func() string {
			if cfg.Notifications.LogFile == nil {
				return ""
			}
			return cfg.Notifications.LogFile.Path
		}(),
	)
}

func setKeybinding(cfg *model.Config, action, value string) error {
	switch action {
	case "nav_up":
		cfg.Keys.NavUp = value
	case "nav_down":
		cfg.Keys.NavDown = value
	case "alt_nav_up":
		cfg.Keys.AltNavUp = value
	case "alt_nav_down":
		cfg.Keys.AltNavDown = value
	case "reorder_up":
		cfg.Keys.ReorderUp = value
	case "reorder_down":
		cfg.Keys.ReorderDown = value
	case "add_task":
		cfg.Keys.AddTask = value
	case "edit_task":
		cfg.Keys.EditTask = value
	case "delete_task":
		cfg.Keys.DeleteTask = value
	case "toggle_done":
		cfg.Keys.ToggleDone = value
	case "start_cycle":
		cfg.Keys.StartCycle = value
	case "settings":
		cfg.Keys.Settings = value
	case "refresh":
		cfg.Keys.Refresh = value
	case "quit":
		cfg.Keys.Quit = value
	case "pause":
		cfg.Keys.Pause = value
	case "end_phase":
		cfg.Keys.EndPhase = value
	case "next_phase":
		cfg.Keys.NextPhase = value
	default:
		return errors.New("unknown action. use config key show to see available actions")
	}
	return nil
}

func parseOnOff(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "on", "true", "1", "yes", "y":
		return true, nil
	case "off", "false", "0", "no", "n":
		return false, nil
	default:
		return false, errors.New("must be on|off")
	}
}

func runPomodoro(store *storage.Store, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	taskID := fs.Int("task", 0, "task id")
	sessions := fs.Int("sessions", 1, "number of focus sessions")
	notifEnabled := fs.String("notifications", "", "notifications on|off")
	notifyWarningBefore := fs.Int("notify-warning-before", 0, "warning minutes before end")
	notifyDesktop := fs.String("notify-desktop", "", "desktop notification on|off")
	notifySound := fs.String("notify-sound", "", "sound notification on|off")
	notifyLog := fs.String("notify-log", "", "log notification on|off")
	notifyLogPath := fs.String("notify-log-path", "", "notification log file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sessions < 1 {
		return errors.New("sessions must be >= 1")
	}

	cfg, err := store.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.Notifications == nil {
		cfg.Notifications = model.DefaultNotificationConfig()
	}
	if *notifyWarningBefore > 0 {
		cfg.Notifications.WarningMinutesBefore = *notifyWarningBefore
	}
	if *notifyLogPath != "" {
		if cfg.Notifications.LogFile == nil {
			cfg.Notifications.LogFile = model.NewLogFileNotifConfig()
		}
		cfg.Notifications.LogFile.Path = strings.TrimSpace(*notifyLogPath)
	}
	if *notifEnabled != "" {
		v, err := parseOnOff(*notifEnabled)
		if err != nil {
			return fmt.Errorf("--notifications: %w", err)
		}
		cfg.Notifications.Enabled = v
	}
	if *notifyDesktop != "" {
		v, err := parseOnOff(*notifyDesktop)
		if err != nil {
			return fmt.Errorf("--notify-desktop: %w", err)
		}
		if cfg.Notifications.Desktop == nil {
			cfg.Notifications.Desktop = model.NewDesktopNotifConfig()
		}
		cfg.Notifications.Desktop.Enabled = v
	}
	if *notifySound != "" {
		v, err := parseOnOff(*notifySound)
		if err != nil {
			return fmt.Errorf("--notify-sound: %w", err)
		}
		if cfg.Notifications.Sound == nil {
			cfg.Notifications.Sound = model.NewSoundNotifConfig()
		}
		cfg.Notifications.Sound.Enabled = v
	}
	if *notifyLog != "" {
		v, err := parseOnOff(*notifyLog)
		if err != nil {
			return fmt.Errorf("--notify-log: %w", err)
		}
		if cfg.Notifications.LogFile == nil {
			cfg.Notifications.LogFile = model.NewLogFileNotifConfig()
		}
		cfg.Notifications.LogFile.Enabled = v
	}
	notifier := notify.NewManagerFromConfig(cfg.Notifications)
	defer notifier.Close()
	ts, err := store.LoadTasks()
	if err != nil {
		return err
	}
	history, err := store.LoadHistory()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	focusDur := time.Duration(cfg.FocusMinutes) * time.Minute
	breakDur := time.Duration(cfg.ShortBreakMinutes) * time.Minute
	targetSess := *sessions

	if *taskID > 0 {
		for _, t := range ts.Tasks {
			if t.ID == *taskID {
				if t.FocusDuration > 0 {
					focusDur = time.Duration(t.FocusDuration) * time.Minute
				}
				if t.BreakDuration > 0 {
					breakDur = time.Duration(t.BreakDuration) * time.Minute
				}
				if t.TargetSessions > 0 && *sessions == 1 {
					remaining := t.TargetSessions - t.CompletedPomodoros
					if remaining > 0 {
						targetSess = remaining
					}
				}
			}
		}
	}

	engineCfg := pomodoro.EngineConfig{
		FocusDuration:      focusDur,
		ShortBreakDuration: breakDur,
		LongBreakDuration:  time.Duration(cfg.LongBreakMinutes) * time.Minute,
		LongBreakEvery:     cfg.LongBreakEvery,
		TargetSessions:     targetSess,
		WarningDuration:    time.Duration(cfg.Notifications.WarningMinutesBefore) * time.Minute,
		TickInterval:       time.Second,
	}

	engine := pomodoro.NewSessionEngine(engineCfg)

	doneChan := make(chan error, 1)

	engine.OnPhaseStart = func(state pomodoro.EngineState) {
		switch state.Phase {
		case pomodoro.PhaseFocus:
			fmt.Printf("\nFocus session %d/%d\n", state.SessionCount, state.TotalSessions)
		case pomodoro.PhaseShortBreak:
			fmt.Printf("\nSHORT BREAK\n")
		case pomodoro.PhaseLongBreak:
			fmt.Printf("\nLONG BREAK\n")
		}
	}

	engine.OnSessionWarn = func(state pomodoro.EngineState) {
		if cfg.Notifications == nil || !cfg.Notifications.Enabled || cfg.Notifications.WarningMinutesBefore <= 0 {
			return
		}
		phaseType := string(state.Phase)
		_ = notifier.SendNotification(ctx, model.NotificationEvent{
			Type:       model.NotificationSessionWarn,
			Timestamp:  time.Now(),
			SessionNum: state.SessionCount,
			PhaseType:  phaseType,
			TaskID:     *taskID,
			Message:    fmt.Sprintf("Sisa %s %d menit.", strings.ReplaceAll(phaseType, "_", " "), cfg.Notifications.WarningMinutesBefore),
		})
	}

	engine.OnTick = func(state pomodoro.EngineState) {
		label := "FOCUS"
		switch state.Phase {
		case pomodoro.PhaseShortBreak:
			label = "SHORT BREAK"
		case pomodoro.PhaseLongBreak:
			label = "LONG BREAK"
		}
		mins := int(state.Remaining / time.Minute)
		secs := int((state.Remaining % time.Minute) / time.Second)
		fmt.Printf("\r%s %02d:%02d", label, mins, secs)
	}

	engine.OnPhaseComplete = func(phase pomodoro.Phase, sessionCount int, startedAt, endedAt time.Time, completed bool) {
		if phase == pomodoro.PhaseFocus {
			if completed {
				fmt.Print("\a\n") // Beep + Newline
			} else {
				fmt.Print("\n")
			}
			history = append(history, model.SessionHistory{StartedAt: startedAt, EndedAt: endedAt, TaskID: *taskID, Type: "focus", Completed: completed})
			if completed {
				_ = notifier.SendNotification(ctx, model.NotificationEvent{
					Type:       model.NotificationFocusComplete,
					Timestamp:  time.Now(),
					SessionNum: sessionCount,
					PhaseType:  "focus",
					TaskID:     *taskID,
					Message:    "Sesi fokus selesai. Saatnya istirahat.",
				})

				if cfg.GCalEnabled {
					go func() {
						client, err := gcal.NewClient(store)
						if err != nil {
							return
						}
						taskTitle := "Pomodoro Session"
						if *taskID > 0 {
							for _, t := range ts.Tasks {
								if t.ID == *taskID {
									taskTitle = t.Title
									break
								}
							}
						}
						syncCtx, syncCancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer syncCancel()
						_, _ = client.SyncSessionEvent(syncCtx, taskTitle, startedAt, endedAt, cfg.GCalCalendarName)
					}()
				}

				if *taskID > 0 {
					for ti := range ts.Tasks {
						if ts.Tasks[ti].ID == *taskID {
							ts.Tasks[ti].CompletedPomodoros++
							ts.Tasks[ti].UpdatedAt = time.Now()
							if ts.Tasks[ti].TargetSessions > 0 && ts.Tasks[ti].CompletedPomodoros >= ts.Tasks[ti].TargetSessions {
								ts.Tasks[ti].Done = true
								_ = notifier.SendNotification(ctx, model.NotificationEvent{
									Type:       model.NotificationTaskComplete,
									Timestamp:  time.Now(),
									SessionNum: sessionCount,
									TaskID:     *taskID,
									Message:    "Semua sesi task selesai.",
								})

								// Update GCal event title asynchronously if GCal is enabled
								if cfg.GCalEnabled && ts.Tasks[ti].GCalEventID != "" {
									go func(eventID, calendarName string) {
										client, err := gcal.NewClient(store)
										if err != nil {
											return
										}
										ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
										defer cancel()
										_ = client.MarkEventAsDone(ctx, eventID, calendarName)
									}(ts.Tasks[ti].GCalEventID, cfg.GCalCalendarName)
								}
							}
						}
					}
				}
			}
		} else {
			if completed {
				fmt.Print("\a\n") // Beep + Newline
			} else {
				fmt.Print("\n")
			}
			history = append(history, model.SessionHistory{StartedAt: startedAt, EndedAt: endedAt, TaskID: *taskID, Type: string(phase), Completed: completed})
			if completed {
				_ = notifier.SendNotification(ctx, model.NotificationEvent{
					Type:       model.NotificationBreakComplete,
					Timestamp:  time.Now(),
					SessionNum: sessionCount,
					PhaseType:  string(phase),
					TaskID:     *taskID,
					Message:    "Waktu istirahat selesai. Kembali fokus.",
				})
			}
		}
	}

	engine.OnComplete = func() {
		doneChan <- nil
	}

	engine.Start(ctx)

	var runErr error
	select {
	case <-ctx.Done():
		runErr = fmt.Errorf("interrupted")
		time.Sleep(10 * time.Millisecond) // allow engine to trigger OnPhaseComplete
	case <-doneChan:
	}

	if err := store.SaveHistory(history); err != nil {
		return err
	}
	if err := store.SaveTasks(ts); err != nil {
		return err
	}

	if runErr != nil {
		return runErr
	}

	fmt.Println("\nPomodoro run finished")
	return nil
}

func runSingleTimer(args []string) error {
	fs := flag.NewFlagSet("timer", flag.ContinueOnError)
	minutes := fs.Int("minutes", 25, "timer minutes")
	label := fs.String("label", "TIMER", "label")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *minutes < 1 {
		return errors.New("minutes must be >= 1")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	_, _, _, err := pomodoro.Countdown(ctx, *label, *minutes)
	if err != nil {
		return fmt.Errorf("interrupted")
	}
	fmt.Println("timer finished")
	return nil
}

func runStats(store *storage.Store) error {
	ts, err := store.LoadTasks()
	if err != nil {
		return err
	}
	h, err := store.LoadHistory()
	if err != nil {
		return err
	}
	focusDone := 0
	for _, entry := range h {
		if entry.Type == "focus" && entry.Completed {
			focusDone++
		}
	}
	fmt.Printf("tasks=%d completed-focus-sessions=%d\n", len(ts.Tasks), focusDone)
	return nil
}

func runGCal(store *storage.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gcal <login|logout|status|sync>")
	}

	switch args[0] {
	case "login":
		client, err := gcal.NewClient(store)
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return client.Login(ctx)

	case "logout":
		err := store.DeleteGCalToken()
		if err != nil {
			return err
		}
		fmt.Println("GCal token deleted. Logged out.")
		return nil

	case "status":
		// 1. Check credentials file
		_, err := store.ReadGCalCredentials()
		if err != nil {
			fmt.Println("GCal Status: Credentials NOT configured.")
			fmt.Println("Silakan letakkan file client_credentials.json Anda di ~/.config/focus-cli/gcal_credentials.json")
			return nil
		}
		fmt.Println("GCal Credentials: Configured.")

		// 2. Check token file
		tokenBytes, err := store.LoadGCalToken()
		if err != nil {
			fmt.Println("GCal Token: NOT authenticated. Run 'focus gcal login' to connect.")
			return nil
		}
		_ = tokenBytes // unused
		fmt.Println("GCal Token: Authenticated.")

		// 3. Test API connectivity & display account information
		client, err := gcal.NewClient(store)
		if err != nil {
			fmt.Printf("GCal Client Error: %v\n", err)
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv, err := client.GetCalendarService(ctx)
		if err != nil {
			fmt.Printf("GCal API Connectivity: Failed (%v)\n", err)
			return nil
		}
		cal, err := srv.Calendars.Get("primary").Do()
		if err != nil {
			fmt.Printf("GCal API Connectivity: Failed (%v)\n", err)
			return nil
		}
		fmt.Printf("GCal API Connectivity: Connected (Account: %s)\n", cal.Summary)
		return nil

	case "sync":
		cfg, err := store.LoadConfig()
		if err != nil {
			return err
		}
		if !cfg.GCalEnabled {
			return errors.New("integrasi GCal dinonaktifkan dalam konfigurasi (gunakan 'focus config set --gcal-enabled on' untuk mengaktifkan)")
		}

		client, err := gcal.NewClient(store)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Println("Mengambil tugas dari Google Calendar...")
		gcalTasks, err := client.ImportTasks(ctx, cfg.GCalCalendarName)
		if err != nil {
			return err
		}

		if len(gcalTasks) == 0 {
			fmt.Println("Tidak ada tugas baru untuk diimpor.")
			return nil
		}

		// Load local tasks
		ts, err := store.LoadTasks()
		if err != nil {
			return err
		}

		importedCount := 0
		for _, gt := range gcalTasks {
			// Check if task already exists based on GCalEventID
			exists := false
			for _, lt := range ts.Tasks {
				if lt.GCalEventID == gt.GCalEventID {
					exists = true
					break
				}
			}

			if !exists {
				gt.ID = ts.NextID
				ts.NextID++
				ts.Tasks = append(ts.Tasks, gt)
				fmt.Printf("Mengimpor tugas baru: #%d %s\n", gt.ID, gt.Title)
				importedCount++
			}
		}

		if importedCount > 0 {
			if err := store.SaveTasks(ts); err != nil {
				return err
			}
			fmt.Printf("Sinkronisasi selesai. %d tugas berhasil diimpor.\n", importedCount)
		} else {
			fmt.Println("Semua tugas dari kalender sudah tersinkronisasi.")
		}
		return nil

	default:
		return fmt.Errorf("unknown gcal subcommand: %s", args[0])
	}
}
