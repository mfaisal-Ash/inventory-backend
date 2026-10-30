package laporan

import (
	"testing"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func TestDistinctDocs_SatuDokumenBanyakBaris(t *testing.T) {
	rows := [][]string{
		{"BM-1", "Selesai"}, {"BM-1", "Selesai"}, {"BM-2", "Draft"}, {"-", "Draft"},
	}
	total, per := distinctDocs(rows, 0, 1)
	if total != 2 || per["Selesai"] != 1 || per["Draft"] != 1 {
		t.Fatalf("dokumen harus dihitung unik: total=%d per=%v", total, per)
	}
}

func TestSummaryFor_TerpaduMemakaiSorotan(t *testing.T) {
	rows := [][]string{
		{bagianSorotan, "Stok menipis", "3", "-"},
		{bagianInventaris, "Total stok (unit)", "100", "-"},
		{bagianSorotan, "Stok habis", "1", "-"},
	}
	got := summaryFor(constant.LaporanRingkasanGudang, ringkasanGudangHeaders, rows)
	if len(got) != 2 || got[0] != [2]string{"Stok menipis", "3"} || got[1] != [2]string{"Stok habis", "1"} {
		t.Fatalf("hanya baris Sorotan yang jadi kartu ringkasan, dapat: %v", got)
	}
	if g := summaryFor(constant.LaporanBarangMasuk, []string{"Qty"}, [][]string{{"5"}}); len(g) < 2 {
		t.Fatalf("laporan lain tetap memakai ringkasan generik, dapat: %v", g)
	}
}

func TestRingkasanGudangTerdaftar(t *testing.T) {
	if _, ok := reportTitles[constant.LaporanRingkasanGudang]; !ok {
		t.Fatal("tipe laporan terpadu harus terdaftar di reportTitles")
	}
}
