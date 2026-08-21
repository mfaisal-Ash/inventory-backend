package maintenance

import (
	"time"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
)

type Repository interface {
	Get() (*model.MaintenanceStatus, error)
	Set(active bool, message string, estimatedUntil *time.Time, updatedBy uint) (*model.MaintenanceStatus, error)
}
