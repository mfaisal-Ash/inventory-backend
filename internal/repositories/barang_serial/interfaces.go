package barang_serial

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	BarangID uint
	GudangID uint
	Status   string

	BarangMasukItemID  uint
	BarangKeluarItemID uint

	// Urutan, kalau diisi constant.UrutanFIFO, membuat List() mengurutkan hasil
	// dari unit yang paling lama tercatat (tanggal barang masuk paling awal) ke
	// yang paling baru, alih-alih urutan default (ID terbaru dulu).
	Urutan string
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.BarangSerial, int64, error)
	FindByID(id uint) (*model.BarangSerial, error)

	FindBySerial(serial string) (*model.BarangSerial, error)
	CountByBarang(barangID uint) (tersedia int64, terpasang int64, rusak int64, err error)

	Create(barangID, gudangID uint, serialNumber, catatan string) (*model.BarangSerial, error)

	RiwayatDokumen(s *model.BarangSerial) (nomorMasuk string, nomorKeluar string, err error)

	UpdateStatusManual(id uint, status string, catatan string) (*model.BarangSerial, error)
	// UpdateLokasi memindahkan unit ke gudang lain + ubah catatan (dipakai
	// modal "Ubah Unit" di menu Nomor Seri). Ini terpisah dari
	// UpdateStatusManual karena mengubah gudang, bukan status.
	UpdateLokasi(id uint, gudangID uint, catatan string) (*model.BarangSerial, error)
	Delete(id uint) error
}
