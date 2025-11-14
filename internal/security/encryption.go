package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	// Encryption constants
	keySize     = 32 // AES-256
	nonceSize   = 12 // GCM nonce size
	saltSize    = 32 // Salt size for key derivation
	hmacKeySize = 32 // HMAC key size

	// Argon2id parameters for OTP hashing
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32

	// Encoded format: salt(32) + nonce(12) + ciphertext + hmac(32)
	minEncryptedSize = saltSize + nonceSize + hmacKeySize
)

type CryptoService struct {
	masterKey []byte
}

// NewCryptoService creates a new encryption service
func NewCryptoService() (*CryptoService, error) {
	masterKeyB64 := os.Getenv("ENCRYPTION_MASTER_KEY")
	if masterKeyB64 == "" {
		return nil, fmt.Errorf("ENCRYPTION_MASTER_KEY environment variable not set")
	}

	masterKey, err := base64.StdEncoding.DecodeString(masterKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}

	if len(masterKey) != keySize {
		return nil, fmt.Errorf("master key must be %d bytes, got %d", keySize, len(masterKey))
	}

	return &CryptoService{
		masterKey: masterKey,
	}, nil
}

// GenerateMasterKey generates a new master key for the ENCRYPTION_MASTER_KEY env var
func GenerateMasterKey() (string, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate master key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// deriveKey derives an encryption key from master key + context + salt
func (cs *CryptoService) deriveKey(context string, salt []byte) []byte {
	return argon2.IDKey(
		append(cs.masterKey, []byte(context)...),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		keySize,
	)
}

// deriveHMACKey derives an HMAC key from master key + context + salt
func (cs *CryptoService) deriveHMACKey(context string, salt []byte) []byte {
	return argon2.IDKey(
		append(cs.masterKey, []byte("hmac-"+context)...),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		hmacKeySize,
	)
}

// EncryptEmail encrypts an email with deterministic encryption for queryability
func (cs *CryptoService) EncryptEmail(email string) (string, error) {
	if email == "" {
		return "", fmt.Errorf("email cannot be empty")
	}

	// Use email as salt source for deterministic encryption
	// This allows the same email to always produce the same ciphertext for queries
	emailHash := sha256.Sum256([]byte(email))
	salt := emailHash[:saltSize]

	// Derive encryption key
	key := cs.deriveKey("email", salt)

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate deterministic nonce from email for consistent encryption
	nonceHash := sha256.Sum256(append([]byte("nonce"), []byte(email)...))
	nonce := nonceHash[:nonceSize]

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, []byte(email), nil)

	// Create HMAC for integrity
	hmacKey := cs.deriveHMACKey("email", salt)
	h := hmac.New(sha256.New, hmacKey)
	h.Write(salt)
	h.Write(nonce)
	h.Write(ciphertext)
	tag := h.Sum(nil)

	// Combine: salt + nonce + ciphertext + hmac
	result := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext)+len(tag))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	result = append(result, tag...)

	return base64.StdEncoding.EncodeToString(result), nil
}

// DecryptEmail decrypts an encrypted email
func (cs *CryptoService) DecryptEmail(encryptedEmail string) (string, error) {
	if encryptedEmail == "" {
		return "", fmt.Errorf("encrypted email cannot be empty")
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(encryptedEmail)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted email: %w", err)
	}

	if len(data) < minEncryptedSize {
		return "", fmt.Errorf("encrypted email too short")
	}

	// Extract components
	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	hmacTag := data[len(data)-hmacKeySize:]
	ciphertext := data[saltSize+nonceSize : len(data)-hmacKeySize]

	// Verify HMAC
	hmacKey := cs.deriveHMACKey("email", salt)
	h := hmac.New(sha256.New, hmacKey)
	h.Write(salt)
	h.Write(nonce)
	h.Write(ciphertext)
	expectedTag := h.Sum(nil)

	if subtle.ConstantTimeCompare(hmacTag, expectedTag) != 1 {
		return "", fmt.Errorf("HMAC verification failed - data may be tampered")
	}

	// Derive decryption key
	key := cs.deriveKey("email", salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt email: %w", err)
	}

	return string(plaintext), nil
}

// HashOTP creates a one-way hash of an OTP with random salt
func (cs *CryptoService) HashOTP(otp string) (string, error) {
	if otp == "" {
		return "", fmt.Errorf("OTP cannot be empty")
	}

	// Generate random salt for each OTP
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash with Argon2id
	hash := argon2.IDKey(
		[]byte(otp),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	// Create HMAC for additional integrity
	hmacKey := cs.deriveHMACKey("otp", salt)
	h := hmac.New(sha256.New, hmacKey)
	h.Write(salt)
	h.Write(hash)
	tag := h.Sum(nil)

	// Combine: salt + hash + hmac
	result := make([]byte, 0, len(salt)+len(hash)+len(tag))
	result = append(result, salt...)
	result = append(result, hash...)
	result = append(result, tag...)

	return base64.StdEncoding.EncodeToString(result), nil
}

// VerifyOTP verifies an OTP against its hash
func (cs *CryptoService) VerifyOTP(otp, hashedOTP string) (bool, error) {
	if otp == "" || hashedOTP == "" {
		return false, fmt.Errorf("OTP and hash cannot be empty")
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(hashedOTP)
	if err != nil {
		return false, fmt.Errorf("failed to decode hashed OTP: %w", err)
	}

	expectedSize := saltSize + argonKeyLen + hmacKeySize
	if len(data) != expectedSize {
		return false, fmt.Errorf("invalid hashed OTP length")
	}

	// Extract components
	salt := data[:saltSize]
	storedHash := data[saltSize : saltSize+argonKeyLen]
	hmacTag := data[saltSize+argonKeyLen:]

	// Verify HMAC
	hmacKey := cs.deriveHMACKey("otp", salt)
	h := hmac.New(sha256.New, hmacKey)
	h.Write(salt)
	h.Write(storedHash)
	expectedTag := h.Sum(nil)

	if subtle.ConstantTimeCompare(hmacTag, expectedTag) != 1 {
		return false, fmt.Errorf("HMAC verification failed - data may be tampered")
	}

	// Hash the provided OTP with the same salt
	candidateHash := argon2.IDKey(
		[]byte(otp),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(storedHash, candidateHash) == 1, nil
}

// EncryptSensitiveData encrypts any sensitive data with random encryption
func (cs *CryptoService) EncryptSensitiveData(data string, context string) (string, error) {
	if data == "" {
		return "", fmt.Errorf("data cannot be empty")
	}

	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Derive encryption key
	key := cs.deriveKey(context, salt)

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, []byte(data), nil)

	// Create HMAC for integrity
	hmacKey := cs.deriveHMACKey(context, salt)
	h := hmac.New(sha256.New, hmacKey)
	h.Write(salt)
	h.Write(nonce)
	h.Write(ciphertext)
	tag := h.Sum(nil)

	// Combine: salt + nonce + ciphertext + hmac
	result := make([]byte, 0, len(salt)+len(nonce)+len(ciphertext)+len(tag))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	result = append(result, tag...)

	return base64.StdEncoding.EncodeToString(result), nil
}

// DecryptSensitiveData decrypts sensitive data
func (cs *CryptoService) DecryptSensitiveData(encryptedData string, context string) (string, error) {
	if encryptedData == "" {
		return "", fmt.Errorf("encrypted data cannot be empty")
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	if len(data) < minEncryptedSize {
		return "", fmt.Errorf("encrypted data too short")
	}

	// Extract components
	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	hmacTag := data[len(data)-hmacKeySize:]
	ciphertext := data[saltSize+nonceSize : len(data)-hmacKeySize]

	// Verify HMAC
	hmacKey := cs.deriveHMACKey(context, salt)
	h := hmac.New(sha256.New, hmacKey)
	h.Write(salt)
	h.Write(nonce)
	h.Write(ciphertext)
	expectedTag := h.Sum(nil)

	if subtle.ConstantTimeCompare(hmacTag, expectedTag) != 1 {
		return "", fmt.Errorf("HMAC verification failed - data may be tampered")
	}

	// Derive decryption key
	key := cs.deriveKey(context, salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}

	return string(plaintext), nil
}

// SecureCompare performs a constant-time string comparison
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SecureWipe overwrites sensitive data in memory
func SecureWipe(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// RateLimitKey generates a rate limiting key for timing attack protection
func (cs *CryptoService) RateLimitKey(identifier string) string {
	h := hmac.New(sha256.New, cs.masterKey)
	h.Write([]byte("ratelimit"))
	h.Write([]byte(identifier))
	h.Write([]byte(time.Now().Format("2006-01-02-15"))) // Hour-based rate limiting
	return base64.StdEncoding.EncodeToString(h.Sum(nil))[:16]
}
