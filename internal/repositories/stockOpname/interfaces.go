package stock_opname

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	Status   string
	GudangID uint
}

type ItemInput struct {
	BarangID  uint
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
	MulaiPenghitungan(id uint, userID uint) (*model.StockOpname, error)
	KirimPemeriksaan(id uint, userID uint) (*model.StockOpname, error)
	Setujui(id uint, userID uint) (*model.StockOpname, error)
	Tolak(id uint, userID uint, reason string) (*model.StockOpname, error)
	Riwayat(id uint) ([]model.StockOpnameStatusHistory, error)
	Batalkan(id uint) (*model.StockOpname, error)

	CountByStatus(status string) (int64, error)

	// NextNomor menghasilkan nomor opname berurutan berikutnya
	// (format SO-YYYYMM-0001) lewat pkg/docnumber.
	NextNomor() (string, error)
}
