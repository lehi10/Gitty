package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	currentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("78"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mergedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	treeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type branchNode struct {
	name      string
	isCurrent bool
	isMerged  bool
	children  []*branchNode
	parent    *branchNode
}

type displayBranch struct {
	name      string
	isCurrent bool
	isMerged  bool
	prefix    string
}

type model struct {
	branches []displayBranch
	cursor   int
	selected string
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.selected = m.branches[m.cursor].name
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder
	s.WriteString(dimStyle.Render("Select a branch:") + "\n\n")

	for i, b := range m.branches {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		prefix := treeStyle.Render(b.prefix)

		var line string
		var marker, name string
		var style lipgloss.Style

		if b.isCurrent {
			marker = "● "
			name = b.name + " (current)"
			if i == m.cursor {
				style = selectedStyle
			} else {
				style = currentStyle
			}
		} else if b.isMerged {
			marker = "✓ "
			name = b.name + " (merged)"
			if i == m.cursor {
				style = selectedStyle
			} else {
				style = mergedStyle
			}
		} else {
			marker = "○ "
			name = b.name
			if i == m.cursor {
				style = selectedStyle
			} else {
				style = normalStyle
			}
		}

		line = cursor + prefix + style.Render(marker+name)
		s.WriteString(line + "\n")
	}

	s.WriteString("\n" + dimStyle.Render("↑/k up • ↓/j down • enter select • q quit"))
	return s.String()
}

func getTrunkBranches() []string {
	// First, check if gtt is initialized with a configured trunk
	config, _ := LoadConfig()
	if config != nil && config.Trunk != "" {
		// Verify the configured trunk exists
		cmd := exec.Command("git", "rev-parse", "--verify", config.Trunk)
		if err := cmd.Run(); err == nil {
			return []string{config.Trunk}
		}
	}

	// Fallback to auto-detection
	var trunks []string
	for _, name := range []string{"main", "master", "develop"} {
		cmd := exec.Command("git", "rev-parse", "--verify", name)
		if err := cmd.Run(); err == nil {
			trunks = append(trunks, name)
		}
	}
	if len(trunks) == 0 {
		return []string{"main"}
	}
	return trunks
}

func getPrimaryTrunk(trunks []string) string {
	// If config exists, use it
	config, _ := LoadConfig()
	if config != nil && config.Trunk != "" {
		for _, t := range trunks {
			if t == config.Trunk {
				return t
			}
		}
	}

	// Fallback: prefer main > master > develop as the root
	for _, preferred := range []string{"main", "master", "develop"} {
		for _, t := range trunks {
			if t == preferred {
				return t
			}
		}
	}
	return trunks[0]
}

func getMergeBase(branch1, branch2 string) string {
	cmd := exec.Command("git", "merge-base", branch1, branch2)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getCommitCount(from, to string) int {
	cmd := exec.Command("git", "rev-list", "--count", from+".."+to)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	var count int
	fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &count)
	return count
}

func getBranchCommit(branch string) string {
	cmd := exec.Command("git", "rev-parse", branch)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getMergedBranches(trunks []string) map[string]bool {
	merged := make(map[string]bool)
	trunkSet := make(map[string]bool)
	for _, t := range trunks {
		trunkSet[t] = true
	}

	// Check merged branches against all trunks
	for _, trunk := range trunks {
		cmd := exec.Command("git", "branch", "--merged", trunk, "--format=%(refname:short)")
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, name := range lines {
			name = strings.TrimSpace(name)
			// Don't mark trunk branches as merged
			if name != "" && !trunkSet[name] {
				merged[name] = true
			}
		}
	}
	return merged
}

func buildBranchTree() ([]displayBranch, error) {
	// Get all branches
	cmd := exec.Command("git", "branch", "--format=%(refname:short)|%(HEAD)")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	trunks := getTrunkBranches()
	primaryTrunk := getPrimaryTrunk(trunks)
	mergedBranches := getMergedBranches(trunks)

	// Create a set of trunk branches
	trunkSet := make(map[string]bool)
	for _, t := range trunks {
		trunkSet[t] = true
	}

	type branchInfo struct {
		name      string
		isCurrent bool
		isMerged  bool
	}

	var branches []branchInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		name := parts[0]
		isCurrent := len(parts) > 1 && parts[1] == "*"
		branches = append(branches, branchInfo{
			name:      name,
			isCurrent: isCurrent,
			isMerged:  mergedBranches[name],
		})
	}

	// Build tree structure
	nodes := make(map[string]*branchNode)
	var root *branchNode

	// Create all nodes
	for _, b := range branches {
		nodes[b.name] = &branchNode{
			name:      b.name,
			isCurrent: b.isCurrent,
			isMerged:  b.isMerged,
			children:  []*branchNode{},
		}
		if b.name == primaryTrunk {
			root = nodes[b.name]
		}
	}

	// If no trunk found, use first branch as root
	if root == nil && len(branches) > 0 {
		root = nodes[branches[0].name]
	}

	// Add other trunks (like develop) as children of primary trunk
	for _, t := range trunks {
		if t != primaryTrunk && nodes[t] != nil {
			nodes[t].parent = root
			root.children = append(root.children, nodes[t])
		}
	}

	// Find parent for each branch
	for _, b := range branches {
		// Skip trunk branches - they're handled separately
		if trunkSet[b.name] {
			continue
		}

		node := nodes[b.name]
		bCommit := getBranchCommit(b.name)

		var bestParent *branchNode
		bestDistance := -1
		bestPriority := -1 // Higher priority = better (develop > main > others)

		// For merged branches, find which branch contains them (prioritize trunks)
		if b.isMerged {
			for _, other := range branches {
				if other.name == b.name {
					continue
				}
				mergeBase := getMergeBase(b.name, other.name)
				if mergeBase == bCommit {
					// other contains b (b was merged into it or b is ancestor)
					priority := 0
					if other.name == "develop" {
						priority = 2
					} else if other.name == "main" || other.name == "master" {
						priority = 1
					}
					distance := getCommitCount(b.name, other.name)
					if bestParent == nil || priority > bestPriority ||
						(priority == bestPriority && distance < bestDistance) {
						bestDistance = distance
						bestPriority = priority
						bestParent = nodes[other.name]
					}
				}
			}
		}

		// If no trunk parent found (or not merged), find closest ancestor branch
		if bestParent == nil {
			for _, other := range branches {
				if other.name == b.name {
					continue
				}

				mergeBase := getMergeBase(b.name, other.name)
				if mergeBase == "" {
					continue
				}

				otherCommit := getBranchCommit(other.name)

				// Only consider branches where other is an ancestor of b
				if mergeBase == otherCommit {
					distance := getCommitCount(other.name, b.name)

					// For tiebreaker: prefer branches with similar names (same feature prefix)
					// over trunks, to maintain stack relationships
					similarityScore := 0
					if !trunkSet[other.name] && strings.HasPrefix(b.name, "feature/") && strings.HasPrefix(other.name, "feature/") {
						// Both are features, give bonus for name similarity
						// Extract feature ID (e.g., "ef-42" from "feature/ef-42-...")
						bParts := strings.Split(strings.TrimPrefix(b.name, "feature/"), "-")
						oParts := strings.Split(strings.TrimPrefix(other.name, "feature/"), "-")
						if len(bParts) >= 2 && len(oParts) >= 2 && bParts[0] == oParts[0] && bParts[1] == oParts[1] {
							similarityScore = 1
						}
					}

					if bestParent == nil ||
						distance < bestDistance ||
						(distance == bestDistance && similarityScore > bestPriority) {
						bestDistance = distance
						bestPriority = similarityScore
						bestParent = nodes[other.name]
					}
				}
			}
		}

		// Default to primary trunk if no parent found
		if bestParent == nil {
			bestParent = root
		}

		if bestParent != nil && bestParent != node {
			node.parent = bestParent
			bestParent.children = append(bestParent.children, node)
		}
	}

	// Sort children alphabetically
	var sortChildren func(n *branchNode)
	sortChildren = func(n *branchNode) {
		sort.Slice(n.children, func(i, j int) bool {
			return n.children[i].name < n.children[j].name
		})
		for _, child := range n.children {
			sortChildren(child)
		}
	}
	if root != nil {
		sortChildren(root)
	}

	// Flatten tree to display list with prefixes
	var result []displayBranch
	var flatten func(n *branchNode, prefix string, continuationPrefix string)
	flatten = func(n *branchNode, prefix string, continuationPrefix string) {
		result = append(result, displayBranch{
			name:      n.name,
			isCurrent: n.isCurrent,
			isMerged:  n.isMerged,
			prefix:    prefix,
		})

		for i, child := range n.children {
			isLast := i == len(n.children)-1
			var childPrefix, nextContinuation string

			if isLast {
				childPrefix = continuationPrefix + "└── "
				nextContinuation = continuationPrefix + "    "
			} else {
				childPrefix = continuationPrefix + "├── "
				nextContinuation = continuationPrefix + "│   "
			}

			flatten(child, childPrefix, nextContinuation)
		}
	}

	if root != nil {
		flatten(root, "", "")
	}

	// Add any orphan branches (not connected to tree)
	inResult := make(map[string]bool)
	for _, b := range result {
		inResult[b.name] = true
	}
	for _, b := range branches {
		if !inResult[b.name] {
			result = append(result, displayBranch{
				name:      b.name,
				isCurrent: b.isCurrent,
				isMerged:  b.isMerged,
				prefix:    "",
			})
		}
	}

	return result, nil
}

func gitCheckout(branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var checkoutCmd = &cobra.Command{
	Use:   "checkout [branch]",
	Short: "Switch branches interactively",
	Long:  `Switch branches. If no branch is specified, opens an interactive selector with tree view.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			if err := gitCheckout(args[0]); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			return
		}

		branches, err := buildBranchTree()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error getting branches:", err)
			os.Exit(1)
		}

		if len(branches) == 0 {
			fmt.Println("No branches found")
			return
		}

		startIdx := 0
		for i, b := range branches {
			if b.isCurrent {
				startIdx = i
				break
			}
		}

		m := model{branches: branches, cursor: startIdx}
		p := tea.NewProgram(m)
		finalModel, err := p.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		final := finalModel.(model)
		if final.selected != "" && !final.branches[final.cursor].isCurrent {
			if err := gitCheckout(final.selected); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
