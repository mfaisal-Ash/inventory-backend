package barang_rusak

import (
	barangRepo "github.com/projsonal/gowms/internal/repositories/barang"
	barangRusakRepo "github.com/projsonal/gowms/internal/repositories/barang_rusak"
	notificationRepo "github.com/projsonal/gowms/internal/repositories/notification"
	"github.com/projsonal/gowms/internal/repositories/role"
	"github.com/projsonal/gowms/pkg/constant"
	"github.com/projsonal/gowms/pkg/utils"
)

const Module = constant.ModuleBarangRusak

type Controller struct {
	repo        barangRusakRepo.Repository
	barangRepo  barangRepo.Repository
	roleRepo    role.Repository
	jwtSvc      *utils.JWTService
	storagePath string
	notifRepo   notificationRepo.Repository
}

func New(repo barangRusakRepo.Repository, barangRepo barangRepo.Repository, roleRepo role.Repository,
	jwtSvc *utils.JWTService, storagePath string, notifRepo notificationRepo.Repository) *Controller {
	return &Controller{
		repo: repo, barangRepo: barangRepo, roleRepo: roleRepo,
		jwtSvc: jwtSvc, storagePath: storagePath, notifRepo: notifRepo,
	}
}

// BarangRusakRequest dipakai untuk Create — barang_id opsional (barang
// rusak bisa dilaporkan tanpa terhubung ke katalog barang, mis. aset lain
// yang belum masuk modul Kelola Barang).
type BarangRusakRequest struct {
	BarangID    *uint  `json:"barang_id"`
	LabelBarang string `json:"label_barang" validate:"required,max=60"`
	NamaBarang  string `json:"nama_barang" validate:"required,max=150"`
	Keterangan  string `json:"keterangan" validate:"max=500"`
	JenisBarang string `json:"jenis_barang" validate:"omitempty,max=10"`
}

// UpdateStatusRequest dipakai admin/super_admin untuk memeriksa laporan
// barang rusak dan menentukan tindak lanjutnya.
type UpdateStatusRequest struct {
	Status     string `json:"status" validate:"required,oneof=pengecekan diperbaiki retur dibuang"`
	Keterangan string `json:"keterangan" validate:"max=500"`
}

type SummaryResponse struct {
	Pengecekan int64 `json:"pengecekan"`
	Diperbaiki int64 `json:"diperbaiki"`
	Retur      int64 `json:"retur"`
	Dibuang    int64 `json:"dibuang"`
	Total      int64 `json:"total"`
}
