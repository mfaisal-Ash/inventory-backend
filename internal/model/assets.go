package model

// Assets is the lightweight asset projection used by JSON responses.
// Fields must be exported for encoding/json and GORM to populate them.
type Assets struct {
	ID         uint   `json:"id" gorm:"primaryKey"`
	BarangID   uint   `json:"barang_id" gorm:"index;not null"`
	LabelBarang string `json:"label_barang"`
	NamaBarang  string `json:"nama_barang"`
}
