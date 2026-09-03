package model

import "time"

// PengajuanBarang adalah pengajuan pengeluaran barang dari gudang yang harus
// disetujui/ditolak (mis. oleh bagian General Affairs/GA) sebelum barangnya
// benar-benar keluar. Begitu disetujui, sistem otomatis membuat dokumen
// BarangKeluar dan memotong stok (lihat repositories/pengajuan_barang).
//
// NamaPencatat/JabatanPencatat dan NamaGA/JabatanGA dipakai untuk mengisi
// kop dokumen cetak (dua kolom tanda tangan: Bagian Pencatatan/Gudang &
// Bagian General Affairs) — diisi bebas sebagai teks karena sistem ini
// belum punya role/departemen "GA" formal, jadi siapa pun yang berwenang
// bisa dicatat namanya di sini lalu tanda tangan fisik di atas kertas.
type PengajuanBarang struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	NomorPengajuan string    `json:"nomor_pengajuan" gorm:"size:30;uniqueIndex;not null"`
	GudangID       uint      `json:"gudang_id" gorm:"not null;index"`
	Gudang         *Gudang   `json:"gudang,omitempty" gorm:"foreignKey:GudangID"`
	Tanggal        time.Time `json:"tanggal" gorm:"not null"`
	Keperluan      string    `json:"keperluan" gorm:"size:255;not null"`
	Status         string    `json:"status" gorm:"size:20;not null;default:'diajukan';index"`

	// Jenis: "masuk" | "keluar" | "rusak" | "template" — menentukan dokumen
	// apa yang otomatis dibuat saat pengajuan disetujui. Default "keluar"
	// supaya data lama (sebelum kolom ini ada) tetap berperilaku sama
	// seperti sebelumnya. "template" adalah pengajuan bebas berbasis
	// formulir yang diunggah admin (lihat PengajuanTemplate/TemplateID di
	// bawah) — tidak terkait barang & tidak membuat dokumen otomatis apa
	// pun saat disetujui. Jenis ini menggantikan "umum" (pengajuan ke
	// atasan) yang dulu ada di sini.
	Jenis string `json:"jenis" gorm:"size:10;not null;default:'keluar';index"`

	// Perihal: catatan/subjek singkat opsional untuk jenis apa pun — dulu
	// wajib diisi khusus untuk jenis "umum" (sekarang digantikan
	// "template"), sekarang murni opsional.
	Perihal string `json:"perihal" gorm:"size:150"`

	// TemplateID: wajib diisi (divalidasi di controller) hanya untuk jenis
	// "template", merujuk ke formulir yang dipilih dari PengajuanTemplate.
	// Nil untuk jenis lain.
	TemplateID *uint              `json:"template_id"`
	Template   *PengajuanTemplate `json:"template,omitempty" gorm:"foreignKey:TemplateID"`

	DiajukanOleh    uint   `json:"diajukan_oleh" gorm:"not null"`
	Pengaju         *User  `json:"pengaju,omitempty" gorm:"foreignKey:DiajukanOleh"`
	NamaPencatat    string `json:"nama_pencatat" gorm:"size:150"`
	JabatanPencatat string `json:"jabatan_pencatat" gorm:"size:100"`

	DiprosesOleh  *uint      `json:"diproses_oleh"`
	Pemroses      *User      `json:"pemroses,omitempty" gorm:"foreignKey:DiprosesOleh"`
	NamaGA        string     `json:"nama_ga" gorm:"size:150"`
	JabatanGA     string     `json:"jabatan_ga" gorm:"size:100"`
	DiprosesPada  *time.Time `json:"diproses_pada"`
	CatatanProses string     `json:"catatan_proses" gorm:"size:255"`

	BarangKeluarID *uint         `json:"barang_keluar_id"`
	BarangKeluar   *BarangKeluar `json:"barang_keluar,omitempty" gorm:"foreignKey:BarangKeluarID"`

	BarangMasukID *uint        `json:"barang_masuk_id"`
	BarangMasuk   *BarangMasuk `json:"barang_masuk,omitempty" gorm:"foreignKey:BarangMasukID"`

	// BarangRusak: daftar laporan barang rusak yang dibuat otomatis saat
	// pengajuan jenis "rusak" ini disetujui (satu baris per unit per item).
	BarangRusak []BarangRusak `json:"barang_rusak,omitempty" gorm:"foreignKey:PengajuanID"`

	Items []PengajuanBarangItem `json:"items,omitempty" gorm:"foreignKey:PengajuanID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PengajuanBarang) TableName() string { return "pengajuan_barang" }

func (p *PengajuanBarang) IsFinal() bool {
	return p.Status == "disetujui" || p.Status == "ditolak"
}

type PengajuanBarangItem struct {
	ID          uint    `json:"id" gorm:"primaryKey"`
	PengajuanID uint    `json:"pengajuan_id" gorm:"not null;index"`
	BarangID    uint    `json:"barang_id" gorm:"not null;index"`
	Barang      *Barang `json:"barang,omitempty" gorm:"foreignKey:BarangID"`
	Qty         int     `json:"qty" gorm:"not null"`
}

func (PengajuanBarangItem) TableName() string { return "pengajuan_barang_items" }
