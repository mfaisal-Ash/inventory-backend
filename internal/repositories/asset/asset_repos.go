package asset

import (
	"gorm.io/gorm"

	"github.com/inventory-backend/internal/model"
	"github.com/inventory-backend/pkg/utils"
)

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.JenisAset != "" {
		q = q.Where("jenis_aset = ?", f.JenisAset)
	}
	if f.GudangID != 0 {
		q = q.Where("gudang_id = ?", f.GudangID)
	}
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	return q
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.Asset, int64, error) {
	var list []model.Asset
	var total int64

	q := applyFilter(r.db.Model(&model.Asset{}), f)
	if p.Search != "" {
		q = q.Where("nama ILIKE ? OR label_rsd ILIKE ? OR kode_ba ILIKE ?",
			"%"+p.Search+"%", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(q.Session(&gorm.Session{}).Preload("Gudang").Order("created_at desc")).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *repository) ListForMap(f Filter, tipeGudang string) ([]MapRow, error) {
	var rows []MapRow

	q := r.db.Table("assets a").
		Select(`a.id, a.nama, a.jenis_aset, a.label_rsd, a.latitude, a.longitude, a.status,
			a.ip_address, a.ping_status, a.jumlah_port,
			g.id AS gudang_id, g.nama AS gudang_nama, g.kode AS gudang_kode, g.tipe AS gudang_tipe,
			g.latitude AS gudang_latitude, g.longitude AS gudang_longitude,
			a.parent_asset_id,
			pa.latitude AS parent_latitude, pa.longitude AS parent_longitude,
			COALESCE(pc.terisi, 0) AS port_terisi`).
		Joins("JOIN gudangs g ON g.id = a.gudang_id").
		Joins("LEFT JOIN assets pa ON pa.id = a.parent_asset_id AND pa.deleted_at IS NULL").
		Joins(`LEFT JOIN (
			SELECT asset_id, COUNT(*) AS terisi FROM asset_ports
			WHERE status = 'terisi' AND deleted_at IS NULL
			GROUP BY asset_id
		) pc ON pc.asset_id = a.id`).
		Where("a.latitude IS NOT NULL AND a.longitude IS NOT NULL")

	if f.JenisAset != "" {
		q = q.Where("a.jenis_aset = ?", f.JenisAset)
	}
	if f.GudangID != 0 {
		q = q.Where("a.gudang_id = ?", f.GudangID)
	}
	if f.Status != "" {
		q = q.Where("a.status = ?", f.Status)
	}
	if tipeGudang != "" {
		q = q.Where("g.tipe = ?", tipeGudang)
	}

	if err := q.Order("g.kode asc, a.nama asc").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *repository) FindByID(id uint) (*model.Asset, error) {
	var a model.Asset
	if err := r.db.Preload("Gudang").First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *repository) Create(a *model.Asset) error {
	return r.db.Create(a).Error
}

func (r *repository) Update(a *model.Asset) error {
	return r.db.Save(a).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&model.Asset{}, id).Error
}

func (r *repository) NextRSDNumber(gudangID uint) (int, error) {
	var count int64
	err := r.db.Model(&model.Asset{}).
		Where("gudang_id = ? AND label_rsd <> ''", gudangID).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count) + 1, nil
}

func (r *repository) NextBANumber() (int, error) {
	var count int64
	err := r.db.Model(&model.Asset{}).
		Where("kode_ba <> ''").
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count) + 1, nil
}

func (r *repository) CountByJenis(jenisAset string) (int64, error) {
	var count int64
	err := r.db.Model(&model.Asset{}).Where("jenis_aset = ?", jenisAset).Count(&count).Error
	return count, err
}
