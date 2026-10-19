package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenTTL é a validade da sessão. Como não há refresh token, mantemos
// um prazo generoso para o uso do admin.
const TokenTTL = 24 * time.Hour

// IssueToken assina um JWT HS256 cujo "subject" é o id do usuário.
func IssueToken(secret string, userID int64, email string) (string, time.Time, error) {
	expiresAt := time.Now().Add(TokenTTL)

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    "forms",
		Audience:  jwt.ClaimStrings{email},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("assinando token: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseToken valida assinatura e expiração e devolve o id do usuário.
func ParseToken(secret, raw string) (int64, error) {
	var claims jwt.RegisteredClaims
	// jwt.WithValidMethods evita o ataque de trocar o algoritmo por "none".
	_, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return 0, fmt.Errorf("token inválido: %w", err)
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("subject inválido no token: %w", err)
	}
	return userID, nil
}
