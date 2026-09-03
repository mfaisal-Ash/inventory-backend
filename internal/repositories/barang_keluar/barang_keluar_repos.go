package barang_keluar

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	barangSerial "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/barangstokgudang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/docnumber"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.Status != "" {
		q = q.Where(constant.QueryStatusEq, f.Status)
	}
	if f.GudangID != 0 {
		q = q.Where(constant.QueryGudangIDEq, f.GudangID)
	}
	if f.KategoriID != 0 || f.BarangID != 0 || f.Merek != "" || f.Tipe != "" {
		q = q.Select("barang_keluar.*").Distinct().
			Joins("JOIN barang_keluar_items ON barang_keluar_items.barang_keluar_id = barang_keluar.id").
			Joins("JOIN barang ON barang.id = barang_keluar_items.barang_id")
		if f.KategoriID != 0 {
			q = q.Where("barang.kategori_id = ?", f.KategoriID)
		}
		if f.BarangID != 0 {
			q = q.Where("barang.id = ?", f.BarangID)
		}
		if f.Merek != "" {
			q = q.Where("barang.merek = ?", f.Merek)
		}
		if f.Tipe != "" {
			q = q.Where("barang.tipe = ?", f.Tipe)
		}
	}
	return q
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.BarangKeluar, int64, error) {
	var list []model.BarangKeluar
	var total int64

	q := applyFilter(r.db.Model(&model.BarangKeluar{}), f)
	if p.Search != "" {
		q = q.Where("nomor_pengeluaran ILIKE ?", "%"+p.Search+"%")
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
		list[i].HitungSisaItems()
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.BarangKeluar, error) {
	var bk model.BarangKeluar
	err := r.db.Preload("Gudang").Preload("Items").
		Preload("Items.Barang", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		First(&bk, id).Error
	if err != nil {
		return nil, err
	}
	bk.HitungSisaItems()
	return &bk, nil
}

func (r *repository) FindByNomor(nomor string) (*model.BarangKeluar, error) {
	var bk model.BarangKeluar
	if err := r.db.Where("nomor_pengeluaran = ?", nomor).First(&bk).Error; err != nil {
		return nil, err
	}
	return &bk, nil
}

func (r *repository) Create(bk *model.BarangKeluar) error {
	return r.db.Create(bk).Error
}

func (r *repository) Update(bk *model.BarangKeluar, items []model.BarangKeluarItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("barang_keluar_id = ?", bk.ID).Delete(&model.BarangKeluarItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ID = 0
			items[i].BarangKeluarID = bk.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		// Omit(clause.Associations): cegah Save() menimpa balik gudang_id
		// dengan ID dari relasi Gudang yang ter-preload di FindByID (bug sama
		// seperti repositories/barang — lihat komentar di sana).
		return tx.Omit(clause.Associations).Save(bk).Error
	})
}

func (r *repository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("barang_keluar_id = ?", id).Delete(&model.BarangKeluarItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.BarangKeluar{}, id).Error
	})
}

// SetProtected hanya menulis kolom is_protected saja (bukan Save() atas
// struct penuh) supaya tidak menyentuh/menimpa items ataupun kolom lain.
func (r *repository) SetProtected(id uint, protect bool) error {
	return r.db.Model(&model.BarangKeluar{}).Where("id = ?", id).Update("is_protected", protect).Error
}

func (r *repository) Complete(id uint, userID uint, serials map[uint][]string) (*model.BarangKeluar, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var bk model.BarangKeluar
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Preload("Items").First(&bk, id).Error; err != nil {
			return err
		}
		if bk.Status != constant.StatusBKDraft {
			return errors.New(constant.ErrBKBukanDraft)
		}

		barangByID := make(map[uint]model.Barang, len(bk.Items))
		for _, item := range bk.Items {
			var b model.Barang
			if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&b, item.BarangID).Error; err != nil {
				return err
			}
			barangByID[item.BarangID] = b

			stokGudang, err := barangstokgudang.GetStokGudangTx(tx, item.BarangID, bk.GudangID)
			if err != nil {
				return err
			}
			if stokGudang < item.Qty {
				return fmt.Errorf("%s (barang: %s, tersedia di gudang ini: %d, diminta: %d)",
					constant.ErrBKStokTidakCukup, b.Nama, stokGudang, item.Qty)
			}

			if b.IsSerialized {
				sn := serials[item.ID]
				if len(sn) != item.Qty {
					return fmt.Errorf("%s (barang: %s, qty: %d, sn dipilih: %d)",
						constant.ErrSerialJumlahTidakSesuai, b.Nama, item.Qty, len(sn))
				}
			}
		}

		for _, item := range bk.Items {
			if barangByID[item.BarangID].IsSerialized {
				if err := barangSerial.ConsumeUnitsTx(tx, item.BarangID, bk.GudangID, item.ID, serials[item.ID]); err != nil {
					return err
				}
			}
			if err := tx.Model(&model.Barang{}).Where("id = ?", item.BarangID).
				Update("stok", gorm.Expr("stok - ?", item.Qty)).Error; err != nil {
				return err
			}

			if err := barangstokgudang.AdjustStokGudangTx(tx, item.BarangID, bk.GudangID, -item.Qty); err != nil {
				return err
			}
		}

		now := time.Now()
		return tx.Model(&model.BarangKeluar{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":           constant.StatusBKSelesai,
			"dikeluarkan_oleh": userID,
			"completed_at":     now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(id)
}

func (r *repository) Batalkan(id uint) (*model.BarangKeluar, error) {
	res := r.db.Model(&model.BarangKeluar{}).Where("id = ? AND status = ?", id, constant.StatusBKDraft).
		Update("status", constant.StatusBKDibatalkan)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errors.New(constant.ErrBKBukanDraft)
	}
	return r.FindByID(id)
}

func (r *repository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.BarangKeluar{}).Where(constant.QueryStatusEq, status).Count(&count).Error
	return count, err
}

func (r *repository) NextNomor() (string, error) {
	return docnumber.Next(r.db, "BK")
}

func (r *repository) UpdateSpesifikasi(itemID uint, jumlahTerpasang int, catatan string) (*model.BarangKeluarItem, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var item model.BarangKeluarItem
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&item, itemID).Error; err != nil {
			return err
		}
		var bk model.BarangKeluar
		if err := tx.First(&bk, item.BarangKeluarID).Error; err != nil {
			return err
		}
		if bk.Status != constant.StatusBKSelesai {
			return errors.New(constant.ErrBKBelumSelesai)
		}
		if jumlahTerpasang < 0 || jumlahTerpasang > item.Qty {
			return errors.New(constant.ErrBKJumlahTerpasangLebih)
		}
		return tx.Model(&model.BarangKeluarItem{}).Where("id = ?", itemID).
			Updates(map[string]interface{}{
				"jumlah_terpasang":    jumlahTerpasang,
				"catatan_spesifikasi": catatan,
			}).Error
	})
	if err != nil {
		return nil, err
	}

	var result model.BarangKeluarItem
	if err := r.db.Preload("Barang", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		First(&result, itemID).Error; err != nil {
		return nil, err
	}
	result.HitungSisa()
	return &result, nil
}

func applySpesifikasiFilter(q *gorm.DB, f SpesifikasiFilter) *gorm.DB {
	if f.GudangID != 0 {
		q = q.Where("bk.gudang_id = ?", f.GudangID)
	}
	if f.From != nil {
		q = q.Where("bk.tanggal >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("bk.tanggal <= ?", *f.To)
	}
	return q
}

func (r *repository) RecapSpesifikasi(f SpesifikasiFilter) ([]SpesifikasiRecapRow, error) {
	q := applySpesifikasiFilter(
		r.db.Table("barang_keluar_items bki").
			Joins("JOIN barang_keluar bk ON bk.id = bki.barang_keluar_id").
			Joins("JOIN barang b ON b.id = bki.barang_id").
			Joins("LEFT JOIN satuan s ON s.id = b.satuan_id").
			Where("bk.status = ?", constant.StatusBKSelesai),
		f,
	)

	var rows []SpesifikasiRecapRow
	err := q.Select(`bki.barang_id AS barang_id,
		b.nama AS nama_barang,
		b.kode_barang AS kode_barang,
		COALESCE(s.singkatan, s.nama, '') AS satuan,
		SUM(bki.qty) AS total_terpakai,
		SUM(bki.jumlah_terpasang) AS total_terpasang,
		SUM(bki.qty - bki.jumlah_terpasang) AS total_sisa`).
		Group("bki.barang_id, b.nama, b.kode_barang, s.singkatan, s.nama").
		Order("b.nama ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func applySpesifikasiListFilter(q *gorm.DB, f SpesifikasiListFilter) *gorm.DB {
	if f.BarangID != 0 {
		q = q.Where("bki.barang_id = ?", f.BarangID)
	}
	if f.GudangID != 0 {
		q = q.Where("bk.gudang_id = ?", f.GudangID)
	}
	switch f.Status {
	case "belum":
		q = q.Where("bki.jumlah_terpasang = 0")
	case "sebagian":
		q = q.Where("bki.jumlah_terpasang > 0 AND bki.jumlah_terpasang < bki.qty")
	case "selesai":
		q = q.Where("bki.jumlah_terpasang >= bki.qty")
	}
	return q
}

func (r *repository) baseSpesifikasiListQuery(f SpesifikasiListFilter) *gorm.DB {
	return applySpesifikasiListFilter(
		r.db.Table("barang_keluar_items bki").
			Joins("JOIN barang_keluar bk ON bk.id = bki.barang_keluar_id").
			Joins("JOIN barang b ON b.id = bki.barang_id").
			Joins("LEFT JOIN satuan s ON s.id = b.satuan_id").
			Joins("LEFT JOIN gudangs g ON g.id = bk.gudang_id").
			Where("bk.status = ?", constant.StatusBKSelesai),
		f,
	)
}

// ListSpesifikasi mengembalikan daftar baris item barang keluar (dokumen
// induknya sudah selesai) satu per satu — mirip pola List() di
// repositories/barang_serial: filter + pencarian + paginasi server-side.
func (r *repository) ListSpesifikasi(p utils.PaginationParams, f SpesifikasiListFilter) ([]SpesifikasiListRow, int64, error) {
	countQ := r.baseSpesifikasiListQuery(f)
	if p.Search != "" {
		countQ = countQ.Where("b.nama ILIKE ? OR bk.nomor_pengeluaran ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	var total int64
	if err := countQ.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	listQ := r.baseSpesifikasiListQuery(f)
	if p.Search != "" {
		listQ = listQ.Where("b.nama ILIKE ? OR bk.nomor_pengeluaran ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	var rows []SpesifikasiListRow
	err := p.Apply(listQ.Session(&gorm.Session{}).Select(`bki.id AS item_id,
		bki.barang_keluar_id AS barang_keluar_id,
		bk.nomor_pengeluaran AS nomor_pengeluaran,
		bk.tanggal AS tanggal,
		bk.keperluan AS keperluan,
		bk.gudang_id AS gudang_id,
		COALESCE(g.nama, '') AS nama_gudang,
		bki.barang_id AS barang_id,
		b.nama AS nama_barang,
		b.kode_barang AS kode_barang,
		COALESCE(s.singkatan, s.nama, '') AS satuan,
		bki.qty AS qty,
		bki.jumlah_terpasang AS jumlah_terpasang,
		GREATEST(bki.qty - bki.jumlah_terpasang, 0) AS jumlah_sisa,
		bki.catatan_spesifikasi AS catatan_spesifikasi`).
		Order("bk.tanggal DESC, bki.id DESC")).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
