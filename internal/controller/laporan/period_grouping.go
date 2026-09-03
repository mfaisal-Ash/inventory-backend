package laporan

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

var bulanIndonesia = [...]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// formatTanggalIndonesia: "1 September 2026" — dipakai untuk tampilan
// laporan Harian (per baris) supaya tanggalnya gampang dibaca (bukan ISO
// mentah "2026-09-01").
func formatTanggalIndonesia(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + bulanIndonesia[int(t.Month())-1] + " " + strconv.Itoa(t.Year())
}

// formatBulanIndonesia: "Agustus 2026" — dipakai label periode Bulanan,
// baik untuk chart maupun tabel/export laporan.
func formatBulanIndonesia(t time.Time) string {
	return bulanIndonesia[int(t.Month())-1] + " " + strconv.Itoa(t.Year())
}

// dateColumnCandidatesForTipe: kolom tanggal yang dipakai untuk
// pengelompokan Harian/Bulanan/Tahunan per tipe laporan — dijaga konsisten
// dengan buildChart supaya toggle granularitas yang sama juga berlaku ke
// tabel/export, bukan cuma chart-nya.
func dateColumnCandidatesForTipe(tipe string) []string {
	switch tipe {
	case constant.LaporanBarangKeluar, constant.LaporanBarangMasuk, constant.LaporanStokOpname, constant.LaporanPengajuanBarang, constant.LaporanTrackingAset:
		return []string{"Tanggal"}
	case constant.LaporanBarangRusak:
		return []string{"Tanggal Diperiksa"}
	default:
		return nil
	}
}

// isSummableHeaderForGrouping: kolom dianggap kuantitas nyata yang boleh
// DIJUMLAHKAN saat baris-baris digabung per periode (qty, stok, selisih,
// nilai/total uang) — sengaja TIDAK termasuk kolom "harga satuan"/"harga
// beli" (harga per unit) karena menjumlahkan harga satuan lintas baris
// tidak representatif seperti menjumlahkan quantity.
func isSummableHeaderForGrouping(header string) bool {
	lower := strings.ToLower(header)
	if strings.Contains(lower, "harga") {
		return false
	}
	keywords := []string{"qty", "stok", "kuantitas", "selisih", "nilai", "total"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func parseRowDate(value string) (time.Time, bool) {
	if t, err := time.Parse(dateFormat, value); err == nil {
		return t, true
	}
	if t, err := time.Parse("2 January 2006", value); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func parseAmount(value string) (int64, bool) {
	cleaned := strings.ReplaceAll(value, ".", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(strings.TrimPrefix(cleaned, "Rp"))
	n, err := strconv.ParseInt(cleaned, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// applyGranularityToRows: bentuk ulang headers/rows laporan generik sesuai
// granularitas yang dipilih pengguna — dipakai bareng Preview (tabel di
// layar) dan Export (Excel/PDF/Word) supaya konsisten:
//   - "harian": baris tetap 1:1, kolom tanggal cuma diformat ulang jadi
//     "1 September 2026" supaya gampang dibaca.
//   - "bulanan"/"tahunan": baris DIKELOMPOKKAN per periode (+ per kolom
//     kunci seperti Kode Barang/Gudang kalau ada), kolom kuantitas
//     dijumlahkan, label periode "Agustus 2026" / "2026".
//
// Kalau tipe laporan ini tidak punya kolom tanggal yang dikenal (mis.
// laporan snapshot stok), headers/rows dikembalikan apa adanya.
func applyGranularityToRows(tipe string, headers []string, rows [][]string, granularity string) ([]string, [][]string) {
	dateColCandidates := dateColumnCandidatesForTipe(tipe)
	dateColIdx := -1
	for _, candidate := range dateColCandidates {
		for i, h := range headers {
			if h == candidate {
				dateColIdx = i
				break
			}
		}
		if dateColIdx >= 0 {
			break
		}
	}
	if dateColIdx < 0 {
		return headers, rows
	}

	if granularity == GranularitasHarian {
		newRows := make([][]string, 0, len(rows))
		for _, row := range rows {
			newRow := append([]string{}, row...)
			if dateColIdx < len(newRow) {
				if t, ok := parseRowDate(newRow[dateColIdx]); ok {
					newRow[dateColIdx] = formatTanggalIndonesia(t)
				}
			}
			newRows = append(newRows, newRow)
		}
		return headers, newRows
	}

	// Bulanan/Tahunan: kelompokkan. Kunci pengelompokan tambahan di dalam 1
	// periode dipilih dari kolom paling spesifik yang tersedia, supaya
	// rekapnya tetap actionable per lokasi/item (bukan 1 angka gabungan
	// semua gudang & barang).
	groupColIdx := -1
	for _, candidate := range []string{"Kode Barang", "Gudang"} {
		for i, h := range headers {
			if h == candidate {
				groupColIdx = i
				break
			}
		}
		if groupColIdx >= 0 {
			break
		}
	}

	var sumCols []int
	for i, h := range headers {
		if isSummableHeaderForGrouping(h) {
			sumCols = append(sumCols, i)
		}
	}

	type bucket struct {
		sortKey  string
		periode  string
		groupVal string
		count    int
		sums     map[int]int64
	}
	buckets := map[string]*bucket{}
	order := make([]string, 0)

	for _, row := range rows {
		if dateColIdx >= len(row) {
			continue
		}
		t, ok := parseRowDate(row[dateColIdx])
		if !ok {
			continue
		}
		sortKey, periode := periodKey(t, granularity)
		groupVal := ""
		if groupColIdx >= 0 && groupColIdx < len(row) {
			groupVal = row[groupColIdx]
		}
		key := sortKey + "::" + groupVal
		b, exists := buckets[key]
		if !exists {
			b = &bucket{sortKey: sortKey, periode: periode, groupVal: groupVal, sums: map[int]int64{}}
			buckets[key] = b
			order = append(order, key)
		}
		b.count++
		for _, ci := range sumCols {
			if ci >= len(row) {
				continue
			}
			if n, ok := parseAmount(row[ci]); ok {
				b.sums[ci] += n
			}
		}
	}

	sort.Slice(order, func(i, j int) bool {
		bi, bj := buckets[order[i]], buckets[order[j]]
		if bi.sortKey != bj.sortKey {
			return bi.sortKey < bj.sortKey
		}
		return bi.groupVal < bj.groupVal
	})

	newHeaders := []string{"Periode"}
	if groupColIdx >= 0 {
		newHeaders = append(newHeaders, headers[groupColIdx])
	}
	newHeaders = append(newHeaders, "Jumlah Data")
	for _, ci := range sumCols {
		newHeaders = append(newHeaders, headers[ci])
	}

	newRows := make([][]string, 0, len(order))
	for _, key := range order {
		b := buckets[key]
		row := []string{b.periode}
		if groupColIdx >= 0 {
			row = append(row, b.groupVal)
		}
		row = append(row, strconv.Itoa(b.count))
		for _, ci := range sumCols {
			row = append(row, strconv.FormatInt(b.sums[ci], 10))
		}
		newRows = append(newRows, row)
	}
	return newHeaders, newRows
}
