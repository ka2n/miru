package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/ka2n/miru/api"
	"github.com/pkg/browser"
)

// keyMap defines keybindings for the pager
type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	GotoTop    key.Binding
	GotoBottom key.Binding
	Search     key.Binding
	NextMatch  key.Binding
	PrevMatch  key.Binding
	ShowMenu   key.Binding
	Reload     key.Binding
	Help       key.Binding
	Quit       key.Binding
}

// defaultKeyMap returns the default keybindings
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "b", "shift+space"),
			key.WithHelp("b/shift+space", "previous page"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "f", "space"),
			key.WithHelp("f/space", "next page"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "go to top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "go to bottom"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		NextMatch: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next search result"),
		),
		PrevMatch: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "previous search result"),
		),
		ShowMenu: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "show menu"),
		),
		Reload: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reload"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "show help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Search, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.GotoTop, k.GotoBottom, k.Search, k.NextMatch, k.PrevMatch},
		{k.ShowMenu, k.Reload, k.Help, k.Quit},
	}
}

type reloadStartMsg struct{}

type reloadFinishMsg struct {
	content string
	result  api.Result
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

// 別のフラグとしてヘルプ表示を制御
type displayState struct {
	showHelp bool // ヘルプを表示するかどうか
}

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

// listItem represents an item in the menu list
type listItem struct {
	title    string
	desc     string
	shortcut string
	action   func() error
}

// FilterValue implements list.Item interface
func (i listItem) FilterValue() string { return i.title }

// Title returns the item's title
func (i listItem) Title() string { return i.title }

// Description returns the item's description
func (i listItem) Description() string { return i.desc }

// pagerModel represents the state for the pager component
type pagerModel struct {
	viewport viewport.Model
	content  string
	search   searchState
	keyMap   keyMap     // Keyboard shortcuts
	help     help.Model // Help model
}

// stashModel represents the state for the bottom bar component
type stashModel struct {
	menuItems    []menuItem   // Menu items
	menuList     list.Model   // List model for menu mode
	selectedIdx  int          // Currently selected index
	displayState displayState // 表示状態を管理
}

// model represents the state for the pager UI
type model struct {
	ready       bool
	inputMode   inputMode
	reloadFunc  func() (string, api.Result, error)
	pagerError  string
	isReloading bool
	result      api.Result // Documentation source information

	pager pagerModel // ページャーコンポーネント
	stash stashModel // ボトムバーコンポーネント

	renderer *glamour.TermRenderer // Markdown renderer
}

func (p *pagerModel) View() string {
	return p.viewport.View()
}

func (p *pagerModel) SetContent(content string) {
	p.content = content
	p.viewport.SetContent(content)
}

// View renders the stash component (bottom bar and help view)
func (s *stashModel) View(width int, isReloading bool, pagerError string, resultData api.Result, help help.Model, keyMap keyMap) string {
	// Bottom bar の生成
	defaultStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("252"))

	titleStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("212")). // Magenta
		Foreground(lipgloss.Color("0")).   // Black
		Padding(0, 1)

	fileNameStyle := defaultStyle.Padding(0, 1)

	// Get package name
	packageName := "MIRU"

	// Extract package name from repository URL
	if repo := resultData.GetRepository(); repo != nil {
		path := repo.String()
		if path != "" {
			// Extract path part from URL
			parts := strings.Split(path, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				if lastPart != "" {
					packageName = lastPart
				}
			}
		}
	}

	// Build bottom bar components
	title := titleStyle.Render("MIRU")
	fileName := fileNameStyle.Render(packageName)
	helpText := defaultStyle.Render("? Help")

	// Display status messages (reloading or error)
	var statusBar string
	if isReloading {
		statusBar = " " + defaultStyle.Foreground(lipgloss.Color("110")).Render("Reloading...")
	} else if pagerError != "" {
		statusBar = " " + defaultStyle.Foreground(lipgloss.Color("9")).Render("Error: "+pagerError)
	}

	// Calculate width for padding
	rightPadding := width - lipgloss.Width(title) - lipgloss.Width(fileName) - lipgloss.Width(statusBar) - lipgloss.Width(helpText)
	if rightPadding < 0 {
		rightPadding = 0
	}

	// Construct the complete bottom bar
	bottomBar := defaultStyle.Width(width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Left,
			title,
			fileName,
			statusBar,
			defaultStyle.Render(strings.Repeat(" ", rightPadding)),
			helpText,
		),
	)

	var output string
	if s.displayState.showHelp {
		output = help.View(keyMap) + "\n" + bottomBar
	} else {
		// ボトムバーのみ返す
		output = bottomBar
	}

	return output
}

// NewPager creates a new pager model with the given content
func NewPager(content string, styleName string, reloadFunc func() (string, api.Result, error), result api.Result) (*model, error) {
	// Initialize text input for search
	ti := textinput.New()
	ti.Prompt = "/"
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Initialize keymap
	keys := defaultKeyMap()

	// Initialize help model
	helpModel := help.New()
	helpModel.ShowAll = true

	// Create pager component
	pagerComponent := pagerModel{
		search: searchState{
			input: ti,
		},
		keyMap: keys,
		help:   helpModel,
	}

	// Create stash component
	stashComponent := stashModel{
		displayState: displayState{
			showHelp: false,
		},
	}

	// Create renderer for markdown
	if styleName == "" {
		styleName = styles.AutoStyle
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(100),
		glamour.WithStandardStyle(styleName),
	)
	if err != nil {
		return nil, err
	}

	// Create main model
	m := &model{
		reloadFunc: reloadFunc,
		result:     result,
		inputMode:  normalMode,
		pager:      pagerComponent,
		stash:      stashComponent,
		renderer:   renderer,
	}

	// Setup menu items
	m.setupMenuItems()

	// Initialize list model for menu mode
	m.initMenuList()

	m.SetContent(content)

	return m, nil
}

func (m *model) SetContent(content string) {
	// Render the content using the markdown renderer
	renderedContent, err := m.renderer.Render(content)
	if err != nil {
		m.pagerError = err.Error()
		return
	}
	m.pager.content = renderedContent
}

// initMenuList initializes the list model for menu mode
func (m *model) initMenuList() {
	// Convert menuItems to list items
	items := []list.Item{}

	for _, item := range m.stash.menuItems {
		items = append(items, listItem{
			title:    item.label,
			desc:     fmt.Sprintf("Shortcut: %s", item.shortcut),
			shortcut: item.shortcut,
			action:   item.action,
		})
	}

	// Create delegate
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetSpacing(1)

	// Create list
	l := list.New(items, delegate, 0, 0)
	l.Title = "Menu"
	l.SetShowHelp(true)
	l.SetFilteringEnabled(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(true)
	l.DisableQuitKeybindings()

	// スタイルを設定
	l.Styles.Title = l.Styles.Title.
		Background(lipgloss.Color("62")). // 青緑色の背景
		Foreground(lipgloss.Color("255")) // 白色のテキスト

	// メニュー項目のスタイルを設定
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Background(lipgloss.Color("62")). // 選択項目の背景色
		Foreground(lipgloss.Color("255")) // 選択項目のテキスト色

	// Set custom keybindings
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "open in browser"),
			),
			key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "close menu"),
			),
		}
	}

	m.stash.menuList = l
}

func (m *model) setupMenuItems() {
	items := []menuItem{}

	r := m.result
	repo := r.GetRepository()
	home := r.GetHomepage()
	regi := r.GetRegistry()
	docs := r.GetDocumentation()

	seen := map[string]any{}

	if repo != nil {
		u := repo.String()
		seen[u] = struct{}{}
		items = append(items, menuItem{
			label:    fmt.Sprintf("Repository: %s", repo),
			shortcut: "g",
			action: func() error {
				return browser.OpenURL(repo.String())
			},
		})
	}

	if regi != nil {
		u := regi.String()
		seen[u] = struct{}{}
		items = append(items, menuItem{
			label:    fmt.Sprintf("Registry: %s", regi),
			shortcut: "r",
			action: func() error {
				return browser.OpenURL(regi.String())
			},
		})
	}

	if home != nil {
		u := home.String()
		seen[u] = struct{}{}
		items = append(items, menuItem{
			label:    fmt.Sprintf("Homepage: %s", home),
			shortcut: "h",
			action: func() error {
				return browser.OpenURL(home.String())
			},
		})
	}

	if docs != nil {
		u := docs.String()
		seen[u] = struct{}{}
		items = append(items, menuItem{
			label:    fmt.Sprintf("Documentation: %s", docs),
			shortcut: "d",
			action: func() error {
				return browser.OpenURL(docs.String())
			},
		})
	}

	// Other related sources
	for i, l := range r.Links {
		u := l.URL.String()
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		items = append(items, menuItem{
			label:    fmt.Sprintf("Other: %s: %s", l.Type, l.URL),
			shortcut: fmt.Sprintf("%d", i+1),
			action: func() error {
				return browser.OpenURL(u)
			},
		})
	}

	m.stash.menuItems = items
}

// Init initializes the pager model
func (m *model) Init() tea.Cmd {
	return nil
}

// Update handles user input and updates the model state
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Execute common update processing
	if model, cmd := m.updateCommon(msg); cmd != nil {
		return model, cmd
	}

	// Execute processing according to mode
	switch m.inputMode {
	case searchMode:
		return m.updateSearchMode(msg)
	case menuMode:
		return m.updateMenuMode(msg)
	default: // normalMode
		return m.updateNormalMode(msg)
	}
}

// updateCommon handles common update logic across all modes
func (m *model) updateCommon(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case reloadStartMsg:
		m.isReloading = true
		return m, nil

	case reloadFinishMsg:
		m.isReloading = false
		if msg.err != nil {
			m.pagerError = msg.err.Error()
		} else {
			m.SetContent(msg.content)
			m.pager.viewport.SetContent(m.pager.content)
			m.setupMenuItems()  // Rebuild menu
			m.clearHighlights() // Clear search highlights
			m.pagerError = ""
		}
		return m, nil

	case tea.WindowSizeMsg:
		if !m.ready {
			m.pager.viewport = viewport.New(msg.Width, msg.Height)
			m.pager.viewport.Style = lipgloss.NewStyle().
				PaddingTop(1).
				PaddingLeft(0).
				PaddingRight(1)
			m.pager.viewport.SetContent(m.pager.content)
			m.ready = true
		}
		m.pager.viewport.Width = msg.Width
		m.pager.viewport.Height = msg.Height
		return m, nil
	}

	return m, nil
}

// updateSearchMode handles updates in search mode
func (m *model) updateSearchMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m.inputMode = normalMode
			m.pager.search.input.Reset()
			m.clearHighlights()
			return m, nil
		case tea.KeyEnter:
			if m.pager.search.input.Value() != "" {
				m.performSearch()
				m.inputMode = normalMode
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.pager.search.input, cmd = m.pager.search.input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// updateMenuMode handles updates in menu mode
func (m *model) updateMenuMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle special keys first
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.inputMode = normalMode
			return m, nil
		case "enter":
			// Get selected item from list
			if i, ok := m.stash.menuList.SelectedItem().(listItem); ok {
				if err := i.action(); err != nil {
					m.pagerError = err.Error()
				}
			}
			return m, nil
		default:
			// Check for shortcut keys
			if item, ok := filterMenuItemByShortcut(m.stash.menuItems, msg.String()); ok {
				if err := item.action(); err != nil {
					m.pagerError = err.Error()
				}
			}
		}
	}

	// Update list model
	m.stash.menuList, cmd = m.stash.menuList.Update(msg)

	// Update list dimensions on window resize
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m.stash.menuList.SetSize(m.pager.viewport.Width, m.pager.viewport.Height)
	}

	return m, cmd
}

// updateNormalMode handles updates in normal mode
func (m *model) updateNormalMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if len(m.pager.search.matches) > 0 {
				m.clearHighlights()
				m.pager.search.input.Reset()
			}
		case "R":
			if m.reloadFunc != nil {
				return m, tea.Batch(
					func() tea.Msg { return reloadStartMsg{} },
					func() tea.Msg {
						content, result, err := m.reloadFunc()
						return reloadFinishMsg{content: content, result: result, err: err}
					},
				)
			}
		case "j", "down":
			m.pager.viewport.ScrollDown(1)
		case "k", "up":
			m.pager.viewport.ScrollUp(1)
		case "f", "pagedown", "space":
			m.pager.viewport.ScrollDown(m.pager.viewport.Height)
		case "b", "pageup", "shift+space":
			m.pager.viewport.ScrollUp(m.pager.viewport.Height)
		case "g", "home":
			m.pager.viewport.GotoTop()
		case "G", "end":
			m.pager.viewport.GotoBottom()
		case "/":
			m.inputMode = searchMode
			m.pager.search.input.Focus()
			return m, textinput.Blink
		case "n":
			if len(m.pager.search.matches) > 0 {
				m.nextMatch()
			}
		case "N":
			if len(m.pager.search.matches) > 0 {
				m.previousMatch()
			}
		case "tab":
			if len(m.stash.menuItems) > 0 {
				m.inputMode = menuMode
				// Update menu list dimensions
				m.stash.menuList.SetSize(m.pager.viewport.Width, m.pager.viewport.Height)
			}
			return m, nil
		case "?":
			m.stash.displayState.showHelp = !m.stash.displayState.showHelp
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{
					Width:  m.pager.viewport.Width,
					Height: m.pager.viewport.Height,
				}
			}
		}
	}

	m.pager.viewport, cmd = m.pager.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the current state of the model
func (m *model) View() string {
	if !m.ready {
		return "\nInitializing..."
	}

	// Display menu in menu mode using list.Model
	if m.inputMode == menuMode {
		return m.stash.menuList.View()
	}

	// Display search input in search mode
	if m.inputMode == searchMode {
		return m.pager.viewport.View() + "\n" + m.pager.search.input.View()
	}

	// 通常モードの場合、viewport と stash の View を組み合わせる
	stashView := m.stash.View(
		m.pager.viewport.Width,
		m.isReloading,
		m.pagerError,
		m.result,
		m.pager.help,
		m.pager.keyMap,
	)

	return m.pager.viewport.View() + "\n" + stashView
}

func (m *model) performSearch() {
	if m.pager.search.input.Value() == "" {
		return
	}

	// Reset matches
	m.pager.search.matches = nil
	m.pager.search.currentMatch = 0

	// Determine case sensitivity
	query := m.pager.search.input.Value()
	content := m.pager.content
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
		m.pager.search.matches = append(m.pager.search.matches, strPos+i)

		// Move position after the current match
		pos = matchPos + 1
	}

	if len(m.pager.search.matches) > 0 {
		// Find first match in viewport
		viewportStart := m.pager.viewport.YOffset
		viewportEnd := m.pager.viewport.YOffset + m.pager.viewport.Height
		firstMatch := 0

		for i, pos := range m.pager.search.matches {
			lines := strings.Split(m.pager.content[:pos], "\n")
			line := len(lines) - 1
			if line >= viewportStart && line < viewportEnd {
				firstMatch = i
				break
			}
		}

		m.pager.search.currentMatch = firstMatch
		m.highlightMatches()
		m.scrollToMatch(firstMatch)
	}
}

func (m *model) highlightMatches() {
	if len(m.pager.search.matches) == 0 {
		return
	}

	contentRunes := []rune(m.pager.content)
	queryLen := len([]rune(m.pager.search.input.Value()))
	var resultBuilder strings.Builder

	lastPos := 0
	for i, bytePos := range m.pager.search.matches {
		// Convert byte position to rune position
		runePos := len([]rune(m.pager.content[:bytePos]))

		// Add text before match
		resultBuilder.WriteString(string(contentRunes[lastPos:runePos]))

		// Add highlighted match
		matchText := string(contentRunes[runePos : runePos+queryLen])
		if i == m.pager.search.currentMatch {
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

	m.pager.viewport.SetContent(resultBuilder.String())
}

// isMatchInViewport checks if the given match index is currently visible in the viewport
func (m *model) isMatchInViewport(matchIndex int) bool {
	if matchIndex < 0 || matchIndex >= len(m.pager.search.matches) {
		return false
	}

	pos := m.pager.search.matches[matchIndex]
	lines := strings.Split(m.pager.content[:pos], "\n")
	line := len(lines) - 1

	viewportStart := m.pager.viewport.YOffset
	viewportEnd := m.pager.viewport.YOffset + m.pager.viewport.Height - 2 // Adjust for help text area

	return line >= viewportStart && line < viewportEnd
}

func (m *model) nextMatch() {
	if len(m.pager.search.matches) == 0 {
		return
	}

	// If current match is in viewport, move to next match normally
	if m.isMatchInViewport(m.pager.search.currentMatch) {
		m.pager.search.currentMatch = (m.pager.search.currentMatch + 1) % len(m.pager.search.matches)
		m.highlightMatches()
		m.scrollToMatch(m.pager.search.currentMatch)
		return
	}

	// Current match is not in viewport, find first match below viewport
	viewportStart := m.pager.viewport.YOffset
	nextMatch := -1
	for i, pos := range m.pager.search.matches {
		lines := strings.Split(m.pager.content[:pos], "\n")
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

	m.pager.search.currentMatch = nextMatch
	m.highlightMatches()
	m.scrollToMatch(nextMatch)
}

func (m *model) previousMatch() {
	if len(m.pager.search.matches) == 0 {
		return
	}

	// If current match is in viewport, move to previous match normally
	if m.isMatchInViewport(m.pager.search.currentMatch) {
		m.pager.search.currentMatch--
		if m.pager.search.currentMatch < 0 {
			m.pager.search.currentMatch = len(m.pager.search.matches) - 1
		}
		m.highlightMatches()
		m.scrollToMatch(m.pager.search.currentMatch)
		return
	}

	// Current match is not in viewport, find last match above viewport
	viewportStart := m.pager.viewport.YOffset
	prevMatch := -1
	for i := len(m.pager.search.matches) - 1; i >= 0; i-- {
		pos := m.pager.search.matches[i]
		lines := strings.Split(m.pager.content[:pos], "\n")
		line := len(lines) - 1

		if line <= viewportStart {
			prevMatch = i
			break
		}
	}

	// If no match found above viewport, wrap to end
	if prevMatch == -1 {
		prevMatch = len(m.pager.search.matches) - 1
	}

	m.pager.search.currentMatch = prevMatch
	m.highlightMatches()
	m.scrollToMatch(prevMatch)
}

func (m *model) scrollToMatch(index int) {
	if index < 0 || index >= len(m.pager.search.matches) {
		return
	}

	pos := m.pager.search.matches[index]
	lines := strings.Split(m.pager.content[:pos], "\n")
	targetLine := len(lines) - 1

	// Calculate actual viewport height considering help text area (2 lines)
	viewportHeight := m.pager.viewport.Height - 2

	// Adjust scroll position to ensure highlight is always visible
	if targetLine < m.pager.viewport.YOffset {
		// Target is above the current view
		m.pager.viewport.YOffset = targetLine
	} else if targetLine >= m.pager.viewport.YOffset+viewportHeight {
		// Target is below the current view
		// Add 2 lines padding to avoid overlap with help text
		m.pager.viewport.YOffset = targetLine - viewportHeight + 2
	}
}

// clearHighlights removes all search highlights and resets search state
func (m *model) clearHighlights() {
	m.pager.search.matches = nil
	m.pager.search.currentMatch = 0
	m.pager.viewport.SetContent(m.pager.content)
}

func filterMenuItemByShortcut(items []menuItem, shortcut string) (menuItem, bool) {
	for _, item := range items {
		if item.shortcut == shortcut {
			return item, true
		}
	}
	return menuItem{}, false
}

// RunPager starts the pager program with the given content
func RunPager(content string, styleName string, result api.Result) error {
	return RunPagerWithReload(content, styleName, nil, result)
}

// RunPagerWithReload starts the pager program with the given content and reload function
func RunPagerWithReload(content string, styleName string, reloadFunc func() (string, api.Result, error), result api.Result) error {
	pager, err := NewPager(content, styleName, reloadFunc, result)
	if err != nil {
		return err
	}

	p := tea.NewProgram(
		pager,
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
