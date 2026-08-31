package laporan

import (
	"sort"
	"strconv"
	"time"

	barangKeluarRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_keluar"
	barangMasukRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_masuk"
	barangSerialRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

// fifoRow adalah satu baris kandidat FIFO/FEFO: satu batch barang masuk
// (untuk barang yang tidak ber-nomor-seri) atau satu unit ber-nomor-seri.
type fifoRow struct {
	kodeBarang   string
	nama         string
	merek        string
	tipe         string
	serialNumber string
	referensi    string
	tanggal      time.Time
	qtyMasuk     int
	sisa         int
}

// buildFifoFefo menyusun "Laporan FIFO/FEFO": untuk setiap barang, daftar
// batch (barang biasa) atau unit (barang ber-nomor-seri) diurutkan dari yang
// paling lama tercatat ke yang paling baru, lengkap dengan estimasi sisa dan
// rekomendasi mana yang sebaiknya dikeluarkan lebih dulu.
//
// Sistem ini belum menyimpan tanggal kedaluwarsa per barang, jadi "FEFO"
// (First Expired First Out) di sini didekati dengan tanggal pencatatan
// (First In First Out) — barang yang tercatat lebih dulu diasumsikan juga
// lebih dulu perlu dikeluarkan. Untuk barang ber-nomor-seri, status
// tersedia/terpasang per unit dibaca langsung dari tabel barang_serials
// (otoritatif). Untuk barang tanpa nomor seri, sisa per batch adalah HASIL
// SIMULASI: total qty keluar (yang belum dibatalkan) dikurangkan dari batch
// yang paling lama dulu, karena barang_keluar tidak menyimpan tautan ke
// batch/masuk spesifik mana stoknya diambil.
func (h *Controller) buildFifoFefo(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	masukList, _, err := h.barangMasukRepo.List(bigPagination(), barangMasukRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	keluarList, _, err := h.barangKeluarRepo.List(bigPagination(), barangKeluarRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	serialList, _, err := h.barangSerialRepo.List(bigPagination(), barangSerialRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}

	keluarPerBarang := map[uint]int{}
	for _, bk := range keluarList {
		if bk.Status == constant.StatusBKDibatalkan {
			continue
		}
		for _, it := range bk.Items {
			keluarPerBarang[it.BarangID] += it.Qty
		}
	}

	rowsByBarang := map[uint][]fifoRow{}
	kodeByBarang := map[uint]string{}

	for _, bm := range masukList {
		if bm.Status == constant.StatusBMDibatalkan {
			continue
		}
		if !inRange(bm.Tanggal, dari, sampai) {
			continue
		}
		for _, it := range bm.Items {
			if it.Barang == nil || it.Barang.IsSerialized {
				continue
			}
			rowsByBarang[it.BarangID] = append(rowsByBarang[it.BarangID], fifoRow{
				kodeBarang: it.Barang.KodeBarang,
				nama:       it.Barang.Nama,
				merek:      orDash(it.Barang.Merek),
				tipe:       orDash(it.Barang.Tipe),
				referensi:  bm.NomorPenerimaan,
				tanggal:    bm.Tanggal,
				qtyMasuk:   it.Qty,
			})
			kodeByBarang[it.BarangID] = it.Barang.KodeBarang
		}
	}

	for _, s := range serialList {
		if s.Barang == nil || !s.Barang.IsSerialized {
			continue
		}
		tanggal := s.CreatedAt
		referensi := "-"
		if s.BarangMasukItem != nil && s.BarangMasukItem.BarangMasuk != nil {
			tanggal = s.BarangMasukItem.BarangMasuk.Tanggal
			referensi = s.BarangMasukItem.BarangMasuk.NomorPenerimaan
		}
		if !inRange(tanggal, dari, sampai) {
			continue
		}
		sisa := 0
		if s.Status == constant.StatusSerialTersedia {
			sisa = 1
		}
		rowsByBarang[s.BarangID] = append(rowsByBarang[s.BarangID], fifoRow{
			kodeBarang:   s.Barang.KodeBarang,
			nama:         s.Barang.Nama,
			merek:        orDash(s.Barang.Merek),
			tipe:         orDash(s.Barang.Tipe),
			serialNumber: s.SerialNumber,
			referensi:    referensi,
			tanggal:      tanggal,
			qtyMasuk:     1,
			sisa:         sisa,
		})
		kodeByBarang[s.BarangID] = s.Barang.KodeBarang
	}

	headers = []string{
		"Kode Barang", "Nama Barang", "Merek", "Tipe", "Serial Number",
		"No. Referensi Masuk", "Tanggal Tercatat", "Qty Masuk", "Sisa Saat Ini",
		"Urutan (1=Tertua)", "Rekomendasi",
	}

	barangIDs := make([]uint, 0, len(rowsByBarang))
	for id := range rowsByBarang {
		barangIDs = append(barangIDs, id)
	}
	sort.Slice(barangIDs, func(i, j int) bool {
		return kodeByBarang[barangIDs[i]] < kodeByBarang[barangIDs[j]]
	})

	for _, barangID := range barangIDs {
		group := rowsByBarang[barangID]
		sort.SliceStable(group, func(i, j int) bool { return group[i].tanggal.Before(group[j].tanggal) })

		isSerialized := group[0].serialNumber != ""
		sisaKeluar := keluarPerBarang[barangID]

		rekomendasiDiberikan := false
		for idx := range group {
			g := &group[idx]
			if !isSerialized {
				sisa := g.qtyMasuk
				switch {
				case sisaKeluar >= sisa:
					sisaKeluar -= sisa
					sisa = 0
				case sisaKeluar > 0:
					sisa -= sisaKeluar
					sisaKeluar = 0
				}
				g.sisa = sisa
			}

			var rekomendasi string
			switch {
			case g.sisa <= 0:
				rekomendasi = "Sudah Keluar / Habis"
			case !rekomendasiDiberikan:
				rekomendasi = "Keluarkan Duluan (FIFO)"
				rekomendasiDiberikan = true
			default:
				rekomendasi = "Menunggu Giliran"
			}

			rows = append(rows, []string{
				g.kodeBarang, g.nama, g.merek, g.tipe, orDash(g.serialNumber),
				g.referensi, g.tanggal.Format(dateFormat), strconv.Itoa(g.qtyMasuk),
				strconv.Itoa(g.sisa), strconv.Itoa(idx + 1), rekomendasi,
			})
		}
	}

	return headers, rows, nil
}
