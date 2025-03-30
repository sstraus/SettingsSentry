package backup

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecrypt_Success(t *testing.T) {
	password := "correct-password"
	plaintext := []byte("this is secret data")

	encryptedData, err := encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("encrypt() failed: %v", err)
	}

	if len(encryptedData) <= saltSize+nonceSize {
		t.Fatalf("encrypted data is too short: len=%d", len(encryptedData))
	}

	decryptedData, err := decrypt(encryptedData, password)
	if err != nil {
		t.Fatalf("decrypt() failed: %v", err)
	}

	if !bytes.Equal(plaintext, decryptedData) {
		t.Errorf("decrypted data does not match original plaintext. Got '%s', want '%s'", string(decryptedData), string(plaintext))
	}
}

func TestDecrypt_WrongPassword(t *testing.T) {
	password := "correct-password"
	wrongPassword := "wrong-password"
	plaintext := []byte("this is secret data")

	encryptedData, err := encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("encrypt() failed: %v", err)
	}

	_, err = decrypt(encryptedData, wrongPassword)
	if err == nil {
		t.Errorf("decrypt() succeeded with wrong password, expected error")
	} else {
		// Optionally check for a specific error type or message if decrypt provides one
		t.Logf("Got expected error on wrong password: %v", err)
	}
}

func TestDecrypt_CorruptData(t *testing.T) {
	password := "correct-password"
	plaintext := []byte("this is secret data")

	encryptedData, err := encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("encrypt() failed: %v", err)
	}

	if len(encryptedData) <= saltSize+nonceSize {
		t.Fatalf("encrypted data is too short to corrupt: len=%d", len(encryptedData))
	}

	// Corrupt the ciphertext part (after salt and nonce)
	corruptIndex := saltSize + nonceSize + (len(encryptedData)-(saltSize+nonceSize))/2 // Middle of ciphertext
	if corruptIndex >= len(encryptedData) {
		corruptIndex = len(encryptedData) - 1 // Ensure index is valid
	}
	corruptedData := make([]byte, len(encryptedData))
	copy(corruptedData, encryptedData)
	corruptedData[corruptIndex] ^= 0xff // Flip some bits

	_, err = decrypt(corruptedData, password)
	if err == nil {
		t.Errorf("decrypt() succeeded with corrupted data, expected error")
	} else {
		// AES-GCM should detect corruption via authentication tag
		t.Logf("Got expected error on corrupted data: %v", err)
	}

	// Also test corrupting the nonce
	if len(encryptedData) > saltSize {
		corruptedNonceData := make([]byte, len(encryptedData))
		copy(corruptedNonceData, encryptedData)
		corruptedNonceData[saltSize] ^= 0xff // Flip a bit in the nonce
		_, err = decrypt(corruptedNonceData, password)
		if err == nil {
			t.Errorf("decrypt() succeeded with corrupted nonce, expected error")
		} else {
			t.Logf("Got expected error on corrupted nonce: %v", err)
		}
	}

	// Also test corrupting the salt (should lead to wrong key -> auth failure)
	if len(encryptedData) > 0 {
		corruptedSaltData := make([]byte, len(encryptedData))
		copy(corruptedSaltData, encryptedData)
		corruptedSaltData[0] ^= 0xff // Flip a bit in the salt
		_, err = decrypt(corruptedSaltData, password)
		if err == nil {
			t.Errorf("decrypt() succeeded with corrupted salt, expected error")
		} else {
			t.Logf("Got expected error on corrupted salt: %v", err)
		}
	}
}

func TestEncrypt_EmptyPassword(t *testing.T) {
	plaintext := []byte("some data")
	_, err := encrypt(plaintext, "")
	if err == nil {
		t.Errorf("encrypt() succeeded with empty password, expected error")
	} else {
		t.Logf("Got expected error for empty password encryption: %v", err)
	}
}

func TestDecrypt_EmptyPassword(t *testing.T) {
	// Need some dummy data that looks like encrypted data structure
	dummyData := make([]byte, saltSize+nonceSize+16) // 16 bytes dummy ciphertext
	rand.Read(dummyData)                             // Fill with random bytes

	_, err := decrypt(dummyData, "")
	if err == nil {
		t.Errorf("decrypt() succeeded with empty password, expected error")
	} else {
		t.Logf("Got expected error for empty password decryption: %v", err)
	}
}

func TestDecrypt_ShortData(t *testing.T) {
	password := "correct-password"
	shortData := make([]byte, saltSize+nonceSize-1) // Too short
	rand.Read(shortData)

	_, err := decrypt(shortData, password)
	if err == nil {
		t.Errorf("decrypt() succeeded with data shorter than salt+nonce, expected error")
	} else if !bytes.Contains([]byte(err.Error()), []byte("too short")) { // Check if error message indicates the length issue
		t.Errorf("decrypt() with short data returned unexpected error: %v", err)
	} else {
		t.Logf("Got expected error for short data: %v", err)
	}

	justEnoughData := make([]byte, saltSize+nonceSize) // Just salt+nonce, no ciphertext
	rand.Read(justEnoughData)
	_, err = decrypt(justEnoughData, password)
	if err == nil {
		t.Errorf("decrypt() succeeded with data containing only salt+nonce, expected error")
	} else {
		// Expecting an error, likely related to GCM authentication failure on empty ciphertext
		t.Logf("Got expected error for data with only salt+nonce: %v", err)
	}
}
