package model

import (
	"time"

	"gorm.io/gorm"
)

// ReturPenggantian mencatat penggantian alat (mis. modem) yang rusak di klien.
// Alur: dilaporkan -> dicek teknisi (perlu_ganti / tidak_perlu_ganti) ->
// selesai (unit baru Normal keluar ke klien, unit lama ditarik ke gudang
// sebagai barang rusak). Lihat repositories/retur.Ganti.
type ReturPenggantian struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	NomorRetur string `json:"nomor_retur" gorm:"size:30;uniqueIndex;not null"`

	BarangID uint    `json:"barang_id" gorm:"not null;index"`
	Barang   *Barang `json:"barang,omitempty" gorm:"foreignKey:BarangID"`
	GudangID uint    `json:"gudang_id" gorm:"not null;index"`
	Gudang   *Gudang `json:"gudang,omitempty" gorm:"foreignKey:GudangID"`

	Klien       string `json:"klien" gorm:"size:150;not null;index"`
	AlamatKlien string `json:"alamat_klien" gorm:"size:255"`

	SerialLama string `json:"serial_lama" gorm:"size:100;not null;index"`
	SerialBaru string `json:"serial_baru" gorm:"size:100;index"`
	Keluhan    string `json:"keluhan" gorm:"size:500;not null"`

	Status          string `json:"status" gorm:"size:20;not null;default:'dilaporkan';index"`
	HasilPengecekan string `json:"hasil_pengecekan" gorm:"size:20"`
	CatatanTeknisi  string `json:"catatan_teknisi" gorm:"size:500"`

	DilaporkanOleh uint       `json:"dilaporkan_oleh" gorm:"not null"`
	Pelapor        *User      `json:"pelapor,omitempty" gorm:"foreignKey:DilaporkanOleh"`
	DicekOleh      *uint      `json:"dicek_oleh"`
	Pemeriksa      *User      `json:"pemeriksa,omitempty" gorm:"foreignKey:DicekOleh"`
	DicekPada      *time.Time `json:"dicek_pada"`
	DigantiOleh    *uint      `json:"diganti_oleh"`
	DigantiPada    *time.Time `json:"diganti_pada"`

	// BarangRusakID: laporan Barang Rusak yang dibuat otomatis untuk unit lama
	// yang ditarik ke gudang.
	BarangRusakID *uint `json:"barang_rusak_id" gorm:"index"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ReturPenggantian) TableName() string { return "retur_penggantian" }
