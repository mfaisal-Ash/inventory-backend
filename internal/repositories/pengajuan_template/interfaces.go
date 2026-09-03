package pengajuan_template

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	OnlyActive bool
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.PengajuanTemplate, int64, error)
	FindByID(id uint) (*model.PengajuanTemplate, error)
	Create(t *model.PengajuanTemplate) error
	Delete(id uint) error
}
