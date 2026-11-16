package barangstokgudang

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"gorm.io/gorm"
)

func newStockTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("membuka database test: %v", err)
	}
	if err := db.AutoMigrate(&model.Barang{}, &model.BarangStokGudang{}); err != nil {
		t.Fatalf("migrasi database test: %v", err)
	}
	return db
}

func TestSyncBarangStokTotalTxMatchesWarehouseRows(t *testing.T) {
	db := newStockTestDB(t)
	barang := model.Barang{IDBarang: "ID-STOCK", KodeBarang: "STOCK-1", Nama: "Kabel", KategoriID: 1, SatuanID: 1, Stok: 99}
	if err := db.Create(&barang).Error; err != nil {
		t.Fatalf("menyimpan barang: %v", err)
	}
	rows := []model.BarangStokGudang{
		{BarangID: barang.ID, GudangID: 1, Stok: 4},
		{BarangID: barang.ID, GudangID: 2, Stok: 6},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("menyimpan stok gudang: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return SyncBarangStokTotalTx(tx, barang.ID)
	}); err != nil {
		t.Fatalf("sinkronisasi stok: %v", err)
	}

	var updated model.Barang
	if err := db.First(&updated, barang.ID).Error; err != nil {
		t.Fatalf("membaca barang: %v", err)
	}
	if updated.Stok != 10 {
		t.Fatalf("stok total = %d, yang diharapkan 10", updated.Stok)
	}
}

func TestSyncBarangStokTotalTxSetsZeroWhenNoWarehouseRows(t *testing.T) {
	db := newStockTestDB(t)
	barang := model.Barang{IDBarang: "ID-EMPTY", KodeBarang: "EMPTY-1", Nama: "Kabel", KategoriID: 1, SatuanID: 1, Stok: 8}
	if err := db.Create(&barang).Error; err != nil {
		t.Fatalf("menyimpan barang: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		return SyncBarangStokTotalTx(tx, barang.ID)
	}); err != nil {
		t.Fatalf("sinkronisasi stok: %v", err)
	}

	var updated model.Barang
	if err := db.First(&updated, barang.ID).Error; err != nil {
		t.Fatalf("membaca barang: %v", err)
	}
	if updated.Stok != 0 {
		t.Fatalf("stok total = %d, yang diharapkan 0", updated.Stok)
	}
}
