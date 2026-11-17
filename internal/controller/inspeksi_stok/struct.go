package inspeksi_stok

import (
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/inspeksi_stok"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Controller struct {
	repo     inspeksi_stok.Repository
	roleRepo role.Repository
	jwtSvc   *utils.JWTService
}

func New(repo inspeksi_stok.Repository, roleRepo role.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{repo: repo, roleRepo: roleRepo, jwtSvc: jwtSvc}
}

type CreateRequest struct {
	BarangID          uint   `json:"barang_id" validate:"required"`
	GudangID          uint   `json:"gudang_id" validate:"required"`
	LabelBatch        string `json:"label_batch" validate:"max=100"`
	DiperiksaQty      int64  `json:"diperiksa_qty" validate:"min=1"`
	NormalQty         int64  `json:"normal_qty" validate:"min=0"`
	RusakQty          int64  `json:"rusak_qty" validate:"min=0"`
	BelumDiperiksaQty int64  `json:"belum_diperiksa_qty" validate:"min=0"`
	Catatan           string `json:"catatan" validate:"max=500"`
}
