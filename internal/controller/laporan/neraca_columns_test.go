package laporan

import (
	"testing"
	"time"

	neracaRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/neraca"
)

type fakeNeraca struct {
	rows []neracaRepo.Row
	got  neracaRepo.Filter
}

func (f *fakeNeraca) Hitung(filter neracaRepo.Filter) (*neracaRepo.Result, error) {
	f.got = filter
	return &neracaRepo.Result{Rows: f.rows}, nil
}

func newTestController(rows []neracaRepo.Row) (*Controller, *fakeNeraca) {
	f := &fakeNeraca{rows: rows}
	return &Controller{neracaRepo: f}, f
}

func TestAppendNeracaColumnsPerGudang(t *testing.T) {
	h, _ := newTestController([]neracaRepo.Row{
		{KodeBarang: "OP-001", NamaGudang: "Manggahang", SaldoAwal: 5, Masuk: 10, Keluar: 3, SaldoAkhir: 12},
		{KodeBarang: "OP-001", NamaGudang: "Wisma", SaldoAwal: 0, Masuk: 4, SaldoAkhir: 4},
	})
	headers := []string{"Nomor", "Gudang", "Kode Barang", "Qty"}
	rows := [][]string{
		{"BM-1", "Manggahang", "OP-001", "10"},
		{"BM-2", "wisma", "op-001", "4"},
		{"BM-3", "Manggahang", "-", "0"},
	}
	nh, nr, err := h.appendNeracaColumns(headers, rows, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if nh[len(nh)-2] != colSaldoAwalNeraca || nh[len(nh)-1] != colSaldoAkhirNeraca {
		t.Fatalf("header tidak sesuai: %v", nh)
	}
	want := [][2]string{{"5", "12"}, {"0", "4"}, {"-", "-"}}
	for i, w := range want {
		got := [2]string{nr[i][4], nr[i][5]}
		if got != w {
			t.Errorf("baris %d: got %v want %v", i, got, w)
		}
	}
}

func TestAppendNeracaColumnsWithoutGudangSumsAllWarehouses(t *testing.T) {
	h, _ := newTestController([]neracaRepo.Row{
		{KodeBarang: "OP-001", NamaGudang: "A", SaldoAwal: 5, SaldoAkhir: 12},
		{KodeBarang: "OP-001", NamaGudang: "B", SaldoAwal: 1, SaldoAkhir: 4},
	})
	_, nr, err := h.appendNeracaColumns([]string{"Kode Barang (SKU)"}, [][]string{{"OP-001"}, {"TIDAK-ADA"}}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if nr[0][1] != "6" || nr[0][2] != "16" {
		t.Errorf("gabungan gudang salah: %v", nr[0])
	}
	if nr[1][1] != "0" || nr[1][2] != "0" {
		t.Errorf("barang tanpa pergerakan harus 0: %v", nr[1])
	}
}

func TestAppendNeracaColumnsSkipsReportsWithoutKodeBarang(t *testing.T) {
	h, _ := newTestController(nil)
	headers := []string{"Tanggal", "Nama/Label Aset"}
	nh, nr, err := h.appendNeracaColumns(headers, [][]string{{"2026-10-01", "Tiang"}}, nil, nil)
	if err != nil || len(nh) != 2 || len(nr[0]) != 2 {
		t.Fatalf("laporan tanpa kode barang tidak boleh berubah: %v %v %v", nh, nr, err)
	}
}

func TestNeracaTotalsKeluarIncludesKabelDanRetur(t *testing.T) {
	var tot neracaTotals
	tot.add(neracaRepo.Row{Masuk: 10, Keluar: 2, PakaiKabel: 3, ReturGanti: 1, Opname: -1, SaldoAwal: 4, SaldoAkhir: 7})
	// 4 + 10 - 6 - 1 = 7
	if tot.Keluar != 6 || tot.SaldoAwal+tot.Masuk-tot.Keluar+tot.Opname != tot.SaldoAkhir {
		t.Errorf("rumus neraca tidak konsisten: %+v", tot)
	}
}

func TestNeracaPeriode(t *testing.T) {
	dari := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	sampai := time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)
	from, to := neracaPeriode(&dari, &sampai)
	if from.Day() != 1 || to.Day() != 31 || to.Month() != time.October {
		t.Errorf("periode salah: %v %v", from, to)
	}
	from2, to2 := neracaPeriode(nil, nil)
	if from2.Year() != 2000 || to2.Before(from2) {
		t.Errorf("default salah: %v %v", from2, to2)
	}
}
