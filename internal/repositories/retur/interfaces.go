package retur

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	Status   string
	BarangID uint
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.ReturPenggantian, int64, error)
	FindByID(id uint) (*model.ReturPenggantian, error)
	CountByStatus(status string) (int64, error)

	// Create membuat laporan retur (nomor RT-YYYYMM-xxxx) dalam 1 transaksi.
	Create(rt *model.ReturPenggantian) error

	// Pengecekan: hasil teknisi. hasil = perlu_ganti | tidak_perlu_ganti.
	Pengecekan(id uint, hasil, catatan string, userID uint) (*model.ReturPenggantian, error)

	// Ganti: eksekusi penggantian dalam 1 transaksi — unit baru (harus Normal,
	// di gudang retur) keluar ke klien & stok gudang -1; unit lama ditarik ke
	// gudang berstatus rusak (tidak menambah stok tersedia) dan dicatat di
	// Barang Rusak.
	Ganti(id uint, serialBaru string, userID uint) (*model.ReturPenggantian, error)
}
