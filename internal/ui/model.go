package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/uikamu/pomodoro-tui/internal/config"
	"github.com/uikamu/pomodoro-tui/internal/storage"
)

type viewMode int

const (
	viewTimer viewMode = iota
	viewHistory
	viewTasks
	viewAddTask
)

type phase int

const (
	phaseWork phase = iota
	phaseShortBreak
	phaseLongBreak
)

func (p phase) label() string {
	switch p {
	case phaseWork:
		return "WORK"
	case phaseShortBreak:
		return "BREAK"
	default:
		return "LONG BREAK"
	}
}

func (p phase) kind() string {
	if p == phaseWork {
		return "work"
	}
	return "break"
}

type tickMsg time.Time

type Model struct {
	cfg   config.Config
	store *storage.Store

	phase      phase
	total      time.Duration
	remaining  time.Duration
	paused     bool
	cyclesDone int
	startedAt  time.Time

	view     viewMode
	history  []storage.Session
	tasks    []storage.Task
	taskCur  int
	taskErr  string
	input    textinput.Model
	progress progress.Model
	quitting bool
}

func New(cfg config.Config, store *storage.Store) Model {
	p := progress.New(progress.WithDefaultGradient())
	ti := textinput.New()
	ti.Placeholder = "新しいタスクを入力..."
	ti.CharLimit = 100
	m := Model{
		cfg:       cfg,
		store:     store,
		phase:     phaseWork,
		total:     cfg.WorkDuration,
		remaining: cfg.WorkDuration,
		startedAt: time.Now(),
		progress:  p,
		input:     ti,
	}
	if store != nil {
		m.tasks, _ = store.ListTasks()
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) durationFor(p phase) time.Duration {
	switch p {
	case phaseWork:
		return m.cfg.WorkDuration
	case phaseShortBreak:
		return m.cfg.BreakDuration
	default:
		return m.cfg.LongBreakDuration
	}
}

func (m *Model) recordCompletion(completed bool) {
	if m.store == nil {
		return
	}
	_ = m.store.Record(storage.Session{
		Kind:      m.phase.kind(),
		Duration:  m.total,
		StartedAt: m.startedAt,
		Completed: completed,
	})
}

func (m *Model) advancePhase() {
	if m.phase == phaseWork {
		m.cyclesDone++
		if m.cyclesDone%m.cfg.CyclesBeforeLong == 0 {
			m.phase = phaseLongBreak
		} else {
			m.phase = phaseShortBreak
		}
	} else {
		m.phase = phaseWork
	}
	m.total = m.durationFor(m.phase)
	m.remaining = m.total
	m.startedAt = time.Now()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 4
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}
		return m, nil

	case tea.KeyMsg:
		if m.view == viewAddTask {
			switch msg.String() {
			case "esc":
				m.view = viewTasks
				m.input.Reset()
				m.input.Blur()
			case "enter":
				title := m.input.Value()
				if title != "" && m.store != nil {
					if _, err := m.store.AddTask(title); err != nil {
						m.taskErr = err.Error()
					} else {
						m.tasks, _ = m.store.ListTasks()
					}
				}
				m.input.Reset()
				m.input.Blur()
				m.view = viewTasks
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "t":
			if m.view == viewTasks {
				m.view = viewTimer
			} else {
				m.view = viewTasks
				if m.store != nil {
					m.tasks, _ = m.store.ListTasks()
				}
				m.taskCur = 0
			}
		case "h":
			if m.view == viewHistory {
				m.view = viewTimer
			} else {
				m.view = viewHistory
				if m.store != nil {
					m.history, _ = m.store.Recent(10)
				}
			}
		}

		if m.view == viewTasks {
			switch msg.String() {
			case "a":
				m.view = viewAddTask
				m.input.Focus()
				return m, nil
			case "up", "k":
				if m.taskCur > 0 {
					m.taskCur--
				}
			case "down", "j":
				if m.taskCur < len(m.tasks)-1 {
					m.taskCur++
				}
			case " ", "x":
				if m.store != nil && m.taskCur < len(m.tasks) {
					_ = m.store.ToggleTask(m.tasks[m.taskCur].ID)
					m.tasks, _ = m.store.ListTasks()
				}
			case "d":
				if m.store != nil && m.taskCur < len(m.tasks) {
					_ = m.store.DeleteTask(m.tasks[m.taskCur].ID)
					m.tasks, _ = m.store.ListTasks()
					if m.taskCur >= len(m.tasks) && m.taskCur > 0 {
						m.taskCur--
					}
				}
			}
			return m, nil
		}

		if m.view == viewTimer {
			switch msg.String() {
			case " ":
				m.paused = !m.paused
			case "r":
				m.remaining = m.total
				m.startedAt = time.Now()
			case "n":
				m.recordCompletion(false)
				m.advancePhase()
			}
		}
		return m, nil

	case tickMsg:
		if !m.paused && m.view == viewTimer {
			m.remaining -= time.Second
			if m.remaining <= 0 {
				m.recordCompletion(true)
				m.advancePhase()
			}
		}
		return m, tick()
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return "ポモドーロを終了しました。お疲れさまでした。\n"
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	switch m.phase {
	case phaseWork:
		titleStyle = titleStyle.Foreground(lipgloss.Color("#FF5F87"))
	case phaseShortBreak:
		titleStyle = titleStyle.Foreground(lipgloss.Color("#5FD7FF"))
	default:
		titleStyle = titleStyle.Foreground(lipgloss.Color("#87FF5F"))
	}

	switch m.view {
	case viewHistory:
		return m.historyView()
	case viewTasks, viewAddTask:
		return m.tasksView()
	}

	pct := 0.0
	if m.total > 0 {
		pct = 1 - float64(m.remaining)/float64(m.total)
	}

	status := "実行中"
	if m.paused {
		status = "一時停止"
	}

	body := fmt.Sprintf(
		"%s\n\n  %s   [%s]\n%s\n\n%s\n\nサイクル: %d  (long breakまで %d/%d)\n\nspace: 一時停止/再開  n: スキップ  r: リセット  h: 履歴  q: 終了",
		titleStyle.Render(m.phase.label()),
		formatDuration(m.remaining),
		status,
		m.topTasksView(),
		m.progress.ViewAs(pct),
		m.cyclesDone,
		m.cyclesDone%m.cfg.CyclesBeforeLong, m.cfg.CyclesBeforeLong,
	) + "\n\nt: タスク一覧"

	return lipgloss.NewStyle().Padding(1, 2).Render(body)
}

func (m Model) topTasksView() string {
	header := lipgloss.NewStyle().Bold(true).Render("タスク (上位3件)")
	if len(m.tasks) == 0 {
		return "\n" + header + "\n  タスクはありません。't' で追加できます。"
	}

	var lines []string
	count := 0
	for _, t := range m.tasks {
		if t.Done {
			continue
		}
		lines = append(lines, fmt.Sprintf("  [ ] %s", t.Title))
		count++
		if count == 3 {
			break
		}
	}
	if count == 0 {
		return "\n" + header + "\n  未完了のタスクはありません。"
	}
	return "\n" + lipgloss.JoinVertical(lipgloss.Left, append([]string{header}, lines...)...)
}

func (m Model) historyView() string {
	header := lipgloss.NewStyle().Bold(true).Render("最近のセッション履歴 (h: 戻る)")
	lines := []string{header, ""}
	if len(m.history) == 0 {
		lines = append(lines, "履歴はまだありません。")
	}
	for _, s := range m.history {
		mark := "✗"
		if s.Completed {
			mark = "✓"
		}
		lines = append(lines, fmt.Sprintf("%s %-5s %5s  %s", mark, s.Kind, formatDuration(s.Duration), s.StartedAt.Format("01/02 15:04")))
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) tasksView() string {
	if m.view == viewAddTask {
		header := lipgloss.NewStyle().Bold(true).Render("新しいタスク (enter: 追加 / esc: キャンセル)")
		return lipgloss.NewStyle().Padding(1, 2).Render(
			lipgloss.JoinVertical(lipgloss.Left, header, "", m.input.View()),
		)
	}

	header := lipgloss.NewStyle().Bold(true).Render("タスク一覧 (a: 追加  space: 完了切替  d: 削除  t: 戻る)")
	lines := []string{header, ""}
	if len(m.tasks) == 0 {
		lines = append(lines, "タスクはまだありません。'a' で追加できます。")
	}
	for i, t := range m.tasks {
		cursor := "  "
		if i == m.taskCur {
			cursor = "> "
		}
		mark := "[ ]"
		if t.Done {
			mark = "[x]"
		}
		lines = append(lines, fmt.Sprintf("%s%s %s", cursor, mark, t.Title))
	}
	if m.taskErr != "" {
		lines = append(lines, "", "エラー: "+m.taskErr)
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", mins, secs)
}
