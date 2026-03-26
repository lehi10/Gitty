package cmd

import (
	"os/exec"
	"strings"
)

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getAllBranches() ([]string, error) {
	output, err := runGitCommand("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}
	return strings.Split(output, "\n"), nil
}

func getCurrentBranch() (string, error) {
	return runGitCommand("rev-parse", "--abbrev-ref", "HEAD")
}
