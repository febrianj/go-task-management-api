package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID uuid.UUID
	TeamID uuid.UUID
}

type Issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) *Issuer {
	return &Issuer{secret: []byte(secret), ttl: ttl}
}

func (i *Issuer) Issue(userID, teamID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(i.ttl)

	claims := jwt.MapClaims{
		"sub":     userID.String(),
		"team_id": teamID.String(),
		"iat":     now.Unix(),
		"exp":     expiresAt.Unix(),
		"jti":     uuid.NewString(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (i *Issuer) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return i.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	mc, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(stringClaim(mc, "sub"))
	if !ok {
		return nil, ErrInvalidToken
	}

	teamID, err := uuid.Parse(stringClaim(mc, "team_id"))
	if !ok {
		return nil, ErrInvalidToken
	}

	return &Claims{UserID: userID, TeamID: teamID}, nil
}

func stringClaim(m jwt.MapClaims, key string) string {
	s, _ := m[key].(string)
	return s
}
