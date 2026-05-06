package cmd

import (
	"fmt"
	"os"

	"sshc/internal/config"
	"sshc/internal/ssh"

	"github.com/spf13/cobra"
)

var (
	createKey    bool
	keyType      string
	keySize      int
	keyComment   string
	host         string
	hostname     string
	user         string
	port         int
	identity     string
	forwardAgent string
	proxyJump    string
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new SSH configuration",
}

var addConfigCmd = &cobra.Command{
	Use:   "config NAME",
	Short: "Add a new SSH config entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		m, err := config.NewManager()
		if err != nil {
			return err
		}
		m.SetDryRun(dryRun)

		idFile := identity
		if idFile != "" && !createKey {
			if _, err := os.Stat(idFile); os.IsNotExist(err) {
				fmt.Printf("Warning: IdentityFile %s does not exist\n", idFile)
			}
		}

		if createKey {
			if idFile == "" {
				idFile, err = ssh.GetDefaultKeyPath(name)
				if err != nil {
					return err
				}
			}
			if dryRun {
				fmt.Printf("[Dry-run] Would generate %s key at %s...\n", keyType, idFile)
			} else {
				fmt.Printf("Generating %s key at %s...\n", keyType, idFile)
			}
			err = ssh.GenerateKey(idFile, ssh.KeyType(keyType), keySize, keyComment, dryRun)
			if err != nil {
				return fmt.Errorf("failed to generate key: %w", err)
			}
		}

		if host == "" {
			host = name
		}

		opts := config.ConfigOptions{
			Host:         host,
			Hostname:     hostname,
			User:         user,
			Port:         port,
			IdentityFile: idFile,
			ForwardAgent: forwardAgent,
			ProxyJump:    proxyJump,
		}

		if err := opts.Validate(); err != nil {
			return err
		}

		if err := m.AddConfig(name, opts.String()); err != nil {
			return err
		}

		if dryRun {
			fmt.Printf("Config %s would be added successfully (dry-run)\n", name)
		} else {
			fmt.Printf("Config %s added successfully\n", name)
		}
		return nil
	},
}

func init() {
	addConfigCmd.Flags().BoolVar(&createKey, "create-key", false, "Create a new SSH key")
	addConfigCmd.Flags().StringVar(&keyType, "key-type", "ed25519", "SSH key type (rsa, ed25519, ecdsa)")
	addConfigCmd.Flags().IntVar(&keySize, "key-size", 0, "SSH key size (bits, only for rsa and ecdsa)")
	addConfigCmd.Flags().StringVar(&keyComment, "key-comment", "", "SSH key comment")
	addConfigCmd.Flags().StringVar(&host, "host", "", "SSH Host alias")
	addConfigCmd.Flags().StringVar(&hostname, "hostname", "", "SSH Hostname (real address)")
	addConfigCmd.Flags().StringVar(&user, "user", "", "SSH User")
	addConfigCmd.Flags().IntVar(&port, "port", 0, "SSH Port")
	addConfigCmd.Flags().StringVar(&identity, "identity", "", "Path to identity file (if not creating one)")
	addConfigCmd.Flags().StringVar(&forwardAgent, "forward-agent", "", "Forward SSH Agent (yes/no)")
	addConfigCmd.Flags().StringVar(&proxyJump, "proxy-jump", "", "SSH ProxyJump")

	addCmd.AddCommand(addConfigCmd)
	rootCmd.AddCommand(addCmd)
}
