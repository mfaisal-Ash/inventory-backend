package barang_rusak

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	Status   string
	BarangID uint
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.BarangRusak, int64, error)
	FindByID(id uint) (*model.BarangRusak, error)
	Create(b *model.BarangRusak) error
	Update(b *model.BarangRusak) error
	Delete(id uint) error

	CountByStatus(status string) (int64, error)

	// SimpanKeGudang: pengganti fitur retur-ke-supplier yang sudah dihapus —
	// menambahkan kembali 1 unit ke stok gudang tujuan (barang_stok_gudang +
	// total stok barang) dan mengunci baris ini ke status akhir
	// "disimpan_gudang" dalam satu transaksi, supaya stoknya tidak bisa
	// dobel-ditambahkan lewat panggilan berulang.
	SimpanKeGudang(id uint, gudangID uint) (*model.BarangRusak, error)
}
