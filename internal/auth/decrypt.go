package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"
)

// traeMagic is the 6-byte magic header of an encrypted Trae CN auth blob
// ("tc\x05\x10\x00\x00"). The 32-byte salt follows, then the AES-128-CBC
// ciphertext.
var traeMagic = []byte{0x74, 0x63, 0x05, 0x10, 0x00, 0x00}

var traeKeyA = []byte{82, 9, 106, 213, 48, 54, 165, 56, 191, 64, 163, 158, 129, 243, 215, 251, 124, 227, 57, 130, 155, 47, 255, 135, 52, 142, 67, 68, 196, 222, 233, 203, 84, 123, 148, 50, 166, 194, 35, 61, 238, 76, 149, 11, 66, 250, 195, 78, 8, 46, 161, 102, 40, 217, 36, 178, 118, 91, 162, 73, 109, 139, 209, 37}
var traeKeyB = []byte{31, 221, 168, 51, 136, 7, 199, 49, 177, 18, 16, 89, 39, 128, 236, 95, 96, 81, 127, 169, 25, 181, 74, 13, 45, 229, 122, 159, 147, 201, 156, 239, 160, 224, 59, 77, 174, 42, 245, 176, 200, 235, 187, 60, 131, 83, 153, 97, 23, 43, 4, 126, 186, 119, 214, 38, 225, 105, 20, 99, 85, 33, 12, 125}

// deriveKey derives the AES-128 key and IV from the blob salt using the
// Trae CN double-SHA-512 KDF over the XORed static key pair.
func deriveKey(salt []byte) (key, iv []byte) {
	pw := make([]byte, 64)
	for i := 0; i < 64; i++ {
		pw[i] = traeKeyA[i] ^ traeKeyB[i]
	}
	hSalt := sha512.Sum512(salt)
	kdfInput := append(hSalt[:], pw...)
	kdf := sha512.Sum512(kdfInput)
	return kdf[0:16], kdf[16:32]
}

// pkcs7Unpad strips and validates PKCS#7 padding.
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > len(data) || padLen > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding size: %d", padLen)
	}
	for i := len(data) - padLen; i < len(data); i++ {
		if data[i] != byte(padLen) {
			return nil, fmt.Errorf("invalid padding byte")
		}
	}
	return data[:len(data)-padLen], nil
}

// DecryptTraeBlob decrypts a Trae CN encrypted auth blob (AES-128-CBC).
// Layout: base64( magic[6] | salt[32] | ciphertext ) — after decryption the
// plaintext carries a 64-byte prefix followed by the JSON auth payload.
func DecryptTraeBlob(enc string) ([]byte, error) {
	blob, err := base64.StdEncoding.DecodeString(strings.TrimSpace(enc))
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	if len(blob) < len(traeMagic)+32+aes.BlockSize {
		return nil, fmt.Errorf("blob too short: %d", len(blob))
	}

	// Magic header validation.
	if !bytes.Equal(blob[:len(traeMagic)], traeMagic) {
		return nil, fmt.Errorf("bad magic header: % x", blob[:len(traeMagic)])
	}

	salt := blob[len(traeMagic) : len(traeMagic)+32]
	ct := blob[len(traeMagic)+32:]
	if len(ct)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext length not multiple of block size")
	}

	key, iv := deriveKey(salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	pt := make([]byte, len(ct))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(pt, ct)

	unpadded, err := pkcs7Unpad(pt)
	if err != nil {
		return nil, fmt.Errorf("unpad failed: %w", err)
	}

	if len(unpadded) < 64 {
		return nil, fmt.Errorf("unpadded data too short: %d", len(unpadded))
	}

	return unpadded[64:], nil
}
