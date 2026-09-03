package barangstokgudang

import (
	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func AdjustStokGudangTx(tx *gorm.DB, barangID, gudangID uint, delta int) error {
	var row model.BarangStokGudang
	err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("barang_id = ? AND gudang_id = ?", barangID, gudangID).
		First(&row).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}

		stok := delta
		if stok < 0 {
			stok = 0
		}
		return tx.Create(&model.BarangStokGudang{BarangID: barangID, GudangID: gudangID, Stok: stok}).Error
	}
	newStok := row.Stok + delta
	if newStok < 0 {
		newStok = 0
	}
	return tx.Model(&model.BarangStokGudang{}).Where("id = ?", row.ID).Update("stok", newStok).Error
}

func GetStokGudangTx(tx *gorm.DB, barangID, gudangID uint) (int, error) {
	var row model.BarangStokGudang
	err := tx.Set("gorm:query_option", "FOR UPDATE").
		Where("barang_id = ? AND gudang_id = ?", barangID, gudangID).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return row.Stok, nil
}

func SetStokGudangTx(tx *gorm.DB, barangID, gudangID uint, stok int) error {
	if stok < 0 {
		stok = 0
	}
	res := tx.Model(&model.BarangStokGudang{}).
		Where("barang_id = ? AND gudang_id = ?", barangID, gudangID).
		Update("stok", stok)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return tx.Create(&model.BarangStokGudang{BarangID: barangID, GudangID: gudangID, Stok: stok}).Error
	}
	return nil
}

func SyncBarangStokTotalTx(tx *gorm.DB, barangID uint) error {
	var total int64
	if err := tx.Model(&model.BarangStokGudang{}).
		Where("barang_id = ?", barangID).
		Select("COALESCE(SUM(stok), 0)").Scan(&total).Error; err != nil {
		return err
	}
	return tx.Model(&model.Barang{}).Where("id = ?", barangID).Update("stok", total).Error
}

// SumMasukKeluar menghitung total qty barang masuk & keluar (all-time, cuma
// dari dokumen yang sudah SELESAI) untuk 1 barang — dihitung fresh setiap
// dipanggil (bukan nilai yang di-cache) supaya selalu real-time. Dipakai
// panel detail "spesifikasi real-time" di Kelola Barang.
func SumMasukKeluar(db *gorm.DB, barangID uint) (masuk int64, keluar int64, err error) {
	if err = db.Table("barang_masuk_items bmi").
		Joins("JOIN barang_masuk bm ON bm.id = bmi.barang_masuk_id").
		Where("bmi.barang_id = ? AND bm.status = ?", barangID, constant.StatusBMSelesai).
		Select("COALESCE(SUM(bmi.qty), 0)").Scan(&masuk).Error; err != nil {
		return 0, 0, err
	}
	if err = db.Table("barang_keluar_items bki").
		Joins("JOIN barang_keluar bk ON bk.id = bki.barang_keluar_id").
		Where("bki.barang_id = ? AND bk.status = ?", barangID, constant.StatusBKSelesai).
		Select("COALESCE(SUM(bki.qty), 0)").Scan(&keluar).Error; err != nil {
		return 0, 0, err
	}
	return masuk, keluar, nil
}

type StokRow struct {
	BarangID   uint   `json:"barang_id"`
	KodeBarang string `json:"kode_barang"`
	NamaBarang string `json:"nama_barang"`
	Merek      string `json:"merek"`
	Tipe       string `json:"tipe"`
	GudangID   uint   `json:"gudang_id"`
	NamaGudang string `json:"nama_gudang"`
	Stok       int    `json:"stok"`
}

func ListAll(db *gorm.DB) ([]StokRow, error) {
	var rows []StokRow
	err := db.Table("barang_stok_gudang bsg").
		Select("bsg.barang_id, b.kode_barang, b.nama AS nama_barang, b.merek, b.tipe, bsg.gudang_id, g.nama AS nama_gudang, bsg.stok").
		Joins("JOIN barang b ON b.id = bsg.barang_id").
		Joins("JOIN gudangs g ON g.id = bsg.gudang_id").
		Where("bsg.stok > 0").
		Order("b.nama, g.nama").
		Scan(&rows).Error
	return rows, err
}

func ListByGudang(db *gorm.DB, gudangID uint) ([]StokRow, error) {
	var rows []StokRow
	err := db.Table("barang_stok_gudang bsg").
		Select("bsg.barang_id, b.kode_barang, b.nama AS nama_barang, bsg.gudang_id, bsg.stok").
		Joins("JOIN barang b ON b.id = bsg.barang_id").
		Where("bsg.gudang_id = ? AND bsg.stok > 0", gudangID).
		Order("b.nama").
		Scan(&rows).Error
	return rows, err
}

func ListByBarang(db *gorm.DB, barangID uint) ([]StokRow, error) {
	var rows []StokRow
	err := db.Table("barang_stok_gudang bsg").
		Select("bsg.barang_id, bsg.gudang_id, g.nama AS nama_gudang, bsg.stok").
		Joins("JOIN gudangs g ON g.id = bsg.gudang_id").
		Where("bsg.barang_id = ? AND bsg.stok > 0", barangID).
		Order("g.nama").
		Scan(&rows).Error
	return rows, err
}
