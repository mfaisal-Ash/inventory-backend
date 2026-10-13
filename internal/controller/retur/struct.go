package retur

import (
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	gudangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/gudang"
	notificationRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/notifikasi"
	returRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/retur"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// Retur memakai izin modul Barang Rusak (view/tambah/edit) — retur
// penggantian adalah kelanjutan alur kerusakan, jadi tidak perlu modul RBAC
// baru yang harus di-seed ulang ke role yang sudah ada.
const Module = constant.ModuleBarangRusak

type Controller struct {
	repo       returRepo.Repository
	barangRepo barangRepo.Repository
	gudangRepo gudangRepo.Repository
	roleRepo   role.Repository
	jwtSvc     *utils.JWTService
	notifRepo  notificationRepo.Repository
}

func New(repo returRepo.Repository, barangRepo barangRepo.Repository, gudangRepo gudangRepo.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService, notifRepo notificationRepo.Repository) *Controller {
	return &Controller{repo: repo, barangRepo: barangRepo, gudangRepo: gudangRepo, roleRepo: roleRepo, jwtSvc: jwtSvc, notifRepo: notifRepo}
}

type CreateRequest struct {
	BarangID    uint   `json:"barang_id" validate:"required"`
	GudangID    uint   `json:"gudang_id" validate:"required"`
	Klien       string `json:"klien" validate:"required,max=150"`
	AlamatKlien string `json:"alamat_klien" validate:"max=255"`
	SerialLama  string `json:"serial_lama" validate:"required,max=100"`
	Keluhan     string `json:"keluhan" validate:"required,max=500"`
}

type PengecekanRequest struct {
	Hasil   string `json:"hasil" validate:"required,oneof=perlu_ganti tidak_perlu_ganti"`
	Catatan string `json:"catatan" validate:"max=500"`
}

type GantiRequest struct {
	SerialBaru string `json:"serial_baru" validate:"required,max=100"`
}

type SummaryResponse struct {
	Dilaporkan      int64 `json:"dilaporkan"`
	PerluGanti      int64 `json:"perlu_ganti"`
	TidakPerluGanti int64 `json:"tidak_perlu_ganti"`
	Selesai         int64 `json:"selesai"`
	Total           int64 `json:"total"`
}
