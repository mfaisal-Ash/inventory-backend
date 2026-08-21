package barang_serial

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	barangSerialRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Controller struct {
	repo       barangSerialRepo.Repository
	barangRepo barangRepo.Repository
	roleRepo   role.Repository
	jwtSvc     *utils.JWTService
}

func New(repo barangSerialRepo.Repository, barangRepo barangRepo.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{repo: repo, barangRepo: barangRepo, roleRepo: roleRepo, jwtSvc: jwtSvc}
}

type UpdateStatusRequest struct {
	Status  string `json:"status" validate:"required,oneof=tersedia terpasang rusak"`
	Catatan string `json:"catatan" validate:"max=255"`
}

type CreateRequest struct {
	BarangID     uint   `json:"barang_id" validate:"required"`
	GudangID     uint   `json:"gudang_id" validate:"required"`
	RakID        *uint  `json:"rak_id"`
	SerialNumber string `json:"serial_number" validate:"required,max=100"`
	Catatan      string `json:"catatan" validate:"max=255"`
}

type RingkasanResponse struct {
	BarangID  uint  `json:"barang_id"`
	Tersedia  int64 `json:"tersedia"`
	Terpasang int64 `json:"terpasang"`
	Rusak     int64 `json:"rusak"`
}

type DetailResponse struct {
	*model.BarangSerial
	NomorBarangMasuk  string `json:"nomor_barang_masuk"`
	NomorBarangKeluar string `json:"nomor_barang_keluar"`
}
