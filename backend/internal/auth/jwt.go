package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 24 * time.Hour
	Issuer          = "grokforge"
)

// TokenPair is returned on login/refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Claims are embedded in access and refresh JWTs.
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	Kind     string `json:"kind"` // "access" | "refresh"
	jwt.RegisteredClaims
}

// TokenService signs and verifies JWTs.
type TokenService struct {
	secret []byte
	now    func() time.Time
}

func NewTokenService(secret string) (*TokenService, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("jwt secret must be at least 16 characters")
	}
	return &TokenService{
		secret: []byte(secret),
		now:    time.Now,
	}, nil
}

func (s *TokenService) Issue(userID int64, username string) (*TokenPair, error) {
	now := s.now()
	access, err := s.sign(userID, username, "access", now, AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.sign(userID, username, "refresh", now, RefreshTokenTTL)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(AccessTokenTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *TokenService) sign(userID int64, username, kind string, now time.Time, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Kind:     kind,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.secret)
}

// ParseAccess validates an access token.
func (s *TokenService) ParseAccess(token string) (*Claims, error) {
	return s.parse(token, "access")
}

// ParseRefresh validates a refresh token.
func (s *TokenService) ParseRefresh(token string) (*Claims, error) {
	return s.parse(token, "refresh")
}

func (s *TokenService) parse(token, wantKind string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	if claims.Kind != wantKind {
		return nil, fmt.Errorf("token kind mismatch")
	}
	return claims, nil
}
