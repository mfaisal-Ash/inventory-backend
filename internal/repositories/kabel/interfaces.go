package kabel

import (
	"time"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	GudangID uint
	BarangID uint
	// Status: "" (semua) | "aktif" | "habis".
	Status string
}

type Summary struct {
	TotalGulungan int64 `json:"total_gulungan"`
	GulunganAktif int64 `json:"gulungan_aktif"`
	GulunganHabis int64 `json:"gulungan_habis"`
	TotalPanjang  int64 `json:"total_panjang"`
	TotalTerpakai int64 `json:"total_terpakai"`
	TotalSisa     int64 `json:"total_sisa"`
}

type PemakaianInput struct {
	Jumlah    int
	Tanggal   time.Time
	Klien     string
	Keperluan string
	UserID    uint
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.KabelGulungan, int64, error)
	FindByID(id uint) (*model.KabelGulungan, error)
	Summary(f Filter) (*Summary, error)

	// Create mendaftarkan gulungan baru. Tidak mengubah stok (stok kabel
	// masuk lewat Barang Masuk dalam meter); hanya divalidasi supaya total
	// sisa gulungan aktif di gudang tidak melebihi stok gudang.
	Create(g *model.KabelGulungan) error

	// Pakai mencatat pemakaian: sisa gulungan berkurang dan stok gudang
	// berkurang dalam satu transaksi.
	Pakai(gulunganID uint, in PemakaianInput) (*model.KabelGulungan, error)

	// BatalPakai membatalkan satu pemakaian: meter kembali ke gulungan dan
	// stok gudang bertambah lagi.
	BatalPakai(gulunganID, pemakaianID, userID uint) (*model.KabelGulungan, error)

	// Hapus menghapus (soft delete) gulungan yang belum pernah dipakai.
	Hapus(id uint) error
}
