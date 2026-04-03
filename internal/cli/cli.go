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
	"focus-cli/internal/pomodoro"
	"focus-cli/internal/tui"
	"focus-cli/internal/storage"
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
	fmt.Println("  config key show")
	fmt.Println("  config key set <action> <key>")
	fmt.Println("  run [--task ID] [--sessions N]")
	fmt.Println("  timer [--minutes N] [--label text]")
	fmt.Println("  stats")
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
		fmt.Printf("focus=%d short=%d long=%d long-every=%d theme=%s\n", cfg.FocusMinutes, cfg.ShortBreakMinutes, cfg.LongBreakMinutes, cfg.LongBreakEvery, cfg.Theme)
		return nil
	case "set":
		fs := flag.NewFlagSet("config set", flag.ContinueOnError)
		focus := fs.Int("focus", 0, "focus minutes")
		short := fs.Int("short", 0, "short break minutes")
		long := fs.Int("long", 0, "long break minutes")
		longEvery := fs.Int("long-every", 0, "long break every n focus sessions")
		theme := fs.String("theme", "", "theme preset: sunrise|forest|mono")
		if err := fs.Parse(args[1:]); err != nil {
			return err
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
	default:
		return fmt.Errorf("unknown config command: %s", args[0])
	}
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

func runPomodoro(store *storage.Store, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	taskID := fs.Int("task", 0, "task id")
	sessions := fs.Int("sessions", 1, "number of focus sessions")
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

	for i := 1; i <= *sessions; i++ {
		fmt.Printf("\nFocus session %d/%d\n", i, *sessions)
		s, e, done, runErr := pomodoro.Countdown(ctx, "FOCUS", cfg.FocusMinutes)
		history = append(history, model.SessionHistory{StartedAt: s, EndedAt: e, TaskID: *taskID, Type: "focus", Completed: done})
		if runErr != nil {
			_ = store.SaveHistory(history)
			return fmt.Errorf("interrupted")
		}

		if *taskID > 0 {
			for ti := range ts.Tasks {
				if ts.Tasks[ti].ID == *taskID {
					ts.Tasks[ti].CompletedPomodoros++
					ts.Tasks[ti].UpdatedAt = time.Now()
				}
			}
		}

		if i == *sessions {
			break
		}

		breakMin := cfg.ShortBreakMinutes
		label := "SHORT BREAK"
		if i%cfg.LongBreakEvery == 0 {
			breakMin = cfg.LongBreakMinutes
			label = "LONG BREAK"
		}
		fmt.Printf("\n%s\n", label)
		s, e, done, runErr = pomodoro.Countdown(ctx, label, breakMin)
		history = append(history, model.SessionHistory{StartedAt: s, EndedAt: e, TaskID: *taskID, Type: strings.ToLower(strings.ReplaceAll(label, " ", "_")), Completed: done})
		if runErr != nil {
			_ = store.SaveHistory(history)
			_ = store.SaveTasks(ts)
			return fmt.Errorf("interrupted")
		}
	}

	if err := store.SaveHistory(history); err != nil {
		return err
	}
	if err := store.SaveTasks(ts); err != nil {
		return err
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
