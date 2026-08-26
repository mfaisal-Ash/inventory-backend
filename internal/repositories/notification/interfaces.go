package notification

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Row struct {
	model.Notification
	IsRead bool `json:"is_read"`
}

type Repository interface {
	Create(n *model.Notification) error

	List(userID uint, userRole string, p utils.PaginationParams) ([]Row, int64, error)

	UnreadCount(userID uint, userRole string) (int64, error)

	MarkRead(notificationID, userID uint) error

	MarkAllRead(userID uint, userRole string) error

	Dismiss(notificationID, userID uint) error
}
