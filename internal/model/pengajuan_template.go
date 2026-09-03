package model

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PengajuanTemplate adalah berkas formulir kosong (docx/pdf) yang diunggah
// admin/super admin untuk kebutuhan pergudangan & inventaris di luar 3 jenis
// baku (Barang Masuk/Keluar/Rusak) — mis. formulir cuti, peminjaman alat,
// permintaan ATK, dsb. Isi berkasnya disimpan sebagai bytea di database
// (pola yang sama dipakai foto bukti BarangRusak & avatar User di app ini —
// lihat BarangRusak.FotoData), bukan di disk, supaya tidak perlu volume
// penyimpanan terpisah.
//
// PengajuanBarang dengan Jenis == JenisPengajuanTemplate merujuk ke satu
// baris di sini lewat TemplateID — "mencetak" pengajuan semacam ini berarti
// mengunduh berkas asli formulir ini apa adanya (kosong, untuk diisi/
// ditandatangani manual di atas kertas), bukan dokumen hasil generate
// sistem, karena sistem tidak tahu struktur internal tiap formulir yang
// diunggah admin.
type PengajuanTemplate struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	Nama      string `json:"nama" gorm:"size:150;not null"`
	Deskripsi string `json:"deskripsi" gorm:"size:255"`
	IsActive  bool   `json:"is_active" gorm:"not null;default:true;index"`

	FileName        string `json:"file_name" gorm:"size:255;not null"`
	FileContentType string `json:"-" gorm:"size:150;not null"`
	FileSize        int64  `json:"file_size" gorm:"not null;default:0"`
	FileData        []byte `json:"-" gorm:"type:bytea;not null"`

	UploadedBy uint  `json:"uploaded_by" gorm:"not null"`
	Pengunggah *User `json:"pengunggah,omitempty" gorm:"foreignKey:UploadedBy"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (PengajuanTemplate) TableName() string { return "pengajuan_templates" }

// FileURL: URL berkas versi-tercache (query "v" berubah tiap UpdatedAt
// berubah, supaya browser tidak menyimpan cache lama kalau berkasnya pernah
// diganti) — dilayani lewat endpoint terautentikasi, bukan static file.
func (t PengajuanTemplate) FileURL() string {
	return fmt.Sprintf("/pengajuan-templates/%d/file?v=%d", t.ID, t.UpdatedAt.Unix())
}

func (t PengajuanTemplate) MarshalJSON() ([]byte, error) {
	type Alias PengajuanTemplate
	return json.Marshal(struct {
		Alias
		FileURL string `json:"file_url"`
	}{
		Alias:   Alias(t),
		FileURL: t.FileURL(),
	})
}
