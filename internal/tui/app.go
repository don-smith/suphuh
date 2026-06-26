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

	"github.com/don-smith/suphuh/internal/appstate"
	"github.com/don-smith/suphuh/internal/status"
	"github.com/don-smith/suphuh/internal/tmux"
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
	spinnerFrame    int
	followPreview   bool
	artIndex        int
}

var (
	borderColor     = lipgloss.Color("63")
	mutedColor      = lipgloss.Color("241")
	accentColor     = lipgloss.Color("170")
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	mutedStyle      = lipgloss.NewStyle().Foreground(mutedColor)
	listStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
	paneStyle       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(borderColor)
	selected        = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(accentColor).Bold(true)
	pillStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(accentColor).Bold(true).Padding(0, 1)
	headerMetaStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
)

const refreshInterval = 200 * time.Millisecond

func New(panes []tmux.Pane) Model {
	vp := viewport.New(80, 20)
	state := appstate.Load()
	m := Model{panes: panes, previewViewport: vp, viewMode: normalizeViewMode(state.View), followPreview: true, artIndex: normalizeArtIndex(state.ArtIndex)}
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
				m.followPreview = true
				m.saveState()
				return m, m.loadPreview()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.followPreview = true
				m.saveState()
				return m, m.loadPreview()
			}
		case "J":
			m.previewViewport.LineDown(1)
			m.followPreview = m.previewViewport.AtBottom()
			return m, nil
		case "K":
			m.previewViewport.LineUp(1)
			m.followPreview = false
			return m, nil
		case "v":
			selectedPaneID := m.selectedPaneID()
			m.viewMode = m.viewMode.Next()
			m.applyView()
			m.selectPane(selectedPaneID)
			m.followPreview = true
			m.saveState()
			return m, m.loadPreview()
		case "?":
			m.artIndex = (m.artIndex + 1) % len(brandingArts)
			m.saveState()
			return m, nil
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
		wasFollowing := m.followPreview || m.previewViewport.AtBottom()
		m.updatePreviewViewport()
		if wasFollowing {
			m.previewViewport.GotoBottom()
			m.followPreview = true
		}
		return m, nil
	case refreshTickMsg:
		m.spinnerFrame++
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
	right := renderBox(paneStyle, rightWidth, bodyHeight, m.renderPreviewPane(rightWidth, bodyHeight))

	help := mutedStyle.Render(fmt.Sprintf("view: %s • v: switch • ?: art • j/k: move • K/J: scroll • enter: jump • q/esc: close", m.viewMode.Label()))
	if m.jumping {
		help = mutedStyle.Render("jumping…")
	}
	if m.err != nil {
		help = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(m.err.Error())
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.renderTitleBar(width),
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		insetLine(help, width),
	)
}

func (m Model) renderTitleBar(width int) string {
	const padding = 2
	innerWidth := max(1, width-padding*2)
	brandText := "sup?huh?"
	counts := m.agentCounts()
	rightText := fmt.Sprintf("%d panes • %d agents • %d working", len(m.panes), counts.agents, counts.working)

	availableRight := innerWidth - ansi.StringWidth(brandText) - 1
	if availableRight < 1 {
		return insetLine(titleStyle.Render(brandText), width)
	}
	rightText = truncate(rightText, availableRight)
	gap := max(1, innerWidth-ansi.StringWidth(brandText)-ansi.StringWidth(rightText))

	line := strings.Repeat(" ", padding) + titleStyle.Render(brandText) + strings.Repeat(" ", gap) + mutedStyle.Render(rightText) + strings.Repeat(" ", padding)
	return fitLine(line, width)
}

func (m Model) renderList(width int, height int) string {
	lines := make([]string, 0, height)
	visible := m.visiblePanes(height)
	start := m.listStart(height)
	separatorAfter := m.agentGroupEnd()
	for i, pane := range visible {
		idx := start + i
		if m.viewMode == ViewAgentsFirst && idx == separatorAfter && len(lines) < height {
			lines = append(lines, mutedStyle.Render(strings.Repeat("─", max(0, width))))
		}

		isSelected := idx == m.selected
		glyph := m.statusGlyph(pane, isSelected)
		session := fitLine(truncate(pane.SessionName, 16), 16)
		command := fitLine(truncate(displayCommand(pane), 9), 9)
		if !isSelected {
			session = sessionStyle(pane.SessionName).Render(session)
		}
		line := fitLine(fmt.Sprintf("%s %s %s", glyph, session, command), width)
		if isSelected {
			line = selected.Render(ansi.Strip(line))
		}
		lines = append(lines, line)
	}

	lines = m.addBranding(lines, width, height)
	return strings.Join(lines, "\n")
}

var brandingArts = [][]string{
	{
		"██████╗ ",
		"╚═══██║",
		"  ▄██╔╝",
		"  ▀▀═╝ ",
		"  ██╗  ",
		"  ╚═╝  ",
	},
	{
		` ________   `,
		`("      "\  `,
		` \___/   :) `,
		`   /  ___/  `,
		`  //  \     `,
		` ('___/     `,
		`  (___)     `,
	},
}

func normalizeArtIndex(index int) int {
	if len(brandingArts) == 0 || index < 0 || index >= len(brandingArts) {
		return 0
	}
	return index
}

func (m Model) addBranding(lines []string, width int, height int) []string {
	art := brandingArts[normalizeArtIndex(m.artIndex)]
	if len(lines) > height-len(art) || width < 18 {
		return lines
	}
	for len(lines) < height-len(art) {
		lines = append(lines, "")
	}
	styles := []lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("207")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("171")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("135")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true),
		lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true),
	}
	for i, line := range art {
		lines = append(lines, styles[i%len(styles)].Render(centerText(line, width)))
	}
	return lines
}

func (m *Model) updatePreviewViewport() {
	if len(m.panes) == 0 {
		return
	}

	_, rightWidth, bodyHeight := layout(m.effectiveWidth(), m.effectiveHeight())
	m.previewViewport.Width = rightWidth
	m.previewViewport.Height = previewViewportHeight(bodyHeight)
	m.previewViewport.SetContent(m.renderPreviewContent(rightWidth))
}

func (m Model) renderPreviewPane(width int, height int) string {
	if len(m.panes) == 0 {
		return fitBox(mutedStyle.Render("No pane selected."), width, height)
	}

	headerLines := m.renderPreviewHeader(width)
	viewportHeight := previewViewportHeight(height)
	m.previewViewport.Height = viewportHeight
	body := fitBox(m.previewViewport.View(), width, viewportHeight)

	parts := append(headerLines, strings.Split(body, "\n")...)
	return fitBox(strings.Join(parts, "\n"), width, height)
}

func (m Model) renderPreviewHeader(width int) []string {
	pane := m.panes[m.selected]
	path := pane.CurrentPath
	command := displayCommand(pane)
	metaWidth := max(1, width-ansi.StringWidth(command)-4)
	meta := headerMetaStyle.Render(truncate(fmt.Sprintf("%s  %s", pane.SessionName, path), metaWidth))
	location := pillStyle.Render(command) + " " + meta
	return []string{
		fitLine(location, width),
		mutedStyle.Render(strings.Repeat("─", max(0, width))),
	}
}

func previewViewportHeight(totalHeight int) int {
	return max(1, totalHeight-2)
}

func (m Model) renderPreviewContent(width int) string {
	lines := make([]string, 0, 120)
	for _, line := range strings.Split(strings.TrimRight(m.preview, "\n"), "\n") {
		line = cleanPreviewLine(line)
		lines = append(lines, truncate(line, max(1, width)))
	}

	if len(lines) == 0 {
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
	rightOuter := max(40, width-leftOuter-1)
	bodyOuter := max(8, height-2)

	leftWidth = max(1, leftOuter-listStyle.GetHorizontalFrameSize())
	rightWidth = max(1, rightOuter-paneStyle.GetHorizontalFrameSize())
	bodyHeight = max(1, bodyOuter-paneStyle.GetVerticalFrameSize())
	return leftWidth, rightWidth, bodyHeight
}

type agentCounts struct {
	agents  int
	working int
}

func (m Model) agentCounts() agentCounts {
	var counts agentCounts
	for _, pane := range m.panes {
		if !isAgentPane(pane) {
			continue
		}
		counts.agents++
		if pane.HasStatus && pane.Status.State == status.Working {
			counts.working++
		}
	}
	return counts
}

func (m Model) agentGroupEnd() int {
	if m.viewMode != ViewAgentsFirst {
		return -1
	}
	for i, pane := range m.panes {
		if !isAgentPane(pane) {
			if i == 0 || i == len(m.panes) {
				return -1
			}
			return i
		}
	}
	return -1
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
	_ = appstate.Save(appstate.State{SelectedPaneID: m.selectedPaneID(), View: string(m.viewMode), ArtIndex: m.artIndex})
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
	case "pi", "claude", "codex", "aider", "goose", "opencode", "gemini":
		return true
	default:
		return false
	}
}

func (m Model) statusGlyph(pane tmux.Pane, selectedRow bool) string {
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
	case status.Working:
		glyph = spinnerGlyph(m.spinnerFrame)
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	case status.Waiting:
		glyph = waitingGlyph()
		style = waitingGlyphStyle(m.spinnerFrame)
	case status.Idle:
		glyph = "✓"
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	}
	if selectedRow {
		return glyph
	}
	return style.Render(glyph)
}

func spinnerGlyph(frame int) string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[frame%len(frames)]
}

func waitingGlyph() string {
	return "?"
}

func waitingGlyphStyle(frame int) lipgloss.Style {
	colors := []lipgloss.Color{mutedColor, lipgloss.Color("214"), accentColor, lipgloss.Color("214")}
	return lipgloss.NewStyle().Foreground(colors[frame%len(colors)]).Bold(true)
}

func sessionStyle(session string) lipgloss.Style {
	palette := []lipgloss.Color{"81", "216", "183", "114", "219", "153", "222", "147"}
	idx := 0
	for _, r := range session {
		idx = (idx*31 + int(r)) % len(palette)
	}
	return lipgloss.NewStyle().Foreground(palette[idx])
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

func centerText(s string, width int) string {
	textWidth := ansi.StringWidth(s)
	if textWidth >= width {
		return truncate(s, width)
	}
	left := (width - textWidth) / 2
	return strings.Repeat(" ", left) + s
}

func insetLine(s string, width int) string {
	const padding = 2
	if width <= padding*2 {
		return fitLine(s, width)
	}
	return strings.Repeat(" ", padding) + fitLine(s, width-padding*2) + strings.Repeat(" ", padding)
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
