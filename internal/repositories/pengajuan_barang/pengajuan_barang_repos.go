package pengajuan_barang

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/barangstokgudang"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/docnumber"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func applyFilter(q *gorm.DB, f Filter) *gorm.DB {
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.GudangID != 0 {
		q = q.Where("gudang_id = ?", f.GudangID)
	}
	if f.Jenis != "" {
		q = q.Where("jenis = ?", f.Jenis)
	}
	if f.BarangID != 0 {
		q = q.Select("pengajuan_barang.*").Distinct().
			Joins("JOIN pengajuan_barang_items ON pengajuan_barang_items.pengajuan_id = pengajuan_barang.id").
			Where("pengajuan_barang_items.barang_id = ?", f.BarangID)
	}
	return q
}

func preloadPengajuan(q *gorm.DB) *gorm.DB {
	return q.
		Preload("Gudang").
		Preload("Pengaju").
		Preload("Pemroses").
		Preload("Items").
		Preload("Items.Barang", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
		Preload("Items.Satuan").
		Preload("BarangKeluar").
		Preload("BarangMasuk").
		Preload("BarangRusak").
		Preload("Template").
		Preload("TandaTangan", func(db *gorm.DB) *gorm.DB {
			// Tanpa kolom data (gambar) supaya list/detail tetap ringan.
			return db.Select("id", "pengajuan_id", "peran", "user_id", "nama", "jabatan",
				"ip_address", "dokumen_hash", "signed_at").Order("signed_at asc")
		})
}

func (r *repository) List(p utils.PaginationParams, f Filter) ([]model.PengajuanBarang, int64, error) {
	var list []model.PengajuanBarang
	var total int64

	q := applyFilter(r.db.Model(&model.PengajuanBarang{}), f)
	if p.Search != "" {
		q = q.Where("nomor_pengajuan ILIKE ? OR keperluan ILIKE ?", "%"+p.Search+"%", "%"+p.Search+"%")
	}
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := p.Apply(preloadPengajuan(q.Session(&gorm.Session{})).Order("id desc")).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *repository) FindByID(id uint) (*model.PengajuanBarang, error) {
	var pengajuan model.PengajuanBarang
	if err := preloadPengajuan(r.db).First(&pengajuan, id).Error; err != nil {
		return nil, err
	}
	return &pengajuan, nil
}

func (r *repository) Create(pengajuan *model.PengajuanBarang) error {
	return r.db.Create(pengajuan).Error
}

func (r *repository) Update(pengajuan *model.PengajuanBarang, items []model.PengajuanBarangItem, ttd *model.PengajuanTandaTangan) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var current model.PengajuanBarang
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&current, pengajuan.ID).Error; err != nil {
			return err
		}
		if current.Status != constant.StatusPengajuanDiajukan {
			return errors.New(constant.ErrPengajuanBukanDiajukan)
		}
		if err := tx.Where("pengajuan_id = ?", pengajuan.ID).Delete(&model.PengajuanBarangItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].ID = 0
			items[i].PengajuanID = pengajuan.ID
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		// Isi berubah -> tanda tangan lama tidak berlaku lagi.
		if err := tx.Where("pengajuan_id = ?", pengajuan.ID).Delete(&model.PengajuanTandaTangan{}).Error; err != nil {
			return err
		}
		if ttd != nil {
			ttd.ID = 0
			ttd.PengajuanID = pengajuan.ID
			if err := tx.Create(ttd).Error; err != nil {
				return err
			}
		}
		// Omit(clause.Associations): cegah Save() menimpa balik gudang_id/
		// template_id dengan ID dari relasi Gudang/Template yang ter-preload
		// di query lookup sebelumnya (bug sama seperti repositories/barang —
		// lihat komentar di sana).
		return tx.Omit(clause.Associations).Save(pengajuan).Error
	})
}

func (r *repository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var current model.PengajuanBarang
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&current, id).Error; err != nil {
			return err
		}
		if current.Status != constant.StatusPengajuanDiajukan {
			return errors.New(constant.ErrPengajuanBukanDiajukan)
		}
		if err := tx.Where("pengajuan_id = ?", id).Delete(&model.PengajuanBarangItem{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.PengajuanBarang{}, id).Error
	})
}

// setujuiKeluar: perilaku asli (satu-satunya jenis sebelum fitur ini
// diperluas) — bikin dokumen BarangKeluar dari daftar barang pengajuan lalu
// langsung memotong stok (kalau semua barangnya bukan barang ber-nomor-seri)
// atau membiarkan dokumennya draft supaya staf gudang menuntaskannya lewat
// halaman Barang Keluar untuk memilih nomor serinya satu-satu.
func (r *repository) setujuiKeluar(tx *gorm.DB, pengajuan *model.PengajuanBarang, userID uint) (map[string]interface{}, error) {
	anySerialized := false
	for _, item := range pengajuan.Items {
		var b model.Barang
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&b, item.BarangID).Error; err != nil {
			return nil, err
		}
		if b.IsSerialized {
			anySerialized = true
		}

		stokGudang, err := barangstokgudang.GetStokGudangTx(tx, item.BarangID, pengajuan.GudangID)
		if err != nil {
			return nil, err
		}
		if stokGudang < item.Qty {
			return nil, fmt.Errorf("%s (barang: %s, tersedia di gudang ini: %d, diajukan: %d)",
				constant.ErrPengajuanStokTidakCukup, b.Nama, stokGudang, item.Qty)
		}
	}

	bkItems := make([]model.BarangKeluarItem, 0, len(pengajuan.Items))
	for _, item := range pengajuan.Items {
		bkItems = append(bkItems, model.BarangKeluarItem{BarangID: item.BarangID, Qty: item.Qty})
	}
	nomorBK, err := docnumber.Next(tx, "BK")
	if err != nil {
		return nil, err
	}
	bk := &model.BarangKeluar{
		NomorPengeluaran: nomorBK,
		GudangID:         pengajuan.GudangID,
		Status:           constant.StatusBKDraft,
		Tanggal:          time.Now(),
		Keperluan:        fmt.Sprintf("Pengajuan %s — %s", pengajuan.NomorPengajuan, pengajuan.Keperluan),
		Items:            bkItems,
	}
	if err := tx.Create(bk).Error; err != nil {
		return nil, err
	}

	if !anySerialized {
		for _, item := range pengajuan.Items {
			if err := tx.Model(&model.Barang{}).Where("id = ?", item.BarangID).
				Update("stok", gorm.Expr("stok - ?", item.Qty)).Error; err != nil {
				return nil, err
			}
			if err := barangstokgudang.AdjustStokGudangTx(tx, item.BarangID, pengajuan.GudangID, -item.Qty); err != nil {
				return nil, err
			}
		}
		now := time.Now()
		if err := tx.Model(&model.BarangKeluar{}).Where("id = ?", bk.ID).Updates(map[string]interface{}{
			"status":           constant.StatusBKSelesai,
			"dikeluarkan_oleh": userID,
			"completed_at":     now,
		}).Error; err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{"barang_keluar_id": bk.ID}, nil
}

// setujuiMasuk: bikin dokumen BarangMasuk berstatus draft dari daftar barang
// pengajuan. Harga satuan diisi 0 dulu — staf gudang yang melengkapinya saat
// menuntaskan penerimaan di halaman Barang Masuk, karena penyetuju pengajuan
// (GA) biasanya tidak tahu harga beli barangnya.
func (r *repository) setujuiMasuk(tx *gorm.DB, pengajuan *model.PengajuanBarang) (map[string]interface{}, error) {
	bmItems := make([]model.BarangMasukItem, 0, len(pengajuan.Items))
	for _, item := range pengajuan.Items {
		bmItems = append(bmItems, model.BarangMasukItem{BarangID: item.BarangID, Qty: item.Qty, HargaSatuan: 0})
	}
	nomorBM, err := docnumber.Next(tx, "BM")
	if err != nil {
		return nil, err
	}
	bm := &model.BarangMasuk{
		NomorPenerimaan: nomorBM,
		GudangID:        pengajuan.GudangID,
		Status:          constant.StatusBMDraft,
		Tanggal:         time.Now(),
		Catatan:         fmt.Sprintf("Pengajuan %s — %s", pengajuan.NomorPengajuan, pengajuan.Keperluan),
		Items:           bmItems,
	}
	if err := tx.Create(bm).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"barang_masuk_id": bm.ID}, nil
}

// setujuiTemplate: pengajuan berbasis formulir yang diunggah admin (jenis
// "template") tidak terkait barang sama sekali, jadi menyetujuinya tidak
// membuat dokumen apa pun — cukup status yang berubah jadi "disetujui"
// lewat update umum di Setujui.
func (r *repository) setujuiTemplate() (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// setujuiRusak: bikin satu baris BarangRusak per unit per item (Qty=3 berarti
// 3 baris terpisah, karena laporan kerusakan pada dasarnya per unit fisik).
// Statusnya tetap "pengecekan" — persetujuan pengajuan hanya mengesahkan
// laporannya masuk ke alur, bukan menggantikan pengecekan teknis staf yang
// menentukan retur/rusak lewat endpoint Inspeksi yang sudah ada.
func (r *repository) setujuiRusak(tx *gorm.DB, pengajuan *model.PengajuanBarang) (map[string]interface{}, error) {
	for _, item := range pengajuan.Items {
		var b model.Barang
		if err := tx.First(&b, item.BarangID).Error; err != nil {
			return nil, err
		}
		barangID := item.BarangID
		for i := 0; i < item.Qty; i++ {
			br := &model.BarangRusak{
				BarangID:       &barangID,
				LabelBarang:    fmt.Sprintf("%s-%d", pengajuan.NomorPengajuan, i+1),
				NamaBarang:     b.Nama,
				Keterangan:     fmt.Sprintf("Dari Pengajuan %s — %s", pengajuan.NomorPengajuan, pengajuan.Keperluan),
				Status:         constant.StatusPengecekan,
				DilaporkanOleh: pengajuan.DiajukanOleh,
				PengajuanID:    &pengajuan.ID,
			}
			if err := tx.Create(br).Error; err != nil {
				return nil, err
			}
		}
	}
	return map[string]interface{}{}, nil
}

func (r *repository) Setujui(id uint, userID uint, namaGA, jabatanGA, catatan string, ttd *model.PengajuanTandaTangan) (*model.PengajuanBarang, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var pengajuan model.PengajuanBarang
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Preload("Items").First(&pengajuan, id).Error; err != nil {
			return err
		}
		if pengajuan.Status != constant.StatusPengajuanDiajukan {
			return errors.New(constant.ErrPengajuanBukanDiajukan)
		}
		if pengajuan.Jenis != constant.JenisPengajuanTemplate && len(pengajuan.Items) == 0 {
			return errors.New("pengajuan ini belum punya daftar barang")
		}

		if err := r.verifikasiDuaPihak(tx, &pengajuan, userID, ttd); err != nil {
			return err
		}

		var extra map[string]interface{}
		var err error
		switch pengajuan.Jenis {
		case constant.JenisPengajuanMasuk:
			extra, err = r.setujuiMasuk(tx, &pengajuan)
		case constant.JenisPengajuanRusak:
			extra, err = r.setujuiRusak(tx, &pengajuan)
		case constant.JenisPengajuanTemplate, constant.JenisPengajuanBarang:
			// Tidak ada dokumen/stok yang berubah — hanya status yang jadi "disetujui".
			extra, err = r.setujuiTemplate()
		default:
			extra, err = r.setujuiKeluar(tx, &pengajuan, userID)
		}
		if err != nil {
			return err
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status":         constant.StatusPengajuanDisetujui,
			"diproses_oleh":  userID,
			"diproses_pada":  now,
			"nama_ga":        namaGA,
			"jabatan_ga":     jabatanGA,
			"catatan_proses": catatan,
		}
		for k, v := range extra {
			updates[k] = v
		}
		if ttd != nil {
			ttd.ID = 0
			ttd.PengajuanID = id
			ttd.Peran = model.PeranTTDPenyetuju
			ttd.UserID = userID
			ttd.SignedAt = now
			if err := tx.Create(ttd).Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.PengajuanBarang{}).Where("id = ?", id).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return r.FindByID(id)
}

// verifikasiDuaPihak menegakkan alur "pengaju tanda tangan -> penyetuju tanda
// tangan" untuk jenis yang wajib TTD. Dipanggil di dalam transaksi Setujui,
// setelah baris pengajuan dikunci (FOR UPDATE).
func (r *repository) verifikasiDuaPihak(tx *gorm.DB, pengajuan *model.PengajuanBarang, userID uint, ttd *model.PengajuanTandaTangan) error {
	var pencatat *model.PengajuanTandaTangan
	if constant.PengajuanWajibTTD(pengajuan.Jenis) {
		var found model.PengajuanTandaTangan
		err := tx.Where("pengajuan_id = ? AND peran = ?", pengajuan.ID, model.PeranTTDPencatat).First(&found).Error
		switch {
		case err == nil:
			pencatat = &found
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}
	}
	return CekDuaPihak(pengajuan, userID, ttd, pencatat)
}

// CekDuaPihak adalah aturan murni (tanpa database) untuk persetujuan dua pihak:
//   - ttd penyetuju wajib ada,
//   - penyetuju bukan orang yang sama dengan pengaju,
//   - pengaju sudah tanda tangan DAN isi pengajuan sekarang masih sama persis
//     dengan yang ia tandatangani (dicocokkan lewat sidik jari dokumen).
//
// Untuk jenis yang tidak wajib TTD, fungsi ini hanya mengisi DokumenHash pada
// ttd (kalau ada) dan tidak menolak apa pun. pencatat = tanda tangan pengaju
// yang tersimpan (nil kalau belum ada).
func CekDuaPihak(pengajuan *model.PengajuanBarang, userID uint, ttd, pencatat *model.PengajuanTandaTangan) error {
	hash := pengajuan.ComputeDokumenHash()
	if ttd != nil {
		ttd.DokumenHash = hash
	}
	if !constant.PengajuanWajibTTD(pengajuan.Jenis) {
		return nil
	}
	if ttd == nil || len(ttd.Data) == 0 {
		return errors.New(constant.ErrPengajuanTTDPenyetujuWajib)
	}
	if pengajuan.DiajukanOleh == userID {
		return errors.New(constant.ErrPengajuanPenyetujuSamaDgnPengaju)
	}
	if pencatat == nil || pencatat.DokumenHash != hash {
		return errors.New(constant.ErrPengajuanTTDPencatatHilang)
	}
	return nil
}

func (r *repository) FindTandaTangan(pengajuanID uint, peran string) (*model.PengajuanTandaTangan, error) {
	var t model.PengajuanTandaTangan
	if err := r.db.Where("pengajuan_id = ? AND peran = ?", pengajuanID, peran).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *repository) OpenMasukByBarang(barangIDs []uint) (map[uint]string, error) {
	out := make(map[uint]string, len(barangIDs))
	if len(barangIDs) == 0 {
		return out, nil
	}
	type row struct {
		BarangID       uint
		NomorPengajuan string
	}
	var rows []row
	err := r.db.Table("pengajuan_barang_items AS i").
		Select("i.barang_id AS barang_id, p.nomor_pengajuan AS nomor_pengajuan").
		Joins("JOIN pengajuan_barang p ON p.id = i.pengajuan_id").
		Where("p.jenis = ? AND p.status = ? AND i.barang_id IN ?", constant.JenisPengajuanMasuk, constant.StatusPengajuanDiajukan, barangIDs).
		Order("p.id asc").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, rw := range rows {
		out[rw.BarangID] = rw.NomorPengajuan // pengajuan terbaru menimpa yang lama
	}
	return out, nil
}

func (r *repository) Tolak(id uint, userID uint, namaGA, jabatanGA, catatan string) (*model.PengajuanBarang, error) {
	now := time.Now()
	res := r.db.Model(&model.PengajuanBarang{}).
		Where("id = ? AND status = ?", id, constant.StatusPengajuanDiajukan).
		Updates(map[string]interface{}{
			"status":         constant.StatusPengajuanDitolak,
			"diproses_oleh":  userID,
			"diproses_pada":  now,
			"nama_ga":        namaGA,
			"jabatan_ga":     jabatanGA,
			"catatan_proses": catatan,
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errors.New(constant.ErrPengajuanBukanDiajukan)
	}
	return r.FindByID(id)
}

func (r *repository) CountByStatus(status string) (int64, error) {
	var count int64
	err := r.db.Model(&model.PengajuanBarang{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

func (r *repository) NextNomor() (string, error) {
	return docnumber.Next(r.db, "PJ")
}
