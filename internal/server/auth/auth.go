package auth

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/lingyuins/octopus/internal/conf"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/golang-jwt/jwt/v5"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	UserID uint   `json:"user_id,omitempty"`
	Role string `json:"role,omitempty"`
}

func GenerateJWTToken(expiresMin int, userID uint, role string) (string, string, error) {
	now := time.Now()
	claims := &jwtClaims{
		UserID: userID,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    conf.APP_NAME,
		},
	}
	if expiresMin == 0 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(15) * time.Minute))
	} else if expiresMin > 0 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(expiresMin) * time.Minute))
	} else if expiresMin == -1 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(30) * 24 * time.Hour))
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(conf.AppConfig.Auth.JWTSecret))
	if err != nil {
		return "", "", err
	}
	return token, claims.ExpiresAt.Format(time.RFC3339), nil
}

// VerifyJWTToken validates the JWT and returns the user identity in claims.
func VerifyJWTToken(token string) (bool, uint, string) {
	claims := &jwtClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.AppConfig.Auth.JWTSecret), nil
	})
	if err != nil || !jwtToken.Valid {
		return false, 0, ""
	}
	if claims.Role == "" {
		claims.Role = model.UserRoleAdmin
	}
	return true, claims.UserID, claims.Role
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
	return "sk-" + conf.APP_NAME + "-" + string(b)
}
