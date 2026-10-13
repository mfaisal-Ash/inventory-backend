package model

import (
	"time"

	"gorm.io/gorm"
)

// KabelGulungan = satu gulungan kabel fisik di satu gudang. Panjang dalam
// meter. Terpakai/Sisa/Status tidak disimpan (lihat KabelPemakaian) supaya
// tidak ada dua sumber kebenaran yang bisa tidak sinkron.
type KabelGulungan struct {
	ID           uint    `json:"id" gorm:"primaryKey"`
	KodeGulungan string  `json:"kode_gulungan" gorm:"size:30;uniqueIndex;not null"`
	BarangID     uint    `json:"barang_id" gorm:"not null;index"`
	Barang       *Barang `json:"barang,omitempty" gorm:"foreignKey:BarangID"`
	GudangID     uint    `json:"gudang_id" gorm:"not null;index"`
	Gudang       *Gudang `json:"gudang,omitempty" gorm:"foreignKey:GudangID"`
	PanjangAwal  int     `json:"panjang_awal" gorm:"not null"`
	Catatan      string  `json:"catatan" gorm:"size:255"`
	DibuatOleh   uint    `json:"dibuat_oleh" gorm:"not null"`
	Pembuat      *User   `json:"pembuat,omitempty" gorm:"foreignKey:DibuatOleh"`

	Pemakaian []KabelPemakaian `json:"pemakaian,omitempty" gorm:"foreignKey:GulunganID"`

	// Turunan, diisi repository.
	Terpakai int    `json:"terpakai" gorm:"-"`
	Sisa     int    `json:"sisa" gorm:"-"`
	Status   string `json:"status" gorm:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (KabelGulungan) TableName() string { return "kabel_gulungan" }

// KabelPemakaian = satu kali pemakaian (meter) dari sebuah gulungan.
// Dibatalkan = true mengembalikan meter ke gulungan dan stok gudang; baris
// tetap disimpan sebagai jejak audit.
type KabelPemakaian struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	GulunganID  uint      `json:"gulungan_id" gorm:"not null;index"`
	Jumlah      int       `json:"jumlah" gorm:"not null"`
	Tanggal     time.Time `json:"tanggal" gorm:"not null;index"`
	Klien       string    `json:"klien" gorm:"size:150"`
	Keperluan   string    `json:"keperluan" gorm:"size:255"`
	DicatatOleh uint      `json:"dicatat_oleh" gorm:"not null"`
	Petugas     *User     `json:"petugas,omitempty" gorm:"foreignKey:DicatatOleh"`

	Dibatalkan     bool       `json:"dibatalkan" gorm:"not null;default:false;index"`
	DibatalkanOleh *uint      `json:"dibatalkan_oleh"`
	DibatalkanPada *time.Time `json:"dibatalkan_pada"`

	CreatedAt time.Time `json:"created_at"`
}

func (KabelPemakaian) TableName() string { return "kabel_pemakaian" }

// HitungTurunan mengisi Terpakai/Sisa/Status dari pemakaian yang sudah
// di-preload (hanya yang tidak dibatalkan yang dihitung).
func (g *KabelGulungan) HitungTurunan(terpakai int) {
	if terpakai < 0 {
		terpakai = 0
	}
	g.Terpakai = terpakai
	g.Sisa = g.PanjangAwal - terpakai
	if g.Sisa < 0 {
		g.Sisa = 0
	}
	if g.Sisa == 0 {
		g.Status = "habis"
	} else {
		g.Status = "aktif"
	}
}
