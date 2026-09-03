package cod

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	return q
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.CodTransaction, int64, error) {
	var list []model.CodTransaction
	var total int64

	q := applyFilter(r.db.Model(&model.CodTransaction{}), f)
	if p.Search != "" {
		q = q.Where("kode ILIKE ? OR pelanggan ILIKE ? OR kurir ILIKE ?",
			"%"+p.Search+"%", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(q.Session(&gorm.Session{}).Preload("Pengiriman").Order("tanggal desc, id desc")).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.CodTransaction, error) {
	var c model.CodTransaction
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) FindByKode(kode string) (*model.CodTransaction, error) {
	var c model.CodTransaction
	if err := r.db.Where("kode = ?", kode).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) Create(c *model.CodTransaction) error {
	return r.db.Create(c).Error
}

// Update: Omit(clause.Associations) sebagai pencegahan tambahan — FindByID
// di sini tidak Preload("Pengiriman") jadi saat ini belum aktif kena bug,
// tapi List() sudah Preload("Pengiriman") dan pola yang sama di modul lain
// (barang, aset, tugas, dll) terbukti berbahaya begitu ada Preload di jalur
// yang dipakai sebelum Save(). Ditambahkan supaya aman ke depannya.
func (r *repository) Update(c *model.CodTransaction) error {
	return r.db.Omit(clause.Associations).Save(c).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&model.CodTransaction{}, id).Error
}

func (r *repository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.CodTransaction{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *repository) SumNominal() (int64, error) {
	var total int64
	err := r.db.Model(&model.CodTransaction{}).Select("COALESCE(SUM(nominal), 0)").Scan(&total).Error
	return total, err
}
