package utils

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
)

const jwtIssuer = "stockrsd"

 type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	RoleID   uint   `json:"role_id"`
	RoleName string `json:"role_name"`
	TokenType string `json:"token_type"`
	SessionID uint `json:"session_id"`
	jwt.RegisteredClaims
}

type JWTService struct {
	cfg *config.JWTConfig

	sessionChecker func(sessionID uint) (revoked bool, revokedByUsername string, err error)
}

func NewJWTService(cfg *config.JWTConfig) *JWTService {
	return &JWTService{cfg: cfg}
}

func (s *JWTService) SetSessionChecker(checker func(sessionID uint) (revoked bool, revokedByUsername string, err error)) {
	s.sessionChecker = checker
}

func (s *JWTService) CheckSession(sessionID uint) (bool, string, error) {
	if s.sessionChecker == nil || sessionID == 0 {
		return false, "", errors.New("session checker belum dikonfigurasi")
	}
	return s.sessionChecker(sessionID)
}

func (s *JWTService) GenerateAccessToken(userID, roleID, sessionID uint, roleName string) (string, error) {
	claims := JWTClaims{
		UserID:    userID,
		RoleID:    roleID,
		RoleName:  roleName,
		TokenType: "access",
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtIssuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			Audience:  jwt.ClaimStrings{"stockrsd-api"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(s.cfg.AccessExpiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.AccessSecret))
}

func (s *JWTService) GenerateRefreshToken(userID uint) (string, time.Time, error) {
	expiry := time.Now().AddDate(0, 0, s.cfg.RefreshExpiryDays)
	claims := jwt.RegisteredClaims{
		Issuer:    jwtIssuer,
		Subject:   strconv.FormatUint(uint64(userID), 10),
		Audience:  jwt.ClaimStrings{"stockrsd-refresh"},
		ExpiresAt: jwt.NewNumericDate(expiry),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.RefreshSecret))
	return signed, expiry, err
}

func (s *JWTService) ParseAccessToken(tokenStr string) (*JWTClaims, error) {
	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.AccessSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(jwtIssuer), jwt.WithAudience("stockrsd-api"))

	if err != nil || !token.Valid || claims.TokenType != "access" || claims.UserID == 0 || claims.SessionID == 0 {
		return nil, errors.New("access token tidak valid")
	}
	return claims, nil
}

func (s *JWTService) ParseRefreshToken(tokenStr string) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.RefreshSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(jwtIssuer), jwt.WithAudience("stockrsd-refresh"))
	if err != nil || !token.Valid || claims.Subject == "" {
		return nil, errors.New("refresh token tidak valid")
	}
	return claims, nil
}

func ParseUintSubject(subject string) uint {
	id, err := strconv.ParseUint(subject, 10, 64)
	if err != nil || id == 0 || uint64(uint(id)) != id {
		return 0
	}
	return uint(id)
}
