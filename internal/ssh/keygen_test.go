package ssh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshc-test-keys")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		keyType KeyType
		bits    int
		comment string
	}{
		{"rsa", RSA, 2048, "test@rsa"},
		{"ed25519", ED25519, 0, "test@ed25519"},
		{"ecdsa", ECDSA, 256, "test@ecdsa"},
	}

	for _, tt := range tests {
		t.Run(string(tt.keyType), func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name)
			if err := GenerateKey(path, tt.keyType, tt.bits, tt.comment); err != nil {
				t.Errorf("GenerateKey() error = %v", err)
			}

			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Private key not created")
			}
			if _, err := os.Stat(path + ".pub"); os.IsNotExist(err) {
				t.Errorf("Public key not created")
			} else {
				// Verify comment
				pubContent, err := os.ReadFile(path + ".pub")
				if err != nil {
					t.Fatal(err)
				}
				if tt.comment != "" && !os.IsNotExist(err) {
					if !strings.Contains(string(pubContent), tt.comment) {
						t.Errorf("Public key does not contain comment: %s", tt.comment)
					}
				}
			}
		})
	}
}
