package proxy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"time"
)

// masticateKeyHex is the hardcoded AES-256 key the Trae CN client uses for
// llm_raw_chat message encryption (extracted from the bundled server.js,
// module exported as "masticateVegetablesToPulp").
const masticateKeyHex = "6195f24ca4d430f8a4833de7db8dac37d148a084e7464a351ffa68585c16b955"

// masticate encrypts a message payload for the upstream llm_raw_chat API:
// the first 8 key bytes are XORed with a random pin, AES-256-GCM runs with
// a random 12-byte IV and the seconds-level requestAt string as AAD, and the
// wire message is base64(iv || ciphertext || tag). The pin and requestAt must
// be echoed in the X-Request-Pin / X-Requested-At headers.
func masticate(plaintext []byte) (message, pin string, requestAt int64, err error) {
	key, err := hex.DecodeString(masticateKeyHex)
	if err != nil {
		return "", "", 0, err
	}
	pinB := make([]byte, 8)
	if _, err = rand.Read(pinB); err != nil {
		return "", "", 0, err
	}
	for i := range pinB {
		key[i] ^= pinB[i]
	}
	iv := make([]byte, 12)
	if _, err = rand.Read(iv); err != nil {
		return "", "", 0, err
	}
	requestAt = time.Now().Unix()

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", 0, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", 0, err
	}
	ct := gcm.Seal(nil, iv, plaintext, []byte(strconv.FormatInt(requestAt, 10)))
	out := make([]byte, 0, len(iv)+len(ct))
	out = append(out, iv...)
	out = append(out, ct...)
	return base64.StdEncoding.EncodeToString(out), hex.EncodeToString(pinB), requestAt, nil
}

// Demasticate reverses masticate; it is exported so cross-package tests and
// diagnostics tooling can verify what the proxy actually sends upstream.
func Demasticate(message, pin string, requestAt int64) ([]byte, error) {
	key, err := hex.DecodeString(masticateKeyHex)
	if err != nil {
		return nil, err
	}
	pinB, err := hex.DecodeString(pin)
	if err != nil {
		return nil, err
	}
	if len(pinB) != 8 {
		return nil, errors.New("pin must be 8 bytes")
	}
	for i := range pinB {
		key[i] ^= pinB[i]
	}
	raw, err := base64.StdEncoding.DecodeString(message)
	if err != nil {
		return nil, err
	}
	if len(raw) < 12+16 {
		return nil, errors.New("message too short")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, raw[:12], raw[12:], []byte(strconv.FormatInt(requestAt, 10)))
}
