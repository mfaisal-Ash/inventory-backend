package laporan

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	assetRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset"
	barangRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	barangKeluarRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_keluar"
	barangMasukRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_masuk"
	barangRusakRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_rusak"
	barangSerialRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	pengajuanRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_barang"
	stockOpnameRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/stockOpname"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/reportexport"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const Module = constant.ModuleLaporan
const dateFormat = "2006-01-02"

func parseDateRange(c *fiber.Ctx) (dari, sampai *time.Time, err error) {
	if raw := c.Query("dari", ""); raw != "" {
		t, e := time.Parse(dateFormat, raw)
		if e != nil {
			return nil, nil, fmt.Errorf("format tanggal 'dari' tidak valid (gunakan YYYY-MM-DD)")
		}
		dari = &t
	}
	if raw := c.Query("sampai", ""); raw != "" {
		t, e := time.Parse(dateFormat, raw)
		if e != nil {
			return nil, nil, fmt.Errorf("format tanggal 'sampai' tidak valid (gunakan YYYY-MM-DD)")
		}
		t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		sampai = &t
	}
	return dari, sampai, nil
}

func inRange(t time.Time, dari, sampai *time.Time) bool {
	if dari != nil && t.Before(*dari) {
		return false
	}
	if sampai != nil && t.After(*sampai) {
		return false
	}
	return true
}

func formatRupiah(v int64) string {
	neg := v < 0
	s := strconv.FormatInt(v, 10)
	if neg {
		s = s[1:]
	}
	n := len(s)
	var parts []string
	for n > 3 {
		parts = append([]string{s[n-3:]}, parts...)
		s = s[:n-3]
		n = len(s)
	}
	parts = append([]string{s}, parts...)
	res := strings.Join(parts, ".")
	if neg {
		res = "-" + res
	}
	return res
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (h *Controller) userLabel(cache map[uint]string, id *uint) string {
	if id == nil {
		return "-"
	}
	if name, ok := cache[*id]; ok {
		return name
	}
	name := "-"
	if u, err := h.usersRepo.FindByID(*id); err == nil && u != nil {
		if u.FullName != "" {
			name = u.FullName
		} else {
			name = u.Username
		}
	}
	cache[*id] = name
	return name
}

func (h *Controller) buildStokBarang() (headers []string, rows [][]string, err error) {
	list, _, err := h.barangRepo.List(bigPagination(), barangRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	headers = []string{"Kode Barang", "Nama", "Merek", "Tipe", "Kategori", "Satuan", "Stok", "Stok Minimum", "Harga Beli", "Nilai Inventaris", "Status"}
	for _, b := range list {
		kategori, satuan := "-", "-"
		if b.Kategori != nil {
			kategori = b.Kategori.Nama
		}
		if b.Satuan != nil {
			satuan = b.Satuan.Nama
		}
		status := "Aktif"
		if !b.IsActive {
			status = "Nonaktif"
		}
		rows = append(rows, []string{
			b.KodeBarang, b.Nama, orDash(b.Merek), orDash(b.Tipe), kategori, satuan,
			strconv.Itoa(b.Stok), strconv.Itoa(b.StokMinimum),
			formatRupiah(b.HargaBeli), formatRupiah(b.NilaiInventaris()), status,
		})
	}
	return headers, rows, nil
}

const maxSerialsShownInReport = 5

func summarizeSerials(serials []string) string {
	if len(serials) == 0 {
		return "-"
	}
	if len(serials) <= maxSerialsShownInReport {
		return strings.Join(serials, "; ")
	}
	shown := strings.Join(serials[:maxSerialsShownInReport], "; ")
	return fmt.Sprintf("%s; +%d lainnya", shown, len(serials)-maxSerialsShownInReport)
}

func (h *Controller) serialsForMasukItem(itemID uint) string {
	list, _, err := h.barangSerialRepo.List(bigPagination(), barangSerialRepoPkg.Filter{BarangMasukItemID: itemID})
	if err != nil || len(list) == 0 {
		return "-"
	}
	sn := make([]string, 0, len(list))
	for _, s := range list {
		sn = append(sn, s.SerialNumber)
	}
	return summarizeSerials(sn)
}

func (h *Controller) serialsForKeluarItem(itemID uint) string {
	list, _, err := h.barangSerialRepo.List(bigPagination(), barangSerialRepoPkg.Filter{BarangKeluarItemID: itemID})
	if err != nil || len(list) == 0 {
		return "-"
	}
	sn := make([]string, 0, len(list))
	for _, s := range list {
		sn = append(sn, s.SerialNumber)
	}
	return summarizeSerials(sn)
}

func (h *Controller) buildBarangMasuk(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	list, _, err := h.barangMasukRepo.List(bigPagination(), barangMasukRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	headers = []string{"Nomor Penerimaan", "Tanggal", "Gudang", "Kode Barang", "Nama Barang", "Merek", "Tipe", "Qty", "Harga Satuan", "Serial Number", "Status", "Diterima Oleh", "Catatan"}
	userCache := map[uint]string{}
	for _, bm := range list {
		if !inRange(bm.Tanggal, dari, sampai) {
			continue
		}
		gudang := "-"
		if bm.Gudang != nil {
			gudang = bm.Gudang.Nama
		}
		if len(bm.Items) == 0 {
			rows = append(rows, []string{
				bm.NomorPenerimaan, bm.Tanggal.Format(dateFormat), gudang, "-", "-", "-", "-", "-", "-", "-",
				bm.Status, h.userLabel(userCache, bm.DiterimaOleh), bm.Catatan,
			})
			continue
		}
		for _, it := range bm.Items {
			kodeBarang, nama, merek, tipe, sn := "-", "-", "-", "-", "-"
			if it.Barang != nil {
				kodeBarang, nama = it.Barang.KodeBarang, it.Barang.Nama
				merek, tipe = orDash(it.Barang.Merek), orDash(it.Barang.Tipe)
				if it.Barang.IsSerialized {
					sn = h.serialsForMasukItem(it.ID)
				}
			}
			rows = append(rows, []string{
				bm.NomorPenerimaan, bm.Tanggal.Format(dateFormat), gudang, kodeBarang, nama, merek, tipe,
				strconv.Itoa(it.Qty), formatRupiah(it.HargaSatuan), sn, bm.Status, h.userLabel(userCache, bm.DiterimaOleh), bm.Catatan,
			})
		}
	}
	return headers, rows, nil
}

func (h *Controller) buildBarangKeluar(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	list, _, err := h.barangKeluarRepo.List(bigPagination(), barangKeluarRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	headers = []string{"Nomor Pengeluaran", "Tanggal", "Gudang", "Kode Barang", "Nama Barang", "Merek", "Tipe", "Qty", "Serial Number", "Keperluan", "Penerima", "Status", "Dikeluarkan Oleh"}
	userCache := map[uint]string{}
	for _, bk := range list {
		if !inRange(bk.Tanggal, dari, sampai) {
			continue
		}
		gudang := "-"
		if bk.Gudang != nil {
			gudang = bk.Gudang.Nama
		}
		if len(bk.Items) == 0 {
			rows = append(rows, []string{
				bk.NomorPengeluaran, bk.Tanggal.Format(dateFormat), gudang, "-", "-", "-", "-", "-", "-",
				bk.Keperluan, bk.Penerima, bk.Status, h.userLabel(userCache, bk.DikeluarkanOleh),
			})
			continue
		}
		for _, it := range bk.Items {
			kodeBarang, nama, merek, tipe, sn := "-", "-", "-", "-", "-"
			if it.Barang != nil {
				kodeBarang, nama = it.Barang.KodeBarang, it.Barang.Nama
				merek, tipe = orDash(it.Barang.Merek), orDash(it.Barang.Tipe)
				if it.Barang.IsSerialized {
					sn = h.serialsForKeluarItem(it.ID)
				}
			}
			rows = append(rows, []string{
				bk.NomorPengeluaran, bk.Tanggal.Format(dateFormat), gudang, kodeBarang, nama, merek, tipe,
				strconv.Itoa(it.Qty), sn, bk.Keperluan, bk.Penerima, bk.Status, h.userLabel(userCache, bk.DikeluarkanOleh),
			})
		}
	}
	return headers, rows, nil
}

var pengajuanStatusLabelMap = map[string]string{
	constant.StatusPengajuanDiajukan:  "Menunggu Persetujuan",
	constant.StatusPengajuanDisetujui: "Disetujui",
	constant.StatusPengajuanDitolak:   "Ditolak",
}

func pengajuanStatusLabel(status string) string {
	if label, ok := pengajuanStatusLabelMap[status]; ok {
		return label
	}
	return status
}

// buildPengajuanBarang — rekap seluruh pengajuan pengeluaran barang (baik
// yang masih menunggu, sudah disetujui, maupun ditolak) supaya bisa
// direkap per hari/bulan lewat granularitas yang sama dengan laporan lain.
func (h *Controller) buildPengajuanBarang(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	list, _, err := h.pengajuanRepo.List(bigPagination(), pengajuanRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	headers = []string{"Nomor Pengajuan", "Jenis", "Tanggal", "Gudang", "Kode Barang", "Nama Barang", "Qty", "Keperluan", "Status", "Diajukan Oleh", "Diproses Oleh", "Catatan Proses"}
	userCache := map[uint]string{}
	for _, p := range list {
		if !inRange(p.Tanggal, dari, sampai) {
			continue
		}
		gudang := "-"
		if p.Gudang != nil {
			gudang = p.Gudang.Nama
		}
		diajukanOleh := p.DiajukanOleh
		diajukan := h.userLabel(userCache, &diajukanOleh)
		diproses := h.userLabel(userCache, p.DiprosesOleh)
		statusLabel := pengajuanStatusLabel(p.Status)
		jenisLabel := pengajuanJenisLabel(p.Jenis)
		if len(p.Items) == 0 {
			rows = append(rows, []string{
				p.NomorPengajuan, jenisLabel, p.Tanggal.Format(dateFormat), gudang, "-", "-", "-",
				p.Keperluan, statusLabel, diajukan, diproses, orDash(p.CatatanProses),
			})
			continue
		}
		for _, it := range p.Items {
			kodeBarang, nama := "-", "-"
			if it.Barang != nil {
				kodeBarang, nama = it.Barang.KodeBarang, it.Barang.Nama
			}
			rows = append(rows, []string{
				p.NomorPengajuan, jenisLabel, p.Tanggal.Format(dateFormat), gudang, kodeBarang, nama, strconv.Itoa(it.Qty),
				p.Keperluan, statusLabel, diajukan, diproses, orDash(p.CatatanProses),
			})
		}
	}
	return headers, rows, nil
}

func pengajuanJenisLabel(jenis string) string {
	switch jenis {
	case constant.JenisPengajuanMasuk:
		return "Barang Masuk"
	case constant.JenisPengajuanRusak:
		return "Barang Rusak"
	default:
		return "Barang Keluar"
	}
}

func (h *Controller) buildStockOpname(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	list, _, err := h.stockOpnameRepo.List(bigPagination(), stockOpnameRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	headers = []string{"Nomor Opname", "Tanggal", "Gudang", "Kode Barang", "Nama Barang", "Stok Sistem", "Stok Fisik", "Selisih", "Status", "Dilakukan Oleh", "Catatan"}
	userCache := map[uint]string{}
	for _, so := range list {
		if !inRange(so.Tanggal, dari, sampai) {
			continue
		}
		gudang := "-"
		if so.Gudang != nil {
			gudang = so.Gudang.Nama
		}
		dilakukanOleh := h.userLabel(userCache, so.DilakukanOleh)
		if len(so.Items) == 0 {
			rows = append(rows, []string{
				so.NomorOpname, so.Tanggal.Format(dateFormat), gudang, "-", "-", "-", "-", "-",
				so.Status, dilakukanOleh, so.Catatan,
			})
			continue
		}
		for _, it := range so.Items {
			kodeBarang, nama := "-", "-"
			if it.Barang != nil {
				kodeBarang, nama = it.Barang.KodeBarang, it.Barang.Nama
			}
			rows = append(rows, []string{
				so.NomorOpname, so.Tanggal.Format(dateFormat), gudang, kodeBarang, nama,
				strconv.Itoa(it.StokSistem), strconv.Itoa(it.StokFisik), strconv.Itoa(it.Selisih),
				so.Status, dilakukanOleh, so.Catatan,
			})
		}
	}
	return headers, rows, nil
}

var reportTitles = map[string]string{
	constant.LaporanStokBarang:      "Laporan Stok Barang",
	constant.LaporanBarangMasuk:     "Laporan Barang Masuk",
	constant.LaporanBarangKeluar:    "Laporan Barang Keluar",
	constant.LaporanStokOpname:      "Laporan Stock Opname",
	constant.LaporanBarangRusak:     "Laporan Barang Rusak",
	constant.LaporanFifoFefo:        "Laporan FIFO/FEFO",
	constant.LaporanPengajuanBarang: "Laporan Pengajuan Barang",
	constant.LaporanTrackingAset:    "Laporan Tracking Aset",
}

// assetHistoryEventLabel: label Indonesia yang enak dibaca untuk tiap
// EventType riwayat aset, dijaga konsisten dengan EVENT_TYPE_LABEL di
// frontend (AssetTrackingMap.tsx) supaya laporan & UI tracking aset
// memakai istilah yang sama.
var assetHistoryEventLabel = map[string]string{
	"dibuat":            "Dibuat",
	"status":            "Status diubah",
	"lokasi":            "Lokasi diubah",
	"induk":             "Aset induk diubah",
	"gudang":            "Dipindahkan ke gudang lain",
	"port":              "Port diubah",
	"nilai_aset":        "Nilai aset diubah",
	"data_transportasi": "Data transportasi diubah",
}

var barangRusakStatusLabel = map[string]string{
	constant.StatusBarangRusakPengecekan: "Menunggu Pengecekan",
	constant.StatusBarangRusakDiperbaiki: "Bisa Diperbaiki",
	constant.StatusRetur:                 "Retur",
	constant.StatusBarangRusakDibuang:    "Dibuang/Rusak",
}

// buildBarangRusak — laporan SELURUH laporan barang rusak (semua status:
// menunggu pengecekan, bisa diperbaiki, retur, dibuang), berbeda dari
// buildBarangRetur yang cuma mengambil status "retur" saja.
func (h *Controller) buildBarangRusak(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	list, _, err := h.barangRusakRepo.List(bigPagination(), barangRusakRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	headers = []string{"Label/Kode Barang", "Kode Barang (SKU)", "Nama Barang", "Merek", "Tipe", "Serial Number", "Keterangan", "Status", "Dilaporkan Oleh", "Diperiksa Oleh", "Tanggal Diperiksa"}
	for _, b := range list {
		refDate := b.DicekPada
		if refDate == nil {
			refDate = &b.CreatedAt
		}
		if !inRange(*refDate, dari, sampai) {
			continue
		}
		pelapor, pemeriksa, tanggal := "-", "-", "-"
		if b.Pelapor != nil {
			pelapor = b.Pelapor.FullName
		}
		if b.Pemeriksa != nil {
			pemeriksa = b.Pemeriksa.FullName
		}
		if b.DicekPada != nil {
			tanggal = b.DicekPada.Format(dateFormat)
		}
		kodeBarang, merek, tipe := "-", "-", "-"
		if b.Barang != nil {
			kodeBarang, merek, tipe = b.Barang.KodeBarang, orDash(b.Barang.Merek), orDash(b.Barang.Tipe)
		}
		status := b.Status
		if label, ok := barangRusakStatusLabel[b.Status]; ok {
			status = label
		}
		rows = append(rows, []string{
			b.LabelBarang, kodeBarang, b.NamaBarang, merek, tipe, orDash(b.SerialNumber), orDash(b.Keterangan), status, pelapor, pemeriksa, tanggal,
		})
	}
	return headers, rows, nil
}

func (h *Controller) buildReport(tipe string, dari, sampai *time.Time) (title string, headers []string, rows [][]string, err error) {
	title, ok := reportTitles[tipe]
	if !ok {
		return "", nil, nil, errors.New(constant.ErrLaporanTipeTidakDidukung)
	}

	switch tipe {
	case constant.LaporanStokBarang:
		headers, rows, err = h.buildStokBarang()
	case constant.LaporanBarangMasuk:
		headers, rows, err = h.buildBarangMasuk(dari, sampai)
	case constant.LaporanBarangKeluar:
		headers, rows, err = h.buildBarangKeluar(dari, sampai)
	case constant.LaporanStokOpname:
		headers, rows, err = h.buildStockOpname(dari, sampai)
	case constant.LaporanBarangRusak:
		headers, rows, err = h.buildBarangRusak(dari, sampai)
	case constant.LaporanFifoFefo:
		headers, rows, err = h.buildFifoFefo(dari, sampai)
	case constant.LaporanPengajuanBarang:
		headers, rows, err = h.buildPengajuanBarang(dari, sampai)
	case constant.LaporanTrackingAset:
		headers, rows, err = h.buildLaporanTrackingAset(dari, sampai)
	}
	return title, headers, rows, err
}

func (h *Controller) buildBarangRetur(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	list, _, err := h.barangRusakRepo.List(bigPagination(), barangRusakRepoPkg.Filter{Status: constant.StatusRetur})
	if err != nil {
		return nil, nil, err
	}
	headers = []string{"Label/Kode Barang", "Kode Barang (SKU)", "Nama Barang", "Merek", "Tipe", "Serial Number", "Keterangan", "Dilaporkan Oleh", "Diperiksa Oleh", "Tanggal Diperiksa"}
	for _, b := range list {
		if b.DicekPada != nil && !inRange(*b.DicekPada, dari, sampai) {
			continue
		}
		pelapor, pemeriksa, tanggal := "-", "-", "-"
		if b.Pelapor != nil {
			pelapor = b.Pelapor.FullName
		}
		if b.Pemeriksa != nil {
			pemeriksa = b.Pemeriksa.FullName
		}
		if b.DicekPada != nil {
			tanggal = b.DicekPada.Format(dateFormat)
		}
		kodeBarang, merek, tipe := "-", "-", "-"
		if b.Barang != nil {
			kodeBarang, merek, tipe = b.Barang.KodeBarang, orDash(b.Barang.Merek), orDash(b.Barang.Tipe)
		}
		rows = append(rows, []string{
			b.LabelBarang, kodeBarang, b.NamaBarang, merek, tipe, orDash(b.SerialNumber), orDash(b.Keterangan), pelapor, pemeriksa, tanggal,
		})
	}
	return headers, rows, nil
}

// buildLaporanTrackingAset — laporan riwayat/tracking seluruh aset (lintas
// semua gudang), dipakai untuk pengganti "Laporan Barang Retur". Satu baris
// per kejadian riwayat aset (AssetHistory) dalam rentang tanggal dari/sampai.
func (h *Controller) buildLaporanTrackingAset(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	assets, _, err := h.assetRepo.List(bigPagination(), assetRepoPkg.Filter{})
	if err != nil {
		return nil, nil, err
	}
	assetByID := make(map[uint]model.Asset, len(assets))
	for _, a := range assets {
		assetByID[a.ID] = a
	}

	list, err := h.assetHistoryRepo.ListRange(dari, sampai, exportRowLimit)
	if err != nil {
		return nil, nil, err
	}

	headers = []string{"Tanggal", "Nama/Label Aset", "Jenis Aset", "Gudang", "Event", "Keterangan", "Oleh"}
	for _, ev := range list {
		namaLabel, jenisAset, gudang := "-", "-", "-"
		if a, ok := assetByID[ev.AssetID]; ok {
			namaLabel = a.Nama
			if a.LabelRSD != "" {
				namaLabel = fmt.Sprintf("%s (%s)", a.Nama, a.LabelRSD)
			}
			jenisAset = orDash(a.JenisAset)
			if a.Gudang != nil {
				gudang = a.Gudang.Nama
			}
		}

		eventLabel := ev.EventType
		if label, ok := assetHistoryEventLabel[ev.EventType]; ok {
			eventLabel = label
		}

		var ketParts []string
		if ev.FieldLama != "" || ev.FieldBaru != "" {
			ketParts = append(ketParts, fmt.Sprintf("%s -> %s", orDash(ev.FieldLama), orDash(ev.FieldBaru)))
		}
		if ev.Catatan != "" {
			ketParts = append(ketParts, ev.Catatan)
		}
		keterangan := "-"
		if len(ketParts) > 0 {
			keterangan = strings.Join(ketParts, "; ")
		}

		rows = append(rows, []string{
			ev.CreatedAt.Format(dateFormat), namaLabel, jenisAset, gudang, eventLabel, keterangan, orDash(ev.UserNama),
		})
	}
	return headers, rows, nil
}

func computeGenericSummary(headers []string, rows [][]string) [][2]string {
	summary := [][2]string{{"Total Baris", strconv.Itoa(len(rows))}}

	for colIdx, header := range headers {
		lower := strings.ToLower(header)
		switch {
		case strings.Contains(lower, "nilai") || strings.Contains(lower, "harga") || strings.Contains(lower, "total"):
			sum := sumCurrencyColumn(rows, colIdx)
			summary = append(summary, [2]string{"Total " + header, "Rp " + formatRupiah(sum)})
		case strings.Contains(lower, "stok") || strings.Contains(lower, "kuantitas") || strings.Contains(lower, "qty"):
			sum := sumNumericColumn(rows, colIdx)
			summary = append(summary, [2]string{"Total " + header, strconv.FormatInt(sum, 10)})
		case strings.Contains(lower, "gudang"):
			count := countDistinctColumn(rows, colIdx)
			if count > 0 {
				summary = append(summary, [2]string{"Gudang Terlibat", strconv.Itoa(count)})
			}
		}
	}
	return summary
}

func sumCurrencyColumn(rows [][]string, colIdx int) int64 {
	var sum int64
	for _, row := range rows {
		if colIdx >= len(row) {
			continue
		}
		cleaned := strings.ReplaceAll(row[colIdx], ".", "")
		cleaned = strings.ReplaceAll(cleaned, ",", "")
		cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, "Rp"))
		if n, err := strconv.ParseInt(cleaned, 10, 64); err == nil {
			sum += n
		}
	}
	return sum
}

func sumNumericColumn(rows [][]string, colIdx int) int64 {
	var sum int64
	for _, row := range rows {
		if colIdx >= len(row) {
			continue
		}
		if n, err := strconv.ParseInt(strings.TrimSpace(row[colIdx]), 10, 64); err == nil {
			sum += n
		}
	}
	return sum
}

func countDistinctColumn(rows [][]string, colIdx int) int {
	distinct := map[string]struct{}{}
	for _, row := range rows {
		if colIdx < len(row) && row[colIdx] != "" && row[colIdx] != "-" {
			distinct[row[colIdx]] = struct{}{}
		}
	}
	return len(distinct)
}

func (h *Controller) Export(c *fiber.Ctx) error {
	tipe := c.Query("tipe", "")
	format := c.Query("format", "")
	if format != constant.FormatExcel && format != constant.FormatPDF && format != constant.FormatWord {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrLaporanFormatTidakDidukung, nil)
	}

	dari, sampai, err := parseDateRange(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	title, headers, rows, err := h.buildReport(tipe, dari, sampai)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == constant.ErrLaporanTipeTidakDidukung {
			status = fiber.StatusBadRequest
		}
		return utils.Fail(c, status, err.Error(), nil)
	}

	timestamp := time.Now().Format("20060102-150405")
	// Summary & chart selalu dihitung dari data MENTAH (sebelum
	// dikelompokkan) supaya angka totalnya tetap akurat apapun tampilan
	// tabel/export yang dipilih pengguna.
	summary := computeGenericSummary(headers, rows)
	granularity := normalizeGranularity(c.Query("granularitas", ""))
	chart := h.buildChart(tipe, headers, rows, granularity)
	// Tabel/file export mengikuti granularitas yang sama dengan chart:
	// Harian tetap 1 baris per transaksi (tanggal diformat ulang),
	// Bulanan/Tahunan dikelompokkan & dijumlahkan per periode.
	headers, rows = applyGranularityToRows(tipe, headers, rows, granularity)
	switch format {
	case constant.FormatExcel:
		data, err := reportexport.ToExcel(title, summary, headers, rows, toExportChart(chart))
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat file excel", nil)
		}
		c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s-%s.xlsx"`, tipe, timestamp))
		return c.Send(data)
	case constant.FormatPDF:
		data, err := reportexport.ToPDF(title, summary, headers, rows, toExportChart(chart))
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat file pdf", nil)
		}
		c.Set(fiber.HeaderContentType, "application/pdf")
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s-%s.pdf"`, tipe, timestamp))
		return c.Send(data)
	case constant.FormatWord:

		insight := ""
		if chart != nil {
			insight = "Total keseluruhan & rincian: " + chart.Insight()
		}
		data, err := reportexport.ToDocx(title, summary, headers, rows, toExportChart(chart), insight)
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat file docx", nil)
		}
		c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s-%s.docx"`, tipe, timestamp))
		return c.Send(data)
	}
	return utils.Fail(c, fiber.StatusBadRequest, constant.ErrLaporanFormatTidakDidukung, nil)
}

// CustomExportRequest adalah payload untuk fitur "Modifikasi Data": pengguna
// (biasanya atasan/PIC gudang) memilih sendiri baris mana saja dari hasil
// pratinjau laporan yang mau direkap/dicetak, alih-alih selalu mengekspor
// seluruh data. Frontend mengirim ulang header+baris yang sudah dia pilih
// (hasil dari GET /laporan/preview yang sama), lalu endpoint ini merender
// ulang jadi file export memakai builder yang sama persis dengan export biasa
// supaya format Excel/PDF/Word-nya konsisten.
type CustomExportRequest struct {
	Tipe    string     `json:"tipe"`
	Judul   string     `json:"judul"`
	Format  string     `json:"format"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

const customExportRowLimit = 20000

func (h *Controller) ExportCustom(c *fiber.Ctx) error {
	var req CustomExportRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "payload tidak valid", nil)
	}

	if req.Format != constant.FormatExcel && req.Format != constant.FormatPDF && req.Format != constant.FormatWord {
		return utils.Fail(c, fiber.StatusBadRequest, constant.ErrLaporanFormatTidakDidukung, nil)
	}
	if len(req.Headers) == 0 || len(req.Rows) == 0 {
		return utils.Fail(c, fiber.StatusBadRequest, "pilih minimal satu baris data untuk direkap", nil)
	}
	if len(req.Rows) > customExportRowLimit {
		return utils.Fail(c, fiber.StatusBadRequest, "terlalu banyak baris dipilih untuk sekali export", nil)
	}
	for i, row := range req.Rows {
		if len(row) != len(req.Headers) {
			return utils.Fail(c, fiber.StatusBadRequest, fmt.Sprintf("baris ke-%d tidak cocok dengan jumlah kolom", i+1), nil)
		}
	}

	title := req.Judul
	if title == "" {
		title = reportTitles[req.Tipe]
	}
	if title == "" {
		title = "Laporan (Data Terpilih)"
	}
	title += " (Data Terpilih)"

	summary := computeGenericSummary(req.Headers, req.Rows)
	timestamp := time.Now().Format("20060102-150405")
	safeTipe := req.Tipe
	if safeTipe == "" {
		safeTipe = "laporan"
	}

	switch req.Format {
	case constant.FormatExcel:
		data, err := reportexport.ToExcel(title, summary, req.Headers, req.Rows, nil)
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat file excel", nil)
		}
		c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s-terpilih-%s.xlsx"`, safeTipe, timestamp))
		return c.Send(data)
	case constant.FormatPDF:
		data, err := reportexport.ToPDF(title, summary, req.Headers, req.Rows, nil)
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat file pdf", nil)
		}
		c.Set(fiber.HeaderContentType, "application/pdf")
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s-terpilih-%s.pdf"`, safeTipe, timestamp))
		return c.Send(data)
	case constant.FormatWord:
		data, err := reportexport.ToDocx(title, summary, req.Headers, req.Rows, nil, "")
		if err != nil {
			return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat file docx", nil)
		}
		c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
		c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s-terpilih-%s.docx"`, safeTipe, timestamp))
		return c.Send(data)
	}
	return utils.Fail(c, fiber.StatusBadRequest, constant.ErrLaporanFormatTidakDidukung, nil)
}

func toExportChart(cd *ChartData) *reportexport.ChartData {
	if cd == nil {
		return nil
	}
	return &reportexport.ChartData{Title: cd.Title, Type: cd.Type, Labels: cd.Labels, Values: cd.Values}
}

func (h *Controller) Types(c *fiber.Ctx) error {
	types := make([]fiber.Map, 0, len(reportTitles))
	for key, label := range reportTitles {
		types = append(types, fiber.Map{"tipe": key, "label": label})
	}
	return utils.OK(c, "daftar tipe laporan berhasil diambil", types)
}

func (h *Controller) Preview(c *fiber.Ctx) error {
	tipe := c.Query("tipe", "")
	dari, sampai, err := parseDateRange(c)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error(), nil)
	}
	title, headers, rows, err := h.buildReport(tipe, dari, sampai)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == constant.ErrLaporanTipeTidakDidukung {
			status = fiber.StatusBadRequest
		}
		return utils.Fail(c, status, err.Error(), nil)
	}
	summaryPairs := computeGenericSummary(headers, rows)
	summary := make([]fiber.Map, 0, len(summaryPairs))
	for _, kv := range summaryPairs {
		summary = append(summary, fiber.Map{"label": kv[0], "value": kv[1]})
	}
	granularity := normalizeGranularity(c.Query("granularitas", ""))
	chart := h.buildChart(tipe, headers, rows, granularity)
	// Tabel pratinjau di layar mengikuti granularitas yang sama dengan
	// chart-nya (lihat applyGranularityToRows) — summary tetap dari data
	// mentah supaya totalnya akurat.
	viewHeaders, viewRows := applyGranularityToRows(tipe, headers, rows, granularity)
	return utils.OK(c, "pratinjau laporan berhasil diambil", fiber.Map{
		"title":        title,
		"headers":      viewHeaders,
		"rows":         viewRows,
		"summary":      summary,
		"chart":        chart,
		"granularitas": granularity,
	})
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/laporan", middleware.JWTAuth(h.jwtSvc))
	view := middleware.RequirePermission(h.roleRepo, Module, constant.ActionView)
	print := middleware.RequirePermission(h.roleRepo, Module, constant.ActionPrint)

	g.Get("/tipe", view, h.Types)
	g.Get("/preview", view, h.Preview)
	g.Get("/export", print, h.Export)
	g.Post("/export-custom", print, h.ExportCustom)
}
