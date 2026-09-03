package auth

import (
	"time"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"gorm.io/gorm"
)

func (r *repository) SaveRefreshToken(t *model.RefreshToken) error {
	return r.db.Create(t).Error
}

func (r *repository) FindActiveRefreshToken(userID uint, tokenHash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	err := r.db.
		Where("user_id = ? AND token_hash = ? AND revoked = false", userID, tokenHash).
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repository) RevokeAllUserTokens(userID uint) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked", true).Error
}

// ListActiveSessions: dulu cuma filter revoked = false, jadi sesi yang
// sudah KEDALUWARSA (refresh token lewat expires_at, misal karena device
// lama tidak dipakai) tetap muncul di daftar "sedang login" — padahal
// perangkat itu sebenarnya sudah otomatis ter-logout begitu access token +
// refresh token-nya sama-sama habis. Sekarang ikut filter expires_at,
// konsisten dengan OnlineUserIDs di bawah yang sudah benar dari awal.
func (r *repository) ListActiveSessions(userID uint) ([]model.RefreshToken, error) {
	var sessions []model.RefreshToken
	err := r.db.
		Where("user_id = ? AND revoked = false AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *repository) OnlineUserIDs(userIDs []uint) (map[uint]bool, error) {
	result := make(map[uint]bool, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	var onlineIDs []uint
	err := r.db.Model(&model.RefreshToken{}).
		Distinct("user_id").
		Where("user_id IN ? AND revoked = false AND expires_at > ?", userIDs, time.Now()).
		Pluck("user_id", &onlineIDs).Error
	if err != nil {
		return nil, err
	}
	for _, id := range onlineIDs {
		result[id] = true
	}
	return result, nil
}

func (r *repository) RevokeSession(userID, sessionID, revokedByUserID uint) error {
	result := r.db.Model(&model.RefreshToken{}).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Updates(map[string]interface{}{"revoked": true, "revoked_by_user_id": revokedByUserID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *repository) CheckSession(sessionID uint) (bool, string, error) {
	var t model.RefreshToken
	err := r.db.Select("id", "user_id", "revoked", "revoked_by_user_id", "expires_at").
		First(&t, sessionID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, "", nil
		}
		return false, "", err
	}
	revoked := t.Revoked || (!t.ExpiresAt.IsZero() && time.Now().After(t.ExpiresAt))
	if !revoked {
		return false, "", nil
	}
	if t.RevokedByUserID == nil || *t.RevokedByUserID == t.UserID {
		return true, "", nil
	}
	var username string
	if err := r.db.Model(&model.User{}).
		Where("id = ?", *t.RevokedByUserID).
		Pluck("username", &username).Error; err != nil {
		return true, "", nil
	}
	return true, username, nil
}
