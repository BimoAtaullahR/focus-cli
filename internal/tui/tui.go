package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"focus-cli/internal/model"
	"focus-cli/internal/notify"
	"focus-cli/internal/storage"
)

const (
	modeDashboard = iota
	modeForm
	modeConfirmDelete
	modeRunning
)

type runPhase int

const (
	runPhaseFocus runPhase = iota
	runPhaseBreak
)

type formKind int

const (
	formAdd formKind = iota
	formEdit
	formConfig
)

type formField struct {
	label string
	input textinput.Model
}

type formState struct {
	kind   formKind
	fields []formField
	index  int
	taskID int
}

type confirmState struct {
	taskID int
}

type runState struct {
	phase           runPhase
	label           string
	remaining       time.Duration
	phaseDuration   time.Duration
	startedAt       time.Time
	taskID          int
	sessionIndex    int
	totalSessions   int
	paused          bool
	notifiedWarning bool
}

type Model struct {
	store   *storage.Store
	tasks   model.TaskStore
	config  model.Config
	history []model.SessionHistory

	width  int
	height int
	cursor int
	mode   int

	status   string
	form     *formState
	confirm  *confirmState
	run      *runState
	tickID   int
	ready    bool
	notifier *notify.Manager
}

var (
	appTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	panelStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	selectedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Bold(true)
	selectedDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("62")).Bold(true).Strikethrough(true)
	doneTaskStyle     = lipgloss.NewStyle().Strikethrough(true)
	dimStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	accentStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	goodStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	badStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
)

func applyTheme(theme string) {
	switch strings.ToLower(strings.TrimSpace(theme)) {
	case "forest":
		appTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("120"))
		panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("65")).Padding(0, 1)
		selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("28")).Bold(true)
		selectedDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Background(lipgloss.Color("28")).Bold(true).Strikethrough(true)
		doneTaskStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Strikethrough(true)
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("108"))
		helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
		accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("49")).Bold(true)
		goodStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("48"))
		warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("186"))
		badStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	case "mono":
		appTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
		panelStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("245")).Padding(0, 1)
		selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("252")).Bold(true)
		selectedDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("252")).Bold(true).Strikethrough(true)
		doneTaskStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Strikethrough(true)
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
		accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
		goodStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
		badStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	default:
		appTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("221"))
		panelStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("173")).Padding(0, 1)
		selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("222")).Bold(true)
		selectedDoneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(lipgloss.Color("222")).Bold(true).Strikethrough(true)
		doneTaskStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("180")).Strikethrough(true)
		dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("180"))
		helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("216"))
		accentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
		goodStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
		warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("215"))
		badStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	}
}

func keyIs(msg tea.KeyMsg, bindings ...string) bool {
	pressed := normalizeKey(msg.String())
	for _, binding := range bindings {
		if pressed == normalizeKey(binding) {
			return true
		}
	}
	return false
}

func normalizeKey(s string) string {
	raw := strings.ToLower(s)
	if raw == " " {
		return "space"
	}
	trimmed := strings.TrimSpace(raw)
	return trimmed
}

func Run(store *storage.Store) error {
	tasks, err := store.LoadTasks()
	if err != nil {
		return err
	}
	config, err := store.LoadConfig()
	if err != nil {
		return err
	}
	history, err := store.LoadHistory()
	if err != nil {
		return err
	}
	applyTheme(config.Theme)
	m := &Model{store: store, tasks: tasks, config: config, history: history, status: "ready", notifier: notify.NewManagerFromConfig(config.Notifications)}
	defer m.notifier.Close()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}

func (m *Model) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		if m.mode == modeDashboard {
			return m.updateDashboard(msg)
		}
		if m.mode == modeForm {
			return m.updateForm(msg)
		}
		if m.mode == modeConfirmDelete {
			return m.updateConfirm(msg)
		}
		if m.mode == modeRunning {
			return m.updateRun(msg)
		}
	case runTickMsg:
		return m.handleRunTick(msg)
	}
	return m, nil
}

func (m *Model) View() string {
	if !m.ready {
		return "Loading focus-cli..."
	}
	if m.mode == modeForm {
		return m.viewForm()
	}
	if m.mode == modeConfirmDelete {
		return m.viewConfirm()
	}
	if m.mode == modeRunning {
		return m.viewRunning()
	}
	return m.viewDashboard()
}

type runTickMsg struct {
	at    time.Time
	token int
}

func runTickCmd(token int) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return runTickMsg{at: t, token: token}
	})
}

func (m *Model) notifyAsync(event model.NotificationEvent) {
	if m.notifier == nil || m.config.Notifications == nil || !m.config.Notifications.Enabled {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = m.notifier.SendNotification(ctx, event)
	}()
}

func (m *Model) beginFocusCycle(taskID, totalSessions int) (tea.Model, tea.Cmd) {
	if totalSessions < 1 {
		totalSessions = 1
	}
	m.run = &runState{
		phase:           runPhaseFocus,
		label:           "FOCUS",
		remaining:       time.Duration(m.config.FocusMinutes) * time.Minute,
		phaseDuration:   time.Duration(m.config.FocusMinutes) * time.Minute,
		startedAt:       time.Now(),
		taskID:          taskID,
		sessionIndex:    1,
		totalSessions:   totalSessions,
		paused:          false,
		notifiedWarning: false,
	}
	m.mode = modeRunning
	m.status = fmt.Sprintf("Semangat! Fokus sesi 1/%d dimulai.", totalSessions)
	m.tickID++
	return m, runTickCmd(m.tickID)
}

func (m *Model) beginBreakPhase(breakType string) (tea.Model, tea.Cmd) {
	if m.run == nil {
		m.mode = modeDashboard
		return m.reload()
	}
	minutes := m.config.ShortBreakMinutes
	label := "SHORT BREAK"
	if breakType == "long" {
		minutes = m.config.LongBreakMinutes
		label = "LONG BREAK"
	}
	m.run.phase = runPhaseBreak
	m.run.label = label
	m.run.remaining = time.Duration(minutes) * time.Minute
	m.run.phaseDuration = time.Duration(minutes) * time.Minute
	m.run.startedAt = time.Now()
	m.run.paused = false
	m.run.notifiedWarning = false
	if breakType == "long" {
		m.status = "Hebat! Saatnya long break untuk isi ulang energi."
	} else {
		m.status = "Nice! Saatnya short break sebentar."
	}
	m.tickID++
	return m, runTickCmd(m.tickID)
}

func (m *Model) finishFocusSession(manual bool) (tea.Model, tea.Cmd) {
	if m.run == nil {
		return m, nil
	}
	achievementMsg := ""
	history := model.SessionHistory{
		StartedAt: m.run.startedAt,
		EndedAt:   time.Now(),
		TaskID:    m.run.taskID,
		Type:      "focus",
		Completed: true,
	}
	m.history = append(m.history, history)
	_ = m.store.SaveHistory(m.history)
	m.notifyAsync(model.NotificationEvent{
		Type:       model.NotificationFocusComplete,
		Timestamp:  time.Now(),
		SessionNum: m.run.sessionIndex,
		PhaseType:  "focus",
		TaskID:     m.run.taskID,
		Message:    "Sesi fokus selesai. Saatnya istirahat.",
	})
	isTaskComplete := false
	if m.run.taskID > 0 {
		for i := range m.tasks.Tasks {
			if m.tasks.Tasks[i].ID == m.run.taskID {
				m.tasks.Tasks[i].CompletedPomodoros++
				if m.tasks.Tasks[i].TargetSessions > 0 && m.tasks.Tasks[i].CompletedPomodoros >= m.tasks.Tasks[i].TargetSessions {
					m.tasks.Tasks[i].Done = true
					isTaskComplete = true
					achievementMsg = fmt.Sprintf("Luar biasa! Task '%s' selesai 100%%. Kerja bagus!", m.tasks.Tasks[i].Title)
				}
				m.tasks.Tasks[i].UpdatedAt = time.Now()
				break
			}
		}
		_ = m.store.SaveTasks(m.tasks)
	}
	if isTaskComplete {
		m.notifyAsync(model.NotificationEvent{
			Type:       model.NotificationTaskComplete,
			Timestamp:  time.Now(),
			SessionNum: m.run.sessionIndex,
			TaskID:     m.run.taskID,
			Message:    "Semua sesi task selesai.",
		})
	}
	if m.run.sessionIndex >= m.run.totalSessions {
		m.status = "Mantap! Satu cycle pomodoro berhasil kamu selesaikan."
		if achievementMsg != "" {
			m.status = achievementMsg
		}
		m.run = nil
		m.mode = modeDashboard
		m.tickID++
		return m.reload()
	}
	breakType := "short"
	if m.run.sessionIndex%m.config.LongBreakEvery == 0 {
		breakType = "long"
	}
	if manual {
		m.status = "Bagus! Sesi fokus selesai, lanjut break ya."
		if achievementMsg != "" {
			m.status = achievementMsg
		}
	}
	return m.beginBreakPhase(breakType)
}

func (m *Model) finishBreakSession(skip bool) (tea.Model, tea.Cmd) {
	if m.run == nil {
		return m, nil
	}
	history := model.SessionHistory{
		StartedAt: m.run.startedAt,
		EndedAt:   time.Now(),
		TaskID:    m.run.taskID,
		Type:      strings.ToLower(strings.ReplaceAll(m.run.label, " ", "_")),
		Completed: true,
	}
	m.history = append(m.history, history)
	_ = m.store.SaveHistory(m.history)
	m.notifyAsync(model.NotificationEvent{
		Type:       model.NotificationBreakComplete,
		Timestamp:  time.Now(),
		SessionNum: m.run.sessionIndex,
		PhaseType:  strings.ToLower(strings.ReplaceAll(m.run.label, " ", "_")),
		TaskID:     m.run.taskID,
		Message:    "Waktu istirahat selesai. Kembali fokus.",
	})
	if m.run.sessionIndex < m.run.totalSessions {
		m.run.sessionIndex++
		if skip {
			m.status = "Oke, break dilewati. Lanjut fokus berikutnya!"
		} else {
			m.status = fmt.Sprintf("Siap! Fokus sesi %d/%d dimulai.", m.run.sessionIndex, m.run.totalSessions)
		}
		m.run.phase = runPhaseFocus
		m.run.label = "FOCUS"
		m.run.remaining = time.Duration(m.config.FocusMinutes) * time.Minute
		m.run.phaseDuration = time.Duration(m.config.FocusMinutes) * time.Minute
		m.run.startedAt = time.Now()
		m.run.paused = false
		m.run.notifiedWarning = false
		m.tickID++
		return m, runTickCmd(m.tickID)
	}
	m.status = "Keren! Semua sesi pada cycle ini sudah tuntas."
	m.run = nil
	m.mode = modeDashboard
	m.tickID++
	return m.reload()
}

func (m *Model) updateDashboard(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyIs(msg, "ctrl+c"):
		return m, tea.Quit
	case keyIs(msg, m.config.Keys.Quit, "esc"):
		return m, tea.Quit
	case keyIs(msg, m.config.Keys.NavUp, m.config.Keys.AltNavUp):
		if len(m.tasks.Tasks) > 0 && m.cursor > 0 {
			m.cursor--
		}
	case keyIs(msg, m.config.Keys.NavDown, m.config.Keys.AltNavDown):
		if len(m.tasks.Tasks) > 0 && m.cursor < len(m.tasks.Tasks)-1 {
			m.cursor++
		}
	case keyIs(msg, m.config.Keys.ReorderUp):
		m.moveTask(-1)
	case keyIs(msg, m.config.Keys.ReorderDown):
		m.moveTask(1)
	case keyIs(msg, m.config.Keys.AddTask):
		m.openForm(formAdd, nil)
	case keyIs(msg, m.config.Keys.EditTask):
		if task := m.selectedTask(); task != nil {
			m.openForm(formEdit, task)
		}
	case keyIs(msg, m.config.Keys.DeleteTask):
		if task := m.selectedTask(); task != nil {
			m.mode = modeConfirmDelete
			m.confirm = &confirmState{taskID: task.ID}
		}
	case keyIs(msg, m.config.Keys.ToggleDone):
		if task := m.selectedTask(); task != nil {
			m.toggleTask(task.ID)
		}
	case keyIs(msg, m.config.Keys.StartCycle):
		return m.startSelectedCycle()
	case keyIs(msg, m.config.Keys.Settings):
		m.openForm(formConfig, nil)
	case keyIs(msg, m.config.Keys.Refresh):
		return m.reload()
	}
	return m, nil
}

func (m *Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.form == nil {
		m.mode = modeDashboard
		return m, nil
	}
	current := &m.form.fields[m.form.index].input
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))):
		m.mode = modeDashboard
		m.form = nil
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next"))):
		m.form.index = (m.form.index + 1) % len(m.form.fields)
		for i := range m.form.fields {
			m.form.fields[i].input.Blur()
		}
		m.form.fields[m.form.index].input.Focus()
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev"))):
		m.form.index--
		if m.form.index < 0 {
			m.form.index = len(m.form.fields) - 1
		}
		for i := range m.form.fields {
			m.form.fields[i].input.Blur()
		}
		m.form.fields[m.form.index].input.Focus()
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save"))):
		return m.submitForm()
	}
	updated, cmd := current.Update(msg)
	m.form.fields[m.form.index].input = updated
	return m, cmd
}

func (m *Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		if m.confirm != nil {
			m.deleteTask(m.confirm.taskID)
		}
		m.confirm = nil
		m.mode = modeDashboard
		return m, nil
	case "n", "N", "esc":
		m.confirm = nil
		m.mode = modeDashboard
		return m, nil
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) updateRun(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyIs(msg, "ctrl+c", "esc", m.config.Keys.Quit):
		m.status = "Cycle dihentikan. Tidak apa-apa, kamu bisa lanjut kapan saja."
		m.mode = modeDashboard
		m.run = nil
		m.tickID++
		return m.reload()
	}
	if m.run == nil {
		return m, nil
	}
	switch {
	case keyIs(msg, m.config.Keys.Pause, "space"):
		m.run.paused = !m.run.paused
		if m.run.paused {
			m.status = "Timer dijeda. Tarik napas, lanjut saat siap."
		} else {
			m.status = "Lanjut lagi. Fokusmu mantap!"
		}
		return m, nil
	case keyIs(msg, m.config.Keys.EndPhase):
		if m.run.phase == runPhaseFocus {
			return m.finishFocusSession(true)
		}
		return m.finishBreakSession(true)
	case keyIs(msg, m.config.Keys.NextPhase):
		if m.run.phase == runPhaseBreak {
			return m.finishBreakSession(true)
		}
	}
	return m, nil
}

func (m *Model) handleRunTick(msg runTickMsg) (tea.Model, tea.Cmd) {
	if msg.token != m.tickID {
		return m, nil
	}
	if m.run == nil {
		m.mode = modeDashboard
		return m, nil
	}
	if m.run.paused {
		return m, runTickCmd(m.tickID)
	}
	m.run.remaining -= time.Second
	if m.run.phase == runPhaseFocus && !m.run.notifiedWarning && m.config.Notifications != nil {
		warnAt := time.Duration(m.config.Notifications.WarningMinutesBefore) * time.Minute
		if warnAt > 0 && m.run.remaining > 0 && m.run.remaining <= warnAt {
			m.run.notifiedWarning = true
			m.notifyAsync(model.NotificationEvent{
				Type:       model.NotificationSessionWarn,
				Timestamp:  time.Now(),
				SessionNum: m.run.sessionIndex,
				PhaseType:  "focus",
				TaskID:     m.run.taskID,
				Message:    fmt.Sprintf("Sisa fokus %d menit.", m.config.Notifications.WarningMinutesBefore),
			})
		}
	}
	if m.run.remaining <= 0 {
		if m.run.phase == runPhaseFocus {
			return m.finishFocusSession(false)
		}
		return m.finishBreakSession(false)
	}
	return m, runTickCmd(m.tickID)
}

func (m *Model) ViewDashboard() string { return m.viewDashboard() }

func (m *Model) viewDashboard() string {
	left := m.renderTaskList()
	right := m.renderSummary()
	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	header := m.renderHeader()
	footer := helpStyle.Render(fmt.Sprintf("%s/%s navigate  %s/%s reorder  %s run  %s add  %s edit  %s delete  %s done  %s config  %s refresh  %s quit",
		m.config.Keys.NavUp,
		m.config.Keys.NavDown,
		m.config.Keys.ReorderUp,
		m.config.Keys.ReorderDown,
		m.config.Keys.StartCycle,
		m.config.Keys.AddTask,
		m.config.Keys.EditTask,
		m.config.Keys.DeleteTask,
		m.config.Keys.ToggleDone,
		m.config.Keys.Settings,
		m.config.Keys.Refresh,
		m.config.Keys.Quit,
	))
	footer = "Arrows: ↑/↓  Alt nav: j/k\n" + footer
	if m.status != "" {
		footer = footer + "\n" + warnStyle.Render("Status: "+m.status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (m *Model) renderHeader() string {
	now := time.Now().Format("Mon, 02 Jan 2006 15:04:05")
	left := appTitleStyle.Render("focus-cli  interactive pomodoro")
	right := dimStyle.Render(now)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", max(2, m.width-lipgloss.Width(left)-lipgloss.Width(right)-2)), right)
}

func (m *Model) renderTaskList() string {
	rows := []string{accentStyle.Render("Tasks")}
	if len(m.tasks.Tasks) == 0 {
		rows = append(rows, dimStyle.Render("No tasks yet. Press a to add one."))
	} else {
		for i, task := range m.tasks.Tasks {
			marker := "  "
			if i == m.cursor {
				marker = "→ "
			}
			prefix := "  "
			if task.Done {
				prefix = "✓ "
			}
			content := fmt.Sprintf("%s%s%02d. %s [%d/%d]", marker, prefix, i+1, task.Title, task.CompletedPomodoros, task.TargetSessions)
			switch {
			case i == m.cursor && task.Done:
				rows = append(rows, selectedDoneStyle.Render(content))
			case i == m.cursor:
				rows = append(rows, selectedStyle.Render(content))
			case task.Done:
				rows = append(rows, doneTaskStyle.Render(content))
			default:
				rows = append(rows, content)
			}
		}
	}
	width := 52
	if m.width > 110 {
		width = 62
	}
	return panelStyle.Width(width).Render(strings.Join(rows, "\n"))
}

func (m *Model) renderSummary() string {
	selected := "No task selected"
	if task := m.selectedTask(); task != nil {
		status := "todo"
		if task.Done {
			status = "done"
		}
		remaining := task.TargetSessions - task.CompletedPomodoros
		if remaining < 0 {
			remaining = 0
		}
		selected = fmt.Sprintf("ID: %d\nTitle: %s\nStatus: %s\nProgress: %d/%d\nRemaining sessions: %d\nDesc: %s", task.ID, task.Title, status, task.CompletedPomodoros, task.TargetSessions, remaining, blankIf(task.Description))
	}
	summary := fmt.Sprintf("Config\nFocus: %d min\nShort break: %d min\nLong break: %d min\nLong every: %d\n\nHistory\nFocus done: %d\nBreak sessions: %d", m.config.FocusMinutes, m.config.ShortBreakMinutes, m.config.LongBreakMinutes, m.config.LongBreakEvery, m.countCompletedFocus(), m.countCompletedBreaks())
	return lipgloss.JoinVertical(lipgloss.Left, panelStyle.Width(34).Render(selected), panelStyle.Width(34).Render(summary))
}

func (m *Model) viewForm() string {
	if m.form == nil {
		return m.viewDashboard()
	}
	lines := []string{}
	title := ""
	switch m.form.kind {
	case formAdd:
		title = "Add task"
	case formEdit:
		title = "Edit task"
	case formConfig:
		title = "Update config"
	}
	lines = append(lines, appTitleStyle.Render(title))
	for i := range m.form.fields {
		field := m.form.fields[i]
		label := dimStyle.Render(field.label)
		if i == m.form.index {
			label = accentStyle.Render(field.label)
		}
		lines = append(lines, fmt.Sprintf("%s\n%s", label, field.input.View()))
	}
	lines = append(lines, helpStyle.Render("tab/shift+tab pindah field  enter simpan  esc batal"))
	return panelStyle.Render(strings.Join(lines, "\n\n"))
}

func (m *Model) viewConfirm() string {
	if m.confirm == nil {
		return m.viewDashboard()
	}
	return panelStyle.Render(fmt.Sprintf("Delete task #%d?\n\ny = delete  n = cancel", m.confirm.taskID))
}

func (m *Model) viewRunning() string {
	if m.run == nil {
		return m.viewDashboard()
	}
	remaining := m.run.remaining
	mins := int(remaining / time.Minute)
	secs := int((remaining % time.Minute) / time.Second)
	bar := fmt.Sprintf("%02d:%02d", mins, secs)
	phase := m.run.label
	if m.run.phase == runPhaseFocus {
		phase = "FOCUS"
	}
	progress := 0.0
	if m.run.phaseDuration > 0 {
		progress = float64(m.run.phaseDuration-m.run.remaining) / float64(m.run.phaseDuration)
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	line := fmt.Sprintf("%s\nSession %d/%d\n%s %.0f%%", phase+"  "+bar, m.run.sessionIndex, m.run.totalSessions, progressBar(progress, 26), progress*100)
	if m.run.paused {
		line += "\n" + badStyle.Render("PAUSED")
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		appTitleStyle.Render("Session running"),
		panelStyle.Width(52).Render(line+fmt.Sprintf("\n\nControls:\n  %s/space pause-resume\n  %s end current phase\n  %s next focus (when break)\n  %s stop cycle", m.config.Keys.Pause, m.config.Keys.EndPhase, m.config.Keys.NextPhase, m.config.Keys.Quit)),
	)
}

func (m *Model) openForm(kind formKind, task *model.Task) {
	fields := []formField{}
	switch kind {
	case formAdd:
		fields = []formField{
			newTextField("Title", "", true),
			newTextField("Target sessions", "1", false),
			newTextField("Description", "", false),
		}
	case formEdit:
		if task == nil {
			return
		}
		fields = []formField{
			newTextField("Title", task.Title, true),
			newTextField("Target sessions", strconv.Itoa(task.TargetSessions), false),
			newTextField("Completed sessions", strconv.Itoa(task.CompletedPomodoros), false),
			newTextField("Description", task.Description, false),
		}
	case formConfig:
		fields = []formField{
			newTextField("Focus minutes", strconv.Itoa(m.config.FocusMinutes), true),
			newTextField("Short break minutes", strconv.Itoa(m.config.ShortBreakMinutes), false),
			newTextField("Long break minutes", strconv.Itoa(m.config.LongBreakMinutes), false),
			newTextField("Long break every", strconv.Itoa(m.config.LongBreakEvery), false),
			newTextField("Theme (sunrise|forest|mono)", m.config.Theme, false),
		}
	}
	m.form = &formState{kind: kind, fields: fields}
	m.mode = modeForm
}

func newTextField(label, value string, focused bool) formField {
	i := textinput.New()
	i.SetValue(value)
	i.Prompt = ""
	i.CharLimit = 128
	i.Width = 42
	if focused {
		i.Focus()
	}
	return formField{label: label, input: i}
}

func (m *Model) submitForm() (tea.Model, tea.Cmd) {
	if m.form == nil {
		m.mode = modeDashboard
		return m, nil
	}
	switch m.form.kind {
	case formAdd:
		title := strings.TrimSpace(m.form.fields[0].input.Value())
		if title == "" {
			m.status = "title tidak boleh kosong"
			return m, nil
		}
		target, err := strconv.Atoi(strings.TrimSpace(m.form.fields[1].input.Value()))
		if err != nil || target < 1 {
			m.status = "target harus angka >= 1"
			return m, nil
		}
		desc := strings.TrimSpace(m.form.fields[2].input.Value())
		if err := m.addTask(title, desc, target); err != nil {
			m.status = err.Error()
			return m, nil
		}
	case formEdit:
		task := m.selectedTask()
		if task == nil {
			m.status = "task tidak ditemukan"
			return m, nil
		}
		title := strings.TrimSpace(m.form.fields[0].input.Value())
		target, err := strconv.Atoi(strings.TrimSpace(m.form.fields[1].input.Value()))
		if err != nil || target < 1 {
			m.status = "target harus angka >= 1"
			return m, nil
		}
		completed, err := strconv.Atoi(strings.TrimSpace(m.form.fields[2].input.Value()))
		if err != nil || completed < 0 {
			m.status = "completed sessions harus angka >= 0"
			return m, nil
		}
		desc := strings.TrimSpace(m.form.fields[3].input.Value())
		if err := m.editTask(task.ID, title, desc, target, completed); err != nil {
			m.status = err.Error()
			return m, nil
		}
	case formConfig:
		vals := make([]int, 4)
		for i := 0; i < 4; i++ {
			v, err := strconv.Atoi(strings.TrimSpace(m.form.fields[i].input.Value()))
			if err != nil || v < 1 {
				m.status = "semua config harus angka >= 1"
				return m, nil
			}
			vals[i] = v
		}
		m.config.FocusMinutes = vals[0]
		m.config.ShortBreakMinutes = vals[1]
		m.config.LongBreakMinutes = vals[2]
		m.config.LongBreakEvery = vals[3]
		theme := strings.ToLower(strings.TrimSpace(m.form.fields[4].input.Value()))
		if theme != "sunrise" && theme != "forest" && theme != "mono" {
			m.status = "theme harus sunrise|forest|mono"
			return m, nil
		}
		m.config.Theme = theme
		if err := m.store.SaveConfig(m.config); err != nil {
			m.status = err.Error()
			return m, nil
		}
		applyTheme(m.config.Theme)
		m.status = "config tersimpan"
		m.reload()
	}
	m.form = nil
	m.mode = modeDashboard
	return m.reload()
}

func (m *Model) addTask(title, desc string, target int) error {
	now := time.Now()
	task := model.Task{ID: m.tasks.NextID, Title: title, Description: desc, TargetSessions: target, CreatedAt: now, UpdatedAt: now}
	m.tasks.NextID++
	m.tasks.Tasks = append(m.tasks.Tasks, task)
	return m.store.SaveTasks(m.tasks)
}

func (m *Model) editTask(id int, title, desc string, target int, completed int) error {
	for i := range m.tasks.Tasks {
		if m.tasks.Tasks[i].ID == id {
			m.tasks.Tasks[i].Title = title
			m.tasks.Tasks[i].Description = desc
			m.tasks.Tasks[i].TargetSessions = target
			m.tasks.Tasks[i].CompletedPomodoros = completed
			m.tasks.Tasks[i].Done = completed >= target
			m.tasks.Tasks[i].UpdatedAt = time.Now()
			return m.store.SaveTasks(m.tasks)
		}
	}
	return fmt.Errorf("task #%d not found", id)
}

func (m *Model) deleteTask(id int) {
	out := m.tasks.Tasks[:0]
	for _, task := range m.tasks.Tasks {
		if task.ID == id {
			continue
		}
		out = append(out, task)
	}
	m.tasks.Tasks = out
	_ = m.store.SaveTasks(m.tasks)
	if m.cursor >= len(m.tasks.Tasks) && m.cursor > 0 {
		m.cursor--
	}
}

func (m *Model) toggleTask(id int) {
	for i := range m.tasks.Tasks {
		if m.tasks.Tasks[i].ID == id {
			m.tasks.Tasks[i].Done = !m.tasks.Tasks[i].Done
			m.tasks.Tasks[i].UpdatedAt = time.Now()
			_ = m.store.SaveTasks(m.tasks)
			if m.tasks.Tasks[i].Done {
				m.status = fmt.Sprintf("Yeay! Task '%s' ditandai selesai.", m.tasks.Tasks[i].Title)
			} else {
				m.status = fmt.Sprintf("Task '%s' dibuka lagi. Lanjutkan, kamu pasti bisa!", m.tasks.Tasks[i].Title)
			}
			return
		}
	}
}

func (m *Model) selectedTask() *model.Task {
	if len(m.tasks.Tasks) == 0 || m.cursor < 0 || m.cursor >= len(m.tasks.Tasks) {
		return nil
	}
	return &m.tasks.Tasks[m.cursor]
}

func (m *Model) moveTask(delta int) {
	if len(m.tasks.Tasks) < 2 || m.cursor < 0 || m.cursor >= len(m.tasks.Tasks) {
		return
	}
	next := m.cursor + delta
	if next < 0 || next >= len(m.tasks.Tasks) {
		return
	}
	m.tasks.Tasks[m.cursor], m.tasks.Tasks[next] = m.tasks.Tasks[next], m.tasks.Tasks[m.cursor]
	m.cursor = next
	if err := m.store.SaveTasks(m.tasks); err != nil {
		m.status = err.Error()
		return
	}
	m.status = fmt.Sprintf("task moved to position %d", next+1)
}

func (m *Model) startSelectedCycle() (tea.Model, tea.Cmd) {
	totalSessions := 1
	if task := m.selectedTask(); task != nil {
		if task.Done {
			m.status = fmt.Sprintf("Task '%s' sudah selesai. Undo dulu dengan %s kalau mau lanjut sesi.", task.Title, m.config.Keys.ToggleDone)
			return m, nil
		}
		remaining := task.TargetSessions - task.CompletedPomodoros
		if remaining > 0 {
			totalSessions = remaining
		}
		return m.beginFocusCycle(task.ID, totalSessions)
	}
	return m.beginFocusCycle(0, totalSessions)
}

func (m *Model) reload() (tea.Model, tea.Cmd) {
	tasks, err := m.store.LoadTasks()
	if err == nil {
		m.tasks = tasks
	}
	config, err := m.store.LoadConfig()
	if err == nil {
		m.config = config
		applyTheme(m.config.Theme)
	}
	history, err := m.store.LoadHistory()
	if err == nil {
		m.history = history
	}
	if m.cursor >= len(m.tasks.Tasks) && len(m.tasks.Tasks) > 0 {
		m.cursor = len(m.tasks.Tasks) - 1
	}
	if len(m.tasks.Tasks) == 0 {
		m.cursor = 0
	}
	return m, nil
}

func (m *Model) countCompletedFocus() int {
	count := 0
	for _, entry := range m.history {
		if entry.Type == "focus" && entry.Completed {
			count++
		}
	}
	return count
}

func (m *Model) countCompletedBreaks() int {
	count := 0
	for _, entry := range m.history {
		if entry.Completed && (entry.Type == "short_break" || entry.Type == "long_break") {
			count++
		}
	}
	return count
}

func progressBar(p float64, width int) string {
	filled := int(p * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func blankIf(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
