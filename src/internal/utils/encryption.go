package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

func GetEncryptionKey() ([]byte, error) {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return nil, errors.New("titkosítási kulcs nem található a környezeti változókban")
	}

	keyBytes := []byte(key)
	if len(keyBytes) != 16 && len(keyBytes) != 24 && len(keyBytes) != 32 {
		return nil, errors.New("a titkosítási kulcsnak 16, 24 vagy 32 bájt hosszúnak kell lennie")
	}

	return keyBytes, nil
}

func EncryptData(data string) (string, error) {
	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := []byte(data)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func DecryptData(encryptedData string) (string, error) {
	key, err := GetEncryptionKey()
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("a titkosított szöveg túl rövid")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

func EncryptCardID(cardID string) (string, error) {
	return EncryptData(cardID)
}

func DecryptCardID(encryptedCardID string) (string, error) {
	return DecryptData(encryptedCardID)
}

func HashCardID(cardID string) (string, error) {
	return EncryptCardID(cardID)
}

func CompareCardID(plainCardID, encryptedCardID string) (bool, error) {
	decrypted, err := DecryptCardID(encryptedCardID)
	if err != nil {
		return false, err
	}

	return decrypted == plainCardID, nil
}
