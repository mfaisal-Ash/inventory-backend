package barang

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
	"gorm.io/gorm"
)

func testBarangDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("membuka database test: %v", err)
	}
	if err := db.AutoMigrate(&model.Barang{}); err != nil {
		t.Fatalf("migrasi database test: %v", err)
	}
	return db
}

func TestApplyFilterKeepsApprovalConditionsGrouped(t *testing.T) {
	db := testBarangDB(t)
	rows := []model.Barang{
		{IDBarang: "ID-A", KodeBarang: "A-1", Nama: "Kabel", KategoriID: 1, SatuanID: 1, IsActive: true, ApprovalStatus: "menunggu", DiajukanOleh: uintPtr(7)},
		{IDBarang: "ID-B", KodeBarang: "B-1", Nama: "Kabel", KategoriID: 2, SatuanID: 1, IsActive: true, ApprovalStatus: "disetujui"},
		{IDBarang: "ID-C", KodeBarang: "C-1", Nama: "Kabel", KategoriID: 2, SatuanID: 1, IsActive: true, ApprovalStatus: "menunggu", DidelegasikanKe: uintPtr(7)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("menyimpan fixture: %v", err)
	}

	repo := &repository{db: db}
	list, _, err := repo.List(utils.PaginationParams{Page: 1, Limit: 20}, Filter{
		IDKategori: 1,
		HanyaAktif: true,
		Status:     []string{"disetujui"},
		SubmitBy:   7,
		DelegasiTo: 7,
	})
	if err != nil {
		t.Fatalf("mengambil daftar: %v", err)
	}
	if len(list) != 1 || list[0].KodeBarang != "A-1" {
		t.Fatalf("hasil filter tidak sesuai: %+v", list)
	}
}

func uintPtr(v uint) *uint { return &v }
