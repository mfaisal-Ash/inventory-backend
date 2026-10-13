package kabel

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

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

// BizError = pelanggaran aturan bisnis; pesannya aman ditampilkan ke operator.
type BizError struct{ Msg string }

func (e *BizError) Error() string { return e.Msg }

func biz(msg string) error { return &BizError{Msg: msg} }

func bizf(format string, a ...interface{}) error {
	return &BizError{Msg: fmt.Sprintf(format, a...)}
}

// sqlTerpakai = total meter pemakaian aktif (tidak dibatalkan) sebuah gulungan.
const sqlTerpakai = `COALESCE((SELECT SUM(kp.jumlah) FROM kabel_pemakaian kp
	WHERE kp.gulungan_id = kabel_gulungan.id AND kp.dibatalkan = false), 0)`

func preloadDetail(q *gorm.DB) *gorm.DB {
	return q.Preload("Barang").Preload("Barang.Satuan").Preload("Gudang").Preload("Pembuat").
		Preload("Pemakaian", func(db *gorm.DB) *gorm.DB {
			return db.Order("tanggal DESC, id DESC").Preload("Petugas")
		})
}

func preloadList(q *gorm.DB) *gorm.DB {
	return q.Preload("Barang").Preload("Barang.Satuan").Preload("Gudang").Preload("Pembuat")
}

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.GudangID != 0 {
		q = q.Where("kabel_gulungan.gudang_id = ?", f.GudangID)
	}
	if f.BarangID != 0 {
		q = q.Where("kabel_gulungan.barang_id = ?", f.BarangID)
	}
	switch f.Status {
	case constant.StatusKabelAktif:
		q = q.Where("kabel_gulungan.panjang_awal > " + sqlTerpakai)
	case constant.StatusKabelHabis:
		q = q.Where("kabel_gulungan.panjang_awal <= " + sqlTerpakai)
	}
	return q
}

// terpakaiMap menghitung total pemakaian aktif untuk banyak gulungan sekaligus.
func terpakaiMap(db *gorm.DB, ids []uint) (map[uint]int, error) {
	out := make(map[uint]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		GulunganID uint
		Total      int
	}
	if err := db.Model(&model.KabelPemakaian{}).
		Select("gulungan_id, COALESCE(SUM(jumlah), 0) AS total").
		Where("gulungan_id IN ? AND dibatalkan = false", ids).
		Group("gulungan_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.GulunganID] = r.Total
	}
	return out, nil
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.KabelGulungan, int64, error) {
	var list []model.KabelGulungan
	var total int64

	q := applyFilter(r.db.Model(&model.KabelGulungan{}), f)
	if s := strings.TrimSpace(p.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("kabel_gulungan.kode_gulungan ILIKE ? OR kabel_gulungan.barang_id IN (SELECT id FROM barang WHERE nama ILIKE ? OR kode_barang ILIKE ?)", like, like, like)
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(preloadList(q.Session(&gorm.Session{})).Order("kabel_gulungan.created_at DESC, kabel_gulungan.id DESC")).Find(&list).Error; err != nil {
		return nil, 0, err
	}

	ids := make([]uint, len(list))
	for i := range list {
		ids[i] = list[i].ID
	}
	tm, err := terpakaiMap(r.db, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range list {
		list[i].HitungTurunan(tm[list[i].ID])
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.KabelGulungan, error) {
	var g model.KabelGulungan
	if err := preloadDetail(r.db).First(&g, id).Error; err != nil {
		return nil, err
	}
	terpakai := 0
	for _, pm := range g.Pemakaian {
		if !pm.Dibatalkan {
			terpakai += pm.Jumlah
		}
	}
	g.HitungTurunan(terpakai)
	return &g, nil
}

func (r *repository) Summary(f Filter) (*Summary, error) {
	var s Summary
	row := struct {
		TotalGulungan int64
		TotalPanjang  int64
		TotalTerpakai int64
		Aktif         int64
	}{}
	q := applyFilter(r.db.Model(&model.KabelGulungan{}), Filter{GudangID: f.GudangID, BarangID: f.BarangID})
	if err := q.Select(`COUNT(*) AS total_gulungan,
		COALESCE(SUM(kabel_gulungan.panjang_awal), 0) AS total_panjang,
		COALESCE(SUM(` + sqlTerpakai + `), 0) AS total_terpakai,
		COALESCE(SUM(CASE WHEN kabel_gulungan.panjang_awal > ` + sqlTerpakai + ` THEN 1 ELSE 0 END), 0) AS aktif`).
		Scan(&row).Error; err != nil {
		return nil, err
	}
	s.TotalGulungan = row.TotalGulungan
	s.TotalPanjang = row.TotalPanjang
	s.TotalTerpakai = row.TotalTerpakai
	s.GulunganAktif = row.Aktif
	s.GulunganHabis = row.TotalGulungan - row.Aktif
	s.TotalSisa = row.TotalPanjang - row.TotalTerpakai
	if s.TotalSisa < 0 {
		s.TotalSisa = 0
	}
	return &s, nil
}

// cekBarangKabel memastikan barang ada dan berstatuan meter.
func cekBarangKabel(tx *gorm.DB, barangID uint) error {
	var b model.Barang
	if err := tx.Preload("Satuan").First(&b, barangID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return biz("barang tidak ditemukan")
		}
		return err
	}
	if b.Satuan == nil {
		return biz(constant.ErrKabelBukanMeter)
	}
	nama := strings.ToLower(strings.TrimSpace(b.Satuan.Nama))
	sing := strings.ToLower(strings.TrimSpace(b.Satuan.Singkatan))
	switch {
	case nama == "meter", nama == "m", sing == "m", sing == "mtr":
		return nil
	}
	return biz(constant.ErrKabelBukanMeter)
}

// sisaAktifGudang = total sisa semua gulungan (barang+gudang) yang belum habis.
func sisaAktifGudang(tx *gorm.DB, barangID, gudangID uint) (int, error) {
	var sisa int
	err := tx.Model(&model.KabelGulungan{}).
		Where("barang_id = ? AND gudang_id = ?", barangID, gudangID).
		Select("COALESCE(SUM(kabel_gulungan.panjang_awal - " + sqlTerpakai + "), 0)").
		Scan(&sisa).Error
	return sisa, err
}

func (r *repository) Create(g *model.KabelGulungan) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := cekBarangKabel(tx, g.BarangID); err != nil {
			return err
		}
		var gd model.Gudang
		if err := tx.First(&gd, g.GudangID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return biz("gudang tidak ditemukan")
			}
			return err
		}

		stok, err := barangstokgudang.GetStokGudangTx(tx, g.BarangID, g.GudangID)
		if err != nil {
			return err
		}
		sisa, err := sisaAktifGudang(tx, g.BarangID, g.GudangID)
		if err != nil {
			return err
		}
		if sisa+g.PanjangAwal > stok {
			return bizf("%s (stok %d m, sudah tercatat di gulungan %d m, gulungan baru %d m)",
				constant.ErrKabelMelebihiStok, stok, sisa, g.PanjangAwal)
		}

		kode, err := docnumber.Next(tx, "KBL")
		if err != nil {
			return err
		}
		g.KodeGulungan = kode
		return tx.Omit("Barang", "Gudang", "Pembuat", "Pemakaian").Create(g).Error
	})
}

func (r *repository) Pakai(gulunganID uint, in PemakaianInput) (*model.KabelGulungan, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var g model.KabelGulungan
		if err := tx.Set(forUpdate, "FOR UPDATE").First(&g, gulunganID).Error; err != nil {
			return err
		}
		var terpakai int
		if err := tx.Model(&model.KabelPemakaian{}).
			Where("gulungan_id = ? AND dibatalkan = false", g.ID).
			Select("COALESCE(SUM(jumlah), 0)").Scan(&terpakai).Error; err != nil {
			return err
		}
		if in.Jumlah > g.PanjangAwal-terpakai {
			return bizf("%s (sisa %d m, diminta %d m)", constant.ErrKabelSisaKurang, g.PanjangAwal-terpakai, in.Jumlah)
		}

		stok, err := barangstokgudang.GetStokGudangTx(tx, g.BarangID, g.GudangID)
		if err != nil {
			return err
		}
		if in.Jumlah > stok {
			return bizf("%s (stok %d m, diminta %d m)", constant.ErrKabelStokKurang, stok, in.Jumlah)
		}
		if err := barangstokgudang.AdjustStokGudangTx(tx, g.BarangID, g.GudangID, -in.Jumlah); err != nil {
			return err
		}
		if err := barangstokgudang.SyncBarangStokTotalTx(tx, g.BarangID); err != nil {
			return err
		}
		return tx.Create(&model.KabelPemakaian{
			GulunganID:  g.ID,
			Jumlah:      in.Jumlah,
			Tanggal:     in.Tanggal,
			Klien:       in.Klien,
			Keperluan:   in.Keperluan,
			DicatatOleh: in.UserID,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(gulunganID)
}

func (r *repository) BatalPakai(gulunganID, pemakaianID, userID uint) (*model.KabelGulungan, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var g model.KabelGulungan
		if err := tx.Set(forUpdate, "FOR UPDATE").First(&g, gulunganID).Error; err != nil {
			return err
		}
		var pm model.KabelPemakaian
		if err := tx.Set(forUpdate, "FOR UPDATE").
			Where("id = ? AND gulungan_id = ?", pemakaianID, g.ID).First(&pm).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return biz(constant.ErrKabelPemakaianTidakAda)
			}
			return err
		}
		if pm.Dibatalkan {
			return biz(constant.ErrKabelSudahDibatalkan)
		}
		now := time.Now()
		if err := tx.Model(&model.KabelPemakaian{}).Where("id = ?", pm.ID).Updates(map[string]interface{}{
			"dibatalkan":      true,
			"dibatalkan_oleh": userID,
			"dibatalkan_pada": now,
		}).Error; err != nil {
			return err
		}
		if err := barangstokgudang.AdjustStokGudangTx(tx, g.BarangID, g.GudangID, pm.Jumlah); err != nil {
			return err
		}
		return barangstokgudang.SyncBarangStokTotalTx(tx, g.BarangID)
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(gulunganID)
}

func (r *repository) Hapus(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var g model.KabelGulungan
		if err := tx.Set(forUpdate, "FOR UPDATE").First(&g, id).Error; err != nil {
			return err
		}
		var n int64
		if err := tx.Model(&model.KabelPemakaian{}).
			Where("gulungan_id = ? AND dibatalkan = false", g.ID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return biz(constant.ErrKabelMasihAdaPemakaian)
		}
		return tx.Delete(&g).Error
	})
}
