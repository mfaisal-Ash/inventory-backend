package pengajuan_barang

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	Status   string
	GudangID uint
	Jenis    string
	BarangID uint
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.PengajuanBarang, int64, error)
	FindByID(id uint) (*model.PengajuanBarang, error)
	Create(pengajuan *model.PengajuanBarang) error
	Update(pengajuan *model.PengajuanBarang, items []model.PengajuanBarangItem) error
	Delete(id uint) error

	// Setujui menyetujui pengajuan: dalam satu transaksi, membuat dokumen
	// BarangKeluar dari daftar barang pengajuan lalu langsung memotong stok
	// (kalau semua barangnya bukan barang ber-nomor-seri) atau membiarkan
	// dokumen BarangKeluar berstatus draft supaya staf gudang menuntaskannya
	// lewat halaman Barang Keluar (kalau ada barang ber-nomor-seri, karena
	// nomor serinya harus dipilih manual satu-satu di sana).
	Setujui(id uint, userID uint, namaGA, jabatanGA, catatan string) (*model.PengajuanBarang, error)
	Tolak(id uint, userID uint, namaGA, jabatanGA, catatan string) (*model.PengajuanBarang, error)

	CountByStatus(status string) (int64, error)

	// NextNomor menghasilkan nomor pengajuan berurutan berikutnya
	// (format PJ-YYYYMM-0001) lewat pkg/docnumber.
	NextNomor() (string, error)
}
