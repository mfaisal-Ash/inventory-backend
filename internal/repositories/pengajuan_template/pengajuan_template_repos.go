package pengajuan_template

import (
	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.OnlyActive {
		q = q.Where("is_active = ?", true)
	}
	return q
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.PengajuanTemplate, int64, error) {
	var list []model.PengajuanTemplate
	var total int64

	q := applyFilter(r.db.Model(&model.PengajuanTemplate{}), f)
	if p.Search != "" {
		q = q.Where("nama ILIKE ?", "%"+p.Search+"%")
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(q.Session(&gorm.Session{}).Preload("Pengunggah").Order("nama asc")).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.PengajuanTemplate, error) {
	var t model.PengajuanTemplate
	if err := r.db.Preload("Pengunggah").First(&t, id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repository) Create(t *model.PengajuanTemplate) error {
	return r.db.Create(t).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&model.PengajuanTemplate{}, id).Error
}
