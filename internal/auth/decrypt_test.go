package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"testing"
)

// encryptTraeBlob is the test-side inverse of DecryptTraeBlob, using the same
// KDF to produce a valid encrypted blob.
func encryptTraeBlob(t *testing.T, payload []byte) string {
	t.Helper()

	plain := append(make([]byte, 64), payload...)
	// PKCS#7 pad
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	for i := 0; i < pad; i++ {
		plain = append(plain, byte(pad))
	}

	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	key, iv := deriveKey(salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ct := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, plain)

	blob := append(append(append([]byte{}, traeMagic...), salt...), ct...)
	return base64.StdEncoding.EncodeToString(blob)
}

func TestDecryptTraeBlob_RoundTrip(t *testing.T) {
	payload := []byte(`{"token":"secret_jwt","userId":"u-1"}`)
	enc := encryptTraeBlob(t, payload)

	got, err := DecryptTraeBlob(enc)
	if err != nil {
		t.Fatalf("DecryptTraeBlob() error = %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("DecryptTraeBlob() = %q, want %q", got, payload)
	}
}

func TestDecryptTraeBlob_BadMagic(t *testing.T) {
	enc := encryptTraeBlob(t, []byte(`{"token":"x"}`))
	raw, _ := base64.StdEncoding.DecodeString(enc)
	raw[0] ^= 0xff // corrupt magic
	if _, err := DecryptTraeBlob(base64.StdEncoding.EncodeToString(raw)); err == nil {
		t.Error("DecryptTraeBlob() error = nil, want bad magic error")
	}
}

func TestDecryptTraeBlob_BadBase64(t *testing.T) {
	if _, err := DecryptTraeBlob("!!!not-base64!!!"); err == nil {
		t.Error("DecryptTraeBlob() error = nil, want base64 error")
	}
}

func TestDecryptTraeBlob_TooShort(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
	if _, err := DecryptTraeBlob(short); err == nil {
		t.Error("DecryptTraeBlob() error = nil, want short blob error")
	}
}
