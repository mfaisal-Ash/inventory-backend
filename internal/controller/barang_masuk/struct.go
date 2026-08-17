package barang_masuk

import (
	"time"

	barangRepo "github.com/inventory-backend/internal/repositories/barang"
	bmRepo "github.com/inventory-backend/internal/repositories/barang_masuk"
	gudangRepo "github.com/inventory-backend/internal/repositories/gudang"
	notifikasiRepo "github.com/inventory-backend/internal/repositories/notifikasi"
	poRepo "github.com/inventory-backend/internal/repositories/po"
	"github.com/inventory-backend/internal/repositories/role"
	supplierRepo "github.com/inventory-backend/internal/repositories/supplier"
	"github.com/inventory-backend/pkg/utils"
)

type Controller struct {
	repo         bmRepo.Repository
	barangRepo   barangRepo.Repository
	gudangRepo   gudangRepo.Repository
	poRepo       poRepo.Repository
	supplierRepo supplierRepo.Repository
	roleRepo     role.Repository
	jwtSvc       *utils.JWTService
	notifRepo    notifikasiRepo.Repository
}

func New(repo bmRepo.Repository, barangRepo barangRepo.Repository, gudangRepo gudangRepo.Repository,
	poRepo poRepo.Repository, supplierRepo supplierRepo.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService,
	notifRepo notifikasiRepo.Repository) *Controller {
	return &Controller{
		repo: repo, barangRepo: barangRepo, gudangRepo: gudangRepo,
		poRepo: poRepo, supplierRepo: supplierRepo, roleRepo: roleRepo, jwtSvc: jwtSvc,
		notifRepo: notifRepo,
	}
}

type ItemRequest struct {
	BarangID    uint  `json:"barang_id" validate:"required"`
	RakID       *uint `json:"rak_id"`
	Qty         int   `json:"qty" validate:"required,min=1"`
	HargaSatuan int64 `json:"harga_satuan" validate:"min=0"`
}

type BMRequest struct {
	PurchaseOrderID *uint `json:"purchase_order_id"`
	SupplierID      *uint `json:"supplier_id"`
	GudangID        uint  `json:"gudang_id" validate:"required"`

	Tanggal string        `json:"tanggal" validate:"required"`
	Catatan string        `json:"catatan" validate:"max=255"`
	Items   []ItemRequest `json:"items" validate:"required,min=1,dive"`
}

func parseTanggalHarian(raw string) (time.Time, error) {
	return time.Parse("2006-01-02", raw)
}

type SummaryResponse struct {
	TotalDokumen int64 `json:"total_dokumen"`
	Draft        int64 `json:"draft"`
	Selesai      int64 `json:"selesai"`
}
