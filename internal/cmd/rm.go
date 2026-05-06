package cmd

import (
	"fmt"

	"sshc/internal/config"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove SSH configuration or keys",
}

var rmConfigCmd = &cobra.Command{
	Use:   "config NAME",
	Short: "Remove an SSH config entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m, err := config.NewManager()
		if err != nil {
			return err
		}

		if err := m.RemoveConfig(name); err != nil {
			return err
		}

		fmt.Printf("Config %s removed successfully\n", name)
		return nil
	},
}

func init() {
	rmCmd.AddCommand(rmConfigCmd)
	rootCmd.AddCommand(rmCmd)
}
