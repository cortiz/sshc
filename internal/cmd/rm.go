package cmd

import (
	"fmt"

	"sshc/internal/config"

	"github.com/spf13/cobra"
)

var (
	deleteKey bool
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

		idFile, err := m.RemoveConfigWithKey(name, deleteKey)
		if err != nil {
			return err
		}

		if deleteKey {
			if idFile != "" {
				fmt.Printf("Config %s and its key %s removed successfully\n", name, idFile)
			} else {
				fmt.Printf("Config %s removed successfully (no identity file found to delete)\n", name)
			}
		} else {
			fmt.Printf("Config %s removed successfully\n", name)
			if idFile != "" {
				fmt.Printf("Warning: SSH key %s was not deleted. Use --delete-key to remove it.\n", idFile)
			}
		}

		return nil
	},
}

func init() {
	rmConfigCmd.Flags().BoolVar(&deleteKey, "delete-key", false, "Delete the associated SSH key")
	rmCmd.AddCommand(rmConfigCmd)
	rootCmd.AddCommand(rmCmd)
}
