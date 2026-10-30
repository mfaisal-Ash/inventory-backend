package laporan

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

// Laporan Terpadu Gudang: satu laporan yang merangkum SEMUA laporan lain
// (inventaris, barang masuk/keluar, pengajuan, barang rusak, stock opname,
// tracking aset, FIFO/FEFO) supaya kondisi gudang bisa dibaca sekali lihat.
//
// Datanya tidak dihitung ulang dari nol: tiap bagian memakai builder laporan
// yang sama dengan laporan tunggalnya, jadi angkanya pasti konsisten dengan
// laporan detail yang bisa dibuka dari tiap bagian.

const (
	bagianSorotan    = "Sorotan"
	bagianInfo       = "Informasi"
	bagianInventaris = "Inventaris"
	bagianRestock    = "Perlu Restock"
	bagianPengajuan  = "Pengajuan Barang"
	bagianMasuk      = "Barang Masuk"
	bagianKeluar     = "Barang Keluar"
	bagianRusak      = "Barang Rusak"
	bagianOpname     = "Stock Opname"
	bagianAset       = "Tracking Aset"
	bagianFifo       = "FIFO/FEFO"

	maxRestockRows = 15
)

var ringkasanGudangHeaders = []string{"Bagian", "Indikator", "Nilai", "Keterangan"}

type ringkasanRows struct{ rows [][]string }

func (r *ringkasanRows) add(bagian, indikator string, nilai interface{}, ket string) {
	if strings.TrimSpace(ket) == "" {
		ket = "-"
	}
	r.rows = append(r.rows, []string{bagian, indikator, fmt.Sprint(nilai), ket})
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func atoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// distinctDocs menghitung dokumen unik (satu dokumen bisa punya banyak baris
// item) dan memecahnya per nilai kolom groupIdx (mis. Status atau Jenis).
func distinctDocs(rows [][]string, docIdx, groupIdx int) (total int, perGroup map[string]int) {
	perGroup = map[string]int{}
	seen := map[string]struct{}{}
	for _, row := range rows {
		doc := cell(row, docIdx)
		if doc == "" || doc == "-" {
			continue
		}
		if _, dup := seen[doc]; dup {
			continue
		}
		seen[doc] = struct{}{}
		total++
		if groupIdx >= 0 {
			g := cell(row, groupIdx)
			if g == "" {
				g = "-"
			}
			perGroup[g]++
		}
	}
	return total, perGroup
}

func countByColumn(rows [][]string, idx int) map[string]int {
	out := map[string]int{}
	for _, row := range rows {
		v := cell(row, idx)
		if v == "" {
			v = "-"
		}
		out[v]++
	}
	return out
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func periodeLabel(dari, sampai *time.Time) string {
	switch {
	case dari == nil && sampai == nil:
		return "Semua tanggal"
	case dari != nil && sampai != nil:
		return dari.Format(dateFormat) + " s/d " + sampai.Format(dateFormat)
	case dari != nil:
		return "Sejak " + dari.Format(dateFormat)
	default:
		return "Sampai " + sampai.Format(dateFormat)
	}
}

func (h *Controller) buildRingkasanGudang(dari, sampai *time.Time) (headers []string, rows [][]string, err error) {
	var body, sorotan ringkasanRows

	// ── Inventaris + daftar barang yang perlu restock ────────────────────────
	sh, sr, err := h.buildStokBarang(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	iKode, iNama := headerIndex(sh, "Kode Barang"), headerIndex(sh, "Nama")
	iStok, iMin, iStatus := headerIndex(sh, "Stok"), headerIndex(sh, "Stok Minimum"), headerIndex(sh, "Status")

	type lowItem struct {
		label      string
		stok, min  int
		ratioScore float64
	}
	var aktif, totalStok, menipis, habis int
	var low []lowItem
	for _, row := range sr {
		if cell(row, iStatus) != "Aktif" {
			continue
		}
		aktif++
		stok, min := atoi(cell(row, iStok)), atoi(cell(row, iMin))
		totalStok += stok
		if stok == 0 {
			habis++
		}
		if min > 0 && stok <= min {
			menipis++
			low = append(low, lowItem{
				label: cell(row, iKode) + " — " + cell(row, iNama),
				stok:  stok, min: min, ratioScore: float64(stok) / float64(min),
			})
		}
	}
	nilai, nilaiErr := h.barangRepo.SumNilaiInventaris()

	body.add(bagianInfo, "Periode laporan", periodeLabel(dari, sampai), "Mutasi stok dihitung pada periode ini; posisi stok = kondisi terkini")
	body.add(bagianInventaris, "Jenis barang aktif", aktif, "")
	body.add(bagianInventaris, "Total stok (unit)", totalStok, "")
	if nilaiErr == nil {
		body.add(bagianInventaris, "Nilai inventaris (Rp)", nilai, "Stok x harga beli")
	}
	body.add(bagianInventaris, "Stok menipis", menipis, "Stok <= stok minimum")
	body.add(bagianInventaris, "Stok habis", habis, "")
	for _, col := range []string{"Saldo Awal", "Masuk", "Keluar", "Penyesuaian Opname", "Saldo Akhir"} {
		if idx := headerIndex(sh, col); idx >= 0 {
			body.add(bagianInventaris, col+" (periode)", sumNumericColumn(sr, idx), "Total semua barang")
		}
	}

	sort.SliceStable(low, func(i, j int) bool { return low[i].ratioScore < low[j].ratioScore })
	for i, it := range low {
		if i >= maxRestockRows {
			body.add(bagianRestock, "Barang lain yang menipis", len(low)-maxRestockRows,
				"Lihat Ringkasan Stok untuk daftar lengkap")
			break
		}
		ket := "Menipis — segera ajukan barang masuk"
		if it.stok == 0 {
			ket = "HABIS — prioritas utama"
		}
		body.add(bagianRestock, it.label, fmt.Sprintf("stok %d / min %d", it.stok, it.min), ket)
	}

	// ── Pengajuan barang ─────────────────────────────────────────────────────
	ph, pr, err := h.buildPengajuanBarang(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	pDoc, pStatus, pJenis, pQty := headerIndex(ph, "Nomor Pengajuan"), headerIndex(ph, "Status"), headerIndex(ph, "Jenis"), headerIndex(ph, "Qty")
	pTotal, perStatus := distinctDocs(pr, pDoc, pStatus)
	_, perJenis := distinctDocs(pr, pDoc, pJenis)
	body.add(bagianPengajuan, "Total pengajuan (periode)", pTotal, "")
	for _, k := range sortedKeys(perStatus) {
		body.add(bagianPengajuan, "Status: "+k, perStatus[k], "")
	}
	for _, k := range sortedKeys(perJenis) {
		body.add(bagianPengajuan, "Jenis: "+k, perJenis[k], "")
	}
	body.add(bagianPengajuan, "Total qty diajukan", sumNumericColumn(pr, pQty), "")
	menunggu, menungguErr := h.pengajuanRepo.CountByStatus(constant.StatusPengajuanDiajukan)

	// ── Barang masuk ─────────────────────────────────────────────────────────
	mh, mr, err := h.buildBarangMasuk(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	mTotal, mStatus := distinctDocs(mr, headerIndex(mh, "Nomor Penerimaan"), headerIndex(mh, "Status"))
	body.add(bagianMasuk, "Dokumen penerimaan", mTotal, "")
	for _, k := range sortedKeys(mStatus) {
		body.add(bagianMasuk, "Status: "+k, mStatus[k], "")
	}
	body.add(bagianMasuk, "Total qty masuk", sumNumericColumn(mr, headerIndex(mh, "Qty")), "")

	// ── Barang keluar ────────────────────────────────────────────────────────
	kh, kr, err := h.buildBarangKeluar(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	kTotal, kStatus := distinctDocs(kr, headerIndex(kh, "Nomor Pengeluaran"), headerIndex(kh, "Status"))
	body.add(bagianKeluar, "Dokumen pengeluaran", kTotal, "")
	for _, k := range sortedKeys(kStatus) {
		body.add(bagianKeluar, "Status: "+k, kStatus[k], "")
	}
	body.add(bagianKeluar, "Total qty keluar", sumNumericColumn(kr, headerIndex(kh, "Qty")), "")

	// ── Barang rusak ─────────────────────────────────────────────────────────
	rh, rr, err := h.buildBarangRusak(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	body.add(bagianRusak, "Laporan barang rusak", len(rr), "")
	rStatus := countByColumn(rr, headerIndex(rh, "Status"))
	for _, k := range sortedKeys(rStatus) {
		body.add(bagianRusak, "Status: "+k, rStatus[k], "")
	}

	// ── Stock opname ─────────────────────────────────────────────────────────
	oh, or, err := h.buildStockOpname(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	oTotal, _ := distinctDocs(or, headerIndex(oh, "Nomor Opname"), -1)
	iSel := headerIndex(oh, "Selisih")
	beda := 0
	for _, row := range or {
		if atoi(cell(row, iSel)) != 0 {
			beda++
		}
	}
	body.add(bagianOpname, "Sesi opname", oTotal, "")
	body.add(bagianOpname, "Item dengan selisih", beda, "Stok fisik berbeda dari stok sistem")
	body.add(bagianOpname, "Total selisih bersih", sumNumericColumn(or, iSel), "Negatif = fisik kurang dari sistem")

	// ── Tracking aset ────────────────────────────────────────────────────────
	ah, ar, err := h.buildLaporanTrackingAset(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	body.add(bagianAset, "Peristiwa tercatat", len(ar), "")
	body.add(bagianAset, "Aset terlibat", countDistinctColumn(ar, headerIndex(ah, "Nama/Label Aset")), "")
	aEvent := countByColumn(ar, headerIndex(ah, "Event"))
	for _, k := range sortedKeys(aEvent) {
		body.add(bagianAset, "Event: "+k, aEvent[k], "")
	}

	// ── FIFO/FEFO ────────────────────────────────────────────────────────────
	fh, fr, err := h.buildFifoFefo(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	fRek := countByColumn(fr, headerIndex(fh, "Rekomendasi"))
	body.add(bagianFifo, "Batch barang masuk terlacak", len(fr), "")
	body.add(bagianFifo, "Harus dikeluarkan duluan", fRek["Keluarkan Duluan (FIFO)"], "Batch tertua yang masih ada sisa")
	body.add(bagianFifo, "Menunggu giliran", fRek["Menunggu Giliran"], "")

	// ── Sorotan (kartu ringkasan di atas) ───────────────────────────────────
	sorotan.add(bagianSorotan, "Jenis barang aktif", aktif, "")
	sorotan.add(bagianSorotan, "Total stok (unit)", totalStok, "")
	sorotan.add(bagianSorotan, "Stok menipis", menipis, "Perlu pengajuan barang masuk")
	sorotan.add(bagianSorotan, "Stok habis", habis, "")
	if menungguErr == nil {
		sorotan.add(bagianSorotan, "Pengajuan menunggu persetujuan", menunggu, "Semua periode")
	}

	return ringkasanGudangHeaders, append(sorotan.rows, body.rows...), nil
}

// summaryFor memilih kartu ringkasan sesuai tipe laporan: laporan terpadu
// memakai baris "Sorotan"-nya, laporan lain memakai ringkasan generik.
func summaryFor(tipe string, headers []string, rows [][]string) [][2]string {
	if tipe != constant.LaporanRingkasanGudang {
		return computeGenericSummary(headers, rows)
	}
	out := make([][2]string, 0, 6)
	for _, row := range rows {
		if len(row) >= 3 && row[0] == bagianSorotan {
			out = append(out, [2]string{row[1], row[2]})
		}
	}
	return out
}
