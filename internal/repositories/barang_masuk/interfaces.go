package barang_masuk

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	Status   string
	GudangID uint

	KategoriID uint
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.BarangMasuk, int64, error)
	FindByID(id uint) (*model.BarangMasuk, error)
	FindByNomor(nomor string) (*model.BarangMasuk, error)
	Create(bm *model.BarangMasuk) error
	Update(bm *model.BarangMasuk, items []model.BarangMasukItem) error
	Delete(id uint) error

	Complete(id uint, userID uint, serials map[uint][]string) (*model.BarangMasuk, error)
	Batalkan(id uint) (*model.BarangMasuk, error)

	CountByStatus(status string) (int64, error)
}
