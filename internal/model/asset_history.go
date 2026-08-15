package model

import "time"

type AssetHistory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	AssetID   uint      `json:"asset_id" gorm:"not null;index"`
	EventType string    `json:"event_type" gorm:"size:20;"`
	TabelLama string    `json:"tabel_lama" gorm:"size:255"`
	TabelBaru string    `json:"tabel_baru" gorm:"size:255"`
	Catatan   string    `json:"catatan" gorm:"size:255"`
	UserID    *uint     `json:"user_id" `
	NamaUser  string    `json:"nama_user" gorm:"size:200"`
	CreatedAt time.Time `json:"creater_at" `
}

func (AssetHistory) TableName() string { return "asset_history" }
