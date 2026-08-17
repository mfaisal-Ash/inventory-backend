package model

import (
	"time"

	"gorm.io/gorm"
)

type BarangRusak struct {
	ID uint `json:"id" gorm:"primaryKey"`

	BarangID *uint   `json:"barang_id"`
	Barang   *Barang `json:"barang,omitempty" gorm:"foreignKey:BarangID"`

	LabelBarang string `json:"label_barang" gorm:"size:60;not null;index"`
	NamaBarang  string `json:"nama_barang" gorm:"size:150;not null"`
	Keterangan  string `json:"keterangan" gorm:"size:500"`

	FotoURL string `json:"foto_url" gorm:"size:255"`

	JenisBarang string `json:"jenis_barang" gorm:"size:10"`

	Status string `json:"status" gorm:"size:20;not null;default:'pengecekan';index"`

	DilaporkanOleh uint       `json:"dilaporkan_oleh" gorm:"not null"`
	Pelapor        *User      `json:"pelapor,omitempty" gorm:"foreignKey:DilaporkanOleh"`
	DicekOleh      *uint      `json:"dicek_oleh"`
	Pemeriksa      *User      `json:"pemeriksa,omitempty" gorm:"foreignKey:DicekOleh"`
	DicekPada      *time.Time `json:"dicek_pada"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (BarangRusak) TableName() string { return "barang_rusak" }
