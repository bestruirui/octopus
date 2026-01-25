package auth

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/bestruirui/octopus/internal/conf"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWTToken(expiresMin int) (string, string, error) {
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    conf.APP_NAME,
	}
	if expiresMin == 0 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(15) * time.Minute))
	} else if expiresMin > 0 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(expiresMin) * time.Minute))
	} else if expiresMin == -1 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(30) * 24 * time.Hour))
	}
	user := op.UserGet()
	secret := user.Username + user.Password
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	return token, claims.ExpiresAt.Format(time.RFC3339), nil
}

func VerifyJWTToken(token string) bool {
	jwtToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		user := op.UserGet()
		secret := user.Username + user.Password
		return []byte(secret), nil
	})
	if err != nil || !jwtToken.Valid {
		return false
	}
	return true
}

func GenerateAPIKey() string {
	const keyChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 48)
	maxI := big.NewInt(int64(len(keyChars)))
	for i := range b {
		n, err := rand.Int(rand.Reader, maxI)
		if err != nil {
			return ""
		}
		b[i] = keyChars[n.Int64()]
	}
	return "sk-wagix" + "-" + string(b)
}
