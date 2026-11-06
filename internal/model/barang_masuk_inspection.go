package model

import "time"

// BarangMasukInspection records the receiving inspection performed before stock is posted.
type BarangMasukInspection struct {
	ID                uint         `json:"id" gorm:"primaryKey"`
	BarangMasukID     uint         `json:"barang_masuk_id" gorm:"not null;index"`
	BarangMasuk       *BarangMasuk `json:"barang_masuk,omitempty" gorm:"foreignKey:BarangMasukID"`
	BarangMasukItemID uint         `json:"barang_masuk_item_id" gorm:"not null;index"`
	ReceivedQty       int          `json:"received_qty" gorm:"not null"`
	Identifiers       string       `json:"identifiers,omitempty" gorm:"type:jsonb"`
	InspectedBy       uint         `json:"inspected_by" gorm:"not null"`
	InspectedAt       time.Time    `json:"inspected_at"`
}

func (BarangMasukInspection) TableName() string { return "barang_masuk_inspections" }
