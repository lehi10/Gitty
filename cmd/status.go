package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the working tree status",
	Long:  `Displays paths that have differences between the index file and the current HEAD commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		gitCmd := exec.Command("git", "status")
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		gitCmd.Stdin = os.Stdin

		if err := gitCmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
