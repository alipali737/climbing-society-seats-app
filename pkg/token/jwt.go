package token

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func NewJWT(username string, cryptographicKey []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iss":      "uow-climbing-seats",
			"username": username,
			"exp":      time.Now().Add(time.Hour * 6).Unix(), // Token will expire after 6 hours
		})
	tokenString, err := token.SignedString(cryptographicKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateJWT(tokenString string, cryptographicKey []byte) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return cryptographicKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	return token, nil
}
