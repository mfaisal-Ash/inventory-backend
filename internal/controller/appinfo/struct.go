package appinfo

import (
	maintenanceRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/maintenance"
	notificationRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/notifikasi"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type ControllerApp struct {
	cfg             *config.Config
	jwtSvc          *utils.JWTService
	maintenanceRepo maintenanceRepo.Repository
	notifRepo       notificationRepo.Repository
}

func NewControllerApp(
	cfg *config.Config,
	jwtSvc *utils.JWTService,
	maintenanceRepo maintenanceRepo.Repository,
	notifRepo notificationRepo.Repository,
) *ControllerApp {
	return &ControllerApp{
		cfg:             cfg,
		jwtSvc:          jwtSvc,
		maintenanceRepo: maintenanceRepo,
		notifRepo:       notifRepo,
	}
}
