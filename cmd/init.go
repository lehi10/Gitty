package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type initModel struct {
	branches []string
	cursor   int
	selected string
	quitting bool
}

func (m initModel) Init() tea.Cmd {
	return nil
}

func (m initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.branches)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.branches[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m initModel) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var s strings.Builder
	s.WriteString(titleStyle.Render("Initialize gtt") + "\n\n")
	s.WriteString(dimStyle.Render("Select your trunk branch (main development branch):") + "\n\n")

	for i, branch := range m.branches {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		var line string
		if i == m.cursor {
			line = selectedStyle.Render(cursor + "○ " + branch)
		} else {
			line = normalStyle.Render(cursor + "○ " + branch)
		}
		s.WriteString(line + "\n")
	}

	s.WriteString("\n" + dimStyle.Render("↑/k up • ↓/j down • enter select • q quit"))
	return s.String()
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gtt in the current repository",
	Long:  `Initialize gtt by selecting the trunk branch. Configuration is saved to .gttconfig`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check if already initialized
		if ConfigExists() {
			config, _ := LoadConfig()
			if config != nil {
				fmt.Printf("gtt is already initialized with trunk: %s\n", config.Trunk)
				fmt.Println("To reinitialize, delete .gttconfig and run gtt init again.")
				return
			}
		}

		// Get all branches
		branches, err := getAllBranches()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: not a git repository or no branches found")
			os.Exit(1)
		}

		if len(branches) == 0 {
			fmt.Fprintln(os.Stderr, "Error: no branches found in this repository")
			os.Exit(1)
		}

		// Find default selection (prefer main, then master, then develop)
		defaultIdx := 0
		for i, b := range branches {
			if b == "main" {
				defaultIdx = i
				break
			} else if b == "master" && defaultIdx == 0 {
				defaultIdx = i
			} else if b == "develop" && defaultIdx == 0 {
				defaultIdx = i
			}
		}

		// Run interactive selection
		m := initModel{branches: branches, cursor: defaultIdx}
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		final := finalModel.(initModel)
		if final.selected == "" {
			fmt.Println("Initialization cancelled.")
			return
		}

		// Save configuration
		config := &Config{Trunk: final.selected}
		if err := SaveConfig(config); err != nil {
			fmt.Fprintln(os.Stderr, "Error saving configuration:", err)
			os.Exit(1)
		}

		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
		fmt.Println(successStyle.Render("✓") + " Initialized gtt with trunk: " + final.selected)
		fmt.Println("  Configuration saved to .gttconfig")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
