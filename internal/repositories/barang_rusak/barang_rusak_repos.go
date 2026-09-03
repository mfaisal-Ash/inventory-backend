package barang_rusak

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/barangstokgudang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.BarangID != 0 {
		q = q.Where("barang_id = ?", f.BarangID)
	}
	return q
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.BarangRusak, int64, error) {
	var list []model.BarangRusak
	var total int64

	q := applyFilter(r.db.Model(&model.BarangRusak{}), f)
	if p.Search != "" {
		q = q.Where("label_barang ILIKE ? OR nama_barang ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(q.Session(&gorm.Session{}).Preload("Barang").Preload("Pelapor").Preload("Pemeriksa").Order("created_at desc")).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.BarangRusak, error) {
	var b model.BarangRusak
	if err := r.db.Preload("Barang").Preload("Pelapor").Preload("Pemeriksa").First(&b, id).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *repository) Create(b *model.BarangRusak) error {
	return r.db.Create(b).Error
}

// Update: Omit(clause.Associations) mencegah Save() menimpa balik kolom FK
// (barang_id, pelapor_id, pemeriksa_id) dengan ID dari relasi Barang/Pelapor/
// Pemeriksa yang ter-preload di FindByID (bug sama seperti repositories/
// barang — lihat komentar di sana).
func (r *repository) Update(b *model.BarangRusak) error {
	return r.db.Omit(clause.Associations).Save(b).Error
}

func (r *repository) Delete(id uint) error {
	return r.db.Delete(&model.BarangRusak{}, id).Error
}

func (r *repository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.BarangRusak{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *repository) SimpanKeGudang(id uint, gudangID uint) (*model.BarangRusak, error) {
	var b model.BarangRusak
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// FOR UPDATE: kunci baris ini supaya tidak ada dua request bersamaan
		// yang sama-sama lolos pengecekan status dan menambahkan stok dobel.
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&b, id).Error; err != nil {
			return err
		}
		if b.Status != constant.StatusRetur {
			return errors.New("hanya barang berstatus \"Bisa Diretur\" yang bisa disimpan ke gudang")
		}
		if b.BarangID == nil {
			return errors.New("barang ini tidak tertaut ke katalog Kelola Barang, jadi stoknya tidak bisa ditambahkan kembali secara otomatis — tautkan dulu lewat menu Ubah")
		}
		if err := barangstokgudang.AdjustStokGudangTx(tx, *b.BarangID, gudangID, 1); err != nil {
			return err
		}
		if err := barangstokgudang.SyncBarangStokTotalTx(tx, *b.BarangID); err != nil {
			return err
		}
		b.Status = constant.StatusDisimpanGudang
		return tx.Omit(clause.Associations).Save(&b).Error
	})
	if err != nil {
		return nil, err
	}
	return &b, nil
}
