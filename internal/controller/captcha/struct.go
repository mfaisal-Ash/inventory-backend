package captcha

import "github.com/mfaisal-Ash/inventory-backend/pkg/captcha"

type Controller struct {
	svc *captcha.Service
}

func New(svc *captcha.Service) *Controller {
	return &Controller{svc: svc}
}
