package laporan

import (
	assetRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset"
	assetHistoryRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/asset_history"
	barangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang"
	barangKeluarRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_keluar"
	barangMasukRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_masuk"
	barangRusakRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_rusak"
	barangSerialRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/barang_serial"
	pengajuanBarangRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_barang"
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
	pengajuanRepo    pengajuanBarangRepo.Repository
	assetRepo        assetRepo.Repository
	assetHistoryRepo assetHistoryRepo.Repository
	usersRepo        usersRepo.Repository
	roleRepo         role.Repository
	jwtSvc           *utils.JWTService
}

func New(barangRepo barangRepo.Repository, barangMasukRepo barangMasukRepo.Repository,
	barangKeluarRepo barangKeluarRepo.Repository, stockOpnameRepo stockOpnameRepo.Repository, barangRusakRepo barangRusakRepo.Repository,
	barangSerialRepo barangSerialRepo.Repository, pengajuanRepo pengajuanBarangRepo.Repository,
	assetRepo assetRepo.Repository, assetHistoryRepo assetHistoryRepo.Repository, usersRepo usersRepo.Repository,
	roleRepo role.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{
		barangRepo: barangRepo, barangMasukRepo: barangMasukRepo,
		barangKeluarRepo: barangKeluarRepo, stockOpnameRepo: stockOpnameRepo, barangRusakRepo: barangRusakRepo,
		barangSerialRepo: barangSerialRepo, pengajuanRepo: pengajuanRepo,
		assetRepo: assetRepo, assetHistoryRepo: assetHistoryRepo, usersRepo: usersRepo,
		roleRepo: roleRepo, jwtSvc: jwtSvc,
	}
}

func bigPagination() utils.PaginationParams {
	return utils.PaginationParams{Page: 1, Limit: exportRowLimit}
}
