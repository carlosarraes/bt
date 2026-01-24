package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// CredentialStorage defines the interface for secure credential storage
type CredentialStorage interface {
	// Store saves credentials securely
	Store(key string, value interface{}) error

	// Retrieve gets stored credentials
	Retrieve(key string, dest interface{}) error

	// Delete removes stored credentials
	Delete(key string) error

	// Clear removes all stored credentials
	Clear() error

	// Exists checks if a credential exists
	Exists(key string) bool
}

// StoredCredentials represents the structure of stored authentication data
type StoredCredentials struct {
	Method       AuthMethod `json:"method"`
	Username     string     `json:"username,omitempty"`
	Password     string     `json:"password,omitempty"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	TokenExpiry  time.Time  `json:"token_expiry,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// FileCredentialStorage implements CredentialStorage using encrypted files
type FileCredentialStorage struct {
	configDir string
	password  []byte
}

// NewFileCredentialStorage creates a new file-based credential storage
func NewFileCredentialStorage() (CredentialStorage, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate or retrieve encryption key
	password, err := getOrCreateEncryptionKey(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	return &FileCredentialStorage{
		configDir: configDir,
		password:  password,
	}, nil
}

func (s *FileCredentialStorage) Store(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, err := s.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	filePath := filepath.Join(s.configDir, key+".enc")
	if err := os.WriteFile(filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

func (s *FileCredentialStorage) Retrieve(key string, dest interface{}) error {
	filePath := filepath.Join(s.configDir, key+".enc")

	encrypted, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("credentials not found for key: %s", key)
		}
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	data, err := s.decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return nil
}

func (s *FileCredentialStorage) Delete(key string) error {
	filePath := filepath.Join(s.configDir, key+".enc")

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete credentials file: %w", err)
	}

	return nil
}

func (s *FileCredentialStorage) Clear() error {
	files, err := filepath.Glob(filepath.Join(s.configDir, "*.enc"))
	if err != nil {
		return fmt.Errorf("failed to list credential files: %w", err)
	}

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("failed to remove credential file %s: %w", file, err)
		}
	}

	return nil
}

func (s *FileCredentialStorage) Exists(key string) bool {
	filePath := filepath.Join(s.configDir, key+".enc")
	_, err := os.Stat(filePath)
	return err == nil
}

// encrypt encrypts data using AES-GCM
func (s *FileCredentialStorage) encrypt(data []byte) ([]byte, error) {
	// Derive key from password using PBKDF2
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key(s.password, salt, 100000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encrypted := gcm.Seal(nil, nonce, data, nil)

	// Prepend salt and nonce to encrypted data
	result := make([]byte, len(salt)+len(nonce)+len(encrypted))
	copy(result, salt)
	copy(result[len(salt):], nonce)
	copy(result[len(salt)+len(nonce):], encrypted)

	return result, nil
}

// decrypt decrypts data using AES-GCM
func (s *FileCredentialStorage) decrypt(data []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	// Extract salt and nonce
	salt := data[:16]

	// Derive key from password
	key := pbkdf2.Key(s.password, salt, 100000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < 16+nonceSize {
		return nil, fmt.Errorf("encrypted data too short for nonce")
	}

	nonce := data[16 : 16+nonceSize]
	encrypted := data[16+nonceSize:]

	decrypted, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

// getConfigDir returns the configuration directory for the bt CLI
func getConfigDir() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(configDir, "bt")
	default:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config", "bt")
	}

	return configDir, nil
}

// getOrCreateEncryptionKey gets or creates the encryption key for credential storage
func getOrCreateEncryptionKey(configDir string) ([]byte, error) {
	keyFile := filepath.Join(configDir, ".key")

	// Try to read existing key
	if data, err := os.ReadFile(keyFile); err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil && len(decoded) == 32 {
			return decoded, nil
		}
	}

	// Generate new key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Save key to file
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(keyFile, []byte(encoded), 0600); err != nil {
		return nil, fmt.Errorf("failed to save encryption key: %w", err)
	}

	return key, nil
}
