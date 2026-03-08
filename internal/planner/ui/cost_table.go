package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repomap/repomap/internal/planner"
)

// CostTableResult is the outcome of the cost table interaction.
type CostTableResult struct {
	Selected *planner.ScoredModel
	Quit     bool
}

// CostTableModel is the Bubble Tea model for the interactive cost table.
type CostTableModel struct {
	scores   []planner.ScoredModel
	cursor   int
	result   CostTableResult
	done     bool
	width    int
	height   int
}

// NewCostTable creates a new cost table model.
func NewCostTable(scores []planner.ScoredModel) CostTableModel {
	return CostTableModel{
		scores: scores,
		width:  80,
		height: 24,
	}
}

// Result returns the outcome after the model is done.
func (m CostTableModel) Result() CostTableResult {
	return m.result
}

func (m CostTableModel) Init() tea.Cmd {
	return nil
}

func (m CostTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.scores)-1 {
				m.cursor++
			}
		case key.Matches(msg, keys.Enter):
			m.result = CostTableResult{Selected: &m.scores[m.cursor]}
			m.done = true
			return m, tea.Quit
		case key.Matches(msg, keys.Quit):
			m.result = CostTableResult{Quit: true}
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("8"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("4"))

	connectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	unconnectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))

	starStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			MarginTop(1)
)

func (m CostTableModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("  Step 2 of 5 — Cost Analysis"))
	b.WriteString("\n\n")

	// Table header
	header := fmt.Sprintf("  %-3s %-28s %8s %10s   %-12s",
		"", "Model", "Context", "Est. Cost", "Status")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 70)))
	b.WriteString("\n")

	// Find divider position (between connected and unconnected)
	dividerDrawn := false
	prevConnected := true

	for i, sm := range m.scores {
		if prevConnected && !sm.IsConnected && !dividerDrawn {
			b.WriteString("\n")
			b.WriteString(dividerStyle.Render("  OTHER AVAILABLE PROVIDERS  (not connected)"))
			b.WriteString("\n")
			b.WriteString(dividerStyle.Render("  " + strings.Repeat("─", 70)))
			b.WriteString("\n")
			dividerDrawn = true
		}
		prevConnected = sm.IsConnected

		// Build row
		star := "   "
		if sm.IsRecommended {
			star = starStyle.Render(" ⭐")
		}

		name := sm.Model.DisplayName
		ctx := formatContextWindow(sm.Model.ContextWindow)
		cost := fmt.Sprintf("$%.2f", sm.Total)

		var status string
		if sm.IsConnected {
			status = connectedStyle.Render("✓ connected")
		} else {
			status = unconnectedStyle.Render("→ connect")
		}

		warn := ""
		if sm.Warning != "" {
			warn = "  " + warningStyle.Render(sm.Warning)
		}

		row := fmt.Sprintf("  %s %-28s %8s %10s   %-12s%s",
			star, name, ctx, cost, status, warn)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(row))
		} else {
			b.WriteString(row)
		}
		b.WriteString("\n")
	}

	// Hint line
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("  [↑↓] select model   [enter] confirm   [q] quit"))
	b.WriteString("\n")

	// Recommendation note
	for _, sm := range m.scores {
		if sm.IsRecommended {
			b.WriteString("\n")
			b.WriteString(starStyle.Render(fmt.Sprintf(
				"  Recommended: %s — best balance of cost and quality",
				sm.Model.DisplayName)))
			b.WriteString("\n")
			break
		}
	}

	return b.String()
}

func formatContextWindow(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%dM", tokens/1_000_000)
	}
	return fmt.Sprintf("%dK", tokens/1_000)
}

// Key bindings
type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Up:    key.NewBinding(key.WithKeys("up", "k")),
	Down:  key.NewBinding(key.WithKeys("down", "j")),
	Enter: key.NewBinding(key.WithKeys("enter")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
}
