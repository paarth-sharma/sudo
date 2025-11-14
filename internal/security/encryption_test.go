package security

import (
	"os"
	"testing"
)

func TestEncryptionFunctionality(t *testing.T) {
	// Generate a test master key
	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate master key: %v", err)
	}

	// Set it in environment for testing
	os.Setenv("ENCRYPTION_MASTER_KEY", masterKey)
	defer os.Unsetenv("ENCRYPTION_MASTER_KEY")

	// Create crypto service
	crypto, err := NewCryptoService()
	if err != nil {
		t.Fatalf("Failed to create crypto service: %v", err)
	}

	t.Run("EmailEncryption", func(t *testing.T) {
		testEmail := "test@example.com"

		// Test encryption
		encrypted, err := crypto.EncryptEmail(testEmail)
		if err != nil {
			t.Fatalf("Failed to encrypt email: %v", err)
		}

		if encrypted == testEmail {
			t.Error("Encrypted email should not equal plaintext")
		}

		// Test decryption
		decrypted, err := crypto.DecryptEmail(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt email: %v", err)
		}

		if decrypted != testEmail {
			t.Errorf("Decrypted email %s doesn't match original %s", decrypted, testEmail)
		}

		// Test deterministic encryption (same email should produce same ciphertext)
		encrypted2, err := crypto.EncryptEmail(testEmail)
		if err != nil {
			t.Fatalf("Failed to encrypt email second time: %v", err)
		}

		if encrypted != encrypted2 {
			t.Error("Email encryption should be deterministic")
		}
	})

	t.Run("OTPHashing", func(t *testing.T) {
		testOTP := "123456"

		// Test hashing
		hashed, err := crypto.HashOTP(testOTP)
		if err != nil {
			t.Fatalf("Failed to hash OTP: %v", err)
		}

		if hashed == testOTP {
			t.Error("Hashed OTP should not equal plaintext")
		}

		// Test verification with correct OTP
		valid, err := crypto.VerifyOTP(testOTP, hashed)
		if err != nil {
			t.Fatalf("Failed to verify OTP: %v", err)
		}

		if !valid {
			t.Error("Valid OTP should verify successfully")
		}

		// Test verification with incorrect OTP
		valid, err = crypto.VerifyOTP("654321", hashed)
		if err != nil {
			t.Fatalf("Failed to verify incorrect OTP: %v", err)
		}

		if valid {
			t.Error("Invalid OTP should not verify")
		}

		// Test non-deterministic hashing (same OTP should produce different hashes)
		hashed2, err := crypto.HashOTP(testOTP)
		if err != nil {
			t.Fatalf("Failed to hash OTP second time: %v", err)
		}

		if hashed == hashed2 {
			t.Error("OTP hashing should not be deterministic")
		}

		// But both should verify the same OTP
		valid, err = crypto.VerifyOTP(testOTP, hashed2)
		if err != nil {
			t.Fatalf("Failed to verify second hash: %v", err)
		}

		if !valid {
			t.Error("Second hash should also verify the same OTP")
		}
	})

	t.Run("SensitiveDataEncryption", func(t *testing.T) {
		testData := "sensitive information"
		context := "test-context"

		// Test encryption
		encrypted, err := crypto.EncryptSensitiveData(testData, context)
		if err != nil {
			t.Fatalf("Failed to encrypt sensitive data: %v", err)
		}

		if encrypted == testData {
			t.Error("Encrypted data should not equal plaintext")
		}

		// Test decryption
		decrypted, err := crypto.DecryptSensitiveData(encrypted, context)
		if err != nil {
			t.Fatalf("Failed to decrypt sensitive data: %v", err)
		}

		if decrypted != testData {
			t.Errorf("Decrypted data %s doesn't match original %s", decrypted, testData)
		}

		// Test non-deterministic encryption (same data should produce different ciphertext)
		encrypted2, err := crypto.EncryptSensitiveData(testData, context)
		if err != nil {
			t.Fatalf("Failed to encrypt sensitive data second time: %v", err)
		}

		if encrypted == encrypted2 {
			t.Error("Sensitive data encryption should not be deterministic")
		}

		// But both should decrypt to the same value
		decrypted2, err := crypto.DecryptSensitiveData(encrypted2, context)
		if err != nil {
			t.Fatalf("Failed to decrypt second ciphertext: %v", err)
		}

		if decrypted2 != testData {
			t.Error("Second ciphertext should decrypt to same value")
		}
	})

	t.Run("SecurityFeatures", func(t *testing.T) {
		// Test empty inputs
		_, err := crypto.EncryptEmail("")
		if err == nil {
			t.Error("Should reject empty email")
		}

		_, err = crypto.HashOTP("")
		if err == nil {
			t.Error("Should reject empty OTP")
		}

		// Test SecureCompare
		if !SecureCompare("test", "test") {
			t.Error("SecureCompare should return true for identical strings")
		}

		if SecureCompare("test", "different") {
			t.Error("SecureCompare should return false for different strings")
		}

		// Test SecureWipe
		data := []byte("sensitive")
		SecureWipe(data)
		for _, b := range data {
			if b != 0 {
				t.Error("SecureWipe should zero all bytes")
				break
			}
		}
	})

	t.Run("InvalidData", func(t *testing.T) {
		// Test decryption with invalid data
		_, err := crypto.DecryptEmail("invalid-base64!")
		if err == nil {
			t.Error("Should reject invalid base64")
		}

		_, err = crypto.DecryptEmail("dmFsaWRiYXNlNjQ=") // "validbase64" but too short
		if err == nil {
			t.Error("Should reject data that's too short")
		}

		// Test OTP verification with invalid hash
		_, err = crypto.VerifyOTP("123456", "invalid-hash")
		if err == nil {
			t.Error("Should reject invalid OTP hash")
		}
	})
}

func TestMasterKeyGeneration(t *testing.T) {
	key1, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate first master key: %v", err)
	}

	key2, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("Failed to generate second master key: %v", err)
	}

	if key1 == key2 {
		t.Error("Generated master keys should be different")
	}

	if len(key1) == 0 || len(key2) == 0 {
		t.Error("Generated master keys should not be empty")
	}

	t.Logf("Sample master key: %s", key1)
	t.Log("Save this key securely as ENCRYPTION_MASTER_KEY environment variable")
}
