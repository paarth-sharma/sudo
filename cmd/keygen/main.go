package main

import (
	"fmt"
	"log"
	"os"
	"sudo/internal/security"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--test" {
		testEncryption()
		return
	}

	fmt.Println("SUDO Kanban - Encryption Key Generator")
	fmt.Println("=========================================")
	fmt.Println()

	// Generate master key
	masterKey, err := security.GenerateMasterKey()
	if err != nil {
		log.Fatalf("❌ Failed to generate master key: %v", err)
	}

	fmt.Println("Master key generated successfully!")
	fmt.Println()
	fmt.Println("Your ENCRYPTION_MASTER_KEY:")
	fmt.Printf("   %s\n", masterKey)
	fmt.Println()
	fmt.Println("Setup Instructions:")
	fmt.Println("   1. Save this key in a secure location")
	fmt.Println("   2. Set as environment variable:")
	fmt.Println("      export ENCRYPTION_MASTER_KEY=\"" + masterKey + "\"")
	fmt.Println()
	fmt.Println("SECURITY WARNING:")
	fmt.Println("   • Never commit this key to source control")
	fmt.Println("   • Store securely - losing this key means data loss")
	fmt.Println("   • Use different keys for dev/staging/production")
	fmt.Println("   • Consider using cloud key management services")
	fmt.Println()
	fmt.Println("To test the encryption system:")
	fmt.Printf("   export ENCRYPTION_MASTER_KEY=\"%s\"\n", masterKey)
	fmt.Println("   go run cmd/keygen/main.go --test")
}

func testEncryption() {
	fmt.Println("Testing encryption functionality...")
	fmt.Println()

	// Check if master key is set, if not use a test key
	if os.Getenv("ENCRYPTION_MASTER_KEY") == "" {
		// Use a test key for demonstration
		testKey := "PuPdaOvxf9KPbAXwL+ZJ9qf3bW+igdh7SwAQqIci4yQ="
		os.Setenv("ENCRYPTION_MASTER_KEY", testKey)
		fmt.Printf("Using test key: %s...\n", testKey[:20])
		fmt.Println()
	}

	// Create crypto service
	crypto, err := security.NewCryptoService()
	if err != nil {
		fmt.Printf("Failed to create crypto service: %v\n", err)
		return
	}

	// Test email encryption
	fmt.Println("Testing email encryption...")
	testEmail := "test@example.com"

	encrypted, err := crypto.EncryptEmail(testEmail)
	if err != nil {
		fmt.Printf("Email encryption failed: %v\n", err)
		return
	}

	decrypted, err := crypto.DecryptEmail(encrypted)
	if err != nil {
		fmt.Printf("Email decryption failed: %v\n", err)
		return
	}

	if decrypted != testEmail {
		fmt.Printf("Email roundtrip failed: got %s, expected %s\n", decrypted, testEmail)
		return
	}

	fmt.Printf("   Email: %s -> %s... -> %s\n", testEmail, encrypted[:20], decrypted)

	// Test OTP hashing
	fmt.Println("Testing OTP hashing...")
	testOTP := "123456"

	hashed, err := crypto.HashOTP(testOTP)
	if err != nil {
		fmt.Printf("OTP hashing failed: %v\n", err)
		return
	}

	valid, err := crypto.VerifyOTP(testOTP, hashed)
	if err != nil {
		fmt.Printf("OTP verification failed: %v\n", err)
		return
	}

	if !valid {
		fmt.Println("OTP verification returned false for valid OTP")
		return
	}

	// Test with wrong OTP
	wrongValid, err := crypto.VerifyOTP("654321", hashed)
	if err != nil {
		fmt.Printf("Wrong OTP verification failed: %v\n", err)
		return
	}

	if wrongValid {
		fmt.Println("OTP verification returned true for invalid OTP")
		return
	}

	fmt.Printf("   OTP: %s -> %s... -> verified\n", testOTP, hashed[:20])
	fmt.Printf("   Wrong OTP rejected\n")

	fmt.Println()
	fmt.Println("All tests passed! Encryption system is working correctly.")
	fmt.Println()
	fmt.Println("Performance note:")
	fmt.Println("   • Email encryption: Fast (designed for frequent use)")
	fmt.Println("   • OTP hashing: Intentionally slow (prevents brute force)")
	fmt.Println()
	fmt.Println("Ready for deployment!")
}
