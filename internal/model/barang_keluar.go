package model

import "time"

type BarangKeluar struct {
	ID               uint               `json:"id" gorm:"primaryKey"`
	NomorPengeluaran string             `json:"nomor_pengeluaran" gorm:"size:30;uniqueIndex;not null"`
	GudangID         uint               `json:"gudang_id" gorm:"not null;index"`
	Gudang           *Gudang            `json:"gudang,omitempty" gorm:"foreignKey:GudangID"`
	Status           string             `json:"status" gorm:"size:20;not null;default:'draft';index"`
	Tanggal          time.Time          `json:"tanggal" gorm:"not null"`
	Keperluan        string             `json:"keperluan" gorm:"size:255"`
	Penerima         string             `json:"penerima" gorm:"size:150"`
	DikeluarkanOleh  *uint              `json:"dikeluarkan_oleh"`
	CompletedAt      *time.Time         `json:"completed_at"`
	IsProtected      bool               `json:"is_protected" gorm:"not null;default:false"`
	Items            []BarangKeluarItem `json:"items,omitempty" gorm:"foreignKey:BarangKeluarID"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

func (BarangKeluar) TableName() string { return "barang_keluar" }

type BarangKeluarItem struct {
	ID             uint    `json:"id" gorm:"primaryKey"`
	BarangKeluarID uint    `json:"barang_keluar_id" gorm:"not null;index"`
	BarangID       uint    `json:"barang_id" gorm:"not null;index"`
	Barang         *Barang `json:"barang,omitempty" gorm:"foreignKey:BarangID"`
	Qty            int     `json:"qty" gorm:"not null"`

	// Spesifikasi pemasangan (mis. kabel: dari Qty yang dikeluarkan, berapa
	// yang sudah terpasang di lapangan). Diisi/diperbarui belakangan setelah
	// dokumen barang keluar selesai — lihat UpdateSpesifikasi di repository.
	JumlahTerpasang    int    `json:"jumlah_terpasang" gorm:"not null;default:0"`
	CatatanSpesifikasi string `json:"catatan_spesifikasi" gorm:"size:255"`

	// JumlahSisa = Qty - JumlahTerpasang, dihitung on-the-fly (bukan kolom
	// tabel) supaya tidak ada risiko data dobel-sumber-kebenaran yang bisa
	// desinkron dari Qty/JumlahTerpasang. Diisi lewat HitungSisa()/
	// HitungSisaItems() sebelum item dikembalikan ke client.
	JumlahSisa int `json:"jumlah_sisa" gorm:"-"`
}

func (BarangKeluarItem) TableName() string { return "barang_keluar_items" }

// HitungSisa mengisi JumlahSisa = Qty - JumlahTerpasang untuk satu item.
func (i *BarangKeluarItem) HitungSisa() {
	sisa := i.Qty - i.JumlahTerpasang
	if sisa < 0 {
		sisa = 0
	}
	i.JumlahSisa = sisa
}

// HitungSisaItems mengisi JumlahSisa untuk seluruh Items pada dokumen ini.
func (bk *BarangKeluar) HitungSisaItems() {
	for idx := range bk.Items {
		bk.Items[idx].HitungSisa()
	}
}
