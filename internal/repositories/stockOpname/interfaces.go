package stock_opname

import (
	"github.com/inventory-backend/internal/model"
	"github.com/inventory-backend/pkg/utils"
)

type Filter struct {
	Status   string
	GudangID uint
}

type ItemInput struct {
	BarangID  uint
	RakID     *uint
	StokFisik int
	Catatan   string
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.StockOpname, int64, error)
	FindByID(id uint) (*model.StockOpname, error)
	FindByNomor(nomor string) (*model.StockOpname, error)
	Create(so *model.StockOpname, inputs []ItemInput) error
	Update(so *model.StockOpname, inputs []ItemInput) error
	Delete(id uint) error

	Complete(id uint, userID uint) (*model.StockOpname, error)
	Batalkan(id uint) (*model.StockOpname, error)

	CountByStatus(status string) (int64, error)
}
