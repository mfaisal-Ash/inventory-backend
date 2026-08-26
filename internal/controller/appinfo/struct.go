package appinfo

import (
	maintenanceRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/maintenance"
	notificationRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/notification"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Controller struct {
	cfg             *config.Config
	jwtSvc          *utils.JWTService
	maintenanceRepo maintenanceRepo.Repository
	notifRepo       notificationRepo.Repository
}

func New(
	cfg *config.Config,
	jwtSvc *utils.JWTService,
	maintenanceRepo maintenanceRepo.Repository,
	notifRepo notificationRepo.Repository,
) *Controller {
	return &Controller{
		cfg:             cfg,
		jwtSvc:          jwtSvc,
		maintenanceRepo: maintenanceRepo,
		notifRepo:       notifRepo,
	}
}
