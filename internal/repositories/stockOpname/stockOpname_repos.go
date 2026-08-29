package stock_opname

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/barangstokgudang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const queryIDEq = "id = ?"

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.Status != "" {
		q = q.Where(constant.QueryStatusEq, f.Status)
	}
	if f.GudangID != 0 {
		q = q.Where(constant.QueryGudangIDEq, f.GudangID)
	}
	return q
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.StockOpname, int64, error) {
	var list []model.StockOpname
	var total int64

	q := applyFilter(r.db.Model(&model.StockOpname{}), f)
	if p.Search != "" {
		q = q.Where("nomor_opname ILIKE ?", "%"+p.Search+"%")
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(q.Session(&gorm.Session{}).
		Preload("Gudang").Preload("Items").
		Preload("Items.Barang", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Order("id desc")).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	for i := range list {
		if err := refreshDraftStokSistem(r.db, &list[i]); err != nil {
			return nil, 0, err
		}
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.StockOpname, error) {
	var so model.StockOpname

	err := r.db.Preload("Gudang").Preload("Items").
		Preload("Items.Barang", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		First(&so, id).Error
	if err != nil {
		return nil, err
	}
	if err := refreshDraftStokSistem(r.db, &so); err != nil {
		return nil, err
	}
	return &so, nil
}

func (r *repository) FindByNomor(nomor string) (*model.StockOpname, error) {
	var so model.StockOpname
	if err := r.db.Where("nomor_opname = ?", nomor).First(&so).Error; err != nil {
		return nil, err
	}
	return &so, nil
}

func dedupeInputs(inputs []ItemInput) []ItemInput {
	order := make([]uint, 0, len(inputs))
	merged := make(map[uint]*ItemInput)
	for _, in := range inputs {
		if existing, ok := merged[in.BarangID]; ok {
			existing.StokFisik += in.StokFisik
			if in.Catatan != "" {
				if existing.Catatan != "" {
					existing.Catatan += "; " + in.Catatan
				} else {
					existing.Catatan = in.Catatan
				}
			}
			continue
		}
		cp := in
		merged[in.BarangID] = &cp
		order = append(order, in.BarangID)
	}
	out := make([]ItemInput, 0, len(order))
	for _, barangID := range order {
		out = append(out, *merged[barangID])
	}
	return out
}

func buildItems(tx *gorm.DB, soID uint, gudangID uint, inputs []ItemInput) ([]model.StockOpnameItem, error) {
	deduped := dedupeInputs(inputs)
	items := make([]model.StockOpnameItem, 0, len(deduped))
	for _, in := range deduped {
		if err := tx.First(&model.Barang{}, in.BarangID).Error; err != nil {
			return nil, err
		}
		// Stok sistem diambil LIVE saat baris dibuat/diubah. Untuk dokumen draft,
		// nilai ini akan disegarkan ulang lagi setiap kali dibaca (lihat
		// refreshDraftStokSistem) supaya tidak basi jika ada transaksi lain
		// (Barang Masuk/Keluar/opname lain) yang terjadi setelah draft dibuat.
		stokSistem, err := barangstokgudang.GetStokGudangTx(tx, in.BarangID, gudangID)
		if err != nil {
			return nil, err
		}
		item := model.StockOpnameItem{
			StockOpnameID: soID,
			BarangID:      in.BarangID,
			StokSistem:    stokSistem,
			StokFisik:     in.StokFisik,
			Catatan:       in.Catatan,
		}
		item.HitungSelisih()
		items = append(items, item)
	}
	return items, nil
}

func refreshDraftStokSistem(db *gorm.DB, so *model.StockOpname) error {
	if so == nil || so.Status != constant.StatusSODraft || len(so.Items) == 0 {
		return nil
	}
	for i := range so.Items {
		item := &so.Items[i]
		stokSistem, err := barangstokgudang.GetStokGudangTx(db, item.BarangID, so.GudangID)
		if err != nil {
			return err
		}
		item.StokSistem = stokSistem
		item.HitungSelisih()
	}
	return nil
}

func (r *repository) Create(so *model.StockOpname, inputs []ItemInput) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(so).Error; err != nil {
			return err
		}
		items, err := buildItems(tx, so.ID, so.GudangID, inputs)
		if err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		so.Items = items
		return nil
	})
}

func (r *repository) Update(so *model.StockOpname, inputs []ItemInput) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("stock_opname_id = ?", so.ID).Delete(&model.StockOpnameItem{}).Error; err != nil {
			return err
		}
		items, err := buildItems(tx, so.ID, so.GudangID, inputs)
		if err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return tx.Save(so).Error
	})
}

func (r *repository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("stock_opname_id = ?", id).Delete(&model.StockOpnameItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.StockOpname{}, id).Error
	})
}

func (r *repository) Complete(id uint, userID uint) (*model.StockOpname, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var so model.StockOpname
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Preload("Items").First(&so, id).Error; err != nil {
			return err
		}
		if so.Status != constant.StatusSODraft {
			return errors.New(constant.ErrSOBukanDraft)
		}

		for idx := range so.Items {
			item := &so.Items[idx]
			liveStokSistem, err := barangstokgudang.GetStokGudangTx(tx, item.BarangID, so.GudangID)
			if err != nil {
				return err
			}
			item.StokSistem = liveStokSistem
			item.HitungSelisih()
			if err := tx.Model(&model.StockOpnameItem{}).Where(queryIDEq, item.ID).
				Updates(map[string]interface{}{"stok_sistem": item.StokSistem, "selisih": item.Selisih}).Error; err != nil {
				return err
			}
			if item.Selisih == 0 {
				continue
			}
			if err := barangstokgudang.SetStokGudangTx(tx, item.BarangID, so.GudangID, item.StokFisik); err != nil {
				return err
			}
			if err := barangstokgudang.SyncBarangStokTotalTx(tx, item.BarangID); err != nil {
				return err
			}
		}

		now := time.Now()
		return tx.Model(&model.StockOpname{}).Where(queryIDEq, id).Updates(map[string]interface{}{
			"status":         constant.StatusSOSelesai,
			"dilakukan_oleh": userID,
			"completed_at":   now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(id)
}

func (r *repository) Batalkan(id uint) (*model.StockOpname, error) {
	res := r.db.Model(&model.StockOpname{}).Where("id = ? AND status = ?", id, constant.StatusSODraft).
		Update("status", constant.StatusSODibatalkan)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errors.New(constant.ErrSOBukanDraft)
	}
	return r.FindByID(id)
}

func (r *repository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.StockOpname{}).Where(constant.QueryStatusEq, status).Count(&count).Error
	return count, err
}
