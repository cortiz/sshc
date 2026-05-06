package ssh

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

type KeyType string

const (
	RSA     KeyType = "rsa"
	ED25519 KeyType = "ed25519"
	ECDSA   KeyType = "ecdsa"
)

func GenerateKey(path string, keyType KeyType, bits int, comment string, dryRun bool) error {
	var privateKey any
	var publicKey ssh.PublicKey
	var err error

	if dryRun {
		fmt.Printf("[Dry-run] Would generate %s key (%d bits) at %s\n", keyType, bits, path)
		return nil
	}

	switch keyType {
	case RSA:
		if bits == 0 {
			bits = 4096
		}
		priv, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return err
		}
		privateKey = priv
		publicKey, err = ssh.NewPublicKey(&priv.PublicKey)
		if err != nil {
			return err
		}
	case ED25519:
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		privateKey = priv
		publicKey, err = ssh.NewPublicKey(pub)
		if err != nil {
			return err
		}
	case ECDSA:
		var curve elliptic.Curve
		switch bits {
		case 256, 0:
			curve = elliptic.P256()
		case 384:
			curve = elliptic.P384()
		case 521:
			curve = elliptic.P521()
		default:
			return fmt.Errorf("unsupported ecdsa bits: %d (use 256, 384, or 521)", bits)
		}
		priv, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return err
		}
		privateKey = priv
		publicKey, err = ssh.NewPublicKey(&priv.PublicKey)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}

	// Save private key
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}
	privBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(privBlock), 0600); err != nil {
		return err
	}

	// Save public key
	pubBytes := ssh.MarshalAuthorizedKey(publicKey)
	if comment != "" {
		// MarshalAuthorizedKey returns "type key\n"
		content := strings.TrimSpace(string(pubBytes))
		pubBytes = []byte(fmt.Sprintf("%s %s\n", content, comment))
	}
	if err := os.WriteFile(path+".pub", pubBytes, 0644); err != nil {
		return err
	}

	return nil
}

func GetDefaultKeyPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", name), nil
}
