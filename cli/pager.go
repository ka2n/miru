package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/haya14busa/go-openbrowser"
	"github.com/ka2n/miru/api"
)

type reloadStartMsg struct{}

type reloadFinishMsg struct {
	content string
	err     error
}

var (
	searchHighlight = lipgloss.NewStyle().
			Background(lipgloss.Color("228")). // yellow
			Foreground(lipgloss.Color("0"))    // black

	currentMatchHighlight = lipgloss.NewStyle().
				Background(lipgloss.Color("196")). // red
				Foreground(lipgloss.Color("15"))   // white
)

type inputMode int

const (
	normalMode inputMode = iota
	searchMode
	menuMode
)

type searchState struct {
	input        textinput.Model
	matches      []int
	currentMatch int
}

type menuItem struct {
	label    string       // Display name
	shortcut string       // Shortcut key
	action   func() error // Action to execute
}

// pagerModel represents the state for the pager UI
type pagerModel struct {
	viewport    viewport.Model
	content     string
	ready       bool
	inputMode   inputMode
	search      searchState
	reloadFunc  func() (string, error)
	reloadError string
	isReloading bool

	docSource   api.DocSource // Documentation source information
	menuItems   []menuItem    // Menu items
	selectedIdx int           // Currently selected index
}

// NewPager creates a new pager model with the given content
func NewPager(content string, reloadFunc func() (string, error), docSource api.DocSource) *pagerModel {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	m := &pagerModel{
		content:    content,
		reloadFunc: reloadFunc,
		docSource:  docSource,
		inputMode:  normalMode,
		search: searchState{
			input: ti,
		},
	}
	m.setupMenuItems()
	return m
}

func (m *pagerModel) setupMenuItems() {
	m.menuItems = []menuItem{
		{
			label:    "Repository",
			shortcut: "g",
			action: func() error {
				url, err := api.GetDocumentationURL(m.docSource)
				if err != nil {
					return err
				}
				return openbrowser.Start(url.String())
			},
		},
		{
			label:    "Registry",
			shortcut: "r",
			action: func() error {
				// Open registry URL based on source type
				switch m.docSource.Type {
				case api.SourceTypeNPM:
					return openbrowser.Start(fmt.Sprintf("https://www.npmjs.com/package/%s", m.docSource.PackagePath))
				case api.SourceTypeCratesIO:
					return openbrowser.Start(fmt.Sprintf("https://crates.io/crates/%s", m.docSource.PackagePath))
				case api.SourceTypeRubyGems:
					return openbrowser.Start(fmt.Sprintf("https://rubygems.org/gems/%s", m.docSource.PackagePath))
				case api.SourceTypeGoPkgDev:
					return openbrowser.Start(fmt.Sprintf("https://pkg.go.dev/%s", m.docSource.PackagePath))
				default:
					return fmt.Errorf("registry not available for %s", m.docSource.Type)
				}
			},
		},
		{
			label:    "Homepage",
			shortcut: "h",
			action: func() error {
				// Open package homepage
				// Note: Homepage URL retrieval functionality will be added in the future
				return fmt.Errorf("homepage not available")
			},
		},
	}
}

// Init initializes the pager model
func (m *pagerModel) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the model state
func (m *pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inputMode == searchMode {
			var cmd tea.Cmd
			switch msg.Type {
			case tea.KeyType(tea.KeyEscape):
				m.inputMode = normalMode
				m.search.input.Reset()
				m.clearHighlights()
			case tea.KeyEnter:
				if m.search.input.Value() != "" {
					m.performSearch()
					m.inputMode = normalMode
				}
			default:
				m.search.input, cmd = m.search.input.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case tea.KeyType(tea.KeyEscape).String():
			if m.inputMode != normalMode {
				prevMode := m.inputMode
				m.inputMode = normalMode
				if prevMode == searchMode {
					m.search.input.Reset()
					m.clearHighlights()
				}
				return m, nil
			}
			if len(m.search.matches) > 0 {
				m.clearHighlights()
				m.search.input.Reset()
			}
		case "R":
			if m.reloadFunc != nil {
				return m, tea.Batch(
					func() tea.Msg { return reloadStartMsg{} },
					func() tea.Msg {
						content, err := m.reloadFunc()
						return reloadFinishMsg{content: content, err: err}
					},
				)
			}
		case "j", "down":
			if m.inputMode == menuMode {
				m.selectedIdx = (m.selectedIdx + 1) % len(m.menuItems)
				return m, nil
			}
			m.viewport.ScrollDown(1)
		case "k", "up":
			if m.inputMode == menuMode {
				m.selectedIdx--
				if m.selectedIdx < 0 {
					m.selectedIdx = len(m.menuItems) - 1
				}
				return m, nil
			}
			m.viewport.ScrollUp(1)
		case "f", "pagedown", "space":
			m.viewport.ScrollDown(m.viewport.Height)
		case "b", "pageup", "shift+space":
			m.viewport.ScrollUp(m.viewport.Height)
		case "g", "home":
			if m.inputMode == menuMode {
				if err := m.menuItems[0].action(); err != nil {
					m.reloadError = err.Error()
				}
				m.inputMode = normalMode
				return m, nil
			}
			m.viewport.GotoTop()
		case "G", "end":
			m.viewport.GotoBottom()
		case "/":
			m.inputMode = searchMode
			m.search.input.Focus()
			return m, textinput.Blink
		case "n":
			if len(m.search.matches) > 0 {
				m.nextMatch()
			}
		case "N":
			if len(m.search.matches) > 0 {
				m.previousMatch()
			}
		case "tab":
			if m.inputMode != menuMode {
				m.inputMode = menuMode
				return m, nil
			}
			m.selectedIdx = (m.selectedIdx + 1) % len(m.menuItems)
			return m, nil
		case "enter":
			if m.inputMode == menuMode {
				if err := m.menuItems[m.selectedIdx].action(); err != nil {
					m.reloadError = err.Error()
				}
				m.inputMode = normalMode
				return m, nil
			}
		case "r":
			if m.inputMode == menuMode {
				if err := m.menuItems[1].action(); err != nil {
					m.reloadError = err.Error()
				}
				m.inputMode = normalMode
				return m, nil
			}
		case "h":
			if m.inputMode == menuMode {
				if err := m.menuItems[2].action(); err != nil {
					m.reloadError = err.Error()
				}
				m.inputMode = normalMode
				return m, nil
			}
		}

	case reloadStartMsg:
		m.isReloading = true
		return m, nil

	case reloadFinishMsg:
		m.isReloading = false
		if msg.err != nil {
			m.reloadError = msg.err.Error()
		} else {
			m.content = msg.content
			m.clearHighlights() // Clear search highlights
			m.reloadError = ""
		}
		// Force viewport update
		return m, func() tea.Msg {
			return tea.WindowSizeMsg{
				Width:  m.viewport.Width,
				Height: m.viewport.Height + 2,
			}
		}

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				PaddingLeft(2).
				PaddingRight(2)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the current state of the model
func (m *pagerModel) View() string {
	if !m.ready {
		return "\nInitializing..."
	}
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		PaddingLeft(2)

	var help string
	if m.inputMode == searchMode {
		help = m.search.input.View()
	} else if m.inputMode == menuMode {
		var menu strings.Builder
		menu.WriteString("\n")
		for i, item := range m.menuItems {
			prefix := "  "
			if i == m.selectedIdx {
				prefix = "> "
			}
			menu.WriteString(fmt.Sprintf("%s%s (%s)\n", prefix, item.label, item.shortcut))
		}
		menu.WriteString("\nTab: next • Enter: execute • Esc: close")
		help = menu.String()
	} else {
		baseHelp := "↑/k up • ↓/j down • space/f forward • shift+space/b back • g/home top • G/end bottom"
		searchHelp := "/ search • n next • N previous • R reload • Tab menu • q quit"
		if len(m.search.matches) > 0 {
			searchHelp = fmt.Sprintf("/ search (%d/%d) • n next • N previous • R reload • Tab menu • q quit",
				m.search.currentMatch+1, len(m.search.matches))
		}
		help = helpStyle.Render(baseHelp + " • " + searchHelp)
		if m.isReloading {
			help += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("110")).Render("Reloading...")
		} else if m.reloadError != "" {
			help += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: "+m.reloadError)
		}
	}
	return m.viewport.View() + "\n" + help
}

func (m *pagerModel) performSearch() {
	if m.search.input.Value() == "" {
		return
	}

	// Reset matches
	m.search.matches = nil
	m.search.currentMatch = 0

	// Determine case sensitivity
	query := m.search.input.Value()
	content := m.content
	caseSensitive := strings.ContainsAny(query, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if !caseSensitive {
		query = strings.ToLower(query)
		content = strings.ToLower(content)
	}

	// Convert strings to rune slices for proper Unicode handling
	contentRunes := []rune(content)
	queryRunes := []rune(query)

	// Find all matches
	pos := 0
	for pos < len(contentRunes) {
		// Convert current position to string index
		strPos := len(string(contentRunes[:pos]))

		// Find next match
		i := strings.Index(content[strPos:], string(queryRunes))
		if i == -1 {
			break
		}

		// Convert match position back to rune index
		matchPos := len([]rune(content[:strPos+i]))
		m.search.matches = append(m.search.matches, strPos+i)

		// Move position after the current match
		pos = matchPos + 1
	}

	if len(m.search.matches) > 0 {
		// Find first match in viewport
		viewportStart := m.viewport.YOffset
		viewportEnd := m.viewport.YOffset + m.viewport.Height
		firstMatch := 0

		for i, pos := range m.search.matches {
			lines := strings.Split(m.content[:pos], "\n")
			line := len(lines) - 1
			if line >= viewportStart && line < viewportEnd {
				firstMatch = i
				break
			}
		}

		m.search.currentMatch = firstMatch
		m.highlightMatches()
		m.scrollToMatch(firstMatch)
	}
}

func (m *pagerModel) highlightMatches() {
	if len(m.search.matches) == 0 {
		return
	}

	contentRunes := []rune(m.content)
	queryLen := len([]rune(m.search.input.Value()))
	var resultBuilder strings.Builder

	lastPos := 0
	for i, bytePos := range m.search.matches {
		// Convert byte position to rune position
		runePos := len([]rune(m.content[:bytePos]))

		// Add text before match
		resultBuilder.WriteString(string(contentRunes[lastPos:runePos]))

		// Add highlighted match
		matchText := string(contentRunes[runePos : runePos+queryLen])
		if i == m.search.currentMatch {
			resultBuilder.WriteString(currentMatchHighlight.Render(matchText))
		} else {
			resultBuilder.WriteString(searchHighlight.Render(matchText))
		}

		lastPos = runePos + queryLen
	}

	// Add remaining text
	if lastPos < len(contentRunes) {
		resultBuilder.WriteString(string(contentRunes[lastPos:]))
	}

	m.viewport.SetContent(resultBuilder.String())
}

// isMatchInViewport checks if the given match index is currently visible in the viewport
func (m *pagerModel) isMatchInViewport(matchIndex int) bool {
	if matchIndex < 0 || matchIndex >= len(m.search.matches) {
		return false
	}

	pos := m.search.matches[matchIndex]
	lines := strings.Split(m.content[:pos], "\n")
	line := len(lines) - 1

	viewportStart := m.viewport.YOffset
	viewportEnd := m.viewport.YOffset + m.viewport.Height - 2 // Adjust for help text area

	return line >= viewportStart && line < viewportEnd
}

func (m *pagerModel) nextMatch() {
	if len(m.search.matches) == 0 {
		return
	}

	// If current match is in viewport, move to next match normally
	if m.isMatchInViewport(m.search.currentMatch) {
		m.search.currentMatch = (m.search.currentMatch + 1) % len(m.search.matches)
		m.highlightMatches()
		m.scrollToMatch(m.search.currentMatch)
		return
	}

	// Current match is not in viewport, find first match below viewport
	viewportStart := m.viewport.YOffset
	nextMatch := -1
	for i, pos := range m.search.matches {
		lines := strings.Split(m.content[:pos], "\n")
		line := len(lines) - 1

		if line >= viewportStart {
			nextMatch = i
			break
		}
	}

	// If no match found below viewport, wrap to beginning
	if nextMatch == -1 {
		nextMatch = 0
	}

	m.search.currentMatch = nextMatch
	m.highlightMatches()
	m.scrollToMatch(nextMatch)
}

func (m *pagerModel) previousMatch() {
	if len(m.search.matches) == 0 {
		return
	}

	// If current match is in viewport, move to previous match normally
	if m.isMatchInViewport(m.search.currentMatch) {
		m.search.currentMatch--
		if m.search.currentMatch < 0 {
			m.search.currentMatch = len(m.search.matches) - 1
		}
		m.highlightMatches()
		m.scrollToMatch(m.search.currentMatch)
		return
	}

	// Current match is not in viewport, find last match above viewport
	viewportStart := m.viewport.YOffset
	prevMatch := -1
	for i := len(m.search.matches) - 1; i >= 0; i-- {
		pos := m.search.matches[i]
		lines := strings.Split(m.content[:pos], "\n")
		line := len(lines) - 1

		if line <= viewportStart {
			prevMatch = i
			break
		}
	}

	// If no match found above viewport, wrap to end
	if prevMatch == -1 {
		prevMatch = len(m.search.matches) - 1
	}

	m.search.currentMatch = prevMatch
	m.highlightMatches()
	m.scrollToMatch(prevMatch)
}

func (m *pagerModel) scrollToMatch(index int) {
	if index < 0 || index >= len(m.search.matches) {
		return
	}

	pos := m.search.matches[index]
	lines := strings.Split(m.content[:pos], "\n")
	targetLine := len(lines) - 1

	// Calculate actual viewport height considering help text area (2 lines)
	viewportHeight := m.viewport.Height - 2

	// Adjust scroll position to ensure highlight is always visible
	if targetLine < m.viewport.YOffset {
		// Target is above the current view
		m.viewport.YOffset = targetLine
	} else if targetLine >= m.viewport.YOffset+viewportHeight {
		// Target is below the current view
		// Add 2 lines padding to avoid overlap with help text
		m.viewport.YOffset = targetLine - viewportHeight + 2
	}
}

// clearHighlights removes all search highlights and resets search state
func (m *pagerModel) clearHighlights() {
	m.search.matches = nil
	m.search.currentMatch = 0
	m.viewport.SetContent(m.content)
}

// RunPager starts the pager program with the given content
func RunPager(content string, docSource api.DocSource) error {
	return RunPagerWithReload(content, nil, docSource)
}

// RunPagerWithReload starts the pager program with the given content and reload function
func RunPagerWithReload(content string, reloadFunc func() (string, error), docSource api.DocSource) error {
	p := tea.NewProgram(
		NewPager(content, reloadFunc, docSource),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
