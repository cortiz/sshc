package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	SshcDirName = "sshc.d"
	IncludeLine = "Include sshc.d/*"
)

type Manager struct {
	SshDir string
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	return &Manager{
		SshDir: filepath.Join(home, ".ssh"),
	}, nil
}

func (m *Manager) Init() error {
	sshcDir := filepath.Join(m.SshDir, SshcDirName)
	if err := os.MkdirAll(sshcDir, 0700); err != nil {
		return fmt.Errorf("failed to create sshc directory: %w", err)
	}

	configFile := filepath.Join(m.SshDir, "config")
	backupFile := configFile + ".backup"

	if _, err := os.Stat(configFile); err == nil {
		// 1. backup current ssh config if backup doesn't exist
		if _, err := os.Stat(backupFile); os.IsNotExist(err) {
			content, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to read ssh config for backup: %w", err)
			}
			if err := os.WriteFile(backupFile, content, 0600); err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}

		// 2. Create a new ssh config with the include
		content, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read ssh config: %w", err)
		}
		// Check if Include line already exists
		found := false
		for line := range strings.SplitSeq(string(content), "\n") {
			if strings.TrimSpace(line) == IncludeLine {
				found = true
				break
			}
		}

		if !found {
			newContent := IncludeLine + "\n" + string(content)
			if err := os.WriteFile(configFile, []byte(newContent), 0600); err != nil {
				return fmt.Errorf("failed to update ssh config: %w", err)
			}
		}
	} else if os.IsNotExist(err) {
		// Create a new ssh config with the include
		if err := os.WriteFile(configFile, []byte(IncludeLine+"\n"), 0600); err != nil {
			return fmt.Errorf("failed to create ssh config: %w", err)
		}
	} else {
		return fmt.Errorf("failed to stat ssh config: %w", err)
	}

	return nil
}

func (m *Manager) GetConfigPath(name string) string {
	return filepath.Join(m.SshDir, SshcDirName, name)
}

func (m *Manager) AddConfig(name string, content string) error {
	configPath := m.GetConfigPath(name)
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write config %s: %w", name, err)
	}
	return nil
}

func (m *Manager) RemoveConfig(name string) error {
	_, err := m.RemoveConfigWithKey(name, false)
	return err
}

func (m *Manager) RemoveConfigWithKey(name string, deleteKey bool) (string, error) {
	configPath := m.GetConfigPath(name)
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("config %s does not exist", name)
		}
		return "", fmt.Errorf("failed to read config %s: %w", name, err)
	}

	var identityFile string
	for line := range strings.SplitSeq(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(trimmed), "identityfile ") {
			identityFile = strings.TrimSpace(trimmed[len("identityfile "):])
			break
		}
	}

	if err := os.Remove(configPath); err != nil {
		return "", fmt.Errorf("failed to remove config %s: %w", name, err)
	}

	if deleteKey && identityFile != "" {
		// Try to remove the private key
		_ = os.Remove(identityFile)
		// Try to remove the public key
		_ = os.Remove(identityFile + ".pub")
	}

	return identityFile, nil
}

type ConfigOptions struct {
	Host         string
	Hostname     string
	User         string
	Port         int
	IdentityFile string
	ForwardAgent string // "yes", "no", or ""
	ProxyJump    string
}

func (m *Manager) UpdateConfig(name string, opts ConfigOptions) error {
	path := m.GetConfigPath(name)
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	newLines := make([]string, 0, len(lines))

	foundFields := make(map[string]bool)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lowerTrimmed := strings.ToLower(trimmed)

		if strings.HasPrefix(lowerTrimmed, "host ") && opts.Host != "" {
			newLines = append(newLines, "Host "+opts.Host)
			foundFields["host"] = true
		} else if strings.HasPrefix(lowerTrimmed, "hostname ") {
			foundFields["hostname"] = true
			if opts.Hostname != "" {
				indent := line[:strings.Index(strings.ToLower(line), "hostname")]
				newLines = append(newLines, indent+"Hostname "+opts.Hostname)
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "user ") {
			foundFields["user"] = true
			if opts.User != "" {
				indent := line[:strings.Index(strings.ToLower(line), "user")]
				newLines = append(newLines, indent+"User "+opts.User)
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "port ") {
			foundFields["port"] = true
			if opts.Port != 0 {
				indent := line[:strings.Index(strings.ToLower(line), "port")]
				newLines = append(newLines, indent+fmt.Sprintf("Port %d", opts.Port))
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "identityfile ") {
			foundFields["identityfile"] = true
			if opts.IdentityFile != "" {
				identity := opts.IdentityFile
				// Ensure path is absolute for SSH config
				if absPath, err := filepath.Abs(identity); err == nil {
					identity = absPath
				}
				indent := line[:strings.Index(strings.ToLower(line), "identityfile")]
				newLines = append(newLines, indent+"IdentityFile "+identity)
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "forwardagent ") {
			foundFields["forwardagent"] = true
			if opts.ForwardAgent != "" {
				indent := line[:strings.Index(strings.ToLower(line), "forwardagent")]
				newLines = append(newLines, indent+"ForwardAgent "+opts.ForwardAgent)
			} else {
				newLines = append(newLines, line)
			}
		} else if strings.HasPrefix(lowerTrimmed, "proxyjump ") {
			foundFields["proxyjump"] = true
			if opts.ProxyJump != "" {
				indent := line[:strings.Index(strings.ToLower(line), "proxyjump")]
				newLines = append(newLines, indent+"ProxyJump "+opts.ProxyJump)
			} else {
				newLines = append(newLines, line)
			}
		} else {
			if trimmed != "" || line != "" {
				newLines = append(newLines, line)
			}
		}
	}

	// Add fields if they were not present but are now requested
	if !foundFields["user"] && opts.User != "" {
		newLines = append(newLines, "    User "+opts.User)
	}
	if !foundFields["hostname"] && opts.Hostname != "" {
		newLines = append(newLines, "    Hostname "+opts.Hostname)
	}
	if !foundFields["port"] && opts.Port != 0 {
		newLines = append(newLines, fmt.Sprintf("    Port %d", opts.Port))
	}
	if !foundFields["identityfile"] && opts.IdentityFile != "" {
		identity := opts.IdentityFile
		if absPath, err := filepath.Abs(identity); err == nil {
			identity = absPath
		}
		newLines = append(newLines, "    IdentityFile "+identity)
	}
	if !foundFields["forwardagent"] && opts.ForwardAgent != "" {
		newLines = append(newLines, "    ForwardAgent "+opts.ForwardAgent)
	}
	if !foundFields["proxyjump"] && opts.ProxyJump != "" {
		newLines = append(newLines, "    ProxyJump "+opts.ProxyJump)
	}

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0600)
}

func (opts ConfigOptions) String() string {
	var sb strings.Builder
	if opts.Host != "" {
		sb.WriteString("Host " + opts.Host + "\n")
	}
	if opts.Hostname != "" {
		sb.WriteString("    Hostname " + opts.Hostname + "\n")
	}
	if opts.User != "" {
		sb.WriteString("    User " + opts.User + "\n")
	}
	if opts.Port != 0 {
		sb.WriteString(fmt.Sprintf("    Port %d\n", opts.Port))
	}
	if opts.IdentityFile != "" {
		identity := opts.IdentityFile
		if absPath, err := filepath.Abs(identity); err == nil {
			identity = absPath
		}
		sb.WriteString("    IdentityFile " + identity + "\n")
	}
	if opts.ForwardAgent != "" {
		sb.WriteString("    ForwardAgent " + opts.ForwardAgent + "\n")
	}
	if opts.ProxyJump != "" {
		sb.WriteString("    ProxyJump " + opts.ProxyJump + "\n")
	}
	return sb.String()
}

func (m *Manager) ListConfigs() ([]string, error) {
	sshcDir := filepath.Join(m.SshDir, SshcDirName)
	entries, err := os.ReadDir(sshcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read sshc directory: %w", err)
	}

	var configs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			configs = append(configs, entry.Name())
		}
	}
	slices.Sort(configs)
	return configs, nil
}
