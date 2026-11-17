package model

import "time"

// InspeksiStok menyimpan riwayat pemeriksaan kondisi stok non-serial.
type InspeksiStok struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	BarangID          uint      `json:"barang_id" gorm:"not null;index"`
	GudangID          uint      `json:"gudang_id" gorm:"not null;index"`
	LabelBatch        string    `json:"label_batch" gorm:"size:100"`
	DiperiksaQty      int64     `json:"diperiksa_qty" gorm:"not null"`
	NormalQty         int64     `json:"normal_qty" gorm:"not null"`
	RusakQty          int64     `json:"rusak_qty" gorm:"not null"`
	BelumDiperiksaQty int64     `json:"belum_diperiksa_qty" gorm:"not null"`
	Catatan           string    `json:"catatan" gorm:"size:500"`
	DiperiksaOleh     uint      `json:"diperiksa_oleh" gorm:"not null;index"`
	DiperiksaPada     time.Time `json:"diperiksa_pada" gorm:"not null"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (InspeksiStok) TableName() string { return "inspeksi_stok" }
