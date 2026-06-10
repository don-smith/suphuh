package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"suphuh/internal/appstate"
	"suphuh/internal/tmux"
)

type panePreviewMsg struct {
	paneID  string
	content string
	err     error
}

type jumpDoneMsg struct {
	err error
}

type refreshTickMsg time.Time

type panesRefreshedMsg struct {
	panes []tmux.Pane
	err   error
}

type ViewMode string

const (
	ViewAll         ViewMode = "all"
	ViewAgentsFirst ViewMode = "agents-first"
)

type Model struct {
	panes           []tmux.Pane
	selected        int
	viewMode        ViewMode
	preview         string
	previewViewport viewport.Model
	err             error
	width           int
	height          int
	jumping         bool
}

var (
	borderColor = lipgloss.Color("63")
	mutedColor  = lipgloss.Color("241")
	accentColor = lipgloss.Color("170")
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	mutedStyle  = lipgloss.NewStyle().Foreground(mutedColor)
	listStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
	paneStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
	selected    = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(accentColor).Bold(true)
)

const refreshInterval = 750 * time.Millisecond

func New(panes []tmux.Pane) Model {
	vp := viewport.New(80, 20)
	state := appstate.Load()
	m := Model{panes: panes, previewViewport: vp, viewMode: normalizeViewMode(state.View)}
	m.applyView()
	m.selectPane(state.SelectedPaneID)
	return m
}

func Run(panes []tmux.Pane) error {
	_, err := tea.NewProgram(New(panes), tea.WithAltScreen()).Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.HideCursor, m.loadPreview(), scheduleRefresh())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updatePreviewViewport()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.selected < len(m.panes)-1 {
				m.selected++
				m.saveState()
				return m, m.loadPreview()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.saveState()
				return m, m.loadPreview()
			}
		case "ctrl+d", "pgdown":
			m.previewViewport.HalfViewDown()
			return m, nil
		case "ctrl+u", "pgup":
			m.previewViewport.HalfViewUp()
			return m, nil
		case "v":
			selectedPaneID := m.selectedPaneID()
			m.viewMode = m.viewMode.Next()
			m.applyView()
			m.selectPane(selectedPaneID)
			m.saveState()
			return m, m.loadPreview()
		case "enter":
			if len(m.panes) > 0 {
				m.jumping = true
				return m, m.jumpToSelected()
			}
		}
	case panePreviewMsg:
		if len(m.panes) == 0 || msg.paneID != m.panes[m.selected].PaneID {
			return m, nil
		}
		m.err = msg.err
		m.preview = msg.content
		m.updatePreviewViewport()
		m.previewViewport.GotoBottom()
		return m, nil
	case refreshTickMsg:
		return m, refreshPanes()
	case panesRefreshedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, scheduleRefresh()
		}
		m.err = nil
		m.replacePanes(msg.panes)
		return m, tea.Batch(m.loadPreview(), scheduleRefresh())
	case jumpDoneMsg:
		m.err = msg.err
		m.jumping = false
		if msg.err == nil {
			return m, tea.Quit
		}
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if len(m.panes) == 0 {
		return "No tmux panes found.\n\nq: quit"
	}

	width := m.width
	if width <= 0 {
		width = 100
	}
	height := m.height
	if height <= 0 {
		height = 30
	}

	leftWidth, rightWidth, bodyHeight := layout(width, height)

	left := renderBox(listStyle, leftWidth, bodyHeight, m.renderList(leftWidth, bodyHeight))
	right := renderBox(paneStyle, rightWidth, bodyHeight, m.previewViewport.View())

	help := mutedStyle.Render(fmt.Sprintf("view: %s • v: switch • j/k: move • ctrl-u/d: scroll • enter: jump • q/esc: close", m.viewMode.Label()))
	if m.jumping {
		help = mutedStyle.Render("jumping…")
	}
	if m.err != nil {
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(m.err.Error())
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		fitLine(titleStyle.Render("suphuh"), width),
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		fitLine(help, width),
	)
}

func (m Model) renderList(width int, height int) string {
	var b strings.Builder
	visible := m.visiblePanes(height)
	for i, pane := range visible {
		idx := m.listStart(height) + i
		glyph := statusGlyph(pane, idx == m.selected)
		line := fmt.Sprintf("%s %-16s %-9s",
			glyph,
			truncate(pane.SessionName, 16),
			truncate(displayCommand(pane), 9),
		)
		line = fitLine(line, width)
		if idx == m.selected {
			line = selected.Render(ansi.Strip(line))
		}
		b.WriteString(line)
		if i < len(visible)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m *Model) updatePreviewViewport() {
	if len(m.panes) == 0 {
		return
	}

	_, rightWidth, bodyHeight := layout(m.effectiveWidth(), m.effectiveHeight())
	m.previewViewport.Width = rightWidth
	m.previewViewport.Height = bodyHeight
	m.previewViewport.SetContent(m.renderPreviewContent(rightWidth))
}

func (m Model) renderPreviewContent(width int) string {
	pane := m.panes[m.selected]
	header := fmt.Sprintf("%s:%d.%d %s", pane.SessionName, pane.WindowIndex, pane.PaneIndex, pane.CurrentPath)
	lines := []string{titleStyle.Render(truncate(header, max(1, width)))}

	for _, line := range strings.Split(strings.TrimRight(m.preview, "\n"), "\n") {
		line = cleanPreviewLine(line)
		lines = append(lines, truncate(line, max(1, width)))
	}

	if len(lines) == 1 {
		lines = append(lines, mutedStyle.Render("No captured output."))
	}

	return strings.Join(lines, "\n")
}

func (m Model) effectiveWidth() int {
	if m.width <= 0 {
		return 100
	}
	return m.width
}

func (m Model) effectiveHeight() int {
	if m.height <= 0 {
		return 30
	}
	return m.height
}

func layout(width, height int) (leftWidth int, rightWidth int, bodyHeight int) {
	leftOuter := max(30, width/3)
	rightOuter := max(40, width-leftOuter-2)
	bodyOuter := max(8, height-2)

	leftWidth = max(1, leftOuter-listStyle.GetHorizontalFrameSize())
	rightWidth = max(1, rightOuter-paneStyle.GetHorizontalFrameSize())
	bodyHeight = max(1, bodyOuter-paneStyle.GetVerticalFrameSize())
	return leftWidth, rightWidth, bodyHeight
}

func (m Model) visiblePanes(height int) []tmux.Pane {
	start := m.listStart(height)
	end := min(len(m.panes), start+height)
	return m.panes[start:end]
}

func (m Model) listStart(height int) int {
	if len(m.panes) <= height {
		return 0
	}
	start := m.selected - height/2
	if start < 0 {
		return 0
	}
	if start+height > len(m.panes) {
		return len(m.panes) - height
	}
	return start
}

func (m *Model) replacePanes(panes []tmux.Pane) {
	selectedPaneID := m.selectedPaneID()
	m.panes = panes
	m.applyView()
	m.selectPane(selectedPaneID)
	m.updatePreviewViewport()
}

func (m *Model) applyView() {
	if m.viewMode != ViewAgentsFirst {
		return
	}
	sort.SliceStable(m.panes, func(i, j int) bool {
		return isAgentPane(m.panes[i]) && !isAgentPane(m.panes[j])
	})
}

func (m *Model) selectPane(paneID string) {
	if len(m.panes) == 0 {
		m.selected = 0
		m.preview = ""
		m.updatePreviewViewport()
		return
	}

	m.selected = min(m.selected, len(m.panes)-1)
	if paneID == "" {
		return
	}
	for i, pane := range m.panes {
		if pane.PaneID == paneID {
			m.selected = i
			return
		}
	}
}

func (m Model) selectedPaneID() string {
	if len(m.panes) == 0 || m.selected < 0 || m.selected >= len(m.panes) {
		return ""
	}
	return m.panes[m.selected].PaneID
}

func (m Model) saveState() {
	_ = appstate.Save(appstate.State{SelectedPaneID: m.selectedPaneID(), View: string(m.viewMode)})
}

func scheduleRefresh() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return refreshTickMsg(t)
	})
}

func refreshPanes() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		panes, err := tmux.ListPanes(ctx)
		return panesRefreshedMsg{panes: panes, err: err}
	}
}

func (m Model) loadPreview() tea.Cmd {
	if len(m.panes) == 0 {
		return nil
	}
	paneID := m.panes[m.selected].PaneID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		content, err := tmux.CapturePane(ctx, paneID, 120)
		return panePreviewMsg{paneID: paneID, content: content, err: err}
	}
}

func (m Model) jumpToSelected() tea.Cmd {
	pane := m.panes[m.selected]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return jumpDoneMsg{err: tmux.JumpToPane(ctx, pane)}
	}
}

func normalizeViewMode(view string) ViewMode {
	switch ViewMode(view) {
	case ViewAgentsFirst:
		return ViewAgentsFirst
	default:
		return ViewAll
	}
}

func (v ViewMode) Next() ViewMode {
	if v == ViewAgentsFirst {
		return ViewAll
	}
	return ViewAgentsFirst
}

func (v ViewMode) Label() string {
	switch v {
	case ViewAgentsFirst:
		return "agents first"
	default:
		return "all"
	}
}

func isAgentPane(pane tmux.Pane) bool {
	switch displayCommand(pane) {
	case "pi", "claude", "codex", "aider", "goose", "opencode":
		return true
	default:
		return false
	}
}

func statusGlyph(pane tmux.Pane, selectedRow bool) string {
	if !isAgentPane(pane) {
		return " "
	}
	if !pane.HasStatus {
		if selectedRow {
			return "·"
		}
		return mutedStyle.Render("·")
	}

	glyph := "?"
	style := mutedStyle
	switch pane.Status.State {
	case "working":
		glyph = "●"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	case "blocked":
		glyph = "◆"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	case "idle":
		glyph = "✓"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	}
	if selectedRow {
		return glyph
	}
	return style.Render(glyph)
}

func displayCommand(pane tmux.Pane) string {
	if pane.DisplayCommand != "" {
		return pane.DisplayCommand
	}
	return pane.CurrentCommand
}

func renderBox(style lipgloss.Style, width int, height int, content string) string {
	content = fitBox(content, width, height)
	content = lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, content)
	return style.Width(width).Height(height).Render(content)
}

func fitBox(s string, width int, height int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > height {
		lines = lines[:height]
	}

	for i, line := range lines {
		lines[i] = fitLine(line, width)
	}

	return strings.Join(lines, "\n")
}

func cleanPreviewLine(s string) string {
	s = ansi.Strip(s)
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "    ")

	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' {
			return -1
		}
		return r
	}, s)
}

func fitLine(s string, width int) string {
	if width <= 0 {
		return ""
	}

	s = truncate(s, width)
	padding := width - ansi.StringWidth(s)
	if padding > 0 {
		s += strings.Repeat(" ", padding)
	}
	return s
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 || ansi.StringWidth(s) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return "…"
	}
	return ansi.Truncate(s, maxLen, "…")
}
