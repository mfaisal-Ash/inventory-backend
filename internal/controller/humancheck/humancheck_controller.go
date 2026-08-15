package humancheck

import (
	"github.com/gofiber/fiber/v2"

	"github.com/projsonal/gowms/pkg/utils"
)

// IssueResponse adalah payload token human-check yang dikirim ke klien.
type IssueResponse struct {
	Token string `json:"token"`
}

// Issue menerbitkan token human-check baru yang wajib disertakan klien
// pada permintaan berikutnya (mis. registrasi/login) untuk membuktikan
// interaksi manusia tanpa mengandalkan CAPTCHA pihak ketiga.
func (h *Controller) Issue(c *fiber.Ctx) error {
	token, err := h.svc.Issue()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat token verifikasi", nil)
	}
	return utils.OK(c, "token verifikasi berhasil dibuat", IssueResponse{Token: token})
}

// RegisterRoutes mendaftarkan endpoint publik (tanpa JWT) untuk human-check,
// karena token ini justru dipakai SEBELUM user terautentikasi (mis. saat
// registrasi atau login).
func (h *Controller) RegisterRoutes(router fiber.Router) {
	router.Get("/human-check", h.Issue)
}
