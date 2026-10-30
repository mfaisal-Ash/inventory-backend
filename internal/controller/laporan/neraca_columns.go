package laporan

import (
	"strconv"
	"strings"
	"time"

	neracaRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/neraca"
)

// Kolom neraca di tabel laporan.
//
// Metode neraca (rumus yang sama dengan menu Neraca Stok):
//
//	Saldo Akhir = Saldo Awal + Masuk - Keluar +/- Penyesuaian Opname
//
// "Keluar" sudah mencakup semua pengurang stok: Barang Keluar, pemakaian
// kabel, dan retur yang diganti. Saldo dijangkarkan ke stok sistem saat ini
// lalu dihitung mundur per periode laporan (dari/sampai), jadi angkanya
// selalu cocok dengan stok yang tampil di aplikasi.
const (
	colSaldoAwalNeraca  = "Saldo Awal (Neraca)"
	colSaldoAkhirNeraca = "Saldo Akhir (Neraca)"
)

// neracaTotals: jumlah neraca satu barang (seluruh gudang) atau satu barang di satu gudang.
type neracaTotals struct {
	SaldoAwal  int
	Masuk      int
	Keluar     int // semua pengurang: keluar + kabel + retur ganti
	Opname     int
	SaldoAkhir int
}

func (t *neracaTotals) add(r neracaRepo.Row) {
	t.SaldoAwal += r.SaldoAwal
	t.Masuk += r.Masuk
	t.Keluar += r.Keluar + r.PakaiKabel + r.ReturGanti
	t.Opname += r.Opname
	t.SaldoAkhir += r.SaldoAkhir
}

type neracaLookup struct {
	byKodeGudang map[string]neracaTotals
	byKode       map[string]neracaTotals
}

func neracaKodeKey(kode string) string { return strings.ToLower(strings.TrimSpace(kode)) }

func neracaPairKey(kode, gudang string) string {
	return neracaKodeKey(kode) + "|" + strings.ToLower(strings.TrimSpace(gudang))
}

func dayStartLocal(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// neracaPeriode menerjemahkan filter tanggal laporan ke rentang neraca.
// Tanpa "dari" = sejak awal pencatatan; tanpa "sampai" = hari ini.
func neracaPeriode(dari, sampai *time.Time) (time.Time, time.Time) {
	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	if dari != nil {
		from = dayStartLocal(*dari)
	}
	to := dayStartLocal(time.Now())
	if sampai != nil {
		to = dayStartLocal(*sampai)
	}
	if to.Before(from) {
		to = from
	}
	return from, to
}

func (h *Controller) loadNeraca(dari, sampai *time.Time) (*neracaLookup, error) {
	from, to := neracaPeriode(dari, sampai)
	res, err := h.neracaRepo.Hitung(neracaRepo.Filter{Dari: from, Sampai: to})
	if err != nil {
		return nil, err
	}
	lk := &neracaLookup{
		byKodeGudang: make(map[string]neracaTotals, len(res.Rows)),
		byKode:       make(map[string]neracaTotals, len(res.Rows)),
	}
	for _, r := range res.Rows {
		pair := neracaPairKey(r.KodeBarang, r.NamaGudang)
		t := lk.byKodeGudang[pair]
		t.add(r)
		lk.byKodeGudang[pair] = t

		kode := neracaKodeKey(r.KodeBarang)
		k := lk.byKode[kode]
		k.add(r)
		lk.byKode[kode] = k
	}
	return lk, nil
}

// find: neraca satu barang. Bila gudang diketahui memakai neraca barang di gudang itu,
// selain itu gabungan seluruh gudang. Tidak ada pergerakan/stok = semua 0.
func (lk *neracaLookup) find(kode, gudang string) neracaTotals {
	gudang = strings.TrimSpace(gudang)
	if gudang != "" && gudang != "-" {
		return lk.byKodeGudang[neracaPairKey(kode, gudang)]
	}
	return lk.byKode[neracaKodeKey(kode)]
}

func headerIndex(headers []string, names ...string) int {
	for _, name := range names {
		for i, h := range headers {
			if h == name {
				return i
			}
		}
	}
	return -1
}

// appendNeracaColumns menambah kolom Saldo Awal & Saldo Akhir (neraca) di akhir tabel laporan
// yang memuat kolom "Kode Barang" (dan "Gudang" bila ada). Laporan tanpa kode
// barang (mis. tracking aset) dikembalikan apa adanya.
func (h *Controller) appendNeracaColumns(headers []string, rows [][]string, dari, sampai *time.Time) ([]string, [][]string, error) {
	kodeIdx := headerIndex(headers, "Kode Barang", "Kode Barang (SKU)")
	if kodeIdx < 0 {
		return headers, rows, nil
	}
	gudangIdx := headerIndex(headers, "Gudang")
	lk, err := h.loadNeraca(dari, sampai)
	if err != nil {
		return nil, nil, err
	}
	newHeaders := append(append([]string{}, headers...), colSaldoAwalNeraca, colSaldoAkhirNeraca)
	newRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		kode := ""
		if kodeIdx < len(row) {
			kode = row[kodeIdx]
		}
		gudang := ""
		if gudangIdx >= 0 && gudangIdx < len(row) {
			gudang = row[gudangIdx]
		}
		awal, akhir := "-", "-"
		if kode != "" && kode != "-" {
			t := lk.find(kode, gudang)
			awal, akhir = strconv.Itoa(t.SaldoAwal), strconv.Itoa(t.SaldoAkhir)
		}
		newRows = append(newRows, append(append([]string{}, row...), awal, akhir))
	}
	return newHeaders, newRows, nil
}
