package gudang

import (
	"github.com/inventory-backend/internal/model"
	"github.com/inventory-backend/pkg/utils"
)

type Repository interface {
	ListKategori(p utils.PaginationParams) ([]model.Kategori, int64, error)
	FindKategoriByID(id uint) (*model.Kategori, error)
	FindKategoriByNama(nama string) (*model.Kategori, error)
	CreateKategori(k *model.Kategori) error
	UpdateKategori(k *model.Kategori) error
	DeleteKategori(id uint) error
	CountKategori() (int64, error)

	ListSatuan(p utils.PaginationParams) ([]model.Satuan, int64, error)
	FindSatuanByID(id uint) (*model.Satuan, error)
	FindSatuanByNama(nama string) (*model.Satuan, error)
	CreateSatuan(s *model.Satuan) error
	UpdateSatuan(s *model.Satuan) error
	DeleteSatuan(id uint) error

	ListGudang(p utils.PaginationParams) ([]model.Gudang, int64, error)
	FindGudangByID(id uint) (*model.Gudang, error)
	FindGudangByKode(kode string) (*model.Gudang, error)
	CreateGudang(g *model.Gudang) error
	UpdateGudang(g *model.Gudang) error
	DeleteGudang(id uint) error
	CountGudang() (int64, error)

	ListRak(p utils.PaginationParams, gudangID uint) ([]model.Rak, int64, error)
	FindRakByID(id uint) (*model.Rak, error)
	FindRakByKode(kode string) (*model.Rak, error)
	CreateRak(r *model.Rak) error
	UpdateRak(r *model.Rak) error
	DeleteRak(id uint) error
	CountRakAll() (int64, error)
	CountRakByStatus(status string) (int64, error)
	CountRakByGudang(gudangID uint) (int64, error)

	AdjustRakTerisi(rakID uint, delta int) (*model.Rak, error)
}
