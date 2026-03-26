package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	prTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	prDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	prInputStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	prSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	prErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func checkGhInstalled() bool {
	cmd := exec.Command("gh", "--version")
	return cmd.Run() == nil
}

func checkGhAuth() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

func getTrunkForPR() string {
	config, _ := LoadConfig()
	if config != nil && config.Trunk != "" {
		return config.Trunk
	}
	// Default fallback
	for _, name := range []string{"main", "master", "develop"} {
		cmd := exec.Command("git", "rev-parse", "--verify", name)
		if err := cmd.Run(); err == nil {
			return name
		}
	}
	return "main"
}

func createPR(title, body, base string) error {
	args := []string{"pr", "create", "--title", title, "--body", body}
	if base != "" {
		args = append(args, "--base", base)
	}

	cmd := exec.Command("gh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PR Creation interactive model
type prCreateModel struct {
	step       int // 0=title, 1=body, 2=confirm
	title      string
	body       string
	base       string
	branch     string
	inputValue string
	quitting   bool
	creating   bool
}

func (m prCreateModel) Init() tea.Cmd {
	return nil
}

func (m prCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			switch m.step {
			case 0: // Title entered
				if m.inputValue != "" {
					m.title = m.inputValue
					m.inputValue = ""
					m.step = 1
				}
			case 1: // Body entered (can be empty)
				m.body = m.inputValue
				m.inputValue = ""
				m.step = 2
			case 2: // Confirm
				m.creating = true
				return m, tea.Quit
			}
		case "backspace":
			if len(m.inputValue) > 0 {
				m.inputValue = m.inputValue[:len(m.inputValue)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.inputValue += msg.String()
			}
		}
	}
	return m, nil
}

func (m prCreateModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder
	s.WriteString(prTitleStyle.Render("Create Pull Request") + "\n\n")
	s.WriteString(prDimStyle.Render(fmt.Sprintf("Branch: %s → %s", m.branch, m.base)) + "\n\n")

	switch m.step {
	case 0:
		s.WriteString(prDimStyle.Render("Title:") + "\n")
		s.WriteString(prInputStyle.Render("> "+m.inputValue) + "█\n")
	case 1:
		s.WriteString(prDimStyle.Render("Title: ") + prInputStyle.Render(m.title) + "\n\n")
		s.WriteString(prDimStyle.Render("Description (optional):") + "\n")
		s.WriteString(prInputStyle.Render("> "+m.inputValue) + "█\n")
	case 2:
		s.WriteString(prDimStyle.Render("Title: ") + prInputStyle.Render(m.title) + "\n")
		if m.body != "" {
			s.WriteString(prDimStyle.Render("Description: ") + prInputStyle.Render(m.body) + "\n")
		}
		s.WriteString("\n" + prDimStyle.Render("Press Enter to create PR, Esc to cancel"))
	}

	s.WriteString("\n\n" + prDimStyle.Render("esc cancel"))
	return s.String()
}

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create or manage pull requests",
	Long:  `Create a pull request for the current branch or manage existing PRs.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check gh is installed
		if !checkGhInstalled() {
			fmt.Fprintln(os.Stderr, prErrorStyle.Render("✗")+" GitHub CLI (gh) is not installed.")
			fmt.Fprintln(os.Stderr, "  Install it from: https://cli.github.com/")
			os.Exit(1)
		}

		// Check gh is authenticated
		if !checkGhAuth() {
			fmt.Fprintln(os.Stderr, prErrorStyle.Render("✗")+" Not authenticated with GitHub.")
			fmt.Fprintln(os.Stderr, "  Run: gh auth login")
			os.Exit(1)
		}

		// Get current branch
		branch, err := getCurrentBranch()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error getting current branch:", err)
			os.Exit(1)
		}

		trunk := getTrunkForPR()
		if branch == trunk {
			fmt.Fprintln(os.Stderr, prErrorStyle.Render("✗")+" Cannot create PR from trunk branch ("+trunk+")")
			os.Exit(1)
		}

		// Check if branch is pushed
		cmd2 := exec.Command("git", "ls-remote", "--heads", "origin", branch)
		if output, _ := cmd2.Output(); len(output) == 0 {
			fmt.Println(prDimStyle.Render("Pushing branch to origin..."))
			pushCmd := exec.Command("git", "push", "-u", "origin", branch)
			pushCmd.Stdout = os.Stdout
			pushCmd.Stderr = os.Stderr
			if err := pushCmd.Run(); err != nil {
				fmt.Fprintln(os.Stderr, "Error pushing branch:", err)
				os.Exit(1)
			}
		}

		// Interactive PR creation
		m := prCreateModel{
			step:   0,
			base:   trunk,
			branch: branch,
		}
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		final := finalModel.(prCreateModel)
		if final.quitting || !final.creating {
			fmt.Println("PR creation cancelled.")
			return
		}

		// Create the PR
		fmt.Println(prDimStyle.Render("\nCreating PR..."))
		if err := createPR(final.title, final.body, final.base); err != nil {
			fmt.Fprintln(os.Stderr, prErrorStyle.Render("✗")+" Error creating PR:", err)
			os.Exit(1)
		}
	},
}

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List open pull requests",
	Run: func(cmd *cobra.Command, args []string) {
		if !checkGhInstalled() {
			fmt.Fprintln(os.Stderr, prErrorStyle.Render("✗")+" GitHub CLI (gh) is not installed.")
			os.Exit(1)
		}

		if !checkGhAuth() {
			fmt.Fprintln(os.Stderr, prErrorStyle.Render("✗")+" Not authenticated with GitHub.")
			os.Exit(1)
		}

		ghCmd := exec.Command("gh", "pr", "list")
		ghCmd.Stdout = os.Stdout
		ghCmd.Stderr = os.Stderr
		ghCmd.Run()
	},
}

var prViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View the PR for the current branch",
	Run: func(cmd *cobra.Command, args []string) {
		if !checkGhInstalled() {
			fmt.Fprintln(os.Stderr, prErrorStyle.Render("✗")+" GitHub CLI (gh) is not installed.")
			os.Exit(1)
		}

		ghCmd := exec.Command("gh", "pr", "view", "--web")
		ghCmd.Stdout = os.Stdout
		ghCmd.Stderr = os.Stderr
		ghCmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prViewCmd)
}
