package passwordreset

import "github.com/mfaisal-Ash/inventory-backend/internal/model"

type Repository interface {
	Create(*model.PasswordResetRequest) error
	FindPendingByUser(uint) (*model.PasswordResetRequest, error)
	ListPending() ([]model.PasswordResetRequest, error)
	FindByID(uint) (*model.PasswordResetRequest, error)
	Update(*model.PasswordResetRequest) error
}
