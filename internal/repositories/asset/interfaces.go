package asset

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Filter struct {
	JenisAset string
	GudangID  uint
	Status    string

	// Merek/Tipe: filter tambahan supaya daftar aset bisa dipersempit
	// berdasarkan merek/tipe aset (kolom milik Asset sendiri, bukan hasil
	// join ke Barang — lihat model.Asset.Merek/.Tipe).
	Merek string
	Tipe  string
}

type MapRow struct {
	ID              uint
	Nama            string
	JenisAset       string
	LabelRSD        string
	Latitude        float64
	Longitude       float64
	Status          string
	GudangID        uint
	GudangNama      string
	GudangKode      string
	GudangTipe      string
	GudangLatitude  *float64
	GudangLongitude *float64
	ParentAssetID   *uint
	ParentLatitude  *float64
	ParentLongitude *float64
	JumlahPort      int
	PortTerisi      int64

	Merek string
	Tipe  string

	KodeBarang string
}

type Repository interface {
	List(p utils.PaginationParams, f Filter) ([]model.Asset, int64, error)
	FindByID(id uint) (*model.Asset, error)

	ListForMap(f Filter, tipeGudang string) ([]MapRow, error)
	Create(a *model.Asset) error
	Update(a *model.Asset) error
	Delete(id uint) error

	NextRSDNumber(gudangID uint) (int, error)

	NextBANumber() (int, error)

	CountByJenis(jenisAset string) (int64, error)
}
