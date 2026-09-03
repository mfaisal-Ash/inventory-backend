package routes

import (
	"context"
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
	geocodeController "github.com/mfaisal-Ash/inventory-backend/internal/controller/geocode"
	gudangController "github.com/mfaisal-Ash/inventory-backend/internal/controller/gudang"
	humanCheckController "github.com/mfaisal-Ash/inventory-backend/internal/controller/humancheck"
	laporanController "github.com/mfaisal-Ash/inventory-backend/internal/controller/laporan"
	maintenanceController "github.com/mfaisal-Ash/inventory-backend/internal/controller/maintenance"
	notificationController "github.com/mfaisal-Ash/inventory-backend/internal/controller/notifikasi"
	pengajuanBarangController "github.com/mfaisal-Ash/inventory-backend/internal/controller/pengajuan_barang"
	pengajuanTemplateController "github.com/mfaisal-Ash/inventory-backend/internal/controller/pengajuan_template"
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
	pengajuanBarangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_barang"
	pengajuanTemplateRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_template"
	roleRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	stockOpnameRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/stockOpname"
	taskRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/task"
	usersRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/users"
	"github.com/mfaisal-Ash/inventory-backend/pkg/botcheck"
	"github.com/mfaisal-Ash/inventory-backend/pkg/captcha"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geocoding"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geoip"
	"github.com/mfaisal-Ash/inventory-backend/pkg/humancheck"
	"github.com/mfaisal-Ash/inventory-backend/pkg/passwordreset"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
	"github.com/mfaisal-Ash/inventory-backend/pkg/wa"
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

	LaporanController           *laporanController.Controller
	DashboardController         *dashboardController.Controller
	PengajuanBarangController   *pengajuanBarangController.Controller
	PengajuanTemplateController *pengajuanTemplateController.Controller

	CaptchaController     *captchaController.Controller
	HumanCheckController  *humanCheckController.Controller
	SecurityController    *securityController.Controller
	MaintenanceController *maintenanceController.Controller
	GeocodeController     *geocodeController.Controller

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
	rPengajuanBarang := pengajuanBarangRepo.New(db)
	rPengajuanTemplate := pengajuanTemplateRepo.New(db)

	captchaSvc := captcha.NewService(cfg.Captcha.Secret, time.Duration(cfg.Captcha.TTLMinutes)*time.Minute)
	humanCheckSvc := humancheck.NewService(
		cfg.HumanCheck.Secret,
		time.Duration(cfg.HumanCheck.TTLMinutes)*time.Minute,
		time.Duration(cfg.HumanCheck.MinDelaySeconds)*time.Second,
	)
	botCheckSvc := botcheck.NewService(cfg.BotCheck.Secret, time.Duration(cfg.BotCheck.WindowMinutes)*time.Minute)
	geoipSvc := newGeoIPResolver(cfg)

	// waSender dipakai untuk mengirim kode OTP reset password via WhatsApp
	// (lihat pkg/passwordreset). Kalau drivernya belum dikonfigurasi atau
	// gagal diinisialisasi, jatuh ke errSender supaya alur reset password
	// gagal DENGAN JELAS ("whatsapp belum siap: ...") ketimbang diam-diam
	// membiarkan reset password tanpa OTP sama sekali (fail closed, bukan
	// fail open — celah account-takeover yang lama justru dari perilaku
	// fail-open semacam itu).
	var waSender wa.Sender
	switch cfg.WhatsApp.Driver {
	case "whatsmeow":
		sender, err := wa.NewWhatsmeowSender(context.Background(), cfg.WhatsApp.SessionPath)
		if err != nil {
			log.Printf("wa: gagal inisialisasi whatsmeow, reset password via OTP akan gagal sampai ini diperbaiki: %v", err)
			waSender = errSender{reason: err.Error()}
		} else {
			waSender = sender
		}
	default:
		if cfg.WhatsApp.APIURL == "" {
			waSender = errSender{reason: "WHATSAPP_API_URL belum diset"}
		} else {
			waSender = wa.NewClient(cfg.WhatsApp.APIURL, cfg.WhatsApp.APIKey, cfg.WhatsApp.Sender)
		}
	}
	passwordResetSvc := passwordreset.NewService(cfg.PasswordReset.Secret, cfg.PasswordReset.TTLMinutes)

	cAuth := authController.New(authController.Params{
		AuthRepo:         rAuth,
		UserRepo:         rUsers,
		RoleRepo:         rRole,
		JWTSvc:           jwtSvc,
		CaptchaSvc:       captchaSvc,
		HumanCheckSvc:    humanCheckSvc,
		Cfg:              cfg,
		GeoipSvc:         geoipSvc,
		WASender:         waSender,
		PasswordResetSvc: passwordResetSvc,
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
	cBarangRusak := barangRusakController.New(rBarangRusak, rBarang, rGudang, rRole, jwtSvc, rNotification)
	cTask := taskController.New(rTask, rRole, jwtSvc)
	cLaporan := laporanController.New(rBarang, rBarangMasuk, rBarangKeluar, rStockOpname, rBarangRusak, rBarangSerial, rPengajuanBarang, rAsset, rAssetHistory, rUsers, rRole, jwtSvc)
	cDashboard := dashboardController.New(rBarang, rGudang, rBarangMasuk, rBarangKeluar, rStockOpname, rRole, jwtSvc, db)
	cPengajuanBarang := pengajuanBarangController.New(rPengajuanBarang, rBarang, rGudang, rUsers, rRole, jwtSvc, rNotification, rPengajuanTemplate)
	cPengajuanTemplate := pengajuanTemplateController.New(rPengajuanTemplate, jwtSvc)
	cCaptcha := captchaController.New(captchaSvc)
	cHumanCheck := humanCheckController.New(humanCheckSvc)
	cSecurity := securityController.New(botCheckSvc, captchaSvc)
	cMaintenance := maintenanceController.New(rMaintenance, jwtSvc, rNotification)
	cHealth := health.NewController(health.NewChecker(db, cfg.Storage.Path))
	geocodingSvc := geocoding.NewService(cfg.App.Name)
	cGeocode := geocodeController.New(geocodingSvc, jwtSvc)

	return &Dependencies{
		Cfg:                         cfg,
		JWTSvc:                      jwtSvc,
		RoleRepo:                    rRole,
		GudangRepo:                  rGudang,
		MaintenanceRepo:             rMaintenance,
		AuthController:              cAuth,
		UserController:              cUsers,
		RoleController:              cRole,
		GudangController:            cGudang,
		BarangController:            cBarang,
		BarangMasukController:       cBarangMasuk,
		BarangKeluarController:      cBarangKeluar,
		BarangSerialController:      cBarangSerial,
		StockOpnameController:       cStockOpname,
		AssetController:             cAsset,
		BarangRusakController:       cBarangRusak,
		TaskController:              cTask,
		AppInfoController:           appinfoController.New(cfg, jwtSvc, rMaintenance, rNotification),
		TrashController:             trashController.New(db, jwtSvc),
		NotificationController:      notificationController.New(rNotification, jwtSvc),
		NotificationRepo:            rNotification,
		LaporanController:           cLaporan,
		DashboardController:         cDashboard,
		PengajuanBarangController:   cPengajuanBarang,
		PengajuanTemplateController: cPengajuanTemplate,
		CaptchaController:           cCaptcha,
		HumanCheckController:        cHumanCheck,
		SecurityController:          cSecurity,
		MaintenanceController:       cMaintenance,
		GeocodeController:           cGeocode,
		BotCheckSvc:                 botCheckSvc,
		HealthController:            cHealth,
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
