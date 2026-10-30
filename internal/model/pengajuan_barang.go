package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

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

	// Jenis: "masuk" | "keluar" | "rusak" | "pengajuan" | "template" — menentukan dokumen
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

	// TandaTangan: jejak tanda tangan digital kedua pihak (pencatat/gudang dan
	// penyetuju). Hanya metadata yang ikut ke JSON — gambar tanda tangannya
	// sendiri disimpan terpisah dan dilayani lewat endpoint terautentikasi
	// (lihat PengajuanTandaTangan.Data).
	TandaTangan []PengajuanTandaTangan `json:"tanda_tangan,omitempty" gorm:"foreignKey:PengajuanID"`

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

	// SatuanID: satuan yang dipakai pada qty baris ini (mis. roll, meter, pcs),
	// diambil dari master satuan. Opsional — kosong berarti memakai satuan
	// bawaan barangnya (data lama sebelum kolom ini ada).
	SatuanID *uint   `json:"satuan_id" gorm:"index"`
	Satuan   *Satuan `json:"satuan,omitempty" gorm:"foreignKey:SatuanID"`
}

func (PengajuanBarangItem) TableName() string { return "pengajuan_barang_items" }

// Peran penanda tangan pada pengajuan barang.
const (
	PeranTTDPencatat  = "pencatat"  // pihak pertama: gudang/pengaju
	PeranTTDPenyetuju = "penyetuju" // pihak kedua: GA/atasan yang menyetujui
)

// PengajuanTandaTangan adalah tanda tangan digital satu pihak pada satu
// pengajuan. Satu pengajuan maksimal punya satu tanda tangan per peran.
//
// DokumenHash adalah SHA-256 dari isi pengajuan (jenis, gudang, tanggal,
// keperluan, daftar barang+qty) saat ditandatangani. Kalau pengajuan diubah
// setelah pihak pertama menandatangani, hash-nya tidak lagi cocok dan tanda
// tangan lama dibuang — jadi pihak kedua selalu menyetujui isi yang sama
// persis dengan yang ditandatangani pihak pertama.
type PengajuanTandaTangan struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	PengajuanID uint      `json:"pengajuan_id" gorm:"not null;uniqueIndex:idx_pengajuan_ttd_peran"`
	Peran       string    `json:"peran" gorm:"size:20;not null;uniqueIndex:idx_pengajuan_ttd_peran"`
	UserID      uint      `json:"user_id" gorm:"not null"`
	Nama        string    `json:"nama" gorm:"size:150"`
	Jabatan     string    `json:"jabatan" gorm:"size:100"`
	Data        []byte    `json:"-" gorm:"type:bytea;not null"`
	IPAddress   string    `json:"ip_address" gorm:"size:64"`
	DokumenHash string    `json:"dokumen_hash" gorm:"size:64;not null"`
	SignedAt    time.Time `json:"signed_at" gorm:"not null"`
}

func (PengajuanTandaTangan) TableName() string { return "pengajuan_tanda_tangan" }

// ComputeDokumenHash menghitung sidik jari isi pengajuan (tanpa ID/status, jadi
// bisa dihitung sebelum pengajuan disimpan). Items harus sudah terisi.
func (p *PengajuanBarang) ComputeDokumenHash() string {
	items := make([]string, 0, len(p.Items))
	for _, it := range p.Items {
		entry := fmt.Sprintf("%d:%d", it.BarangID, it.Qty)
		// Satuan hanya ikut hash kalau diisi, supaya hash dokumen lama
		// (tanpa satuan) tidak berubah dan tanda tangannya tetap valid.
		if it.SatuanID != nil {
			entry += fmt.Sprintf("@%d", *it.SatuanID)
		}
		items = append(items, entry)
	}
	sort.Strings(items)
	payload := strings.Join([]string{
		p.NomorPengajuan, p.Jenis, fmt.Sprint(p.GudangID), p.Tanggal.UTC().Format("2006-01-02"),
		p.Keperluan, strings.Join(items, ","),
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// TandaTanganPeran mengembalikan tanda tangan untuk peran tertentu (nil kalau belum ada).
func (p *PengajuanBarang) TandaTanganPeran(peran string) *PengajuanTandaTangan {
	for i := range p.TandaTangan {
		if p.TandaTangan[i].Peran == peran {
			return &p.TandaTangan[i]
		}
	}
	return nil
}
