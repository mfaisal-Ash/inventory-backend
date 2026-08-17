package maintenance

import (
	"time"

	"github.com/inventory-backend/internal/model"
)

type Repository interface {
	Get() (*model.MaintenanceStatus, error)
	Set(active bool, message string, estimatedUntil *time.Time, updatedBy uint) (*model.MaintenanceStatus, error)
}
