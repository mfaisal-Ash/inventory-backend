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
	// Update mengganti isi pengajuan yang masih "diajukan". Karena isinya
	// berubah, SEMUA tanda tangan lama dibuang (sidik jari dokumennya sudah
	// tidak cocok); ttd (kalau tidak nil) disimpan sebagai tanda tangan
	// pencatat yang baru.
	Update(pengajuan *model.PengajuanBarang, items []model.PengajuanBarangItem, ttd *model.PengajuanTandaTangan) error
	Delete(id uint) error

	// Setujui menyetujui pengajuan: dalam satu transaksi, membuat dokumen
	// BarangKeluar dari daftar barang pengajuan lalu langsung memotong stok
	// (kalau semua barangnya bukan barang ber-nomor-seri) atau membiarkan
	// dokumen BarangKeluar berstatus draft supaya staf gudang menuntaskannya
	// lewat halaman Barang Keluar (kalau ada barang ber-nomor-seri, karena
	// nomor serinya harus dipilih manual satu-satu di sana).
	//
	// ttd = tanda tangan digital penyetuju. Untuk jenis yang wajib alur dua
	// pihak (constant.PengajuanWajibTTD), Setujui menolak kalau: pengaju belum
	// tanda tangan atau isi pengajuan berubah sesudahnya, ttd kosong, atau
	// penyetuju orang yang sama dengan pengaju. Semua dicek di dalam transaksi
	// yang sama dengan pembuatan dokumen, jadi tidak ada celah balapan.
	Setujui(id uint, userID uint, namaGA, jabatanGA, catatan string, ttd *model.PengajuanTandaTangan) (*model.PengajuanBarang, error)
	Tolak(id uint, userID uint, namaGA, jabatanGA, catatan string) (*model.PengajuanBarang, error)

	CountByStatus(status string) (int64, error)

	// FindTandaTangan mengambil satu tanda tangan LENGKAP dengan gambarnya
	// (FindByID/List sengaja tidak memuat gambar supaya ringan).
	FindTandaTangan(pengajuanID uint, peran string) (*model.PengajuanTandaTangan, error)

	// OpenMasukByBarang: untuk tiap barang_id yang diberikan, nomor pengajuan
	// jenis "masuk" yang masih menunggu persetujuan (status diajukan). Dipakai
	// daftar stok menipis supaya barang yang sudah diajukan tidak diajukan dua kali.
	OpenMasukByBarang(barangIDs []uint) (map[uint]string, error)

	// NextNomor menghasilkan nomor pengajuan berurutan berikutnya
	// (format PJ-YYYYMM-0001) lewat pkg/docnumber.
	NextNomor() (string, error)
}
