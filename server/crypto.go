// FILE: server/crypto.go

package main

import (
	"ahcli/common/logger"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"sync"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// Per-client crypto context
type ClientCrypto struct {
	ClientPublicKey [32]byte
	SharedSecret    [32]byte
	Cipher          cipher.AEAD
	Ready           bool
}

// Server manages crypto for all clients
type ServerCryptoManager struct {
	privateKey [32]byte
	publicKey  [32]byte
	clients    map[string]*ClientCrypto // addr.String() -> crypto
	mutex      sync.RWMutex
}

var serverCrypto *ServerCryptoManager

// InitServerCrypto initializes the global server crypto manager
func InitServerCrypto() error {
	logger.Info("Initializing server crypto manager...")

	serverCrypto = &ServerCryptoManager{
		clients: make(map[string]*ClientCrypto),
	}

	// Generate server key pair
	var err error
	serverCrypto.privateKey, err = generatePrivateKey()
	if err != nil {
		return fmt.Errorf("failed to generate server private key: %v", err)
	}

	// Derive public key
	curve25519.ScalarBaseMult(&serverCrypto.publicKey, &serverCrypto.privateKey)

	logger.Info("Server crypto manager initialized with key pair")
	logger.Debug("Server public key: %s", base64.StdEncoding.EncodeToString(serverCrypto.publicKey[:]))

	return nil
}

// HandleHandshake processes client handshake and establishes shared secret
func (scm *ServerCryptoManager) HandleHandshake(addr *net.UDPAddr, clientPublicKey [32]byte) ([32]byte, error) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	addrStr := addr.String()
	logger.Debug("Processing crypto handshake from %s", addrStr)

	// Compute shared secret using ECDH
	var sharedSecret [32]byte
	curve25519.ScalarMult(&sharedSecret, &scm.privateKey, &clientPublicKey)

	// Derive encryption key using BLAKE2b
	encryptionKey, err := blake2b.New256(nil)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create BLAKE2b hasher: %v", err)
	}
	encryptionKey.Write(sharedSecret[:])
	encryptionKey.Write([]byte("ahcli-chat-encryption"))

	var derivedKey [32]byte
	copy(derivedKey[:], encryptionKey.Sum(nil))

	// Create ChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.NewX(derivedKey[:])
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create ChaCha20-Poly1305 cipher: %v", err)
	}

	// Store client crypto context
	scm.clients[addrStr] = &ClientCrypto{
		ClientPublicKey: clientPublicKey,
		SharedSecret:    sharedSecret,
		Cipher:          aead,
		Ready:           true,
	}

	logger.Info("Established crypto context for client %s", addrStr)
	return scm.publicKey, nil
}

// EncryptForClient encrypts a message for a specific client
func (scm *ServerCryptoManager) EncryptForClient(addr *net.UDPAddr, message string) ([]byte, error) {
	scm.mutex.RLock()
	clientCrypto, exists := scm.clients[addr.String()]
	scm.mutex.RUnlock()

	if !exists || !clientCrypto.Ready {
		return nil, fmt.Errorf("no crypto context for client %s", addr.String())
	}

	// Generate random nonce
	nonce := make([]byte, clientCrypto.Cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %v", err)
	}

	// Encrypt message
	plaintext := []byte(message)
	ciphertext := clientCrypto.Cipher.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	encrypted := make([]byte, len(nonce)+len(ciphertext))
	copy(encrypted[:len(nonce)], nonce)
	copy(encrypted[len(nonce):], ciphertext)

	logger.Debug("Encrypted %d bytes for client %s", len(message), addr.String())
	return encrypted, nil
}

// DecryptFromClient decrypts a message from a specific client
func (scm *ServerCryptoManager) DecryptFromClient(addr *net.UDPAddr, data []byte) (string, error) {
	scm.mutex.RLock()
	clientCrypto, exists := scm.clients[addr.String()]
	scm.mutex.RUnlock()

	if !exists || !clientCrypto.Ready {
		return "", fmt.Errorf("no crypto context for client %s", addr.String())
	}

	nonceSize := clientCrypto.Cipher.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("encrypted data too short")
	}

	// Extract nonce and ciphertext
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	// Decrypt message
	plaintext, err := clientCrypto.Cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %v", err)
	}

	logger.Debug("Decrypted %d bytes from client %s", len(plaintext), addr.String())
	return string(plaintext), nil
}

// GetServerPublicKey returns the server's public key
func (scm *ServerCryptoManager) GetServerPublicKey() [32]byte {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	return scm.publicKey
}

// HasClientCrypto checks if a client has established crypto
func (scm *ServerCryptoManager) HasClientCrypto(addr *net.UDPAddr) bool {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	clientCrypto, exists := scm.clients[addr.String()]
	return exists && clientCrypto.Ready
}

// RemoveClient removes crypto context for a disconnected client
func (scm *ServerCryptoManager) RemoveClient(addr *net.UDPAddr) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	addrStr := addr.String()
	if _, exists := scm.clients[addrStr]; exists {
		delete(scm.clients, addrStr)
		logger.Debug("Removed crypto context for client %s", addrStr)
	}
}

// generatePrivateKey generates a random X25519 private key
func generatePrivateKey() ([32]byte, error) {
	var privateKey [32]byte
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return [32]byte{}, err
	}

	// Clamp private key for X25519
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	return privateKey, nil
}
