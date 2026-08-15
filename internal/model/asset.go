package model

import (
	"time"

	"gorm.io/gorm"

	"github.com/projsonal/gowms/pkg/constant"
)

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

	// JumlahPort menyimpan berapa banyak slot port yang dimiliki aset ini
	// (misal OLT/ODC/ODP). 0 berarti aset ini tidak punya port sama sekali.
	JumlahPort int `json:"jumlah_port" gorm:"not null;default:0"`

	// ParentAssetID mengarah ke aset "induk" tempat aset ini tersambung
	// lewat salah satu port induknya (lihat AssetPort.ChildAssetID).
	ParentAssetID *uint  `json:"parent_asset_id" gorm:"index"`
	ParentAsset   *Asset `json:"parent_asset,omitempty" gorm:"foreignKey:ParentAssetID"`

	IPAddress  string     `json:"ip_address" gorm:"size:45"`
	PingStatus string     `json:"ping_status" gorm:"size:20;not null;default:'unknown'"`
	LastPingAt *time.Time `json:"last_ping_at"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Asset) TableName() string { return "assets" }

func JenisAsetPunyaKoordinat(jenisAset string) bool {
	return jenisAset != "transportasi"
}

// jenisIndukMap mendefinisikan urutan hierarki jaringan fiber yang valid:
// key = jenis aset anak, value = jenis aset induk yang boleh jadi tempatnya nyambung.
// Urutan: OLT -> ODC -> ODP -> ONT.
var jenisIndukMap = map[string]string{
	constant.JenisAsetODC: constant.JenisAsetOLT,
	constant.JenisAsetODP: constant.JenisAsetODC,
	constant.JenisAsetONT: constant.JenisAsetODP,
}

// JenisIndukValid mengecek apakah aset berjenis childJenis boleh dipasang
// sebagai anak (tersambung lewat port) dari aset berjenis indukJenis,
// mengikuti hierarki jaringan fiber: OLT -> ODC -> ODP -> ONT.
func JenisIndukValid(childJenis, indukJenis string) bool {
	induk, ok := jenisIndukMap[childJenis]
	if !ok {
		return false
	}
	return induk == indukJenis
}
