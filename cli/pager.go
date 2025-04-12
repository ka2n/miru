package cli

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// pagerModel represents the state for the pager UI
type pagerModel struct {
	viewport viewport.Model
	content  string
	ready    bool
}

// NewPager creates a new pager model with the given content
func NewPager(content string) *pagerModel {
	return &pagerModel{
		content: content,
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
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.viewport.ScrollDown(1)
		case "k", "up":
			m.viewport.ScrollUp(1)
		case "f", "pagedown", "space":
			m.viewport.ScrollDown(m.viewport.Height)
		case "b", "pageup", "shift+space":
			m.viewport.ScrollUp(m.viewport.Height)
		case "g", "home":
			m.viewport.GotoTop()
		case "G", "end":
			m.viewport.GotoBottom()
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

	help := helpStyle.Render("↑/k up • ↓/j down • space/f forward • shift+space/b back • g/home top • G/end bottom • q quit")
	return m.viewport.View() + "\n" + help
}

// RunPager starts the pager program with the given content
func RunPager(content string) error {
	p := tea.NewProgram(
		NewPager(content),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}
