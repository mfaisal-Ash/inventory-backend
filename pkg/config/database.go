package config

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func NewDatabase(cfg *Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		log.Fatalf("gagal konek database: %v", err)
	}
	return db
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Role{},
		&model.Permission{},
		&model.RolePermission{},
		&model.User{},
		&model.RefreshToken{},

		&model.Kategori{},
		&model.Satuan{},
		&model.Gudang{},
		&model.Rak{},

		&model.Barang{},
		&model.Supplier{},
		&model.PurchaseOrder{},
		&model.PurchaseOrderItem{},
		&model.BarangMasuk{},
		&model.BarangMasukItem{},
		&model.BarangKeluar{},
		&model.BarangKeluarItem{},
		&model.StockOpname{},
		&model.StockOpnameItem{},
		&model.Pengiriman{},
		&model.PengirimanTrackingPoint{},
		&model.CodTransaction{},
		&model.Asset{},
		&model.AssetType{},
		&model.AssetPort{},
		&model.AssetHistory{},
		&model.BarangRusak{},
		&model.MaintenanceStatus{},
		&model.Notification{},
		&model.NotificationRead{},
		&model.NotificationDismissed{},
	)
}

func SeedDefaultAssetTypes(db *gorm.DB) error {
	defaults := []model.AssetType{
		{Kode: "tiang", Label: "Tiang", Color: "#78350f", Abbr: "TG", HasKoordinat: true, HasPort: false, IsSystem: true, Urutan: 1},
		{Kode: "odc", Label: "ODC", Color: "#b5451b", Abbr: "ODC", HasKoordinat: true, HasPort: true, IsSystem: true, Urutan: 2},
		{Kode: "ont", Label: "ONT", Color: "#2563eb", Abbr: "ONT", HasKoordinat: true, HasPort: false, IsSystem: true, Urutan: 3},
		{Kode: "odp", Label: "ODP", Color: "#059669", Abbr: "ODP", HasKoordinat: true, HasPort: true, IsSystem: true, Urutan: 4},
		{Kode: "olt", Label: "OLT", Color: "#7c3aed", Abbr: "OLT", HasKoordinat: true, HasPort: true, IsSystem: true, Urutan: 5},
		{Kode: "modem", Label: "Modem", Color: "#d97706", Abbr: "MDM", HasKoordinat: true, HasPort: false, IsSystem: true, Urutan: 6},
		{Kode: "transportasi", Label: "Transportasi", Color: "#6b7280", Abbr: "TR", HasKoordinat: false, HasPort: false, IsSystem: true, Urutan: 7},
	}

	for _, t := range defaults {
		var existing model.AssetType
		err := db.Where("kode = ?", t.Kode).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := db.Create(&t).Error; err != nil {
			return err
		}
	}
	return nil
}
func SeedDefaultRoles(db *gorm.DB) error {
	defaults := []model.Role{
		{Name: constant.RoleSuperAdmin, Description: "Akses penuh seluruh modul sistem", IsSystem: true},
		{Name: constant.RoleAdmin, Description: "Kelola operasional gudang & pengguna", IsSystem: true},
		{Name: constant.RoleKaryawan, Description: "Role default akun self-register", IsSystem: true},
	}

	for _, r := range defaults {
		var existing model.Role
		err := db.Where("name = ?", r.Name).First(&existing).Error
		if err == nil {
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		if err := db.Create(&r).Error; err != nil {
			return err
		}
	}
	return nil
}
