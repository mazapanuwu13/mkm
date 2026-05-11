package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mkm/internal/config"
	"mkm/internal/parser"
	"mkm/internal/runner"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// -- screen enum --------------------------------------------------------------

type appScreen int

const (
	screenGroups appScreen = iota
	screenMain
)

// -- panel enum (screenMain only) ---------------------------------------------

type panel int

const (
	panelProjects panel = iota
	panelTargets
	panelOutput
	panelMainCount = 3
)

// -- list item adapters -------------------------------------------------------

type groupItem struct{ g config.Group }

func (i groupItem) Title() string       { return i.g.Name }
func (i groupItem) Description() string { return fmt.Sprintf("%d project(s)", len(i.g.Projects)) }
func (i groupItem) FilterValue() string { return i.g.Name }

type projectItem struct{ p parser.Project }

func (i projectItem) Title() string       { return i.p.Name }
func (i projectItem) Description() string { return fmt.Sprintf("%d targets", len(i.p.Targets)) }
func (i projectItem) FilterValue() string { return i.p.Name }

type targetItem struct{ t parser.Target }

func (i targetItem) Title() string       { return i.t.Name }
func (i targetItem) Description() string { return i.t.Description }
func (i targetItem) FilterValue() string { return i.t.Name }

// -- messages -----------------------------------------------------------------

type outputMsg string
type doneMsg struct{ err error }

type pathScannedMsg struct {
	groupIdx int
	dirs     []string
	label    string
	errMsg   string
}

// -- input purpose ------------------------------------------------------------

type inputPurpose int

const (
	inputNone inputPurpose = iota
	inputNewGroup
	inputAddPath
)

// -- streamer -----------------------------------------------------------------

type streamer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	p   *tea.Program
}

func (s *streamer) Write(b []byte) (int, error) {
	s.mu.Lock()
	n, err := s.buf.Write(b)
	snapshot := s.buf.String()
	s.mu.Unlock()
	s.p.Send(outputMsg(snapshot))
	return n, err
}

// -- model --------------------------------------------------------------------

type Model struct {
	cfg *config.Config

	screen      appScreen
	activeGroup int

	groupList   list.Model
	projectList list.Model
	targetList  list.Model
	output      viewport.Model

	projects []parser.Project
	focus    panel

	inputPurpose inputPurpose
	inputPrompt  string
	textInput    textinput.Model

	running    bool
	outputText string
	statusLine string

	width   int
	height  int
	ready   bool
	program *tea.Program
}

func newList(title string) list.Model {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = SelectedItemStyle
	d.Styles.SelectedDesc = DescStyle.Foreground(lipgloss.Color("#81A1C1")).Background(colorHighlight)
	d.Styles.NormalTitle = ItemStyle
	d.Styles.NormalDesc = DescStyle
	d.Styles.DimmedTitle = ItemStyle.Foreground(colorBorder)
	d.Styles.DimmedDesc = DescStyle

	l := list.New(nil, d, 0, 0)
	l.Title = title
	l.Styles.Title = TitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	return l
}

// New creates the TUI model backed by the given config.
func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.CharLimit = 512

	m := Model{
		cfg:         cfg,
		screen:      screenGroups,
		activeGroup: -1,
		groupList:   newList("Groups"),
		projectList: newList("Projects"),
		targetList:  newList("Targets"),
		output:      viewport.New(0, 0),
		textInput:   ti,
		focus:       panelProjects,
	}
	m.refreshGroups()
	return m
}

func (m *Model) SetProgram(p *tea.Program) { m.program = p }

// -- refresh helpers ----------------------------------------------------------

func (m *Model) refreshGroups() {
	items := make([]list.Item, len(m.cfg.Groups))
	for i, g := range m.cfg.Groups {
		items[i] = groupItem{g}
	}
	m.groupList.SetItems(items)
}

func (m *Model) loadGroupProjects(groupIdx int) {
	if groupIdx < 0 || groupIdx >= len(m.cfg.Groups) {
		m.projects = nil
		m.projectList.SetItems(nil)
		m.refreshTargets()
		return
	}
	g := m.cfg.Groups[groupIdx]
	var projects []parser.Project
	for _, projPath := range g.Projects {
		targets, err := parser.ParseTargets(filepath.Join(projPath, "Makefile"))
		if err != nil || len(targets) == 0 {
			continue
		}
		projects = append(projects, parser.Project{
			Name:    filepath.Base(projPath),
			Dir:     projPath,
			Targets: targets,
		})
	}
	m.projects = projects
	items := make([]list.Item, len(projects))
	for i, p := range projects {
		items[i] = projectItem{p}
	}
	m.projectList.SetItems(items)
	m.refreshTargets()
}

func (m *Model) refreshTargets() {
	idx := m.projectList.Index()
	if idx < 0 || idx >= len(m.projects) {
		m.targetList.SetItems(nil)
		return
	}
	proj := m.projects[idx]
	items := make([]list.Item, len(proj.Targets))
	for i, t := range proj.Targets {
		items[i] = targetItem{t}
	}
	m.targetList.SetItems(items)
}

// -- init ---------------------------------------------------------------------

func (m Model) Init() tea.Cmd { return nil }

// -- update -------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.resize()

	case tea.KeyMsg:
		// inline input mode
		if m.inputPurpose != inputNone {
			switch msg.String() {
			case "esc":
				m.inputPurpose = inputNone
				m.inputPrompt = ""
				m.textInput.Blur()
				m.statusLine = ""
				return m, nil
			case "enter":
				val := strings.TrimSpace(m.textInput.Value())
				m.textInput.Reset()
				m.textInput.Blur()
				purpose := m.inputPurpose
				m.inputPurpose = inputNone
				m.inputPrompt = ""
				m.statusLine = ""
				if val == "" {
					return m, nil
				}
				switch purpose {
				case inputNewGroup:
					m.cfg.AddGroup(val)
					_ = m.cfg.Save()
					m.refreshGroups()
					m.statusLine = StatusSuccessStyle.Render("+ Group '" + val + "' created")
				case inputAddPath:
					return m, scanPathCmd(val, m.activeGroup)
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// global quit
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !m.running {
				return m, tea.Quit
			}
		}

		// route to active screen
		switch m.screen {
		case screenGroups:
			return m.updateGroups(msg)
		case screenMain:
			return m.updateMain(msg)
		}

	case pathScannedMsg:
		if msg.errMsg != "" {
			m.statusLine = StatusErrorStyle.Render("x " + msg.errMsg)
			return m, nil
		}
		for _, dir := range msg.dirs {
			m.cfg.AddProject(msg.groupIdx, dir)
		}
		_ = m.cfg.Save()
		m.refreshGroups()
		if m.activeGroup == msg.groupIdx {
			m.loadGroupProjects(m.activeGroup)
		}
		m.statusLine = StatusSuccessStyle.Render("+ Added " + msg.label)

	case outputMsg:
		m.outputText = string(msg)
		m.output.SetContent(m.outputText)
		m.output.GotoBottom()

	case doneMsg:
		m.running = false
		if msg.err != nil {
			m.statusLine = StatusErrorStyle.Render("x failed: " + msg.err.Error())
		} else {
			m.statusLine = StatusSuccessStyle.Render("v done")
		}
	}

	return m, nil
}

func (m Model) updateGroups(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n":
		m.inputPurpose = inputNewGroup
		m.inputPrompt = "New group: "
		promptW := lipgloss.Width(InputPromptStyle.Render(m.inputPrompt))
		m.textInput.Width = max(10, m.width-promptW-4)
		m.textInput.Placeholder = "group name..."
		m.textInput.SetValue("")
		return m, m.textInput.Focus()

	case "d":
		idx := m.groupList.Index()
		if idx >= 0 && idx < len(m.cfg.Groups) {
			name := m.cfg.Groups[idx].Name
			m.cfg.DeleteGroup(idx)
			_ = m.cfg.Save()
			m.refreshGroups()
			m.statusLine = StatusSuccessStyle.Render("- Deleted '" + name + "'")
		}

	case "enter":
		idx := m.groupList.Index()
		if idx >= 0 && idx < len(m.cfg.Groups) {
			m.activeGroup = idx
			m.loadGroupProjects(idx)
			m.screen = screenMain
			m.focus = panelProjects
			m.outputText = ""
			m.output.SetContent("")
			m.statusLine = ""
			m.resize()
		}

	default:
		var cmd tea.Cmd
		m.groupList, cmd = m.groupList.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenGroups
		m.activeGroup = -1
		m.outputText = ""
		m.output.SetContent("")
		m.statusLine = ""
		m.resize()
		return m, nil
	case "tab", "right", "l":
		m.focus = (m.focus + 1) % panelMainCount
		return m, nil
	case "shift+tab", "left", "h":
		m.focus = (m.focus + panelMainCount - 1) % panelMainCount
		return m, nil
	}

	switch m.focus {
	case panelProjects:
		switch msg.String() {
		case "a":
			m.inputPurpose = inputAddPath
			m.inputPrompt = "Add path: "
			promptW := lipgloss.Width(InputPromptStyle.Render(m.inputPrompt))
			m.textInput.Width = max(10, m.width-promptW-4)
			m.textInput.Placeholder = "/path/to/project  or  ~/workspace/monorepo"
			m.textInput.SetValue("")
			return m, m.textInput.Focus()
		case "d":
			pIdx := m.projectList.Index()
			if pIdx >= 0 && pIdx < len(m.projects) {
				m.cfg.RemoveProject(m.activeGroup, pIdx)
				_ = m.cfg.Save()
				m.loadGroupProjects(m.activeGroup)
				m.refreshGroups()
				m.statusLine = StatusSuccessStyle.Render("- Project removed")
			}
		case "enter":
			if len(m.projects) > 0 {
				m.focus = panelTargets
			}
		default:
			prevIdx := m.projectList.Index()
			var cmd tea.Cmd
			m.projectList, cmd = m.projectList.Update(msg)
			if m.projectList.Index() != prevIdx {
				m.refreshTargets()
			}
			return m, cmd
		}

	case panelTargets:
		if msg.String() == "enter" && !m.running {
			return m, m.runTarget()
		}
		var cmd tea.Cmd
		m.targetList, cmd = m.targetList.Update(msg)
		return m, cmd

	case panelOutput:
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		return m, cmd
	}
	return m, nil
}

// -- commands -----------------------------------------------------------------

func scanPathCmd(raw string, groupIdx int) tea.Cmd {
	return func() tea.Msg {
		p := raw
		if strings.HasPrefix(p, "~/") {
			home, _ := os.UserHomeDir()
			p = filepath.Join(home, p[2:])
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return pathScannedMsg{groupIdx: groupIdx, errMsg: "invalid path: " + err.Error()}
		}
		if _, err := os.Stat(filepath.Join(abs, "Makefile")); err == nil {
			return pathScannedMsg{groupIdx: groupIdx, dirs: []string{abs}, label: filepath.Base(abs)}
		}
		projects, err := parser.ScanProjects(abs)
		if err != nil {
			return pathScannedMsg{groupIdx: groupIdx, errMsg: "scan error: " + err.Error()}
		}
		if len(projects) == 0 {
			return pathScannedMsg{groupIdx: groupIdx, errMsg: "no Makefiles found in " + abs}
		}
		dirs := make([]string, len(projects))
		for i, proj := range projects {
			dirs[i] = proj.Dir
		}
		label := fmt.Sprintf("%d project(s) from %s", len(dirs), filepath.Base(abs))
		return pathScannedMsg{groupIdx: groupIdx, dirs: dirs, label: label}
	}
}

func (m *Model) runTarget() tea.Cmd {
	pIdx := m.projectList.Index()
	tIdx := m.targetList.Index()
	if pIdx < 0 || pIdx >= len(m.projects) {
		return nil
	}
	proj := m.projects[pIdx]
	if tIdx < 0 || tIdx >= len(proj.Targets) {
		return nil
	}
	target := proj.Targets[tIdx]

	m.running = true
	m.outputText = ""
	m.output.SetContent("")
	m.statusLine = StatusRunningStyle.Render(
		fmt.Sprintf("running: make %s  [%s]", target.Name, proj.Name),
	)
	// stay on targets panel so the user can queue another run immediately

	s := &streamer{p: m.program}
	return func() tea.Msg {
		err := runner.Run(proj.Dir, target.Name, s)
		return doneMsg{err: err}
	}
}

// -- layout -------------------------------------------------------------------

func (m *Model) resize() {
	const headerH = 1
	const footerH = 1
	const titleH = 2

	listH := m.height - headerH - footerH - titleH
	if listH < 1 {
		listH = 1
	}

	switch m.screen {
	case screenGroups:
		m.groupList.SetSize(m.width-4, listH)
	case screenMain:
		usable := m.width - 12
		if usable < 36 {
			usable = 36
		}
		pW := clamp(usable*25/100, 18, 34)
		tW := clamp(usable*25/100, 18, 34)
		oW := usable - pW - tW
		if oW < 10 {
			oW = 10
		}
		m.projectList.SetSize(pW, listH)
		m.targetList.SetSize(tW, listH)
		m.output.Width = oW
		m.output.Height = max(1, listH-2)
	}
}

// -- view ---------------------------------------------------------------------

func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	header := AppTitleStyle.Render("AutoHost TUI -- Makefile Manager")

	var body string
	switch m.screen {
	case screenGroups:
		body = m.viewGroups()
	case screenMain:
		body = m.viewMain()
	}

	var footer string
	if m.inputPurpose != inputNone {
		footer = InputPromptStyle.Render(m.inputPrompt) + m.textInput.View()
	} else {
		help := m.contextHelp()
		status := m.statusLine
		if status == "" {
			status = HelpStyle.Render("Ready")
		}
		pad := max(0, m.width-lipgloss.Width(help)-lipgloss.Width(status)-2)
		footer = lipgloss.JoinHorizontal(lipgloss.Top,
			help,
			strings.Repeat(" ", pad),
			status,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) viewGroups() string {
	groupContent := m.groupList.View()
	if len(m.cfg.Groups) == 0 {
		groupContent += "\n" + HelpStyle.Render("  Press n to create your first group.")
	}
	return ActivePanelStyle.Width(m.width - 4).Render(groupContent)
}

func (m Model) viewMain() string {
	gName := ""
	if m.activeGroup >= 0 && m.activeGroup < len(m.cfg.Groups) {
		gName = m.cfg.Groups[m.activeGroup].Name
	}

	projStyle := PanelStyle
	targetStyle := PanelStyle
	outStyle := PanelStyle
	switch m.focus {
	case panelProjects:
		projStyle = ActivePanelStyle
	case panelTargets:
		targetStyle = ActivePanelStyle
	case panelOutput:
		outStyle = ActivePanelStyle
	}

	usable := m.width - 12
	if usable < 36 {
		usable = 36
	}
	pW := clamp(usable*25/100, 18, 34)
	tW := clamp(usable*25/100, 18, 34)
	oW := usable - pW - tW
	if oW < 10 {
		oW = 10
	}

	breadcrumb := BreadcrumbStyle.Render("# " + gName)

	projContent := m.projectList.View()
	if len(m.projects) == 0 {
		projContent += "\n" + HelpStyle.Render("  a: add path")
	}
	projPanel := projStyle.Width(pW).Render(projContent)
	targetPanel := targetStyle.Width(tW).Render(m.targetList.View())

	outContent := m.output.View()
	if m.outputText == "" && !m.running {
		outContent = HelpStyle.Render("\n  Select a target\n  and press Enter.")
	}
	outPanel := outStyle.Width(oW).Render(TitleStyle.Render("Output") + "\n" + outContent)

	panels := lipgloss.JoinHorizontal(lipgloss.Top, projPanel, targetPanel, outPanel)
	return lipgloss.JoinVertical(lipgloss.Left, breadcrumb, panels)
}

func (m Model) contextHelp() string {
	switch m.screen {
	case screenGroups:
		return HelpStyle.Render("up/down: navigate  n: new group  d: delete  enter: open  q: quit")
	case screenMain:
		switch m.focus {
		case panelProjects:
			return HelpStyle.Render("a: add path  d: remove  enter: targets  tab: next  esc: groups  q: quit")
		case panelTargets:
			return HelpStyle.Render("enter: run  tab: next  esc: groups  q: quit")
		case panelOutput:
			return HelpStyle.Render("up/down: scroll  tab: next  esc: groups  q: quit")
		}
	}
	return ""
}

// -- helpers ------------------------------------------------------------------

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
