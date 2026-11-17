package model

import "time"

// BalanceStok adalah saldo kondisi terkini untuk barang non-serial per gudang.
type BalanceStok struct {
	BarangID          uint      `json:"barang_id" gorm:"primaryKey"`
	GudangID          uint      `json:"gudang_id" gorm:"primaryKey"`
	NormalQty         int64     `json:"normal_qty" gorm:"not null;default:0"`
	RusakQty          int64     `json:"rusak_qty" gorm:"not null;default:0"`
	BelumDiperiksaQty int64     `json:"belum_diperiksa_qty" gorm:"not null;default:0"`
	ReturQty          int64     `json:"retur_qty" gorm:"not null;default:0"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (BalanceStok) TableName() string { return "balance_stok" }

func (b BalanceStok) TersediaQty() int64 {
	tersedia := b.NormalQty - b.ReturQty
	if tersedia < 0 {
		return 0
	}
	return tersedia
}
