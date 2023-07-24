package token

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

func GenerateNewEncryptedToken(passphrase string, tokenLength int) error {
	secretToken, err := generateRandomToken(tokenLength)
	if err != nil {
		return fmt.Errorf("failed to generate a new random token: %v", err)
	}

	encryptedToken, err := encryptToken(secretToken, passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret token: %v", err)
	}

	err = storeToken(encryptedToken)
	if err != nil {
		return fmt.Errorf("failed to write token to file: %v", err)
	}
	return nil
}

func GetCryptographicKey(passphrase string) ([]byte, error) {
	encryptedToken, err := readToken()
	if err != nil {
		return nil, fmt.Errorf("failed to read token from file: %v", err)
	}

	decryptToken, err := decryptToken(encryptedToken, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %v", err)
	}

	return decryptToken, nil
}

func GenerateRandomPassphrase(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("invalid passphrase length")
	}

	buffer := make([]byte, length)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buffer)[:length], nil
}

func generateRandomToken(length int) ([]byte, error) {
	token := make([]byte, length)
	_, err := rand.Read(token)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func encryptToken(token []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, token, nil)
	return ciphertext, nil
}

func decryptToken(ciphertext []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(passphrase))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func storeToken(encryptedToken []byte) error {
	err := os.WriteFile("token.dat", encryptedToken, 0600)
	if err != nil {
		return err
	}
	return nil
}

func readToken() ([]byte, error) {
	encryptedToken, err := os.ReadFile("token.dat")
	if err != nil {
		return nil, err
	}
	return encryptedToken, nil
}
