package cmd

import (
	"fmt"

	"sshc/internal/config"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize sshc",
	Long:  `Sets up the necessary directory structure and includes in your ~/.ssh/config`,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := config.NewManager()
		if err != nil {
			return err
		}
		if err := m.Init(); err != nil {
			return err
		}
		fmt.Println("sshc initialized successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
