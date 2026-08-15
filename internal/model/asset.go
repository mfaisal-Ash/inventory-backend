package model

import (
	"time"

	"gorm.io/gorm"
)

// Asset merepresentasikan aset gudang (tiang, ODC, OLT, ONT, ODP, modem,
// transportasi, dst — lihat pkg/constant JenisAset*). Aset dengan koordinat
// (lihat JenisAsetPunyaKoordinat) mendapat LabelRSD; aset tanpa koordinat
// (mis. transportasi) mendapat KodeBA sebagai gantinya — keduanya saling
// eksklusif, lihat internal/controller/asset Create().
type Asset struct {
	ID         uint     `json:"id" gorm:"primaryKey"`
	Nama       string   `json:"nama" gorm:"size:150;not null;index"`
	JenisAset  string   `json:"jenis_aset" gorm:"size:20;not null;index"`
	GudangID   uint     `json:"gudang_id" gorm:"not null;index"`
	Gudang     *Gudang  `json:"gudang,omitempty" gorm:"foreignKey:GudangID"`
	LabelRSD   string   `json:"label_rsd" gorm:"size:60;index"`
	KodeBA     string   `json:"kode_ba" gorm:"size:30;index"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Status     string   `json:"status" gorm:"size:20;not null;default:'aktif';index"`
	Keterangan string   `json:"keterangan" gorm:"size:500"`

	// --- Ping monitoring (lihat internal/controller/asset ping_controller.go) ---
	IPAddress  string     `json:"ip_address" gorm:"size:45"`
	PingStatus string     `json:"ping_status" gorm:"size:20;not null;default:'unknown'"`
	LastPingAt *time.Time `json:"last_ping_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Asset) TableName() string { return "assets" }

// JenisAsetPunyaKoordinat menentukan apakah suatu jenis aset dipasang
// permanen (butuh latitude/longitude & mendapat LabelRSD) atau bergerak
// (mis. "transportasi", cukup KodeBA tanpa koordinat tetap).
func JenisAsetPunyaKoordinat(jenisAset string) bool {
	return jenisAset != "transportasi"
}
