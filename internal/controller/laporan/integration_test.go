package laporan

// Tes integrasi Laporan Terpadu Gudang terhadap PostgreSQL sungguhan.
// Dilewati kecuali TEST_DATABASE_DSN diisi (database wajib berakhiran "_test"
// karena tes menghapus schema public).

import (
	"os"
	"regexp"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	assetRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset"
	assetHistoryRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset_history"
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	barangKeluarRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_keluar"
	barangMasukRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_masuk"
	barangRusakRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_rusak"
	barangSerialRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	neracaRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/neraca"
	pengajuanRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_barang"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	stockOpnameRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/stockOpname"
	usersRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/users"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestRingkasanGudang_Integrasi(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN tidak diisi — tes integrasi dilewati")
	}
	if !regexp.MustCompile(`dbname=\S*_test\b`).MatchString(dsn) {
		t.Fatal("nama database tes harus berakhiran _test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	must(t, err)
	must(t, db.Exec("DROP SCHEMA public CASCADE").Error)
	must(t, db.Exec("CREATE SCHEMA public").Error)
	must(t, config.AutoMigrate(db))

	role1 := model.Role{Name: "admin"}
	must(t, db.Create(&role1).Error)
	u := model.User{Username: "u", PasswordHash: "x", RoleID: role1.ID}
	must(t, db.Create(&u).Error)
	g := model.Gudang{Nama: "Gudang A", Kode: "GA", Tipe: "pusat"}
	must(t, db.Create(&g).Error)
	kat := model.Kategori{Nama: "Kabel", Singkatan: "KB"}
	sat := model.Satuan{Nama: "Pieces", Singkatan: "pcs"}
	must(t, db.Create(&kat).Error)
	must(t, db.Create(&sat).Error)
	// Dua barang: satu menipis (3 <= 10), satu habis (0).
	b1 := model.Barang{KodeBarang: "KB-001", Nama: "Router X", NamaGudang: "Gudang A", KategoriID: kat.ID, SatuanID: sat.ID,
		HargaBeli: 1000, StokMinimum: 10, Stok: 3, IsActive: true, ApprovalStatus: constant.ApprovalDisetujui}
	b2 := model.Barang{KodeBarang: "KB-002", Nama: "Switch Y", NamaGudang: "Gudang A", KategoriID: kat.ID, SatuanID: sat.ID,
		HargaBeli: 2000, StokMinimum: 5, Stok: 0, IsActive: true, ApprovalStatus: constant.ApprovalDisetujui}
	must(t, db.Create(&b1).Error)
	must(t, db.Create(&b2).Error)
	must(t, db.Create(&model.BarangStokGudang{BarangID: b1.ID, GudangID: g.ID, Stok: 3}).Error)
	must(t, db.Create(&model.BarangStokGudang{BarangID: b2.ID, GudangID: g.ID, Stok: 0}).Error)
	now := time.Now()
	must(t, db.Create(&model.BarangMasuk{NomorPenerimaan: "BM-1", GudangID: g.ID, Status: constant.StatusBMSelesai, Tanggal: now, CompletedAt: &now,
		Items: []model.BarangMasukItem{{BarangID: b1.ID, Qty: "3", HargaSatuan: 1000}}}).Error)
	must(t, db.Create(&model.PengajuanBarang{NomorPengajuan: "PJ-1", Jenis: constant.JenisPengajuanMasuk, GudangID: g.ID, Tanggal: now,
		Status: constant.StatusPengajuanDiajukan, DiajukanOleh: u.ID, Items: []model.PengajuanBarangItem{{BarangID: b1.ID, Qty: "20"}}}).Error)

	h := New(barangRepo.New(db), barangMasukRepo.New(db), barangKeluarRepo.New(db), stockOpnameRepo.New(db),
		barangRusakRepo.New(db), barangSerialRepo.New(db), pengajuanRepo.New(db), assetRepo.New(db),
		assetHistoryRepo.New(db), usersRepo.New(db), role.New(db), neracaRepo.New(db), nil)

	headers, rows, err := h.buildRingkasanGudang(nil, nil)
	if err != nil {
		t.Fatalf("buildRingkasanGudang: %v", err)
	}
	if len(headers) != 4 {
		t.Fatalf("headers = %v", headers)
	}
	find := func(bagian, indikator string) string {
		for _, r := range rows {
			if r[0] == bagian && r[1] == indikator {
				return r[2]
			}
		}
		t.Fatalf("baris %q/%q tidak ada", bagian, indikator)
		return ""
	}
	expect := map[[2]string]string{
		{bagianSorotan, "Jenis barang aktif"}:             "2",
		{bagianSorotan, "Total stok (unit)"}:              "3",
		{bagianSorotan, "Stok menipis"}:                   "2",
		{bagianSorotan, "Stok habis"}:                     "1",
		{bagianSorotan, "Pengajuan menunggu persetujuan"}: "1",
		{bagianMasuk, "Dokumen penerimaan"}:               "1",
		{bagianMasuk, "Total qty masuk"}:                  "3",
		{bagianPengajuan, "Total pengajuan (periode)"}:    "1",
	}
	for k, want := range expect {
		if got := find(k[0], k[1]); got != want {
			t.Errorf("%s / %s = %q, want %q", k[0], k[1], got, want)
		}
	}
	// Barang yang paling kritis (stok habis) harus muncul di daftar restock.
	adaHabis := false
	for _, r := range rows {
		if r[0] == bagianRestock && r[3] == "HABIS — prioritas utama" {
			adaHabis = true
		}
	}
	if !adaHabis {
		t.Error("daftar Perlu Restock harus menandai barang yang habis")
	}
	if cards := summaryFor("Ringkasan Gudang", headers, rows); len(cards) != 5 {
		t.Errorf("kartu ringkasan = %d, want 5", len(cards))
	}
}
