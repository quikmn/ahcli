// FILE: client/crypto.go
package main

import (
	"ahcli/common/logger"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

type ClientCryptoManager struct {
	privateKey      [32]byte
	publicKey       [32]byte
	serverPublicKey [32]byte
	sharedSecret    [32]byte
	cipher          cipher.AEAD
	ready           bool
}

var clientCrypto *ClientCryptoManager

// InitClientCrypto initializes the client crypto manager
func InitClientCrypto() error {
	logger.Info("Initializing client crypto manager...")

	clientCrypto = &ClientCryptoManager{}

	// Generate client key pair
	var err error
	clientCrypto.privateKey, err = generatePrivateKey()
	if err != nil {
		logger.Error("Failed to generate client private key: %v", err)
		return fmt.Errorf("failed to generate client private key: %v", err)
	}

	// Derive public key
	curve25519.ScalarBaseMult(&clientCrypto.publicKey, &clientCrypto.privateKey)

	logger.Info("Client crypto manager initialized with key pair")
	logger.Debug("Client public key: %s", base64.StdEncoding.EncodeToString(clientCrypto.publicKey[:]))

	return nil
}

// GetPublicKey returns the client's public key for handshake
func (ccm *ClientCryptoManager) GetPublicKey() [32]byte {
	logger.Debug("Providing client public key for handshake")
	return ccm.publicKey
}

// CompleteHandshake completes the key exchange with server public key
func (ccm *ClientCryptoManager) CompleteHandshake(serverPublicKey [32]byte) error {
	logger.Debug("Completing handshake with server public key: %s",
		base64.StdEncoding.EncodeToString(serverPublicKey[:]))

	ccm.serverPublicKey = serverPublicKey

	// Compute shared secret using ECDH
	curve25519.ScalarMult(&ccm.sharedSecret, &ccm.privateKey, &ccm.serverPublicKey)
	logger.Debug("Computed ECDH shared secret")

	// Derive encryption key using BLAKE2b (same as server)
	encryptionKey, err := blake2b.New256(nil)
	if err != nil {
		logger.Error("Failed to create BLAKE2b hasher: %v", err)
		return fmt.Errorf("failed to create BLAKE2b hasher: %v", err)
	}
	encryptionKey.Write(ccm.sharedSecret[:])
	encryptionKey.Write([]byte("ahcli-chat-encryption"))

	var derivedKey [32]byte
	copy(derivedKey[:], encryptionKey.Sum(nil))
	logger.Debug("Derived encryption key using BLAKE2b")

	// Create ChaCha20-Poly1305 cipher
	ccm.cipher, err = chacha20poly1305.NewX(derivedKey[:])
	if err != nil {
		logger.Error("Failed to create ChaCha20-Poly1305 cipher: %v", err)
		return fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %v", err)
	}

	ccm.ready = true
	logger.Info("Crypto handshake completed successfully - E2E encryption ready")

	return nil
}

// EncryptMessage encrypts a message for transmission to server
func (ccm *ClientCryptoManager) EncryptMessage(message string) ([]byte, error) {
	if !ccm.ready {
		logger.Error("Attempted to encrypt message but crypto not ready")
		return nil, fmt.Errorf("crypto not ready - handshake not completed")
	}

	// Generate random nonce
	nonce := make([]byte, ccm.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		logger.Error("Failed to generate nonce for encryption: %v", err)
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt message
	plaintext := []byte(message)
	ciphertext := ccm.cipher.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	encrypted := make([]byte, len(nonce)+len(ciphertext))
	copy(encrypted[:len(nonce)], nonce)
	copy(encrypted[len(nonce):], ciphertext)

	logger.Debug("Encrypted message: %d bytes plaintext -> %d bytes ciphertext", len(message), len(encrypted))
	return encrypted, nil
}

// DecryptMessage decrypts a message received from server
func (ccm *ClientCryptoManager) DecryptMessage(data []byte) (string, error) {
	if !ccm.ready {
		logger.Error("Attempted to decrypt message but crypto not ready")
		return "", fmt.Errorf("crypto not ready - handshake not completed")
	}

	nonceSize := ccm.cipher.NonceSize()
	if len(data) < nonceSize {
		logger.Error("Encrypted data too short: %d bytes (need at least %d)", len(data), nonceSize)
		return "", fmt.Errorf("encrypted data too short")
	}

	// Extract nonce and ciphertext
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	// Decrypt message
	plaintext, err := ccm.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		logger.Error("Decryption failed: %v", err)
		return "", fmt.Errorf("decryption failed: %v", err)
	}

	logger.Debug("Decrypted message: %d bytes ciphertext -> %d bytes plaintext", len(data), len(plaintext))
	return string(plaintext), nil
}

// IsReady returns whether crypto is ready for use
func (ccm *ClientCryptoManager) IsReady() bool {
	ready := ccm != nil && ccm.ready
	logger.Debug("Crypto ready status: %t", ready)
	return ready
}

// generatePrivateKey generates a random X25519 private key
func generatePrivateKey() ([32]byte, error) {
	logger.Debug("Generating new X25519 private key")

	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		logger.Error("Failed to read random bytes for private key: %v", err)
		return [32]byte{}, err
	}

	// Clamp private key for X25519
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	logger.Debug("X25519 private key generated and clamped successfully")
	return privateKey, nil
}
