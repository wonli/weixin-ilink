package ilinkapi

import (
	"bytes"
	"crypto/aes"
	"errors"
)

func aesEcbPaddedSize(plaintextSize int) int {
	return ((plaintextSize / aes.BlockSize) + 1) * aes.BlockSize
}

func pkcs7Pad(buf []byte, blockSize int) []byte {
	padding := blockSize - (len(buf) % blockSize)
	if padding == 0 {
		padding = blockSize
	}
	return append(buf, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func encryptAESECB(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	padded := pkcs7Pad(plaintext, block.BlockSize())
	out := make([]byte, len(padded))
	for start := 0; start < len(padded); start += block.BlockSize() {
		block.Encrypt(out[start:start+block.BlockSize()], padded[start:start+block.BlockSize()])
	}
	return out, nil
}

func decryptAESECB(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}
	out := make([]byte, len(ciphertext))
	for start := 0; start < len(ciphertext); start += block.BlockSize() {
		block.Decrypt(out[start:start+block.BlockSize()], ciphertext[start:start+block.BlockSize()])
	}
	return pkcs7Unpad(out, block.BlockSize())
}

func pkcs7Unpad(buf []byte, blockSize int) ([]byte, error) {
	if len(buf) == 0 || len(buf)%blockSize != 0 {
		return nil, errors.New("invalid padded buffer")
	}
	padding := int(buf[len(buf)-1])
	if padding == 0 || padding > blockSize || padding > len(buf) {
		return nil, errors.New("invalid padding size")
	}
	for i := len(buf) - padding; i < len(buf); i++ {
		if int(buf[i]) != padding {
			return nil, errors.New("invalid padding")
		}
	}
	return buf[:len(buf)-padding], nil
}
