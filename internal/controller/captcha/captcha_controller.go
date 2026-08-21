package captcha

import (
	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func (h *Controller) GenerateCaptcha(c *fiber.Ctx) error {
	challenge, err := h.svc.Generate()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat captcha", nil)
	}
	return utils.OK(c, "captcha berhasil dibuat", challenge)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	router.Get("/captcha", h.GenerateCaptcha)
}
