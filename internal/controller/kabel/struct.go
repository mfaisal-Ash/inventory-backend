package kabel

import (
	kabelRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/kabel"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// Kabel adalah tab di halaman Kelola Barang, jadi izinnya (view/tambah/edit)
// mengikuti modul Kelola Barang — sebelumnya memakai modul Barang Keluar saat
// kabel masih punya menu sendiri. Tidak perlu seed ulang: role operasional
// bawaan sudah punya izin Kelola Barang.
const Module = constant.ModuleKelolaBarang

type Controller struct {
	repo     kabelRepo.Repository
	roleRepo role.Repository
	jwtSvc   *utils.JWTService
}

func New(repo kabelRepo.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{repo: repo, roleRepo: roleRepo, jwtSvc: jwtSvc}
}

type CreateRequest struct {
	BarangID    uint   `json:"barang_id" validate:"required"`
	GudangID    uint   `json:"gudang_id" validate:"required"`
	PanjangAwal int    `json:"panjang_awal" validate:"required,min=1,max=100000"`
	Catatan     string `json:"catatan" validate:"max=255"`
}

type PakaiRequest struct {
	Jumlah    int    `json:"jumlah" validate:"required,min=1,max=100000"`
	Tanggal   string `json:"tanggal" validate:"max=10"`
	Klien     string `json:"klien" validate:"max=150"`
	Keperluan string `json:"keperluan" validate:"max=255"`
}
