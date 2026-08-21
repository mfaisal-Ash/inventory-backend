package cod

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	Status string
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.CodTransaction, int64, error)
	FindByID(id uint) (*model.CodTransaction, error)
	FindByKode(kode string) (*model.CodTransaction, error)
	Create(c *model.CodTransaction) error
	Update(c *model.CodTransaction) error
	Delete(id uint) error

	CountByStatus(status string) (int64, error)
	SumNominal() (int64, error)
}
