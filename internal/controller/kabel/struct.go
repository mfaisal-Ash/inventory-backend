package kabel

import (
	kabelRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/kabel"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// Kabel memakai izin modul Barang Keluar (view/tambah/edit): pemakaian kabel
// adalah barang keluar dalam satuan meter, jadi tidak perlu modul RBAC baru
// yang harus di-seed ulang ke role yang sudah ada.
const Module = constant.ModuleBarangKeluar

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
