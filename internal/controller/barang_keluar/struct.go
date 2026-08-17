package barang_keluar

import (
	"time"

	barangRepo "github.com/inventory-backend/internal/repositories/barang"
	bkRepo "github.com/inventory-backend/internal/repositories/barang_keluar"
	gudangRepo "github.com/inventory-backend/internal/repositories/gudang"
	notifikasiRepo "github.com/inventory-backend/internal/repositories/notifikasi"
	"github.com/inventory-backend/internal/repositories/role"
	"github.com/inventory-backend/pkg/utils"
)

type Controller struct {
	repo       bkRepo.Repository
	barangRepo barangRepo.Repository
	gudangRepo gudangRepo.Repository
	roleRepo   role.Repository
	jwtSvc     *utils.JWTService
	notifRepo  notifikasiRepo.Repository
}

func New(repo bkRepo.Repository, barangRepo barangRepo.Repository, gudangRepo gudangRepo.Repository,
	roleRepo role.Repository, jwtSvc *utils.JWTService, notifRepo notifikasiRepo.Repository) *Controller {
	return &Controller{repo: repo, barangRepo: barangRepo, gudangRepo: gudangRepo, roleRepo: roleRepo, jwtSvc: jwtSvc, notifRepo: notifRepo}
}

type ItemRequest struct {
	BarangID uint  `json:"barang_id" validate:"required"`
	RakID    *uint `json:"rak_id"`
	Qty      int   `json:"qty" validate:"required,min=1"`
}

type BKRequest struct {
	GudangID uint `json:"gudang_id" validate:"required"`

	Tanggal   string        `json:"tanggal" validate:"required"`
	Keperluan string        `json:"keperluan" validate:"required,max=255"`
	Penerima  string        `json:"penerima" validate:"max=150"`
	Items     []ItemRequest `json:"items" validate:"required,min=1,dive"`
}

func parseTanggalHarian(raw string) (time.Time, error) {
	return time.Parse("2006-01-02", raw)
}

type SummaryResponse struct {
	TotalDokumen int64 `json:"total_dokumen"`
	Draft        int64 `json:"draft"`
	Selesai      int64 `json:"selesai"`
}
