package routes

import (
	"log"
	"time"

	"gorm.io/gorm"

	appinfoController "github.com/mfaisal-Ash/inventory-backend/internal/controller/appinfo"
	assetController "github.com/mfaisal-Ash/inventory-backend/internal/controller/asset_gudang"
	authController "github.com/mfaisal-Ash/inventory-backend/internal/controller/auth"
	barangController "github.com/mfaisal-Ash/inventory-backend/internal/controller/barang"
	barangKeluarController "github.com/mfaisal-Ash/inventory-backend/internal/controller/barang_keluar"
	barangMasukController "github.com/mfaisal-Ash/inventory-backend/internal/controller/barang_masuk"
	barangRusakController "github.com/mfaisal-Ash/inventory-backend/internal/controller/barang_rusak"
	barangSerialController "github.com/mfaisal-Ash/inventory-backend/internal/controller/barang_serial"
	captchaController "github.com/mfaisal-Ash/inventory-backend/internal/controller/captcha"
	dashboardController "github.com/mfaisal-Ash/inventory-backend/internal/controller/dashboard"
	gudangController "github.com/mfaisal-Ash/inventory-backend/internal/controller/gudang"
	humanCheckController "github.com/mfaisal-Ash/inventory-backend/internal/controller/humancheck"
	laporanController "github.com/mfaisal-Ash/inventory-backend/internal/controller/laporan"
	maintenanceController "github.com/mfaisal-Ash/inventory-backend/internal/controller/maintenance"
	notificationController "github.com/mfaisal-Ash/inventory-backend/internal/controller/notifikasi"
	roleController "github.com/mfaisal-Ash/inventory-backend/internal/controller/role"
	securityController "github.com/mfaisal-Ash/inventory-backend/internal/controller/security"
	stockOpnameController "github.com/mfaisal-Ash/inventory-backend/internal/controller/stockOpname"
	taskController "github.com/mfaisal-Ash/inventory-backend/internal/controller/task"
	trashController "github.com/mfaisal-Ash/inventory-backend/internal/controller/trash"
	usersController "github.com/mfaisal-Ash/inventory-backend/internal/controller/users"
	"github.com/mfaisal-Ash/inventory-backend/internal/health"
	assetRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset"
	assetHistoryRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset_history"
	assetPortRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset_port"
	authRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/auth"
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	barangKeluarRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_keluar"
	barangMasukRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_masuk"
	barangRusakRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_rusak"
	barangSerialRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	gudangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/gudang"
	maintenanceRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/maintenance"
	notificationRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/notifikasi"
	roleRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	stockOpnameRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/stockOpname"
	taskRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/task"
	usersRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/users"
	"github.com/mfaisal-Ash/inventory-backend/pkg/botcheck"
	"github.com/mfaisal-Ash/inventory-backend/pkg/captcha"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geoip"
	"github.com/mfaisal-Ash/inventory-backend/pkg/humancheck"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Dependencies struct {
	Cfg        *config.Config
	JWTSvc     *utils.JWTService
	RoleRepo   roleRepo.Repository
	GudangRepo gudangRepo.Repository

	MaintenanceRepo maintenanceRepo.Repository

	AuthController *authController.Controller
	UserController *usersController.Controller
	RoleController *roleController.Controller

	GudangController *gudangController.Controller
	BarangController *barangController.Controller

	BarangMasukController  *barangMasukController.Controller
	BarangKeluarController *barangKeluarController.Controller
	BarangSerialController *barangSerialController.Controller
	StockOpnameController  *stockOpnameController.Controller
	AssetController        *assetController.Controller
	BarangRusakController  *barangRusakController.Controller
	TaskController         *taskController.Controller
	AppInfoController      *appinfoController.Controller
	TrashController        *trashController.Controller
	NotificationController *notificationController.Controller
	NotificationRepo       notificationRepo.Repository

	LaporanController   *laporanController.Controller
	DashboardController *dashboardController.Controller

	CaptchaController     *captchaController.Controller
	HumanCheckController  *humanCheckController.Controller
	SecurityController    *securityController.Controller
	MaintenanceController *maintenanceController.Controller

	BotCheckSvc *botcheck.Service

	HealthController *health.Controller
}

func New(db *gorm.DB, cfg *config.Config) *Dependencies {
	jwtSvc := utils.NewJWTService(&cfg.JWT)

	rRole := roleRepo.New(db)
	rUsers := usersRepo.New(db)
	rNotification := notificationRepo.New(db)
	rAuth := authRepo.New(db)
	jwtSvc.SetSessionChecker(rAuth.CheckSession)
	rGudang := gudangRepo.New(db)
	rBarang := barangRepo.New(db)
	rBarangMasuk := barangMasukRepo.New(db)
	rBarangKeluar := barangKeluarRepo.New(db)
	rBarangSerial := barangSerialRepo.New(db)
	rStockOpname := stockOpnameRepo.New(db)
	rAsset := assetRepo.New(db)
	rAssetPort := assetPortRepo.New(db)
	rAssetHistory := assetHistoryRepo.New(db)
	rBarangRusak := barangRusakRepo.New(db)
	rTask := taskRepo.New(db)
	rMaintenance := maintenanceRepo.New(db)

	captchaSvc := captcha.NewService(cfg.Captcha.Secret, time.Duration(cfg.Captcha.TTLMinutes)*time.Minute)
	humanCheckSvc := humancheck.NewService(
		cfg.HumanCheck.Secret,
		time.Duration(cfg.HumanCheck.TTLMinutes)*time.Minute,
		time.Duration(cfg.HumanCheck.MinDelaySeconds)*time.Second,
	)
	botCheckSvc := botcheck.NewService(cfg.BotCheck.Secret, time.Duration(cfg.BotCheck.WindowMinutes)*time.Minute)
	geoipSvc := newGeoIPResolver(cfg)

	cAuth := authController.New(authController.Params{
		AuthRepo:      rAuth,
		UserRepo:      rUsers,
		RoleRepo:      rRole,
		JWTSvc:        jwtSvc,
		CaptchaSvc:    captchaSvc,
		HumanCheckSvc: humanCheckSvc,
		Cfg:           cfg,
		GeoipSvc:      geoipSvc,
	})
	cUsers := usersController.New(usersController.Params{
		UserRepo:      rUsers,
		RoleRepo:      rRole,
		AuthRepo:      rAuth,
		JWTSvc:        jwtSvc,
		HumanCheckSvc: humanCheckSvc,
		StoragePath:   cfg.Storage.Path,
	})
	cRole := roleController.New(rRole, jwtSvc)
	cGudang := gudangController.New(rGudang, rRole, jwtSvc)
	cBarang := barangController.New(rBarang, rGudang, rRole, rUsers, jwtSvc, db, rNotification)
	cBarangMasuk := barangMasukController.New(rBarangMasuk, rBarang, rGudang, rRole, jwtSvc, rNotification)
	cBarangKeluar := barangKeluarController.New(rBarangKeluar, rBarang, rGudang, rRole, jwtSvc, rNotification)
	cBarangSerial := barangSerialController.New(rBarangSerial, rBarang, rRole, jwtSvc)
	cStockOpname := stockOpnameController.New(rStockOpname, rBarang, rGudang, rRole, jwtSvc, rNotification)
	cAsset := assetController.New(rAsset, rGudang, rAssetPort, rAssetHistory, rUsers, rRole, rBarang, jwtSvc, rNotification)
	cBarangRusak := barangRusakController.New(rBarangRusak, rBarang, rRole, jwtSvc, rNotification)
	cTask := taskController.New(rTask, rRole, jwtSvc)
	cLaporan := laporanController.New(rBarang, rBarangMasuk, rBarangKeluar, rStockOpname, rBarangRusak, rBarangSerial, rUsers, rRole, jwtSvc)
	cDashboard := dashboardController.New(rBarang, rGudang, rBarangMasuk, rBarangKeluar, rStockOpname, rRole, jwtSvc, db)
	cCaptcha := captchaController.New(captchaSvc)
	cHumanCheck := humanCheckController.New(humanCheckSvc)
	cSecurity := securityController.New(botCheckSvc, captchaSvc)
	cMaintenance := maintenanceController.New(rMaintenance, jwtSvc, rNotification)
	cHealth := health.NewController(health.NewChecker(db, cfg.Storage.Path))

	return &Dependencies{
		Cfg:                    cfg,
		JWTSvc:                 jwtSvc,
		RoleRepo:               rRole,
		GudangRepo:             rGudang,
		MaintenanceRepo:        rMaintenance,
		AuthController:         cAuth,
		UserController:         cUsers,
		RoleController:         cRole,
		GudangController:       cGudang,
		BarangController:       cBarang,
		BarangMasukController:  cBarangMasuk,
		BarangKeluarController: cBarangKeluar,
		BarangSerialController: cBarangSerial,
		StockOpnameController:  cStockOpname,
		AssetController:        cAsset,
		BarangRusakController:  cBarangRusak,
		TaskController:         cTask,
		AppInfoController:      appinfoController.New(cfg, jwtSvc, rMaintenance, rNotification),
		TrashController:        trashController.New(db, jwtSvc),
		NotificationController: notificationController.New(rNotification, jwtSvc),
		NotificationRepo:       rNotification,
		LaporanController:      cLaporan,
		DashboardController:    cDashboard,
		CaptchaController:      cCaptcha,
		HumanCheckController:   cHumanCheck,
		SecurityController:     cSecurity,
		MaintenanceController:  cMaintenance,
		BotCheckSvc:            botCheckSvc,
		HealthController:       cHealth,
	}
}

func newGeoIPResolver(cfg *config.Config) geoip.Resolver {
	if !cfg.GeoIP.Enabled {
		return geoip.NoopResolver{}
	}

	resolver, err := geoip.NewHTTPResolver(cfg.GeoIP.BaseURL)
	if err != nil {
		log.Printf("geoip: konfigurasi GEOIP_BASE_URL tidak valid, fallback ke NoopResolver: %v", err)
		return geoip.NoopResolver{}
	}
	return resolver
}
