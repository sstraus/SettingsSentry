package backup

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io" // Needed for rand.Read

	"golang.org/x/crypto/pbkdf2"
)

const (
	// Salt size in bytes
	saltSize = 16
	// Nonce size for AES-GCM (standard size)
	nonceSize = 12
	// PBKDF2 iteration count (adjust as needed, higher is more secure but slower)
	pbkdf2Iterations = 600000 // Increased iteration count
	// Key size for AES-256
	aesKeySize = 32
)

// encrypt encrypts plaintext using AES-GCM with a key derived from the password via PBKDF2.
// The output format is: salt (16 bytes) + nonce (12 bytes) + ciphertext.
func encrypt(plaintext []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty for encryption")
	}

	// 1. Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// 2. Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, aesKeySize, sha256.New)

	// 3. Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// 4. Create AES-GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM cipher: %w", err)
	}

	// 5. Generate random nonce
	// Nonce size must be standard for GCM
	if gcm.NonceSize() != nonceSize {
		// This should ideally not happen with standard Go crypto/cipher
		return nil, fmt.Errorf("unexpected GCM nonce size: expected %d, got %d", nonceSize, gcm.NonceSize())
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 6. Encrypt data using GCM.Seal
	// Seal prepends the nonce for us if we pass nil for nonce, but we need salt too.
	// So, we manage nonce explicitly and prepend salt manually.
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil) // Seal appends auth tag automatically

	// 7. Combine salt, nonce, and ciphertext
	encryptedData := append(salt, nonce...)
	encryptedData = append(encryptedData, ciphertext...)

	return encryptedData, nil
}

// decrypt decrypts data encrypted by the encrypt function.
// Expects input format: salt (16 bytes) + nonce (12 bytes) + ciphertext.
func decrypt(encryptedData []byte, password string) ([]byte, error) {
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty for decryption")
	}

	// 1. Extract salt, nonce, and ciphertext
	if len(encryptedData) < saltSize+nonceSize {
		return nil, fmt.Errorf("invalid encrypted data: too short")
	}
	salt := encryptedData[:saltSize]
	nonce := encryptedData[saltSize : saltSize+nonceSize]
	ciphertext := encryptedData[saltSize+nonceSize:]

	// 2. Derive key using PBKDF2 (must use same salt and parameters as encryption)
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, aesKeySize, sha256.New)

	// 3. Create AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher for decryption: %w", err)
	}

	// 4. Create AES-GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM cipher for decryption: %w", err)
	}

	// 5. Check nonce size consistency
	if len(nonce) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce size during decryption: expected %d, got %d", gcm.NonceSize(), len(nonce))
	}

	// 6. Decrypt data using GCM.Open
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Authentication failure likely means wrong password or corrupted data
		return nil, fmt.Errorf("decryption failed (wrong password or data corruption?): %w", err)
	}

	return plaintext, nil
}
