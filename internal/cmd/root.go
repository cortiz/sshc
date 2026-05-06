package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sshc",
	Short: "sshc is a CLI tool to manage your SSH configuration and keys",
	Long: `A simple SSH config helper/manager that allows you to:
1. Configure the main SSH config
2. Add ssh config using the include
3. Manage SSH key creation
4. Use git-like commands to manage your configs`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Root flags can be added here
}
