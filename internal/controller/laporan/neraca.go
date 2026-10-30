package laporan

import (
	"strconv"
	"time"

	neracaRepoPkg "github.com/mfaisal-Ash/inventory-backend/internal/repositories/neraca"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

// Metode neraca stok:
//
//	Stok Akhir = Stok Awal + Barang Masuk - Barang Keluar
//
// Kolom neraca ini ditempelkan ke tabel semua menu laporan yang punya kolom
// Kode Barang, supaya angka stok di laporan selalu dihitung dengan rumus yang
// sama dengan menu Neraca Stok & detail produk di Kelola Barang.
//
// Penyesuaian di sisi dokumen (supaya rumus 3 suku tetap berlaku persis):
//   - Barang Masuk  = dokumen Barang Masuk selesai (+ selisih opname positif).
//   - Barang Keluar = dokumen Barang Keluar selesai + pemakaian kabel +
//     penggantian retur (+ selisih opname negatif).
//   - Stok Awal     = saldo sebelum periode dimulai (dihitung mundur dari stok
//     sistem, jadi stok awal tanpa dokumen ikut terhitung).
//
// Di laporan transaksi (Barang Masuk, Keluar, dst.) kolomnya diberi nama
// "Saldo ..." — sengaja TIDAK memakai kata "Stok" karena nilainya snapshot
// per periode yang berulang di setiap baris barang yang sama, jadi tidak
// boleh ikut dijumlahkan di baris total Excel/PDF/ringkasan. Hanya Laporan
// Stok Barang (satu baris = satu barang) memakai nama "Stok Awal/Masuk/...".
var neracaHeaders = []string{"Saldo Awal", "Saldo Masuk", "Saldo Keluar", "Saldo Akhir"}

const (
	headerKodeBarang    = "Kode Barang"
	headerKodeBarangSKU = "Kode Barang (SKU)"
	headerGudang        = "Gudang"
	headerPeriode       = "Periode"

	// Batas banyaknya periode berbeda yang dihitung per satu laporan (satu
	// periode = satu query neraca). Di atas batas ini sel neraca diisi "-"
	// supaya laporan harian dengan rentang sangat panjang tidak membebani DB.
	maxNeracaPeriods = 60
)

type neracaFigure struct {
	Awal, Masuk, Keluar, Akhir int
}

func (f *neracaFigure) add(o neracaFigure) {
	f.Awal += o.Awal
	f.Masuk += o.Masuk
	f.Keluar += o.Keluar
	f.Akhir += o.Akhir
}

func figureFromNeracaRow(r neracaRepoPkg.Row) neracaFigure {
	masuk := r.Masuk
	keluar := r.Keluar + r.PakaiKabel + r.ReturGanti
	if r.Opname >= 0 {
		masuk += r.Opname
	} else {
		keluar += -r.Opname
	}
	return neracaFigure{Awal: r.SaldoAwal, Masuk: masuk, Keluar: keluar, Akhir: r.SaldoAkhir}
}

func (f neracaFigure) cells() []string {
	return []string{strconv.Itoa(f.Awal), strconv.Itoa(f.Masuk), strconv.Itoa(f.Keluar), strconv.Itoa(f.Akhir)}
}

func dashCells() []string { return []string{"-", "-", "-", "-"} }

// neracaSnapshot: hasil neraca satu periode, diindeks per kode barang
// (gabungan semua gudang) dan per kode barang + nama gudang.
type neracaSnapshot struct {
	byBarang       map[string]neracaFigure
	byBarangGudang map[string]neracaFigure
}

func gudangKey(kode, gudang string) string { return kode + "\x00" + gudang }

func (s *neracaSnapshot) lookup(kode, gudang string) neracaFigure {
	if gudang != "" && gudang != "-" {
		if f, ok := s.byBarangGudang[gudangKey(kode, gudang)]; ok {
			return f
		}
		return neracaFigure{}
	}
	return s.byBarang[kode]
}

type neracaResolver struct {
	h     *Controller
	cache map[string]*neracaSnapshot
	calls int
}

func (h *Controller) newNeracaResolver() *neracaResolver {
	return &neracaResolver{h: h, cache: map[string]*neracaSnapshot{}}
}

// snapshot menghitung neraca untuk [dari, sampai] (keduanya inklusif, per hari).
// Mengembalikan nil bila repo belum terpasang, query gagal, atau batas jumlah
// periode terlampaui.
func (r *neracaResolver) snapshot(dari, sampai time.Time) *neracaSnapshot {
	key := dari.Format(dateFormat) + "|" + sampai.Format(dateFormat)
	if s, ok := r.cache[key]; ok {
		return s
	}
	if r.h.neracaRepo == nil || r.calls >= maxNeracaPeriods {
		r.cache[key] = nil
		return nil
	}
	r.calls++
	res, err := r.h.neracaRepo.Hitung(neracaRepoPkg.Filter{Dari: dari, Sampai: sampai})
	if err != nil {
		r.cache[key] = nil
		return nil
	}
	snap := &neracaSnapshot{byBarang: map[string]neracaFigure{}, byBarangGudang: map[string]neracaFigure{}}
	for _, row := range res.Rows {
		f := figureFromNeracaRow(row)
		snap.byBarangGudang[gudangKey(row.KodeBarang, row.NamaGudang)] = f
		agg := snap.byBarang[row.KodeBarang]
		agg.add(f)
		snap.byBarang[row.KodeBarang] = agg
	}
	r.cache[key] = snap
	return snap
}

func startOfLocalDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// rangeBounds: rentang neraca untuk seluruh laporan. Tanpa `dari` dipakai
// 2000-01-01 (seluruh riwayat), tanpa `sampai` dipakai hari ini.
func rangeBounds(dari, sampai *time.Time) (time.Time, time.Time) {
	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	if dari != nil {
		from = startOfLocalDay(*dari)
	}
	to := startOfLocalDay(time.Now())
	if sampai != nil {
		to = startOfLocalDay(*sampai)
	}
	if to.Before(from) {
		to = from
	}
	return from, to
}

// periodBounds: rentang neraca untuk satu periode laporan (hari/bulan/tahun).
func periodBounds(start time.Time, granularity string) (time.Time, time.Time) {
	start = startOfLocalDay(start)
	switch granularity {
	case GranularitasHarian:
		return start, start
	case GranularitasTahunan:
		first := time.Date(start.Year(), time.January, 1, 0, 0, 0, 0, time.Local)
		return first, time.Date(start.Year(), time.December, 31, 0, 0, 0, 0, time.Local)
	default:
		first := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.Local)
		return first, first.AddDate(0, 1, -1)
	}
}

func indexOfHeader(headers []string, names ...string) int {
	for _, name := range names {
		for i, h := range headers {
			if h == name {
				return i
			}
		}
	}
	return -1
}

// appendNeraca menempelkan kolom Stok Awal/Masuk/Keluar/Stok Akhir ke tabel
// laporan yang punya kolom Kode Barang. Periode tiap baris mengikuti
// granularitas tabel (harian = hari itu, bulanan = bulan itu, tahunan =
// tahun itu); baris tanpa tanggal memakai seluruh rentang laporan. Laporan
// tanpa kolom kode barang (mis. Tracking Aset) dikembalikan apa adanya.
func (h *Controller) appendNeraca(tipe string, headers []string, rows [][]string, periods []time.Time, granularity string, dari, sampai *time.Time) ([]string, [][]string) {
	if tipe == constant.LaporanStokBarang || tipe == constant.LaporanTrackingAset {
		return headers, rows
	}
	kodeIdx := indexOfHeader(headers, headerKodeBarang, headerKodeBarangSKU)
	if kodeIdx < 0 {
		return headers, rows
	}
	gudangIdx := -1
	if len(headers) == 0 || headers[0] != headerPeriode {
		gudangIdx = indexOfHeader(headers, headerGudang)
	}

	resolver := h.newNeracaResolver()
	fullFrom, fullTo := rangeBounds(dari, sampai)

	newHeaders := append(append([]string{}, headers...), neracaHeaders...)
	newRows := make([][]string, 0, len(rows))
	for i, row := range rows {
		kode := ""
		if kodeIdx < len(row) {
			kode = row[kodeIdx]
		}
		gudang := ""
		if gudangIdx >= 0 && gudangIdx < len(row) {
			gudang = row[gudangIdx]
		}

		cells := dashCells()
		if kode != "" && kode != "-" {
			from, to := fullFrom, fullTo
			if i < len(periods) && !periods[i].IsZero() {
				from, to = periodBounds(periods[i], granularity)
			}
			if snap := resolver.snapshot(from, to); snap != nil {
				cells = snap.lookup(kode, gudang).cells()
			}
		}
		newRows = append(newRows, append(append([]string{}, row...), cells...))
	}
	return newHeaders, newRows
}

// neracaPerBarang: neraca seluruh barang untuk rentang laporan, dijumlahkan
// lintas gudang (dipakai Laporan Stok Barang).
func (h *Controller) neracaPerBarang(dari, sampai *time.Time) map[string]neracaFigure {
	from, to := rangeBounds(dari, sampai)
	resolver := h.newNeracaResolver()
	snap := resolver.snapshot(from, to)
	if snap == nil {
		return map[string]neracaFigure{}
	}
	return snap.byBarang
}
