package passwordreset

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"gorm.io/gorm"
)

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) Repository                                 { return &repository{db: db} }
func (r *repository) Create(v *model.PasswordResetRequest) error { return r.db.Create(v).Error }
func (r *repository) FindPendingByUser(id uint) (*model.PasswordResetRequest, error) {
	var v model.PasswordResetRequest
	err := r.db.Preload("User").Where("user_id = ? AND status = ?", id, model.PasswordResetPending).First(&v).Error
	return &v, err
}
func (r *repository) ListPending() ([]model.PasswordResetRequest, error) {
	var v []model.PasswordResetRequest
	err := r.db.Preload("User").Where("status = ?", model.PasswordResetPending).Order("created_at ASC").Find(&v).Error
	return v, err
}
func (r *repository) FindByID(id uint) (*model.PasswordResetRequest, error) {
	var v model.PasswordResetRequest
	err := r.db.Preload("User").First(&v, id).Error
	return &v, err
}
func (r *repository) Update(v *model.PasswordResetRequest) error { return r.db.Save(v).Error }
