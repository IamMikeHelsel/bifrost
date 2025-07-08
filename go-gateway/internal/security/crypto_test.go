package security

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCryptoManager(t *testing.T) {
	// Test key generation
	key, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key, 32)

	// Test crypto manager creation
	cm, err := NewCryptoManager(key)
	require.NoError(t, err)
	assert.NotNil(t, cm)

	// Test encryption/decryption
	plaintext := "Hello, Industrial IoT Security!"
	
	// Test byte encryption
	encrypted, err := cm.Encrypt([]byte(plaintext))
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, string(encrypted))

	decrypted, err := cm.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, string(decrypted))

	// Test string encryption
	encryptedStr, err := cm.EncryptString(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, encryptedStr)

	decryptedStr, err := cm.DecryptString(encryptedStr)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decryptedStr)
}

func TestCryptoManagerFromPassword(t *testing.T) {
	password := "test-password"
	salt := "test-salt"

	cm, err := NewCryptoManagerFromPassword(password, salt)
	require.NoError(t, err)
	assert.NotNil(t, cm)

	// Test encryption/decryption works
	plaintext := "Test data"
	encrypted, err := cm.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := cm.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGenerateKeyBase64(t *testing.T) {
	keyB64, err := GenerateKeyBase64()
	require.NoError(t, err)
	assert.NotEmpty(t, keyB64)
	
	// Verify it's valid base64 and correct length when decoded
	key, err := base64.StdEncoding.DecodeString(keyB64)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}