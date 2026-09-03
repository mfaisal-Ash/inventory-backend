package barang_keluar

import (
	"time"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	Status   string
	GudangID uint

	KategoriID uint
	BarangID   uint

	// Merek/Tipe: filter tambahan (join ke tabel barang, sama seperti
	// KategoriID/BarangID) supaya daftar barang keluar bisa dipersempit
	// berdasarkan merek/tipe barang di dalam dokumennya.
	Merek string
	Tipe  string
}

// SpesifikasiFilter membatasi rekap spesifikasi (terpakai/terpasang/sisa)
// per barang — cuma dihitung dari dokumen barang keluar yang sudah selesai.
type SpesifikasiFilter struct {
	GudangID uint
	From     *time.Time
	To       *time.Time
}

// SpesifikasiRecapRow adalah 1 baris rekap per barang, mis. "kabel: dipakai
// 100, terpasang 80, sisa 20".
type SpesifikasiRecapRow struct {
	BarangID       uint   `json:"barang_id"`
	NamaBarang     string `json:"nama_barang"`
	KodeBarang     string `json:"kode_barang"`
	Satuan         string `json:"satuan"`
	TotalTerpakai  int    `json:"total_terpakai"`
	TotalTerpasang int    `json:"total_terpasang"`
	TotalSisa      int    `json:"total_sisa"`
}

// SpesifikasiListFilter membatasi daftar baris spesifikasi per-item (bukan
// rekap agregat) — dipakai tab "Spesifikasi" di halaman Kelola Barang, mirip
// pola Filter di repositories/barang_serial.
type SpesifikasiListFilter struct {
	BarangID uint
	GudangID uint

	// Status: "" (semua), "belum" (jumlah_terpasang = 0),
	// "sebagian" (0 < jumlah_terpasang < qty), atau
	// "selesai" (jumlah_terpasang >= qty).
	Status string
}

// SpesifikasiListRow adalah 1 baris item barang keluar yang sudah selesai,
// dipadukan dengan info dokumen & barangnya supaya bisa ditampilkan sebagai
// daftar flat (mirip "Daftar Unit" di Nomor Seri) tanpa perlu buka detail
// dokumennya satu-satu.
type SpesifikasiListRow struct {
	ItemID             uint      `json:"item_id"`
	BarangKeluarID     uint      `json:"barang_keluar_id"`
	NomorPengeluaran   string    `json:"nomor_pengeluaran"`
	Tanggal            time.Time `json:"tanggal"`
	Keperluan          string    `json:"keperluan"`
	GudangID           uint      `json:"gudang_id"`
	NamaGudang         string    `json:"nama_gudang"`
	BarangID           uint      `json:"barang_id"`
	NamaBarang         string    `json:"nama_barang"`
	KodeBarang         string    `json:"kode_barang"`
	Satuan             string    `json:"satuan"`
	Qty                int       `json:"qty"`
	JumlahTerpasang    int       `json:"jumlah_terpasang"`
	JumlahSisa         int       `json:"jumlah_sisa"`
	CatatanSpesifikasi string    `json:"catatan_spesifikasi"`
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.BarangKeluar, int64, error)
	FindByID(id uint) (*model.BarangKeluar, error)
	FindByNomor(nomor string) (*model.BarangKeluar, error)
	Create(bk *model.BarangKeluar) error
	Update(bk *model.BarangKeluar, items []model.BarangKeluarItem) error
	Delete(id uint) error
	SetProtected(id uint, protect bool) error

	Complete(id uint, userID uint, serials map[uint][]string) (*model.BarangKeluar, error)
	Batalkan(id uint) (*model.BarangKeluar, error)

	CountByStatus(status string) (int64, error)

	// UpdateSpesifikasi mencatat progres pemasangan (mis. kabel yang sudah
	// terpasang) untuk 1 item barang keluar. Cuma boleh dilakukan setelah
	// dokumen induknya berstatus selesai.
	UpdateSpesifikasi(itemID uint, jumlahTerpasang int, catatan string) (*model.BarangKeluarItem, error)

	// RecapSpesifikasi meringkas total terpakai/terpasang/sisa per barang
	// dari seluruh dokumen barang keluar yang sudah selesai.
	RecapSpesifikasi(f SpesifikasiFilter) ([]SpesifikasiRecapRow, error)

	// ListSpesifikasi mengembalikan daftar baris item barang keluar (yang
	// dokumen induknya sudah selesai) satu per satu, buat tab "Spesifikasi"
	// di halaman Kelola Barang — beda dari RecapSpesifikasi yang meringkas
	// per barang.
	ListSpesifikasi(p utils.PaginationParams, f SpesifikasiListFilter) ([]SpesifikasiListRow, int64, error)

	// NextNomor menghasilkan nomor pengeluaran berurutan berikutnya
	// (format BK-YYYYMM-0001) lewat pkg/docnumber.
	NextNomor() (string, error)
}
