package humancheck

import "github.com/mfaisal-Ash/inventory-backend/pkg/humancheck"

type Controller struct {
	svc *humancheck.Service
}

func New(svc *humancheck.Service) *Controller {
	return &Controller{svc: svc}
}

type IssueResponse struct {
	Token string `json:"human_check_token"`
}
