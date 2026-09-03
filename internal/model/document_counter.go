package model

import "time"

// DocumentCounter menyimpan counter berurutan per prefix+periode (mis. kunci
// "BM-202609" untuk Barang Masuk bulan September 2026), dipakai
// pkg/docnumber untuk menghasilkan nomor dokumen (nomor penerimaan,
// pengeluaran, pengajuan, stock opname) yang berurutan dan tidak bentrok,
// menggantikan skema lama yang menempel ke time.Now().UnixNano() (kelihatan
// acak ke pengguna dan secara teori bisa tabrakan).
type DocumentCounter struct {
	Key       string    `json:"key" gorm:"primaryKey;size:32"`
	Counter   int       `json:"counter" gorm:"not null;default:0"`
	UpdatedAt time.Time `json:"updated_at"`
}
