package retur

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/barangstokgudang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/docnumber"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) Repository { return &repository{db: db} }

const forUpdate = "gorm:query_option"

// BizError = pelanggaran aturan bisnis (bukan kegagalan DB); pesannya aman
// ditampilkan ke operator. Error lain (DB dsb.) tidak boleh bocor ke client.
type BizError struct{ Msg string }

func (e *BizError) Error() string { return e.Msg }

func biz(msg string) error { return &BizError{Msg: msg} }

func bizf(format string, a ...interface{}) error { return &BizError{Msg: fmt.Sprintf(format, a...)} }

func preload(q *gorm.DB) *gorm.DB {
	return q.Preload("Barang").Preload("Gudang").Preload("Pelapor").Preload("Pemeriksa")
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.ReturPenggantian, int64, error) {
	var list []model.ReturPenggantian
	var total int64

	q := r.db.Model(&model.ReturPenggantian{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.BarangID != 0 {
		q = q.Where("barang_id = ?", f.BarangID)
	}
	if p.Search != "" {
		like := "%" + p.Search + "%"
		q = q.Where("nomor_retur ILIKE ? OR klien ILIKE ? OR serial_lama ILIKE ? OR serial_baru ILIKE ?", like, like, like, like)
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(preload(q.Session(&gorm.Session{})).Order("created_at desc")).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.ReturPenggantian, error) {
	var rt model.ReturPenggantian
	if err := preload(r.db).First(&rt, id).Error; err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *repository) CountByStatus(status string) (int64, error) {
	var n int64
	err := r.db.Model(&model.ReturPenggantian{}).Where("status = ?", status).Count(&n).Error
	return n, err
}

func (r *repository) Create(rt *model.ReturPenggantian) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var open int64
		if err := tx.Model(&model.ReturPenggantian{}).
			Where("serial_lama = ? AND status IN ?", rt.SerialLama,
				[]string{constant.StatusReturDilaporkan, constant.StatusReturPerluGanti}).
			Count(&open).Error; err != nil {
			return err
		}
		if open > 0 {
			return biz(constant.ErrReturSerialLamaDobel)
		}

		// Kalau unit lama sudah terdaftar di sistem, harus unit barang yang
		// sama dan sedang terpasang (di klien), bukan di gudang / sudah rusak.
		var lama model.BarangSerial
		err := tx.Where("serial_number = ?", rt.SerialLama).First(&lama).Error
		switch {
		case err == nil:
			if lama.BarangID != rt.BarangID {
				return biz(constant.ErrSerialBarangTidakSesuai)
			}
			if lama.Status != constant.StatusSerialTerpasang {
				return bizf("unit %s berstatus %q, bukan terpasang di klien", rt.SerialLama, lama.Status)
			}
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		nomor, err := docnumber.Next(tx, "RT")
		if err != nil {
			return err
		}
		rt.NomorRetur = nomor
		rt.Status = constant.StatusReturDilaporkan
		return tx.Create(rt).Error
	})
}

func (r *repository) Pengecekan(id uint, hasil, catatan string, userID uint) (*model.ReturPenggantian, error) {
	var rt model.ReturPenggantian
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set(forUpdate, "FOR UPDATE").First(&rt, id).Error; err != nil {
			return err
		}
		if rt.Status != constant.StatusReturDilaporkan {
			return biz(constant.ErrReturStatusTidakSesuai)
		}
		now := time.Now()
		rt.HasilPengecekan = hasil
		rt.CatatanTeknisi = catatan
		rt.DicekOleh = &userID
		rt.DicekPada = &now
		if hasil == constant.HasilReturPerluGanti {
			rt.Status = constant.StatusReturPerluGanti
		} else {
			rt.Status = constant.StatusReturTidakPerlu
		}
		return tx.Omit(clause.Associations).Save(&rt).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(rt.ID)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func (r *repository) Ganti(id uint, serialBaru string, userID uint) (*model.ReturPenggantian, error) {
	serialBaru = strings.TrimSpace(serialBaru)
	var rt model.ReturPenggantian

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set(forUpdate, "FOR UPDATE").First(&rt, id).Error; err != nil {
			return err
		}
		if rt.Status != constant.StatusReturPerluGanti {
			return biz(constant.ErrReturStatusTidakSesuai)
		}
		if strings.EqualFold(serialBaru, rt.SerialLama) {
			return biz("unit pengganti tidak boleh sama dengan unit yang rusak")
		}

		// 1) Unit pengganti: wajib Normal (tersedia), barang sama, di gudang retur.
		var baru model.BarangSerial
		if err := tx.Set(forUpdate, "FOR UPDATE").Where("serial_number = ?", serialBaru).First(&baru).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return bizf("%s: %s", constant.ErrSerialTidakDitemukan, serialBaru)
			}
			return err
		}
		if baru.BarangID != rt.BarangID {
			return bizf("%s: %s", constant.ErrSerialBarangTidakSesuai, serialBaru)
		}
		if baru.Status != constant.StatusSerialTersedia || baru.GudangID == nil || *baru.GudangID != rt.GudangID {
			return bizf("%s: %s", constant.ErrReturPenggantiBukanNormal, serialBaru)
		}
		catBaru := truncate(fmt.Sprintf("Pengganti %s di klien %s (%s)", rt.SerialLama, rt.Klien, rt.NomorRetur), 255)
		if err := tx.Model(&model.BarangSerial{}).Where("id = ?", baru.ID).Updates(map[string]interface{}{
			"status":    constant.StatusSerialTerpasang,
			"gudang_id": nil,
			"catatan":   catBaru,
		}).Error; err != nil {
			return err
		}
		if err := barangstokgudang.AdjustStokGudangTx(tx, rt.BarangID, rt.GudangID, -1); err != nil {
			return err
		}
		if err := barangstokgudang.SyncBarangStokTotalTx(tx, rt.BarangID); err != nil {
			return err
		}

		// 2) Unit lama: WAJIB ditarik ke gudang sebagai rusak. Status rusak
		// tidak menambah stok tersedia (barang_stok_gudang tidak disentuh).
		catLama := truncate(fmt.Sprintf("Ditarik dari klien %s, rusak, diganti %s (%s)", rt.Klien, serialBaru, rt.NomorRetur), 255)
		var lama model.BarangSerial
		err := tx.Set(forUpdate, "FOR UPDATE").Where("serial_number = ?", rt.SerialLama).First(&lama).Error
		switch {
		case err == nil:
			if lama.BarangID != rt.BarangID {
				return bizf("%s: %s", constant.ErrSerialBarangTidakSesuai, rt.SerialLama)
			}
			if err := tx.Model(&model.BarangSerial{}).Where("id = ?", lama.ID).Updates(map[string]interface{}{
				"status":    constant.StatusSerialRusak,
				"gudang_id": rt.GudangID,
				"catatan":   catLama,
			}).Error; err != nil {
				return err
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			// Unit lama belum pernah tercatat (dipasang sebelum sistem ini
			// dipakai) — daftarkan langsung sebagai unit rusak di gudang.
			gid := rt.GudangID
			if err := tx.Create(&model.BarangSerial{
				BarangID: rt.BarangID, SerialNumber: rt.SerialLama,
				Status: constant.StatusSerialRusak, GudangID: &gid, Catatan: catLama,
			}).Error; err != nil {
				return err
			}
		default:
			return err
		}

		// 3) Catat di Barang Rusak supaya muncul di alur & laporan yang ada.
		var brg model.Barang
		if err := tx.First(&brg, rt.BarangID).Error; err != nil {
			return err
		}
		bid := rt.BarangID
		now := time.Now()
		br := model.BarangRusak{
			BarangID:       &bid,
			LabelBarang:    truncate(rt.SerialLama, 60),
			NamaBarang:     brg.Nama,
			Merek:          brg.Merek,
			KodeBarang:     brg.KodeBarang,
			SerialNumber:   rt.SerialLama,
			Keterangan:     truncate(fmt.Sprintf("%s — klien %s: %s", rt.NomorRetur, rt.Klien, rt.Keluhan), 500),
			JenisBarang:    constant.StatusRusak,
			Status:         constant.StatusRusak,
			DilaporkanOleh: rt.DilaporkanOleh,
			DicekOleh:      rt.DicekOleh,
			DicekPada:      rt.DicekPada,
		}
		if err := tx.Create(&br).Error; err != nil {
			return err
		}

		// 4) Tutup retur.
		rt.SerialBaru = serialBaru
		rt.Status = constant.StatusReturSelesai
		rt.DigantiOleh = &userID
		rt.DigantiPada = &now
		rt.BarangRusakID = &br.ID
		return tx.Omit(clause.Associations).Save(&rt).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(rt.ID)
}
