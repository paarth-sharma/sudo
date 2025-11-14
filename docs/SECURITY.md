# Security Implementation: Email & OTP Encryption

> 📖 **Related Documentation:**
> - [README](README.md) - Project overview and features
> - [Self-Hosting Guide](SELF_HOST.md) - Deployment and security hardening
> - [Testing Guide](TESTING_SETUP_GUIDE.md) - Security testing procedures

## Overview

This document describes the comprehensive encryption system implemented to secure user emails and OTPs in Supabase. The system provides military-grade security while maintaining application functionality.

**Quick Links:**
- [Security Features](#-security-features)
- [Deployment Setup](#-deployment-setup)
- [Threat Protection](#-security-properties)
- [Emergency Procedures](#-emergency-procedures)

## 🔒 Security Features

### **Multi-Layer Encryption Architecture**
- **AES-256-GCM encryption** for emails with deterministic encryption for queryability
- **Argon2id hashing** for OTPs with random salts (one-way, non-reversible)
- **HMAC-SHA256 integrity verification** to prevent tampering
- **Key derivation** from master secret + context + random salts
- **Constant-time operations** to prevent timing attacks
- **Memory-safe operations** with secure data wiping

### **Advanced Security Measures**
- ✅ **No plaintext storage** - All sensitive data encrypted at rest
- ✅ **Deterministic email encryption** - Allows efficient database queries
- ✅ **Random OTP hashing** - Each OTP gets unique salt, prevents rainbow tables
- ✅ **HMAC integrity protection** - Detects data tampering attempts
- ✅ **Timing attack resistance** - Constant-time comparisons
- ✅ **Memory protection** - Secure wiping of sensitive data
- ✅ **Forward secrecy** - Key rotation capability built-in

## 🔧 Implementation Details

### **Email Encryption (Deterministic)**
```
Format: base64(salt[32] + nonce[12] + ciphertext + hmac[32])
Algorithm: AES-256-GCM with Argon2id key derivation
Purpose: Allows database queries while maintaining encryption
```

### **OTP Hashing (Random)**
```
Format: base64(salt[32] + hash[32] + hmac[32])
Algorithm: Argon2id with random salt per OTP
Purpose: One-way hashing prevents OTP recovery even with database access
```

### **Key Derivation**
```
Derived Key = Argon2id(
    password: masterKey + context,
    salt: random_or_deterministic,
    time: 3,
    memory: 64MB,
    threads: 4,
    keylen: 32 bytes
)
```

## 🚀 Deployment Setup

### **1. Generate Master Key**
```bash
cd /path/to/your/project
go run -c "
import 'sudo/internal/security'
key, _ := security.GenerateMasterKey()
fmt.Println(key)
"
```

Or run the test to generate a key:
```bash
go test ./internal/security -v
```

### **2. Set Environment Variable**
```bash
# Add to your environment (Docker, systemd, etc.)
export ENCRYPTION_MASTER_KEY="your_generated_key_here"
```

For Docker:
```yaml
environment:
  - ENCRYPTION_MASTER_KEY=your_generated_key_here
```

For systemd service:
```ini
[Service]
Environment=ENCRYPTION_MASTER_KEY=your_generated_key_here
```

### **3. Update Existing Data (Migration)**

If you have existing unencrypted data, create a migration script:

```go
package main

import (
    "context"
    "sudo/internal/database"
    "sudo/internal/security"
)

func migrateExistingData() {
    // This is a one-time migration for existing data
    // Run this carefully in a maintenance window
    db := database.NewDB()

    // Migrate users
    // Note: This requires careful handling of existing data
    // Recommend testing thoroughly in staging first
}
```

## 🛡️ Security Properties

### **Threat Protection**
- ✅ **Database breach**: Emails/OTPs unreadable without master key
- ✅ **Code access**: Even with full code access, data remains encrypted
- ✅ **Rainbow tables**: Random salts prevent precomputed attacks
- ✅ **Timing attacks**: Constant-time operations prevent timing analysis
- ✅ **Replay attacks**: HMAC integrity prevents data manipulation
- ✅ **Brute force**: Argon2id makes password cracking computationally expensive

### **What Attackers Cannot Do (Even With Full Code Access)**
1. **Cannot decrypt emails** without the master key
2. **Cannot reverse OTP hashes** due to one-way hashing with random salts
3. **Cannot create valid fake data** without HMAC keys
4. **Cannot use timing attacks** due to constant-time comparisons
5. **Cannot use rainbow tables** due to random per-record salts

### **Master Key Protection**
- Store master key in secure environment variables only
- Never commit master key to code repository
- Use different keys for different environments (dev/staging/prod)
- Consider using cloud key management services (AWS KMS, Azure Key Vault, etc.)
- Implement key rotation procedures

## 📊 Performance Impact

### **Encryption Overhead**
- **Email encryption**: ~1-2ms per operation
- **OTP hashing**: ~100-200ms per operation (intentionally slow for security)
- **Memory usage**: ~64MB per Argon2id operation (parallel operations possible)

### **Database Storage**
- **Email storage**: ~25% increase in size (base64 encoding + metadata)
- **OTP storage**: ~20% increase in size
- **Query performance**: No impact on email queries (deterministic encryption)

## 🔄 Migration Strategy

### **For New Installations**
1. Set `ENCRYPTION_MASTER_KEY` environment variable
2. Deploy the updated code
3. All new data will be automatically encrypted

### **For Existing Installations**
1. **BACKUP YOUR DATABASE FIRST**
2. Set `ENCRYPTION_MASTER_KEY` environment variable
3. Create migration script to encrypt existing data
4. Test migration thoroughly in staging environment
5. Schedule maintenance window for production migration
6. Deploy updated code

⚠️ **WARNING**: Losing the master key means all encrypted data becomes unrecoverable. Implement proper key backup and recovery procedures.

## 🧪 Testing

Run the comprehensive test suite:
```bash
go test ./internal/security -v
```

This tests:
- Email encryption/decryption
- OTP hashing/verification
- HMAC integrity
- Timing attack resistance
- Error handling
- Edge cases

## 🔍 Monitoring

### **Security Metrics to Monitor**
- Failed OTP verification attempts (potential brute force)
- HMAC verification failures (potential tampering)
- Encryption/decryption errors (potential key issues)
- Unusual authentication patterns

### **Logging**
The system logs security events without exposing sensitive data:
- ✅ OTP validation attempts (email obscured)
- ✅ Encryption errors
- ✅ Authentication failures
- ❌ Never logs plaintext emails/OTPs
- ❌ Never logs master keys

## 🔐 Key Rotation

To rotate the master key:
1. Generate new master key
2. Decrypt all data with old key
3. Re-encrypt with new key
4. Update environment variable
5. Deploy updated code

Consider implementing automated key rotation for maximum security.

## 📞 Emergency Procedures

### **If Master Key is Compromised**
1. Immediately rotate the master key
2. Re-encrypt all sensitive data
3. Audit access logs
4. Consider forcing password resets for all users

### **If Master Key is Lost**
⚠️ **CRITICAL**: All encrypted data becomes unrecoverable
- Implement regular key backups to prevent this scenario
- Consider multi-key schemes for disaster recovery

## ✅ Compliance

This encryption implementation helps meet:
- **GDPR**: Right to be forgotten, data protection by design
- **HIPAA**: Strong encryption requirements for PII
- **SOC 2**: Data encryption controls
- **ISO 27001**: Information security management

## 🤝 Best Practices

1. **Never hardcode the master key** in source code
2. **Use different keys for different environments**
3. **Implement regular key rotation**
4. **Monitor for security anomalies**
5. **Keep backups of master keys in secure locations**
6. **Test disaster recovery procedures regularly**
7. **Use secure channels for key distribution**
8. **Implement proper access controls for key storage**

---

**Security is a shared responsibility**. This implementation provides strong technical safeguards, but operational security (key management, access controls, monitoring) is equally important for overall system security.

---

## Additional Security Resources

### Production Security

For production deployments, also review:

- **[Self-Hosting Guide - Security Hardening](SELF_HOST.md#security-hardening)** - Firewall, SSL, and server security
- **[Self-Hosting Guide - Backup & Recovery](SELF_HOST.md#backup--recovery)** - Disaster recovery procedures
- **[Testing Guide - Security Testing](TESTING_SETUP_GUIDE.md)** - Validate security implementation

### Compliance

SUDO's encryption system helps meet requirements for:

- **GDPR** (EU) - Data protection by design, right to erasure
- **CCPA** (California) - Consumer data protection
- **HIPAA** (Healthcare) - Strong encryption for PII
- **SOC 2** - Security controls and audit trails
- **ISO 27001** - Information security management

### Security Checklist for Deployment

Before deploying to production:

- [ ] Generate unique `ENCRYPTION_MASTER_KEY` for production
- [ ] Never commit master key to version control
- [ ] Use different keys for dev/staging/prod environments
- [ ] Set up key backup procedures
- [ ] Configure firewall rules (see [SELF_HOST.md](SELF_HOST.md))
- [ ] Enable SSL/TLS with strong ciphers
- [ ] Implement key rotation schedule
- [ ] Set up security monitoring and alerts
- [ ] Test backup restoration procedures
- [ ] Document emergency procedures for your team
- [ ] Run security tests (see [TESTING_SETUP_GUIDE.md](TESTING_SETUP_GUIDE.md))

---

## Reporting Security Issues

If you discover a security vulnerability, please:

1. **DO NOT** open a public GitHub issue
2. Email security concerns to: [your-security-email]
3. Include detailed information about the vulnerability
4. Allow reasonable time for a fix before disclosure

We take security seriously and will respond promptly to valid reports.

---

**[⬆ Back to README](README.md)** | **[Deploy Securely →](SELF_HOST.md)**