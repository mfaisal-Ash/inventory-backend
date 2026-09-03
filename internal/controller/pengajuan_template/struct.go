package pengajuan_template

import (
	pengajuanTemplateRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/pengajuan_template"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

type Controller struct {
	repo   pengajuanTemplateRepo.Repository
	jwtSvc *utils.JWTService
}

func New(repo pengajuanTemplateRepo.Repository, jwtSvc *utils.JWTService) *Controller {
	return &Controller{repo: repo, jwtSvc: jwtSvc}
}
