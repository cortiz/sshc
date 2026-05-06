package cmd

import (
	"fmt"
	"os"

	"sshc/internal/config"

	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show NAME",
	Short: "Show the content of an SSH config entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m, err := config.NewManager()
		if err != nil {
			return err
		}

		configPath := m.GetConfigPath(name)
		content, err := os.ReadFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("config %s does not exist", name)
			}
			return err
		}

		fmt.Println(string(content))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(showCmd)
}
