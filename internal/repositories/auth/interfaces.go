package auth

import "github.com/mfaisal-Ash/inventory-backend/internal/model"

type Repository interface {
	SaveRefreshToken(t *model.RefreshToken) error
	FindActiveRefreshToken(userID uint, tokenHash string) (*model.RefreshToken, error)
	RevokeAllUserTokens(userID uint) error

	ListActiveSessions(userID uint) ([]model.RefreshToken, error)
	RevokeSession(userID, sessionID, revokedByUserID uint) error
	CheckSession(sessionID uint) (revoked bool, revokedByUsername string, err error)

	OnlineUserIDs(userIDs []uint) (map[uint]bool, error)
}
