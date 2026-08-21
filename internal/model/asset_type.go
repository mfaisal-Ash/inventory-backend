package model

import "time"

// AssetType adalah "jenis aset" yang bisa dikelola (tambah/ubah/hapus) lewat
// menu Manajemen Aset Gudang. 7 jenis bawaan (tiang, odc, ont, odp, olt,
// modem, transportasi) di-seed otomatis saat aplikasi pertama kali jalan
// (lihat pkg/config/seed_asset_types.go) dengan IsSystem=true supaya tidak
// bisa dihapus. Jenis baru yang ditambahkan user otomatis muncul sebagai
// pilihan di form Tambah Aset & sebagai layer baru di peta Tracking Aset,
// karena frontend mengambil daftar ini dari API alih-alih hardcode.
type AssetType struct {
	ID    uint   `json:"id" gorm:"primaryKey"`
	Kode  string `json:"kode" gorm:"size:30;uniqueIndex;not null"`
	Label string `json:"label" gorm:"size:60;not null"`

	// Warna & singkatan dipakai untuk render marker di peta Tracking Aset.
	Color string `json:"color" gorm:"size:10;not null;default:'#6b7280'"`
	Abbr  string `json:"abbr" gorm:"size:6;not null"`

	// HasKoordinat: jenis aset ini punya titik lokasi tetap (dapat label
	// RSD) atau tidak (dapat kode BA, mis. transportasi/aset bergerak).
	HasKoordinat bool `json:"has_koordinat" gorm:"not null;default:true"`
	// HasPort: jenis aset ini punya slot port fisik (splitter/switch).
	HasPort bool `json:"has_port" gorm:"not null;default:false"`

	// IsSystem: jenis bawaan sistem, tidak bisa dihapus (tapi label/warna
	// tetap bisa diubah).
	IsSystem bool `json:"is_system" gorm:"not null;default:false"`
	Urutan   int  `json:"urutan" gorm:"not null;default:0"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AssetType) TableName() string { return "asset_types" }
