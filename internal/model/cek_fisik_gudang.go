package model

import "time"

// CekFisikGudang mencatat satu kali pengecekan fisik sebuah barang di sebuah
// gudang: jumlah di sistem (Kelola Barang + mutasi) dibandingkan dengan jumlah
// yang benar-benar ada di rak. Baris bersifat append-only (riwayat); yang
// dipakai untuk status terkini adalah baris terbaru per barang+gudang.
//
// Pengecekan TIDAK mengubah stok. Kalau fisik berbeda, koreksi stok dilakukan
// lewat Stock Opname supaya tercatat dengan dokumen & persetujuan.
type CekFisikGudang struct {
	ID       uint `json:"id" gorm:"primaryKey"`
	GudangID uint `json:"gudang_id" gorm:"not null;index:idx_cek_gudang_barang,priority:1"`
	BarangID uint `json:"barang_id" gorm:"not null;index:idx_cek_gudang_barang,priority:2"`

	JumlahSistem int  `json:"jumlah_sistem" gorm:"not null;default:0"`
	JumlahFisik  int  `json:"jumlah_fisik" gorm:"not null;default:0"`
	Sesuai       bool `json:"sesuai" gorm:"not null;default:false"`

	Catatan   string `json:"catatan" gorm:"size:255"`
	DicekOleh uint   `json:"dicek_oleh" gorm:"index"`
	NamaCek   string `json:"nama_cek" gorm:"size:100"`

	DicekPada time.Time `json:"dicek_pada" gorm:"not null;index"`
}

func (CekFisikGudang) TableName() string { return "cek_fisik_gudang" }
