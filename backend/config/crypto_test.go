package config

import (
	"testing"
)

func init() {
	// 初始化配置，测试用
	C = Config{
		JWTSecret: "test-secret-key-for-unit-tests-only",
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	plaintext := "sk_abcdef1234567890abcdef12345678"
	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if encrypted == "" {
		t.Fatal("Encrypt returned empty string")
	}
	if encrypted == plaintext {
		t.Fatal("Encrypt returned plaintext unchanged")
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("Decrypt mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	encrypted, err := Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt empty failed: %v", err)
	}
	if encrypted != "" {
		t.Fatalf("Encrypt empty should return empty, got %q", encrypted)
	}

	decrypted, err := Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt empty failed: %v", err)
	}
	if decrypted != "" {
		t.Fatalf("Decrypt empty should return empty, got %q", decrypted)
	}
}

func TestDecrypt_UnencryptedFallback(t *testing.T) {
	// 未加密的旧数据应原样返回（优雅降级）
	oldPlaintext := "my-old-password-123"
	result, err := Decrypt(oldPlaintext)
	if err != nil {
		t.Fatalf("Decrypt old data failed: %v", err)
	}
	if result != oldPlaintext {
		t.Fatalf("Decrypt fallback mismatch: got %q, want %q", result, oldPlaintext)
	}
}

func TestEncrypt_DifferentCiphertexts(t *testing.T) {
	// 同一明文每次加密应产生不同密文（因为随机 nonce）
	plaintext := "test-password"
	enc1, _ := Encrypt(plaintext)
	enc2, _ := Encrypt(plaintext)
	if enc1 == enc2 {
		t.Fatal("Two encryptions of same plaintext should produce different ciphertexts")
	}

	// 但解密后应相同
	dec1, _ := Decrypt(enc1)
	dec2, _ := Decrypt(enc2)
	if dec1 != dec2 {
		t.Fatalf("Decrypted values should match: %q vs %q", dec1, dec2)
	}
}

func TestEncryptDecrypt_SpecialChars(t *testing.T) {
	cases := []string{
		"p@ssw0rd!#$%^&*()",
		"密码测试123",
		"🔑🔐",
		"a\"b'c\\d",
		string([]byte{0, 1, 2, 255}),
	}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			enc, err := Encrypt(tc)
			if err != nil {
				t.Fatalf("Encrypt failed for %q: %v", tc, err)
			}
			dec, err := Decrypt(enc)
			if err != nil {
				t.Fatalf("Decrypt failed for %q: %v", tc, err)
			}
			if dec != tc {
				t.Fatalf("Round-trip failed: got %q, want %q", dec, tc)
			}
		})
	}
}
