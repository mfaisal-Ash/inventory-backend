package barang

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

type idTotal struct {
	BarangID uint
	Total    int64
}

// fillListStats mengisi statistik turunan untuk satu halaman daftar barang
// dengan beberapa query ber-grup (bukan N+1):
//   - TotalKeluar : total qty barang keluar yang sudah selesai. Di tabel,
//     Kuantitas = Stok + TotalKeluar (semua yang pernah tercatat) dan
//     Sisa = Stok.
//   - JumlahRusak : barang ber-SN -> unit serial berstatus rusak; barang
//     tanpa SN -> laporan Barang Rusak berstatus rusak.
//   - NamaPengaju : nama petugas yang mendaftarkan barang (kolom "Nama").
//
// Best-effort: kalau salah satu query gagal, daftar barang tetap tampil
// (statistiknya kosong/0) — jangan sampai tabel utama ikut gagal.
func (h *Controller) fillListStats(list []model.Barang) {
	if len(list) == 0 {
		return
	}
	ids := make([]uint, 0, len(list))
	userIDs := make([]uint, 0, len(list))
	for i := range list {
		ids = append(ids, list[i].ID)
		if list[i].DiajukanOleh != nil {
			userIDs = append(userIDs, *list[i].DiajukanOleh)
		}
	}

	keluar := map[uint]int64{}
	var rows []idTotal
	if err := h.db.Table("barang_keluar_items bki").
		Joins("JOIN barang_keluar bk ON bk.id = bki.barang_keluar_id").
		Where("bki.barang_id IN ? AND bk.status = ?", ids, constant.StatusBKSelesai).
		Select("bki.barang_id AS barang_id, COALESCE(SUM(bki.qty), 0) AS total").
		Group("bki.barang_id").Scan(&rows).Error; err == nil {
		for _, r := range rows {
			keluar[r.BarangID] = r.Total
		}
	}

	rusakSerial := map[uint]int64{}
	rows = nil
	if err := h.db.Model(&model.BarangSerial{}).
		Where("barang_id IN ? AND status = ?", ids, constant.StatusSerialRusak).
		Select("barang_id AS barang_id, COUNT(*) AS total").
		Group("barang_id").Scan(&rows).Error; err == nil {
		for _, r := range rows {
			rusakSerial[r.BarangID] = r.Total
		}
	}

	rusakLaporan := map[uint]int64{}
	rows = nil
	if err := h.db.Model(&model.BarangRusak{}).
		Where("barang_id IN ? AND status = ?", ids, constant.StatusRusak).
		Select("barang_id AS barang_id, COUNT(*) AS total").
		Group("barang_id").Scan(&rows).Error; err == nil {
		for _, r := range rows {
			rusakLaporan[r.BarangID] = r.Total
		}
	}

	nama := map[uint]string{}
	if len(userIDs) > 0 {
		var users []model.User
		if err := h.db.Select("id", "full_name").Where("id IN ?", userIDs).Find(&users).Error; err == nil {
			for _, u := range users {
				nama[u.ID] = u.FullName
			}
		}
	}

	for i := range list {
		b := &list[i]
		b.TotalKeluar = keluar[b.ID]
		if b.IsSerialized {
			b.JumlahRusak = rusakSerial[b.ID]
		} else {
			b.JumlahRusak = rusakLaporan[b.ID]
		}
		if b.DiajukanOleh != nil {
			b.NamaPengaju = nama[*b.DiajukanOleh]
		}
	}
}
