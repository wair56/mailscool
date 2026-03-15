package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// deriveKey 从 JWT_SECRET 派生 AES-256 密钥（SHA-256 哈希）
func deriveKey() []byte {
	h := sha256.Sum256([]byte(C.JWTSecret))
	return h[:]
}

// Encrypt 使用 AES-GCM 加密明文，返回 hex 编码的密文
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(deriveKey())
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt 解密 hex 编码的 AES-GCM 密文，返回明文
func Decrypt(ciphertextHex string) (string, error) {
	if ciphertextHex == "" {
		return "", nil
	}

	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		// 可能是未加密的旧数据，原样返回
		return ciphertextHex, nil
	}

	block, err := aes.NewCipher(deriveKey())
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		// 数据太短，可能是未加密的旧数据
		return ciphertextHex, nil
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		// 解密失败，可能是未加密的旧数据，原样返回
		return ciphertextHex, nil
	}

	return string(plaintext), nil
}
