// Package auth выдаёт и проверяет JWT пользователей provider_routers.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims — полезные данные токена: кто вошёл.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Manager подписывает токены секретом HS256 и проверяет их.
type Manager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl, issuer: "provider_routers"}
}

// TTL — время жизни токена (столько же он хранится в blacklist после выхода).
func (m *Manager) TTL() time.Duration { return m.ttl }

// Issue выдаёт токен пользователю после успешной проверки пароля.
func (m *Manager) Issue(userID uint, email string) (string, time.Time, error) {
	now := time.Now()
	expires := now.Add(m.ttl)
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   fmt.Sprint(userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, expires, err
}

// Parse проверяет подпись, алгоритм и срок действия токена.
func (m *Manager) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(m.issuer))
	if err != nil {
		return nil, err
	}
	if !parsed.Valid || claims.UserID == 0 {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
