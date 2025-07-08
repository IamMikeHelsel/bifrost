package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// CryptoManager handles encryption and decryption operations
type CryptoManager struct {
	gcm cipher.AEAD
}

// NewCryptoManager creates a new crypto manager with AES-256-GCM
func NewCryptoManager(key []byte) (*CryptoManager, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM mode: %w", err)
	}

	return &CryptoManager{gcm: gcm}, nil
}

// NewCryptoManagerFromPassword creates a crypto manager using PBKDF2 key derivation
func NewCryptoManagerFromPassword(password, salt string) (*CryptoManager, error) {
	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}

	// Use PBKDF2 with SHA-256, 100,000 iterations for key derivation
	key := pbkdf2.Key([]byte(password), []byte(salt), 100000, 32, sha256.New)
	return NewCryptoManager(key)
}

// Encrypt encrypts plaintext using AES-256-GCM
func (cm *CryptoManager) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, cm.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := cm.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (cm *CryptoManager) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := cm.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := cm.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64 encoded result
func (cm *CryptoManager) EncryptString(plaintext string) (string, error) {
	encrypted, err := cm.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// DecryptString decrypts a base64 encoded string
func (cm *CryptoManager) DecryptString(ciphertext string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := cm.Decrypt(decoded)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// GenerateKey generates a random 256-bit key for AES-256
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 generates a random 256-bit key and returns it as base64
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}