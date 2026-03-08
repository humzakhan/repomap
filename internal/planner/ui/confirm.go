package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repomap/repomap/internal/planner"
	"github.com/repomap/repomap/internal/scanner"
)

// ConfirmResult is the outcome of the confirmation screen.
type ConfirmResult struct {
	Confirmed bool
}

// ConfirmModel is the Bubble Tea model for the confirmation screen.
type ConfirmModel struct {
	model         planner.ModelConfig
	cost          float64
	output        string
	budget        scanner.TokenBudget
	moduleCount   int
	skipSynthesis bool
	deep          bool
	cursor        int // 0 = Yes, 1 = No
	result        ConfirmResult
	done          bool
}

// ConfirmOptions contains all the information to display on the confirmation screen.
type ConfirmOptions struct {
	Model         planner.ModelConfig
	Cost          float64
	Output        string
	Budget        scanner.TokenBudget
	ModuleCount   int
	SkipSynthesis bool
	Deep          bool
}

// NewConfirm creates a new confirmation screen model.
func NewConfirm(opts ConfirmOptions) ConfirmModel {
	return ConfirmModel{
		model:         opts.Model,
		cost:          opts.Cost,
		output:        opts.Output,
		budget:        opts.Budget,
		moduleCount:   opts.ModuleCount,
		skipSynthesis: opts.SkipSynthesis,
		deep:          opts.Deep,
	}
}

// Result returns the outcome after the model is done.
func (m ConfirmModel) Result() ConfirmResult {
	return m.result
}

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, confirmKeys.Left):
			m.cursor = 0
		case key.Matches(msg, confirmKeys.Right):
			m.cursor = 1
		case key.Matches(msg, confirmKeys.Yes):
			m.result = ConfirmResult{Confirmed: true}
			m.done = true
			return m, tea.Quit
		case key.Matches(msg, confirmKeys.No):
			m.result = ConfirmResult{Confirmed: false}
			m.done = true
			return m, tea.Quit
		case key.Matches(msg, confirmKeys.Enter):
			m.result = ConfirmResult{Confirmed: m.cursor == 0}
			m.done = true
			return m, tea.Quit
		case key.Matches(msg, confirmKeys.Quit):
			m.result = ConfirmResult{Confirmed: false}
			m.done = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m ConfirmModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("  Step 3 of 5 — Confirm"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Model:        %s  (%s)\n", m.model.DisplayName, m.model.Provider))
	b.WriteString(fmt.Sprintf("  Est. cost:    ~$%.2f\n", m.cost))
	b.WriteString(fmt.Sprintf("  Output:       %s\n", m.output))
	b.WriteString("\n")

	b.WriteString("  Stages:\n")

	check := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓")
	cross := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("✗")

	b.WriteString(fmt.Sprintf("    %s  Module summaries       (%d modules)\n", check, m.moduleCount))

	if m.skipSynthesis {
		b.WriteString(fmt.Sprintf("    %s  Architecture synthesis  (skipped)\n", cross))
	} else {
		b.WriteString(fmt.Sprintf("    %s  Architecture synthesis\n", check))
	}

	b.WriteString(fmt.Sprintf("    %s  Documentation ingestion\n", check))

	if m.deep {
		b.WriteString(fmt.Sprintf("    %s  Deep flow tracing\n", check))
	} else {
		b.WriteString(fmt.Sprintf("    %s  Deep flow tracing      (add --deep)\n", cross))
	}

	b.WriteString("\n")

	// Proceed prompt
	yes := "  [Y]es  "
	no := "  [N]o  "
	if m.cursor == 0 {
		yes = selectedStyle.Render(yes)
	} else {
		no = selectedStyle.Render(no)
	}
	b.WriteString(fmt.Sprintf("  Proceed? %s %s\n", yes, no))

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("  [y] proceed   [n] cancel   [←→] select   [q] quit"))
	b.WriteString("\n")

	return b.String()
}

type confirmKeyMap struct {
	Left  key.Binding
	Right key.Binding
	Yes   key.Binding
	No    key.Binding
	Enter key.Binding
	Quit  key.Binding
}

var confirmKeys = confirmKeyMap{
	Left:  key.NewBinding(key.WithKeys("left", "h")),
	Right: key.NewBinding(key.WithKeys("right", "l")),
	Yes:   key.NewBinding(key.WithKeys("y")),
	No:    key.NewBinding(key.WithKeys("n")),
	Enter: key.NewBinding(key.WithKeys("enter")),
	Quit:  key.NewBinding(key.WithKeys("q", "ctrl+c")),
}
