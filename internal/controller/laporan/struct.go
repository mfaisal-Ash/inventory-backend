package laporan

import (
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	barangKeluarRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_keluar"
	barangMasukRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_masuk"
	barangRusakRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_rusak"
	barangSerialRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	stockOpnameRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/stockOpname"
	usersRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/users"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const exportRowLimit = 20000

type Controller struct {
	barangRepo       barangRepo.Repository
	barangMasukRepo  barangMasukRepo.Repository
	barangKeluarRepo barangKeluarRepo.Repository
	stockOpnameRepo  stockOpnameRepo.Repository
	barangRusakRepo  barangRusakRepo.Repository
	barangSerialRepo barangSerialRepo.Repository
	usersRepo        usersRepo.Repository
	roleRepo         role.Repository
	jwtSvc           *utils.JWTService
}

func New(barangRepo barangRepo.Repository, barangMasukRepo barangMasukRepo.Repository,
	barangKeluarRepo barangKeluarRepo.Repository, stockOpnameRepo stockOpnameRepo.Repository, barangRusakRepo barangRusakRepo.Repository,
	barangSerialRepo barangSerialRepo.Repository, usersRepo usersRepo.Repository,
	roleRepo role.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{
		barangRepo: barangRepo, barangMasukRepo: barangMasukRepo,
		barangKeluarRepo: barangKeluarRepo, stockOpnameRepo: stockOpnameRepo, barangRusakRepo: barangRusakRepo,
		barangSerialRepo: barangSerialRepo, usersRepo: usersRepo,
		roleRepo: roleRepo, jwtSvc: jwtSvc,
	}
}

func bigPagination() utils.PaginationParams {
	return utils.PaginationParams{Page: 1, Limit: exportRowLimit}
}
