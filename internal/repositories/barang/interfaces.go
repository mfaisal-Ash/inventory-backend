package barang

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	IDKategori uint
	IDSatuan   uint
	StatusStok string
	HanyaAktif bool
	Merek      string

	Status   []string
	SubmitBy uint
	// DelegasiTo menambahkan data yang ditugaskan kepada user ke hasil filter.
	DelegasiTo uint

	// HanyaDelegasiTo digunakan untuk menampilkan data yang hanya didelegasikan.
	HanyaDelegasiTo uint
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.Barang, int64, error)
	FindByID(id uint) (*model.Barang, error)
	FindByKode(kode string) (*model.Barang, error)
	Create(b *model.Barang) error
	CreateWithInitialStock(b *model.Barang, gudangID uint, stok int) error
	Update(b *model.Barang) error
	Delete(id uint) error

	AdjustStok(id uint, delta int) (*model.Barang, error)

	SetStokGudangAwal(barangID, gudangID uint, stok int) error

	// UpdateWithStokKoreksi: simpan perubahan field non-stok SEKALIGUS
	// mengoreksi stok — delta diterapkan ke BarangStokGudang milik
	// stokGudangID, lalu Barang.Stok di-derive ulang dari
	// SUM(BarangStokGudang) supaya total tidak pernah menyimpang dari
	// rincian per-gudangnya (real-time-safe, lihat SyncBarangStokTotalTx).
	UpdateWithStokKoreksi(b *model.Barang, stokGudangID uint, delta int) error

	// NextSKUNumber: nomor urut berikutnya untuk kategoriCode di periode
	// (MMYYYY) tertentu — lihat pkg/skucode.
	NextSKUNumber(kategoriCode, periode string) (int, error)

	CountAll() (int64, error)
	CountStokMenipis() (int64, error)
	SumNilaiInventaris() (int64, error)
	SumStok() (int64, error)
}
