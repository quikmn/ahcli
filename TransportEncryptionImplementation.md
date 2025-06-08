# AHCLI Transport Encryption Implementation Guide
**Chat Messages Only - Senior Engineering Approach**

## 🚨 CRITICAL WARNING FOR FUTURE DEVELOPERS

**READ THIS BEFORE TOUCHING ANYTHING:**

This is **TRANSPORT ENCRYPTION** - NOT end-to-end encryption. We encrypt data between client and server to secure the network connection. The server can read all messages in plaintext (which is correct for our use case).

**DO NOT:**
- Add "fancy" crypto libraries or complex key exchange protocols
- Touch the voice/audio pipeline (it works perfectly, leave it alone)
- Add E2E encryption (not needed for self-hosted)
- Optimize anything that isn't broken
- Change working chat/UI code (only add encryption layer)

**CORE PHILOSOPHY:**
- **Bulletproof fundamentals** - voice quality is sacred
- **One logical change** - only add chat encryption
- **No band-aids** - if it goes in, it goes in right
- **Self-hosted focus** - we control both ends

---

## 🎯 What We're Actually Building

**Simple Transport Security:**
Client Chat Message → Encrypt → Network → Server Decrypt → Store Plaintext → Re-encrypt → Broadcast

**NOT Building:**
- Complex key management
- Perfect forward secrecy
- Message authentication beyond basic AEAD
- Voice encryption (separate project)
- Multi-key scenarios

---

## 🏗️ Implementation Architecture

### Core Components

1. **Client Crypto Manager** - Handles encryption to server
2. **Server Crypto Manager** - Handles per-client encryption contexts
3. **Simple Handshake** - X25519 ECDH key exchange
4. **Transparent Integration** - Existing chat flow unchanged

### Crypto Stack
- **Key Exchange:** X25519 ECDH
- **Encryption:** ChaCha20-Poly1305 AEAD
- **Key Derivation:** BLAKE2b

**Why these choices:**
- Fast, modern, audited algorithms
- Built into Go's crypto libraries
- Simple to implement correctly

---

## 📁 File Structure
client/
├── crypto.go          # NEW - Client-side crypto
├── net.go             # MODIFY - Add encrypted chat handlers
└── ...existing files...
server/
├── crypto.go          # NEW - Server-side crypto
├── net.go             # MODIFY - Add encrypted chat handlers
└── ...existing files...

**DO NOT MODIFY:**
- Any audio-related files
- State management
- WebSocket/UI code
- Existing chat display logic

---

## 🔧 Implementation Steps

### Step 1: Server Crypto Foundation

Create `server/crypto.go`:

```go
package main

import (
    "crypto/cipher"
    "crypto/rand"
    "net"
    "sync"
    "golang.org/x/crypto/chacha20poly1305"
    "golang.org/x/crypto/curve25519"
    "golang.org/x/crypto/blake2b"
)

// Per-client crypto context
type ClientCrypto struct {
    PublicKey    [32]byte
    SharedSecret [32]byte
    Cipher       cipher.AEAD
    Ready        bool
}

// Server manages crypto for all clients
type ServerCryptoManager struct {
    privateKey [32]byte
    publicKey  [32]byte
    clients    map[string]*ClientCrypto // addr.String() -> crypto
    mutex      sync.RWMutex
}

var serverCrypto *ServerCryptoManager

func InitServerCrypto() error {
    // Generate server key pair
    // Initialize manager
    // Return any errors
}

func (scm *ServerCryptoManager) HandleHandshake(addr *net.UDPAddr, clientPublicKey [32]byte) error {
    // Compute shared secret
    // Create cipher
    // Store client crypto context
}

func (scm *ServerCryptoManager) EncryptForClient(addr *net.UDPAddr, message string) ([]byte, error) {
    // Get client crypto context
    // Encrypt message
    // Return encrypted bytes
}

func (scm *ServerCryptoManager) DecryptFromClient(addr *net.UDPAddr, data []byte) (string, error) {
    // Get client crypto context  
    // Decrypt message
    // Return plaintext
}
Step 2: Client Crypto Foundation
Create client/crypto.go:
gopackage main

import (
    "crypto/cipher"
    "crypto/rand"
    "golang.org/x/crypto/chacha20poly1305"
    "golang.org/x/crypto/curve25519"
    "golang.org/x/crypto/blake2b"
)

type ClientCryptoManager struct {
    privateKey    [32]byte
    publicKey     [32]byte
    serverPublicKey [32]byte
    sharedSecret  [32]byte
    cipher        cipher.AEAD
    ready         bool
}

var clientCrypto *ClientCryptoManager

func InitClientCrypto() error {
    // Generate client key pair
    // Initialize manager
}

func (ccm *ClientCryptoManager) CompleteHandshake(serverPublicKey [32]byte) error {
    // Compute shared secret
    // Create cipher
    // Mark ready
}

func (ccm *ClientCryptoManager) EncryptMessage(message string) ([]byte, error) {
    // Encrypt for server
}

func (ccm *ClientCryptoManager) DecryptMessage(data []byte) (string, error) {
    // Decrypt from server
}
Step 3: Handshake Protocol
Client initiates after connection:
json{
    "type": "crypto_handshake",
    "public_key": [32 bytes as base64]
}
Server responds:
json{
    "type": "crypto_handshake_response", 
    "status": "success",
    "public_key": [32 bytes as base64]
}
Step 4: Modify Chat Flow
Client sends encrypted chat:
json{
    "type": "encrypted_chat",
    "channel": "General",
    "encrypted": true,
    "payload": [encrypted bytes]
}
Server broadcasts to channel:
json{
    "type": "encrypted_chat",
    "channel": "General", 
    "username": "sender",
    "encrypted": true,
    "payload": [re-encrypted for recipient],
    "timestamp": "15:04:05"
}

🧪 Testing Strategy
Phase 1: Crypto Foundation

Compile test - ensure crypto files build
Key generation test - verify key pair creation
Handshake test - test key exchange

Phase 2: Chat Integration

Encryption test - encrypt/decrypt roundtrip
Single user test - send encrypted chat to self
Multi-user test - encrypted chat between clients

Phase 3: Regression Testing

Voice quality test - ensure audio unaffected
UI functionality test - channel switching, etc.
Chat history test - ensure messages display correctly


⚠️ Critical Success Criteria
Must Work Perfectly:

✅ Voice quality unchanged
✅ Existing chat functionality
✅ Channel switching
✅ UI responsiveness

Must Be Secure:

✅ Chat messages encrypted in transit
✅ Unique keys per client session
✅ Graceful crypto failure handling

Must Be Maintainable:

✅ Clean error messages
✅ Minimal code complexity
✅ No crypto in audio pipeline
✅ Easy to disable/debug


🚫 ABSOLUTELY DO NOT

Touch audio files - voice quality is sacred
Add complex key management - simple is better
Optimize working code - don't fix what isn't broken
Add E2E encryption - transport encryption is the goal
Change UI/state management - only add crypto layer
Add dependencies - use Go standard library + golang.org/x/crypto


🎯 Success Definition
When done correctly:

User sends chat message
Message encrypted transparently
Other users receive decrypted message
Zero impact on voice quality
Clean error handling for crypto failures

If you find yourself:

Touching audio code → STOP
Adding complex protocols → STOP
Breaking existing functionality → STOP
Adding external dependencies → STOP


📝 Implementation Notes

Store crypto state separately from AppState
Use existing error handling patterns
Keep crypto functions pure (no side effects)
Test each step before proceeding
Fallback gracefully on crypto failures

Remember: We're building a bulletproof VOIP tool, not a crypto research project. The encryption should be invisible to users and transparent to existing functionality.