package maintenance

import (
	"time"

	maintenanceRepo "github.com/inventory-backend/internal/repositories/maintenance"
	notifikasiRepo "github.com/inventory-backend/internal/repositories/notifikasi"
	"github.com/inventory-backend/pkg/utils"
)

type Controller struct {
	repo      maintenanceRepo.Repository
	jwtSvc    *utils.JWTService
	notifRepo notifikasiRepo.Repository
}

func New(repo maintenanceRepo.Repository, jwtSvc *utils.JWTService, notifRepo notifikasiRepo.Repository) *Controller {
	return &Controller{repo: repo, jwtSvc: jwtSvc, notifRepo: notifRepo}
}

type SetRequest struct {
	IsActive       bool       `json:"is_active"`
	Message        string     `json:"message" validate:"max=500"`
	EstimatedUntil *time.Time `json:"estimated_until"`
}

type StatusResponse struct {
	IsActive         bool       `json:"is_active"`
	Message          string     `json:"message,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	EstimatedUntil   *time.Time `json:"estimated_until,omitempty"`
	RemainingSeconds int64      `json:"remaining_seconds,omitempty"`
}
